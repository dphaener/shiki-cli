package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	agentpkg "github.com/darinhaener/collab/internal/agent"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// TasksOrchestrator manages the tasks generation workflow with a single agent
type TasksOrchestrator struct {
	session           *types.WorkflowSession
	agent             *agentpkg.Agent
	apiKey            string
	ctx               context.Context
	cancel            context.CancelFunc
	tasksInstructions string // Instructions sent as first message to prime the agent
	instructionsSent  bool   // Whether we've sent the initial instructions
	eventBus          *events.EventBus // Event bus for publishing assistant events
}

// NewTasksOrchestrator creates a new tasks orchestrator
func NewTasksOrchestrator(session *types.WorkflowSession, apiKey string, eventBus *events.EventBus) *TasksOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	return &TasksOrchestrator{
		session:  session,
		apiKey:   apiKey,
		ctx:      ctx,
		cancel:   cancel,
		eventBus: eventBus,
	}
}

// Initialize sets up the agent
func (o *TasksOrchestrator) Initialize() error {
	// Create debug logger for initialization
	logger := newDebugLogger()
	defer logger.close()

	logger.log("========== INITIALIZING TASKS AGENT ==========")
	logger.log("Session ID: %s", o.session.ID)
	logger.log("Feature: %s (#%03d)", o.session.FriendlyName, o.session.FeatureNumber)
	logger.log("Slug: %s", o.session.Slug)
	logger.log("SpecFile: %s", o.session.SpecFile)
	logger.log("PlanFile: %s", o.session.PlanFile)
	logger.log("TasksFile: %s", o.session.TasksFile)

	// Create tasks agent configuration - this generates the instructions
	agentCfg := agentpkg.NewTasksAgent(o.session)

	// Store instructions to send as first message
	o.tasksInstructions = agentCfg.SystemPrompt
	o.instructionsSent = false

	logger.log("========== TASKS INSTRUCTIONS ==========")
	logger.log("%s", o.tasksInstructions)
	logger.log("========== END TASKS INSTRUCTIONS ==========")

	// Create agent instance
	o.agent = agentpkg.NewAgent(agentCfg)

	// Start agent (built-in tools only)
	if err := o.agent.Start(o.ctx, o.apiKey, agentCfg.ID); err != nil {
		return fmt.Errorf("failed to start tasks agent: %w", err)
	}

	logger.log("Agent started successfully")
	return nil
}

