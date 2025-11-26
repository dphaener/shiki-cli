package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	agentpkg "github.com/darinhaener/collab/internal/agent"
	"github.com/darinhaener/collab/pkg/types"
)

// debugLogger handles debug logging to a file
type debugLogger struct {
	file *os.File
}

// newDebugLogger creates a new debug logger
func newDebugLogger() *debugLogger {
	// Create log file in current directory
	logPath := filepath.Join(".", "collab-debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return &debugLogger{file: nil}
	}
	return &debugLogger{file: f}
}

// log writes a message to the debug log
func (d *debugLogger) log(format string, args ...interface{}) {
	if d.file == nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(d.file, "[%s] %s\n", timestamp, msg)
	d.file.Sync()
}

// close closes the debug log file
func (d *debugLogger) close() {
	if d.file != nil {
		d.file.Close()
	}
}

// SpecifyOrchestrator manages the specification workflow with a single agent
type SpecifyOrchestrator struct {
	session             *types.SpecifySession
	agent               *agentpkg.Agent
	apiKey              string
	ctx                 context.Context
	cancel              context.CancelFunc
	specifyInstructions string // Instructions sent as first message to prime the agent
	instructionsSent    bool   // Whether we've sent the initial instructions
}

// NewSpecifyOrchestrator creates a new specify orchestrator
func NewSpecifyOrchestrator(session *types.SpecifySession, apiKey string) *SpecifyOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	return &SpecifyOrchestrator{
		session: session,
		apiKey:  apiKey,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Initialize sets up the agent
func (o *SpecifyOrchestrator) Initialize() error {
	// Create debug logger for initialization
	logger := newDebugLogger()
	defer logger.close()

	logger.log("========== INITIALIZING SPECIFY AGENT ==========")
	logger.log("Session ID: %s", o.session.ID)
	logger.log("Feature: %s (#%03d)", o.session.FriendlyName, o.session.FeatureNumber)
	logger.log("Slug: %s", o.session.Slug)
	logger.log("SpecFile: %s", o.session.SpecFile)
	logger.log("SpecDir: %s", o.session.SpecDir)
	logger.log("ChecklistDir: %s", o.session.ChecklistDir)
	logger.log("SkipDiscovery: %v", o.session.SkipDiscoveryQuestions)

	// Create specify agent configuration - this generates the instructions
	agentCfg := agentpkg.NewSpecifyAgent(o.session)

	// Store instructions to send as first message
	o.specifyInstructions = agentCfg.SystemPrompt
	o.instructionsSent = false

	logger.log("========== SPECIFY INSTRUCTIONS ==========")
	logger.log("%s", o.specifyInstructions)
	logger.log("========== END SPECIFY INSTRUCTIONS ==========")

	// Create agent instance
	o.agent = agentpkg.NewAgent(agentCfg)

	// Start agent (built-in tools only)
	if err := o.agent.Start(o.ctx, o.apiKey, agentCfg.ID); err != nil {
		return fmt.Errorf("failed to start specify agent: %w", err)
	}

	logger.log("Agent started successfully")
	return nil
}

// MessageUpdate represents a streaming update from the agent
type MessageUpdate struct {
	Type    string // "text", "tool_use", "complete"
	Content string
}

// maxRetries is the maximum number of times to retry on timeout
const maxRetries = 3

// retryDelay is the delay between retry attempts
const retryDelay = 2 * time.Second

// SendMessage sends a user message to the agent and returns streaming updates
func (o *SpecifyOrchestrator) SendMessage(userMessage string) (<-chan MessageUpdate, <-chan error) {
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

		// If this is the first message, prepend the specify instructions
		if !o.instructionsSent && o.specifyInstructions != "" {
			logger.log("Prepending specify instructions to first message")
			messageToSend = o.specifyInstructions + "\n\n---\n\n## User Request\n\n" + userMessage
			o.instructionsSent = true
		}

		// Retry loop for handling timeouts
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				logger.log("Retry attempt %d/%d after timeout", attempt, maxRetries)
				// Brief delay before retry
				time.Sleep(retryDelay)
			}

			// Create query context with timeout for this attempt
			queryCtx, cancel := context.WithTimeout(o.ctx, 3*time.Minute)

			// Send query to agent
			logger.log("Sending query to agent (message length: %d chars)...", len(messageToSend))
			if err := client.Query(queryCtx, messageToSend); err != nil {
				cancel()
				logger.log("ERROR: failed to send query: %v", err)
				errorChan <- fmt.Errorf("failed to send query: %w", err)
				return
			}
			logger.log("Query sent successfully")

			// Receive messages
			msgChan, errChan := client.ReceiveMessages(queryCtx)
			logger.log("Waiting for messages...")

			var responseBuilder strings.Builder
			var currentMessageID string // Track the current assistant message ID
			msgCount := 0

			// Track pending tool uses to provide context for tool results
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
						// Send interrupt to stop the current operation and clean up
						logger.log("Sending interrupt to clean up after timeout...")
						if interruptErr := client.Interrupt(context.Background()); interruptErr != nil {
							logger.log("Interrupt failed (may be expected): %v", interruptErr)
						}
						// Check if we should retry
						if attempt < maxRetries {
							logger.log("Will retry after timeout (attempt %d/%d)", attempt+1, maxRetries)
							shouldRetry = true
							break messageLoop
						}
						// No more retries, send error
						errorChan <- fmt.Errorf("query timeout after %d retries: %w", maxRetries, queryCtx.Err())
					}
					cancel()
					return

				case err, ok := <-errChan:
					logger.log("Error channel: ok=%v, err=%v", ok, err)
					if !ok {
						// Error channel closed - set to nil to remove from select
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
					// nil error received, continue
					logger.log("Nil error received, continuing...")

				case msg, ok := <-msgChan:
					msgCount++
					logger.log("--- Message #%d ---", msgCount)
					logger.log("Channel ok=%v, msg==nil: %v", ok, msg == nil)

					if !ok {
						logger.log("Channel closed (ok=false), sending completion")
						// Channel truly closed - send completion signal
						if responseBuilder.Len() > 0 {
							updateChan <- MessageUpdate{
								Type:    "complete",
								Content: responseBuilder.String(),
							}
						}
						completed = true
						break messageLoop
					}

					// Skip nil messages - SDK may send these during normal operation
					if msg == nil {
						logger.log("Skipping nil message")
						continue
					}

					// Process message based on type
					msgType := msg.Type()
					logger.log("Message type: %s", msgType)

					// Try to marshal the entire message to JSON for debugging
					if rawJSON, err := json.MarshalIndent(msg, "", "  "); err == nil {
						logger.log("Raw message JSON:\n%s", string(rawJSON))
					} else {
						logger.log("Could not marshal message: %v", err)
						logger.log("Message value: %+v", msg)
					}

					switch msgType {
					case "assistant":
						logger.log("Processing assistant message")

						// Type assert to SDKAssistantMessage to access content
						if assistantMsg, ok := msg.(*claude.SDKAssistantMessage); ok {
							messageID := assistantMsg.Message.ID
							logger.log("Assistant message ID: %s (current: %s)", messageID, currentMessageID)
							logger.log("Assistant message has %d content blocks", len(assistantMsg.Message.Content))
							logger.log("Stop reason: %v", assistantMsg.Message.StopReason)
							logger.log("Stop sequence: %v", assistantMsg.Message.StopSequence)

							// Check if this is a NEW message (different ID) - signals a new assistant turn
							if currentMessageID != "" && messageID != currentMessageID {
								logger.log("New message detected (ID changed), completing previous message")
								// Complete the previous message before starting the new one
								// Use "message_done" (not "complete") so TUI keeps listening
								if responseBuilder.Len() > 0 {
									updateChan <- MessageUpdate{
										Type:    "message_done",
										Content: responseBuilder.String(),
									}
									responseBuilder.Reset()
								}
							}
							currentMessageID = messageID

							// Log usage info
							logger.log("Usage - Input: %d, Output: %d",
								assistantMsg.Message.Usage.InputTokens,
								assistantMsg.Message.Usage.OutputTokens)

							// Process each content block in the message
							for i, block := range assistantMsg.Message.Content {
								logger.log("Content block %d type: %T", i, block)
								switch content := block.(type) {
								case claude.TextContentBlock:
									logger.log("Text block content (len=%d): %s", len(content.Text), truncateForLog(content.Text, 200))
									// Stream text content immediately
									if content.Text != "" {
										responseBuilder.WriteString(content.Text)
										updateChan <- MessageUpdate{
											Type:    "text",
											Content: content.Text,
										}
									}
								case claude.ToolUseContentBlock:
									logger.log("Tool use block: %s (ID: %s)", content.Name, content.ID)

									// Extract and format tool arguments
									var argsStr string
									if len(content.Input) > 0 {
										argsStr = string(content.Input)
									}

									// Track this tool use for error context
									pendingTools[content.ID] = struct {
										Name string
										Args string
									}{
										Name: content.Name,
										Args: argsStr,
									}

									// Send tool use notification immediately
									updateChan <- MessageUpdate{
										Type:    "tool_use",
										Content: content.Name,
									}
								default:
									logger.log("Unknown content block type: %T", block)
								}
							}
						} else {
							logger.log("Failed to type assert to SDKAssistantMessage, actual type: %T", msg)
						}

					case "result":
						logger.log("Processing result message - COMPLETING")
						// Result message signals completion
						if responseBuilder.Len() > 0 {
							updateChan <- MessageUpdate{
								Type:    "complete",
								Content: responseBuilder.String(),
							}
						}
						completed = true
						break messageLoop

					case "system":
						// System messages (init, etc.) - just log and continue
						logger.log("System message received, continuing...")

					case "user":
						// User messages contain tool results - check for errors and surface them
						logger.log("User message (tool result) received")

						// Try to extract tool result details to surface errors to user
						if userMsg, ok := msg.(*claude.SDKUserMessage); ok {
							for _, block := range userMsg.Message.Content {
								if toolResult, ok := block.(claude.ToolResultContentBlock); ok {
									// Extract text content from ToolResultContent
									var contentStr string
									if toolResult.Content != nil {
										if toolResult.Content.Text != nil {
											contentStr = *toolResult.Content.Text
										} else if len(toolResult.Content.Blocks) > 0 {
											// Try to extract text from blocks
											for _, b := range toolResult.Content.Blocks {
												if textBlock, ok := b.(claude.TextContentBlock); ok {
													contentStr += textBlock.Text
												}
											}
										}
									}
									if contentStr == "" {
										contentStr = "(no content)"
									}

									logger.log("Tool result - ID: %s, IsError: %v, Content: %s",
										toolResult.ToolUseID, toolResult.IsError, truncateForLog(contentStr, 100))

									if toolResult.IsError {
										// Build error message with context from the tool use
										errorMsg := contentStr
										if toolInfo, ok := pendingTools[toolResult.ToolUseID]; ok {
											errorMsg = fmt.Sprintf("%s\nTool: %s\nArgs: %s",
												contentStr, toolInfo.Name, toolInfo.Args)
										}

										// Surface tool error to the user
										updateChan <- MessageUpdate{
											Type:    "tool_error",
											Content: errorMsg,
										}
									}

									// Clean up tracked tool
									delete(pendingTools, toolResult.ToolUseID)
								}
							}
						}
						logger.log("Waiting for agent response...")

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

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Stop gracefully shuts down the orchestrator
func (o *SpecifyOrchestrator) Stop() error {
	o.cancel()

	if o.agent != nil {
		return o.agent.Stop()
	}

	return nil
}

// GetAPIKey gets the API key from environment or returns empty string
func GetAPIKey() string {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		// Try to use Claude Code's saved credentials
		// The SDK will handle this automatically
		return ""
	}
	return apiKey
}
