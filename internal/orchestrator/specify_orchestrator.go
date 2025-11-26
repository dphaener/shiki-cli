package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	agentpkg "github.com/darinhaener/collab/internal/agent"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// Pre-compiled regex patterns for error message cleaning (compiled once at package init)
var (
	// Stack traces (lines starting with 'at ' or containing file paths)
	stackTraceLineRegex    = regexp.MustCompile(`(?m)^\s*at\s+.*$`)
	stackTraceEntryRegex   = regexp.MustCompile(`(?m)^\s*\w+\.\w+\([^)]*\):\d+.*$`)

	// File paths - more specific patterns
	absoluteFilePathRegex  = regexp.MustCompile(`/[\w\-./]+\.\w+`)
	stackTraceAtRegex      = regexp.MustCompile(`\s+at\s+[\w./\\]+:\d+`)

	// Error codes - more specific to avoid false positives
	errorCodeLabelRegex    = regexp.MustCompile(`(?i)\berror code[:\s]+[A-Za-z0-9_]+`)
	errorCodeConstRegex    = regexp.MustCompile(`\b[A-Z]{3,}_[A-Z0-9_]+\b`)
	hexValueRegex          = regexp.MustCompile(`\b0x[0-9A-Fa-f]+\b`)

	// Debug information and verbose details
	stackTraceKeywordRegex = regexp.MustCompile(`\bstack trace:.*`)
	traceKeywordRegex      = regexp.MustCompile(`\btrace:.*`)
	debugKeywordRegex      = regexp.MustCompile(`\bdebug:\s*[^\n]*`)

	// Technical JSON/XML fragments
	jsonFragmentRegex      = regexp.MustCompile(`\{[^{}]*"[^"]*"[^{}]*\}`)
	xmlTagRegex            = regexp.MustCompile(`<[^>]*>`)

	// Clean up multiple whitespace but preserve structure
	multipleBlankLinesRegex = regexp.MustCompile(`\n\s*\n\s*\n+`)
	multipleSpacesRegex     = regexp.MustCompile(`\s+`)
)

// errorCleaningPattern pairs a pre-compiled regex with its replacement string
type errorCleaningPattern struct {
	regex       *regexp.Regexp
	replacement string
}

// errorCleaningPatterns is the ordered list of patterns to apply for cleaning error messages
var errorCleaningPatterns = []errorCleaningPattern{
	{stackTraceLineRegex, ""},
	{stackTraceEntryRegex, ""},
	{absoluteFilePathRegex, ""},
	{stackTraceAtRegex, ""},
	{errorCodeLabelRegex, ""},
	{errorCodeConstRegex, ""},
	{hexValueRegex, ""},
	{stackTraceKeywordRegex, ""},
	{traceKeywordRegex, ""},
	{debugKeywordRegex, ""},
	{jsonFragmentRegex, ""},
	{xmlTagRegex, ""},
	{multipleBlankLinesRegex, "\n"},
	{multipleSpacesRegex, " "},
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
	logger              *debugLogger // Singleton logger managed by orchestrator lifecycle
	eventBus            *events.EventBus // Event bus for publishing assistant events
}

// NewSpecifyOrchestrator creates a new specify orchestrator
func NewSpecifyOrchestrator(session *types.SpecifySession, apiKey string, eventBus *events.EventBus) *SpecifyOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	return &SpecifyOrchestrator{
		session:  session,
		apiKey:   apiKey,
		ctx:      ctx,
		cancel:   cancel,
		eventBus: eventBus,
	}
}

