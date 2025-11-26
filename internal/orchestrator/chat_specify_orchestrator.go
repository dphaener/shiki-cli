package orchestrator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	agentpkg "github.com/darinhaener/collab/internal/agent"
	"github.com/darinhaener/collab/internal/broker"
	"github.com/darinhaener/collab/internal/chatservice"
	"github.com/darinhaener/collab/internal/conversation"
	"github.com/darinhaener/collab/pkg/types"
)

// ChatSpecifyOrchestrator manages the specification workflow using the new chat infrastructure.
// It provides the same API as SpecifyOrchestrator for backwards compatibility.
type ChatSpecifyOrchestrator struct {
	// Legacy session (for backwards compatibility)
	legacySession *types.SpecifySession

	// New chat infrastructure
	session    *conversation.Session
	broker     *broker.Broker
	messageSvc chatservice.MessageService
	sessionSvc chatservice.SessionService

	// Agent
	agent  *agentpkg.Agent
	apiKey string

	// Context management
	ctx    context.Context
	cancel context.CancelFunc

	// Current turn tracking
	currentTurn int
}

// NewChatSpecifyOrchestrator creates a new specify orchestrator with the new chat infrastructure.
func NewChatSpecifyOrchestrator(
	legacySession *types.SpecifySession,
	apiKey string,
	eventBroker *broker.Broker,
	messageSvc chatservice.MessageService,
	sessionSvc chatservice.SessionService,
) *ChatSpecifyOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	return &ChatSpecifyOrchestrator{
		legacySession: legacySession,
		broker:        eventBroker,
		messageSvc:    messageSvc,
		sessionSvc:    sessionSvc,
		apiKey:        apiKey,
		ctx:           ctx,
		cancel:        cancel,
		currentTurn:   0,
	}
}

// Initialize sets up the agent and creates a conversation session.
func (o *ChatSpecifyOrchestrator) Initialize() error {
	// Create a conversation session using the new infrastructure
	opts := chatservice.CreateSessionOpts{
		Title:       o.legacySession.FriendlyName,
		FeatureID:   o.legacySession.ID,
		FeatureName: o.legacySession.FriendlyName,
	}

	session, err := o.sessionSvc.Create(o.ctx, conversation.ModeSpecify, opts)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	o.session = session

	// Add user as a participant
	userParticipant := conversation.NewUserParticipant("User")
	o.session.AddParticipant(userParticipant)

	// Create and add the specify agent as a participant
	agentCfg := agentpkg.NewSpecifyAgent(o.legacySession)
	agentParticipant := conversation.NewAgentParticipant(
		agentCfg.Name,
		agentCfg.Role,
		agentCfg.SystemPrompt,
		agentCfg.Model,
	)
	o.session.AddParticipant(agentParticipant)

	// Create agent instance
	o.agent = agentpkg.NewAgent(agentCfg)

	// Start agent (built-in tools only)
	if err := o.agent.Start(o.ctx, o.apiKey, agentCfg.ID); err != nil {
		return fmt.Errorf("failed to start specify agent: %w", err)
	}

	// Start the session
	if err := o.sessionSvc.Start(o.ctx, o.session.ID); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	return nil
}

