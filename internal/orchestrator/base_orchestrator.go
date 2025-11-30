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

// BaseOrchestrator provides the common implementation for all phase orchestrators.
// It handles agent communication, message streaming, retry logic, and event publishing.
// Specific orchestrators embed this and provide session-specific configuration.
type BaseOrchestrator struct {
	// Agent management
	agent  *agentpkg.Agent
	apiKey string
	ctx    context.Context
	cancel context.CancelFunc

	// Instructions management
	instructions     string
	instructionsSent bool

	// Configuration
	sessionID     string
	phaseType     string        // "plan", "tasks", "implement", "specify", "bug"
	timeout       time.Duration // per-phase timeout
	eventMetadata map[string]interface{}

	// Dependencies
	eventBus *events.EventBus
	logger   *debugLogger
}

// NewBaseOrchestrator creates a new base orchestrator with the given configuration.
func NewBaseOrchestrator(
	sessionID string,
	phaseType string,
	timeout time.Duration,
	apiKey string,
	eventBus *events.EventBus,
) *BaseOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	return &BaseOrchestrator{
		apiKey:        apiKey,
		ctx:           ctx,
		cancel:        cancel,
		sessionID:     sessionID,
		phaseType:     phaseType,
		timeout:       timeout,
		eventBus:      eventBus,
		eventMetadata: make(map[string]interface{}),
	}
}

// SetInstructions sets the system prompt/instructions to be injected in the first message.
func (o *BaseOrchestrator) SetInstructions(instructions string) {
	o.instructions = instructions
	o.instructionsSent = false
}

// SetEventMetadata sets additional metadata to include in published events.
func (o *BaseOrchestrator) SetEventMetadata(metadata map[string]interface{}) {
	o.eventMetadata = metadata
}

// InitializeWithAgent initializes the orchestrator with a pre-configured agent.
// The agent config should include the system prompt as instructions.
func (o *BaseOrchestrator) InitializeWithAgent(agentCfg *types.Agent) error {
	o.logger = newDebugLogger()

	o.logger.log("========== INITIALIZING %s AGENT ==========", strings.ToUpper(o.phaseType))
	o.logger.log("Session ID: %s", o.sessionID)
	o.logger.log("Phase Type: %s", o.phaseType)
	o.logger.log("Timeout: %v", o.timeout)

	// Store instructions to send as first message
	o.instructions = agentCfg.SystemPrompt
	o.instructionsSent = false

	o.logger.log("========== %s INSTRUCTIONS ==========", strings.ToUpper(o.phaseType))
	o.logger.log("%s", o.instructions)
	o.logger.log("========== END %s INSTRUCTIONS ==========", strings.ToUpper(o.phaseType))

	// Create agent instance
	o.agent = agentpkg.NewAgent(agentCfg)

	// Start agent (built-in tools only)
	if err := o.agent.Start(o.ctx, o.apiKey, agentCfg.ID); err != nil {
		return fmt.Errorf("failed to start %s agent: %w", o.phaseType, err)
	}

	o.logger.log("Agent started successfully")
	return nil
}