// Initialize sets up the agent
func (o *SpecifyOrchestrator) Initialize() error {
	// Create singleton debug logger for orchestrator lifecycle
	o.logger = newDebugLogger()

	o.logger.log("========== INITIALIZING SPECIFY AGENT ==========")
	o.logger.log("Session ID: %s", o.session.ID)
	o.logger.log("Feature: %s (#%03d)", o.session.FriendlyName, o.session.FeatureNumber)
	o.logger.log("Slug: %s", o.session.Slug)
	o.logger.log("SpecFile: %s", o.session.SpecFile)
	o.logger.log("SpecDir: %s", o.session.SpecDir)
	o.logger.log("ChecklistDir: %s", o.session.ChecklistDir)
	o.logger.log("SkipDiscovery: %v", o.session.SkipDiscoveryQuestions)

	// Create specify agent configuration - this generates the instructions
	agentCfg := agentpkg.NewSpecifyAgent(o.session)

	// Store instructions to send as first message
	o.specifyInstructions = agentCfg.SystemPrompt
	o.instructionsSent = false

	o.logger.log("========== SPECIFY INSTRUCTIONS ==========")
	o.logger.log("%s", o.specifyInstructions)
	o.logger.log("========== END SPECIFY INSTRUCTIONS ==========")

	// Create agent instance
	o.agent = agentpkg.NewAgent(agentCfg)

	// Start agent (built-in tools only)
	if err := o.agent.Start(o.ctx, o.apiKey, agentCfg.ID); err != nil {
		return fmt.Errorf("failed to start specify agent: %w", err)
	}

	o.logger.log("Agent started successfully")
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

// interruptTimeout is the maximum time to wait for an interrupt call to complete.
// If the SDK's Interrupt method hangs, we don't want to block forever.
const interruptTimeout = 10 * time.Second

// interruptClient attempts to interrupt the SDK client with a timeout.
// This prevents hanging if the SDK's Interrupt method blocks.
// Returns true if interrupt succeeded, false if it timed out or failed.
func interruptClient(client sdkClient, logger *debugLogger) bool {
	// Create a timeout context for the interrupt
	ctx, cancel := context.WithTimeout(context.Background(), interruptTimeout)
	defer cancel()

	// Channel to signal interrupt completion
	done := make(chan error, 1)
	go func() {
		done <- client.Interrupt(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			if logger != nil {
				logger.log("Interrupt failed (may be expected): %v", err)
			}
		}
		return err == nil
	case <-ctx.Done():
		if logger != nil {
			logger.log("Interrupt timed out after %v - forcing continuation", interruptTimeout)
		}
		return false
	}
}

// sdkClient interface for interrupt capability (allows testing)
type sdkClient interface {
	Interrupt(ctx context.Context) error
}

// SendMessage sends a user message to the agent and returns streaming updates
func (o *SpecifyOrchestrator) SendMessage(userMessage string) (<-chan MessageUpdate, <-chan error) {
	updateChan := make(chan MessageUpdate, 100)
	errorChan := make(chan error, 1)

	go func() {
		// Use orchestrator's singleton logger (no defer close - managed by orchestrator lifecycle)
		logger := o.logger

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

				// Publish assistant error event if eventBus is available
				if o.eventBus != nil {
					errorEvent := events.NewAssistantError(
						o.session.ID,
						"specify",
						o.session.ID, // Using session ID as agent ID for now
						"query_failed",
						err.Error(),
						fmt.Sprintf("Failed to send query to agent (attempt %d)", attempt+1),
						"retry", // Will retry if attempts remain
						map[string]interface{}{
							"feature_number": o.session.FeatureNumber,
							"slug":          o.session.Slug,
							"attempt":       attempt + 1,
							"max_retries":   maxRetries,
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
					"specify",
					o.session.ID, // Using session ID as agent ID for now
					o.session.ID, // Using session ID as conversation ID
					messageToSend,
					map[string]interface{}{
						"feature_number": o.session.FeatureNumber,
						"slug":          o.session.Slug,
						"attempt":       attempt + 1,
						"message_length": len(messageToSend),
					},
				)
				o.eventBus.Publish(assistantEvent)
			}

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
						interruptClient(client, logger)
						// Check if we should retry
						if attempt < maxRetries {
							logger.log("Will retry after timeout (attempt %d/%d)", attempt+1, maxRetries)
							shouldRetry = true
							break messageLoop
						}
						// Publish timeout error event if eventBus is available
						if o.eventBus != nil {
							timeoutEvent := events.NewAssistantError(
								o.session.ID,
								"specify",
								o.session.ID,
								"timeout",
								queryCtx.Err().Error(),
								fmt.Sprintf("Query timed out after %d retries", maxRetries),
								"abort", // No more retries
								map[string]interface{}{
									"feature_number": o.session.FeatureNumber,
									"slug":          o.session.Slug,
									"max_retries":   maxRetries,
								},
							)
							o.eventBus.Publish(timeoutEvent)
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
							finalResponse := responseBuilder.String()
							updateChan <- MessageUpdate{
								Type:    "complete",
								Content: finalResponse,
							}

							// Publish assistant response event if eventBus is available
							if o.eventBus != nil {
								responseEvent := events.NewAssistantResponse(
									o.session.ID,
									"specify",
									o.session.ID, // Using session ID as agent ID for now
									o.session.ID, // Using session ID as conversation ID
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

							// Publish assistant metadata event for token usage if eventBus is available
							if o.eventBus != nil && assistantMsg.Message.Usage.OutputTokens > 0 {
								metadataEvent := events.NewAssistantMetadata(
									o.session.ID,
									"specify",
									o.session.ID, // Using session ID as agent ID for now
									int(assistantMsg.Message.Usage.InputTokens + assistantMsg.Message.Usage.OutputTokens),
									0, // Cost not available
									0, // Duration not available
									"", // Model not specified
									nil, // No performance data
									map[string]interface{}{
										"feature_number": o.session.FeatureNumber,
										"slug":          o.session.Slug,
										"input_tokens":  assistantMsg.Message.Usage.InputTokens,
										"output_tokens": assistantMsg.Message.Usage.OutputTokens,
										"message_id":    messageID,
									},
								)
								o.eventBus.Publish(metadataEvent)
							}

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
										// Clean the error message to make it user-friendly
										cleanedError := cleanErrorMessage(contentStr)

										// Add context if available, using friendly names
										if toolInfo, ok := pendingTools[toolResult.ToolUseID]; ok {
											context := formatErrorContext(toolInfo.Name, toolInfo.Args)
											if context != "" {
												cleanedError = context + ": " + cleanedError
											}
										}

										// Surface tool error to the user
										updateChan <- MessageUpdate{
											Type:    "tool_error",
											Content: cleanedError,
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

// cleanErrorMessage sanitizes error messages by removing technical details
// and providing user-friendly error descriptions
func cleanErrorMessage(errorMsg string) string {
	if errorMsg == "" {
		return "An unknown error occurred"
	}

	cleaned := errorMsg

	// Apply all pre-compiled cleaning patterns
	for _, p := range errorCleaningPatterns {
		cleaned = p.regex.ReplaceAllString(cleaned, p.replacement)
	}

	// Clean up the result
	cleaned = strings.TrimSpace(cleaned)

	// Remove leading/trailing punctuation artifacts
	cleaned = strings.Trim(cleaned, ":.,; ")

	// Handle empty or too short result after cleaning
	if cleaned == "" || len(cleaned) < 3 {
		return "An error occurred. Please check your input and try again."
	}

	// Ensure the message starts with a capital letter and ends properly
	cleaned = capitalizeFirst(cleaned)
	if !strings.HasSuffix(cleaned, ".") && !strings.HasSuffix(cleaned, "!") && !strings.HasSuffix(cleaned, "?") {
		cleaned += "."
	}

	return cleaned
}

// formatErrorContext creates a user-friendly context description for tool errors
func formatErrorContext(toolName, toolArgs string) string {
	if toolName == "" {
		return ""
	}

	// Map technical tool names to user-friendly descriptions
	friendlyNames := map[string]string{
		"bash":         "running command",
		"edit":         "editing file",
		"write":        "writing file",
		"read":         "reading file",
		"grep":         "searching content",
		"glob":         "finding files",
		"web_fetch":    "fetching web content",
		"web_search":   "searching web",
	}

	friendlyName := friendlyNames[strings.ToLower(toolName)]
	if friendlyName == "" {
		friendlyName = strings.ToLower(toolName)
	}

	return fmt.Sprintf("Error while %s", friendlyName)
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Stop gracefully shuts down the orchestrator
func (o *SpecifyOrchestrator) Stop() error {
	o.cancel()

	// Close the singleton logger
	if o.logger != nil {
		o.logger.close()
		o.logger = nil
	}

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