// SendMessage sends a user message and returns streaming updates.
// This maintains the same API as SpecifyOrchestrator for backwards compatibility.
func (o *ChatSpecifyOrchestrator) SendMessage(userMessage string) (<-chan MessageUpdate, <-chan error) {
	updateChan := make(chan MessageUpdate, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(updateChan)
		defer close(errorChan)

		client := o.agent.GetClient()
		if client == nil {
			errorChan <- fmt.Errorf("agent not initialized")
			return
		}

		// Increment turn
		o.currentTurn++

		// Get user participant ID
		userID := ""
		agentID := ""
		for _, p := range o.session.Participants {
			if p.Type == conversation.ParticipantUser {
				userID = p.ID
			} else if p.Type == conversation.ParticipantAgent {
				agentID = p.ID
			}
		}

		// Create user message
		userMsg := conversation.NewMessage(o.session.ID, userID, conversation.RoleUser, o.currentTurn)
		userMsg.AddText(userMessage)
		if err := o.messageSvc.Create(o.ctx, o.session.ID, userMsg); err != nil {
			errorChan <- fmt.Errorf("failed to create user message: %w", err)
			return
		}

		// Start assistant streaming message
		assistantMsg, err := o.messageSvc.StartStream(o.ctx, o.session.ID, agentID, o.currentTurn)
		if err != nil {
			errorChan <- fmt.Errorf("failed to start stream: %w", err)
			return
		}

		// Emit turn started event
		if o.broker != nil {
			o.broker.Publish(broker.NewTurnStartedEvent(o.session.ID, agentID, o.currentTurn))
		}

		// Retry loop for handling timeouts
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				// Brief delay before retry
				time.Sleep(retryDelay)
			}

			// Create query context with timeout for this attempt
			queryCtx, cancel := context.WithTimeout(o.ctx, 3*time.Minute)

			// Send query to agent
			if err := client.Query(queryCtx, userMessage); err != nil {
				cancel()
				o.messageSvc.FinishStream(o.ctx, assistantMsg.ID, conversation.FinishReasonError, 0, 0)
				errorChan <- fmt.Errorf("failed to send query: %w", err)
				return
			}

			// Receive messages
			msgChan, errChan := client.ReceiveMessages(queryCtx)

			var responseBuilder strings.Builder
			var inputTokens, outputTokens, toolCalls int
			completed := false
			shouldRetry := false

		messageLoop:
			for {
				select {
				case <-queryCtx.Done():
					if queryCtx.Err() != nil {
						// Send interrupt to stop the current operation and clean up
						if interruptErr := client.Interrupt(context.Background()); interruptErr != nil {
							_ = interruptErr
						}
						// Check if we should retry
						if attempt < maxRetries {
							shouldRetry = true
							break messageLoop
						}
						// No more retries, finish with error
						o.messageSvc.FinishStream(o.ctx, assistantMsg.ID, conversation.FinishReasonError, inputTokens, outputTokens)
						errorChan <- fmt.Errorf("query timeout after %d retries: %w", maxRetries, queryCtx.Err())
					}
					cancel()
					return

				case err := <-errChan:
					if err != nil {
						cancel()
						o.messageSvc.FinishStream(o.ctx, assistantMsg.ID, conversation.FinishReasonError, inputTokens, outputTokens)
						errorChan <- err
						return
					}

				case msg, ok := <-msgChan:
					if !ok || msg == nil {
						// Channel closed, send completion signal
						o.finishMessage(assistantMsg.ID, responseBuilder.String(), inputTokens, outputTokens, toolCalls)
						if responseBuilder.Len() > 0 {
							updateChan <- MessageUpdate{
								Type:    "complete",
								Content: responseBuilder.String(),
							}
						}
						completed = true
						break messageLoop
					}

					// Process message based on type
					msgType := msg.Type()

					switch msgType {
					case "assistant":
						if assistantSDKMsg, ok := msg.(*claude.SDKAssistantMessage); ok {
							// Process each content block
							for _, block := range assistantSDKMsg.Message.Content {
								switch content := block.(type) {
								case claude.TextContentBlock:
									if content.Text != "" {
										responseBuilder.WriteString(content.Text)

										// Append to streaming message
										o.messageSvc.AppendDelta(o.ctx, assistantMsg.ID, content.Text)

										// Send legacy update
										updateChan <- MessageUpdate{
											Type:    "text",
											Content: content.Text,
										}
									}

								case claude.ToolUseContentBlock:
									toolCalls++
									displayName := strings.TrimPrefix(content.Name, "mcp__")

									// Add tool call part to message
									toolCallPart := conversation.NewToolCallPart(content.ID, displayName, nil)
									o.messageSvc.AddPart(o.ctx, assistantMsg.ID, toolCallPart)

									// Send legacy update
									updateChan <- MessageUpdate{
										Type:    "tool_use",
										Content: displayName,
									}
								}
							}

							// Extract token usage
							usage := assistantSDKMsg.Message.Usage
							if usage.InputTokens > 0 || usage.OutputTokens > 0 {
								inputTokens = usage.InputTokens
								outputTokens = usage.OutputTokens
							}
						}

					case "result":
						// Result message signals completion
						if resultMsg, ok := msg.(*claude.SDKResultMessage); ok {
							usage := resultMsg.Usage
							if usage.InputTokens > 0 || usage.OutputTokens > 0 {
								inputTokens = usage.InputTokens
								outputTokens = usage.OutputTokens
							}
						}

						o.finishMessage(assistantMsg.ID, responseBuilder.String(), inputTokens, outputTokens, toolCalls)
						if responseBuilder.Len() > 0 {
							updateChan <- MessageUpdate{
								Type:    "complete",
								Content: responseBuilder.String(),
							}
						}
						completed = true
						break messageLoop
					}
				}
			}

			cancel()

			if completed {
				return
			}

			if !shouldRetry {
				return
			}
		}
	}()

	return updateChan, errorChan
}

// finishMessage completes a streaming message and emits events.
func (o *ChatSpecifyOrchestrator) finishMessage(messageID, content string, inputTokens, outputTokens, toolCalls int) {
	finishReason := conversation.FinishReasonStop
	if toolCalls > 0 {
		finishReason = conversation.FinishReasonToolUse
	}

	o.messageSvc.FinishStream(o.ctx, messageID, finishReason, inputTokens, outputTokens)

	// Emit turn completed event
	if o.broker != nil {
		agentID := ""
		for _, p := range o.session.Participants {
			if p.Type == conversation.ParticipantAgent {
				agentID = p.ID
				break
			}
		}

		cost := agentpkg.CalculateCostDetailed(inputTokens, outputTokens, o.agent.Model)
		o.broker.Publish(broker.NewTurnCompletedEvent(
			o.session.ID,
			agentID,
			o.currentTurn,
			inputTokens,
			outputTokens,
			cost,
			toolCalls,
			0, // duration - would need to track
		))
	}
}

// GetSession returns the conversation session.
func (o *ChatSpecifyOrchestrator) GetSession() *conversation.Session {
	return o.session
}

// GetMessages returns all messages for the session.
func (o *ChatSpecifyOrchestrator) GetMessages() ([]*conversation.Message, error) {
	return o.messageSvc.List(o.ctx, o.session.ID, chatservice.ListMessagesOpts{Order: "asc"})
}

// Stop gracefully shuts down the orchestrator.
func (o *ChatSpecifyOrchestrator) Stop() error {
	o.cancel()

	// Complete the session
	if o.session != nil && o.sessionSvc != nil {
		o.sessionSvc.Pause(context.Background(), o.session.ID)
	}

	if o.agent != nil {
		return o.agent.Stop()
	}

	return nil
}

// GetAPIKeyFromEnv gets the API key from environment.
func GetAPIKeyFromEnv() string {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return ""
	}
	return apiKey
}