// SendMessage sends a user message to the agent and returns streaming updates
func (o *TasksOrchestrator) SendMessage(userMessage string) (<-chan MessageUpdate, <-chan error) {
	updateChan := make(chan MessageUpdate, 100)
	errorChan := make(chan error, 1)

	go func() {
		// Create debug logger
		logger := newDebugLogger()
		defer logger.close()

		logger.log("========== NEW MESSAGE ==========")
		logger.log("User message: %s", userMessage)

		defer close(updateChan)
		defer close(errorChan)

		client := o.agent.GetClient()
		if client == nil {
			logger.log("ERROR: agent not initialized")
			errorChan <- fmt.Errorf("agent not initialized")
			return
		}

		// Build the message to send
		messageToSend := userMessage

		// If this is the first message, prepend the tasks instructions
		if !o.instructionsSent && o.tasksInstructions != "" {
			logger.log("Prepending tasks instructions to first message")
			messageToSend = o.tasksInstructions + "\n\n---\n\n## User Request\n\n" + userMessage
			o.instructionsSent = true
		}

		// Retry loop for handling timeouts
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				logger.log("Retry attempt %d/%d after timeout", attempt, maxRetries)
				time.Sleep(retryDelay)
			}

			// Create query context with timeout for this attempt
			queryCtx, cancel := context.WithTimeout(o.ctx, 5*time.Minute) // Longer timeout for tasks

			// Send query to agent
			logger.log("Sending query to agent (message length: %d chars)...", len(messageToSend))
			if err := client.Query(queryCtx, messageToSend); err != nil {
				cancel()
				logger.log("ERROR: failed to send query: %v", err)

				// Publish assistant error event if eventBus is available
				if o.eventBus != nil {
					errorEvent := events.NewAssistantError(
						o.session.ID,
						"tasks",
						o.session.ID,
						"query_failed",
						err.Error(),
						"Failed to send query to tasks agent",
						"abort",
						map[string]interface{}{
							"feature_number": o.session.FeatureNumber,
							"slug":          o.session.Slug,
						},
					)
					o.eventBus.Publish(errorEvent)
				}

				errorChan <- fmt.Errorf("failed to send query: %w", err)
				return
			}
			logger.log("Query sent successfully")

			// Publish assistant request event if eventBus is available
			if o.eventBus != nil {
				assistantEvent := events.NewAssistantRequest(
					o.session.ID,
					"tasks",
					o.session.ID,
					o.session.ID,
					messageToSend,
					map[string]interface{}{
						"feature_number": o.session.FeatureNumber,
						"slug":          o.session.Slug,
						"message_length": len(messageToSend),
					},
				)
				o.eventBus.Publish(assistantEvent)
			}

			// Receive messages
			msgChan, errChan := client.ReceiveMessages(queryCtx)
			logger.log("Waiting for messages...")

			var responseBuilder strings.Builder
			var currentMessageID string
			msgCount := 0

			pendingTools := make(map[string]struct {
				Name string
				Args string
			})

			completed := false
			shouldRetry := false

		messageLoop:
			for {
				select {
				case <-queryCtx.Done():
					logger.log("Context done: %v", queryCtx.Err())
					if queryCtx.Err() != nil {
						logger.log("Sending interrupt to clean up after timeout...")
						interruptClient(client, logger)
						if attempt < maxRetries {
							logger.log("Will retry after timeout (attempt %d/%d)", attempt+1, maxRetries)
							shouldRetry = true
							break messageLoop
						}
						errorChan <- fmt.Errorf("query timeout after %d retries: %w", maxRetries, queryCtx.Err())
					}
					cancel()
					return

				case err, ok := <-errChan:
					logger.log("Error channel: ok=%v, err=%v", ok, err)
					if !ok {
						logger.log("Error channel closed, removing from select...")
						errChan = nil
						continue
					}
					if err != nil {
						logger.log("Actual error received, forwarding and returning")
						cancel()
						errorChan <- err
						return
					}
					logger.log("Nil error received, continuing...")

				case msg, ok := <-msgChan:
					msgCount++
					logger.log("--- Message #%d ---", msgCount)
					logger.log("Channel ok=%v, msg==nil: %v", ok, msg == nil)

					if !ok {
						logger.log("Channel closed (ok=false), sending completion")
						if responseBuilder.Len() > 0 {
							finalResponse := responseBuilder.String()
							updateChan <- MessageUpdate{
								Type:    "complete",
								Content: finalResponse,
							}

							// Publish assistant response event if eventBus is available
							if o.eventBus != nil {
								responseEvent := events.NewAssistantResponse(
									o.session.ID,
									"tasks",
									o.session.ID,
									o.session.ID,
									finalResponse,
									0, // Token count not available here
									0, // Timing not available here
									"", // Model not available here
									map[string]interface{}{
										"feature_number": o.session.FeatureNumber,
										"slug":          o.session.Slug,
										"response_length": len(finalResponse),
									},
								)
								o.eventBus.Publish(responseEvent)
							}
						}
						completed = true
						break messageLoop
					}

					if msg == nil {
						logger.log("Skipping nil message")
						continue
					}

					msgType := msg.Type()
					logger.log("Message type: %s", msgType)

					if rawJSON, err := json.MarshalIndent(msg, "", "  "); err == nil {
						logger.log("Raw message JSON:\n%s", string(rawJSON))
					}

					switch msgType {
					case "assistant":
						logger.log("Processing assistant message")

						if assistantMsg, ok := msg.(*claude.SDKAssistantMessage); ok {
							messageID := assistantMsg.Message.ID
							logger.log("Assistant message ID: %s (current: %s)", messageID, currentMessageID)

							if currentMessageID != "" && messageID != currentMessageID {
								logger.log("New message detected (ID changed), completing previous message")
								if responseBuilder.Len() > 0 {
									updateChan <- MessageUpdate{
										Type:    "message_done",
										Content: responseBuilder.String(),
									}
									responseBuilder.Reset()
								}
							}
							currentMessageID = messageID

							for i, block := range assistantMsg.Message.Content {
								logger.log("Content block %d type: %T", i, block)
								switch content := block.(type) {
								case claude.TextContentBlock:
									logger.log("Text block content (len=%d)", len(content.Text))
									if content.Text != "" {
										responseBuilder.WriteString(content.Text)
										updateChan <- MessageUpdate{
											Type:    "text",
											Content: content.Text,
										}
									}
								case claude.ToolUseContentBlock:
									logger.log("Tool use block: %s (ID: %s)", content.Name, content.ID)

									var argsStr string
									if len(content.Input) > 0 {
										argsStr = string(content.Input)
									}

									pendingTools[content.ID] = struct {
										Name string
										Args string
									}{
										Name: content.Name,
										Args: argsStr,
									}

									updateChan <- MessageUpdate{
										Type:    "tool_use",
										Content: content.Name,
									}
								}
							}
						}

					case "result":
						logger.log("Processing result message - COMPLETING")
						if responseBuilder.Len() > 0 {
							updateChan <- MessageUpdate{
								Type:    "complete",
								Content: responseBuilder.String(),
							}
						}
						completed = true
						break messageLoop

					case "system":
						logger.log("System message received, continuing...")

					case "user":
						logger.log("User message (tool result) received")

						if userMsg, ok := msg.(*claude.SDKUserMessage); ok {
							for _, block := range userMsg.Message.Content {
								if toolResult, ok := block.(claude.ToolResultContentBlock); ok {
									var contentStr string
									if toolResult.Content != nil {
										if toolResult.Content.Text != nil {
											contentStr = *toolResult.Content.Text
										} else if len(toolResult.Content.Blocks) > 0 {
											for _, b := range toolResult.Content.Blocks {
												if textBlock, ok := b.(claude.TextContentBlock); ok {
													contentStr += textBlock.Text
												}
											}
										}
									}

									if toolResult.IsError {
										errorMsg := contentStr
										if toolInfo, ok := pendingTools[toolResult.ToolUseID]; ok {
											errorMsg = fmt.Sprintf("%s\nTool: %s\nArgs: %s",
												contentStr, toolInfo.Name, toolInfo.Args)
										}
										updateChan <- MessageUpdate{
											Type:    "tool_error",
											Content: errorMsg,
										}
									}
									delete(pendingTools, toolResult.ToolUseID)
								}
							}
						}

					default:
						logger.log("Unknown message type: %s, continuing...", msgType)
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

// Stop gracefully shuts down the orchestrator
func (o *TasksOrchestrator) Stop() error {
	o.cancel()

	if o.agent != nil {
		return o.agent.Stop()
	}

	return nil
}