// SendMessage sends a user message to the agent and returns streaming updates.
// This is the unified implementation that replaces the duplicated code in each orchestrator.
func (o *BaseOrchestrator) SendMessage(userMessage string) (<-chan MessageUpdate, <-chan error) {
	updateChan := make(chan MessageUpdate, 100)
	errorChan := make(chan error, 1)

	go func() {
		logger := o.logger
		if logger == nil {
			logger = newDebugLogger()
			defer logger.close()
		}

		logger.log("========== NEW MESSAGE ==========")
		logger.log("User message: %s", userMessage)

		defer close(updateChan)
		defer close(errorChan)

		// Build the message to send
		messageToSend := userMessage

		// If this is the first message, prepend the instructions
		if !o.instructionsSent && o.instructions != "" {
			logger.log("Prepending %s instructions to first message", o.phaseType)
			messageToSend = o.instructions + "\n\n---\n\n## User Request\n\n" + userMessage
			o.instructionsSent = true
		}

		// Track last error for recovery decisions
		var lastError error

		// Retry loop for handling timeouts and broken pipes
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				logger.log("Retry attempt %d/%d", attempt, maxRetries)

				// Restart client on recoverable errors (broken pipes, etc.) or if client is nil
				if agentpkg.IsRecoverableError(lastError) || o.agent.GetClient() == nil {
					logger.log("Attempting client recreation due to: %v", lastError)
					// Use a fresh background context - the orchestrator context may be cancelled
					if err := o.agent.Restart(context.Background()); err != nil {
						logger.log("Client recreation failed: %v", err)
						// Continue anyway - next query will fail if still broken
					} else {
						logger.log("Client recreated successfully")
					}
				}

				time.Sleep(retryDelay)
			}

			// Get client inside the loop (may have been recreated)
			client := o.agent.GetClient()
			if client == nil {
				logger.log("ERROR: agent client is nil, will retry")
				lastError = fmt.Errorf("agent client is nil")
				if attempt < maxRetries {
					continue
				}
				errorChan <- fmt.Errorf("agent not initialized after %d retries", maxRetries)
				return
			}

			// Create query context with timeout for this attempt
			queryCtx, cancel := context.WithTimeout(o.ctx, o.timeout)

			// Send query to agent
			logger.log("Sending query to agent (message length: %d chars)...", len(messageToSend))
			if err := client.Query(queryCtx, messageToSend); err != nil {
				cancel()
				logger.log("ERROR: failed to send query: %v", err)
				lastError = err

				// Check if this is a recoverable error (broken pipe, etc.)
				if agentpkg.IsRecoverableError(err) && attempt < maxRetries {
					logger.log("Recoverable error detected, will retry with client restart")
					continue
				}

				// Publish error event
				if o.eventBus != nil {
					metadata := o.buildEventMetadata(map[string]interface{}{
						"attempt":     attempt + 1,
						"max_retries": maxRetries,
					})
					errorEvent := events.NewAssistantError(
						o.sessionID,
						o.phaseType,
						o.sessionID,
						"query_failed",
						err.Error(),
						fmt.Sprintf("Failed to send query to agent (attempt %d)", attempt+1),
						"retry",
						metadata,
					)
					o.eventBus.Publish(errorEvent)
				}

				errorChan <- fmt.Errorf("failed to send query: %w", err)
				return
			}
			logger.log("Query sent successfully")

			// Publish assistant request event
			if o.eventBus != nil {
				metadata := o.buildEventMetadata(map[string]interface{}{
					"attempt":        attempt + 1,
					"message_length": len(messageToSend),
				})
				assistantEvent := events.NewAssistantRequest(
					o.sessionID,
					o.phaseType,
					o.sessionID,
					o.sessionID,
					messageToSend,
					metadata,
				)
				o.eventBus.Publish(assistantEvent)
			}

			// Receive messages
			msgChan, errChan := client.ReceiveMessages(queryCtx)
			logger.log("Waiting for messages...")

			var responseBuilder strings.Builder
			var currentMessageID string
			msgCount := 0

			// Track pending tool uses
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
						// Publish timeout error event
						if o.eventBus != nil {
							metadata := o.buildEventMetadata(map[string]interface{}{
								"max_retries": maxRetries,
							})
							timeoutEvent := events.NewAssistantError(
								o.sessionID,
								o.phaseType,
								o.sessionID,
								"timeout",
								queryCtx.Err().Error(),
								fmt.Sprintf("Query timed out after %d retries", maxRetries),
								"abort",
								metadata,
							)
							o.eventBus.Publish(timeoutEvent)
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

							// Publish response event
							if o.eventBus != nil {
								metadata := o.buildEventMetadata(map[string]interface{}{
									"response_length": len(finalResponse),
								})
								responseEvent := events.NewAssistantResponse(
									o.sessionID,
									o.phaseType,
									o.sessionID,
									o.sessionID,
									finalResponse,
									0, 0, "",
									metadata,
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

					// Capture session ID for resume capability
					if sessionIDProvider, ok := msg.(interface{ SessionID() string }); ok {
						if sid := sessionIDProvider.SessionID(); sid != "" {
							o.agent.SetSessionID(sid)
						}
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
							logger.log("Assistant message has %d content blocks", len(assistantMsg.Message.Content))

							// Check if this is a NEW message (different ID)
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

							// Publish metadata event for token usage
							if o.eventBus != nil && assistantMsg.Message.Usage.OutputTokens > 0 {
								metadata := o.buildEventMetadata(map[string]interface{}{
									"input_tokens":  assistantMsg.Message.Usage.InputTokens,
									"output_tokens": assistantMsg.Message.Usage.OutputTokens,
									"message_id":    messageID,
								})
								metadataEvent := events.NewAssistantMetadata(
									o.sessionID,
									o.phaseType,
									o.sessionID,
									int(assistantMsg.Message.Usage.InputTokens+assistantMsg.Message.Usage.OutputTokens),
									0, 0, "", nil,
									metadata,
								)
								o.eventBus.Publish(metadataEvent)
							}

							// Process each content block
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
									var argsMap map[string]interface{}
									if len(content.Input) > 0 {
										argsStr = string(content.Input)
										_ = json.Unmarshal(content.Input, &argsMap)
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
										Args:    argsMap,
									}
								default:
									logger.log("Unknown content block type: %T", block)
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

									logger.log("Tool result - ID: %s, IsError: %v",
										toolResult.ToolUseID, toolResult.IsError)

									if toolResult.IsError {
										cleanedError := cleanErrorMessage(contentStr)
										if toolInfo, ok := pendingTools[toolResult.ToolUseID]; ok {
											context := formatErrorContext(toolInfo.Name, toolInfo.Args)
											if context != "" {
												cleanedError = context + ": " + cleanedError
											}
										}

										updateChan <- MessageUpdate{
											Type:    "tool_error",
											Content: cleanedError,
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

// Interrupt gracefully interrupts the current agent operation.
func (o *BaseOrchestrator) Interrupt() error {
	if o.agent != nil {
		client := o.agent.GetClient()
		if client != nil {
			if o.logger != nil {
				o.logger.log("User requested interrupt")
			}
			interruptClient(client, o.logger)
		}
	}
	return nil
}

// Stop gracefully shuts down the orchestrator.
func (o *BaseOrchestrator) Stop() error {
	o.cancel()

	if o.logger != nil {
		o.logger.close()
		o.logger = nil
	}

	if o.agent != nil {
		return o.agent.Stop()
	}

	return nil
}

// GetAgent returns the underlying agent (for orchestrators that need direct access).
func (o *BaseOrchestrator) GetAgent() *agentpkg.Agent {
	return o.agent
}

// GetContext returns the orchestrator's context.
func (o *BaseOrchestrator) GetContext() context.Context {
	return o.ctx
}

// buildEventMetadata merges the base event metadata with additional fields.
func (o *BaseOrchestrator) buildEventMetadata(additional map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range o.eventMetadata {
		result[k] = v
	}
	for k, v := range additional {
		result[k] = v
	}
	return result
}
