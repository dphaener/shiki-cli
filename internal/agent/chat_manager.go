package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/darinhaener/collab/internal/broker"
	"github.com/darinhaener/collab/internal/chatservice"
	"github.com/darinhaener/collab/internal/conversation"
	"github.com/darinhaener/collab/internal/diff"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// ChatManager handles agent lifecycle with the new chat infrastructure.
// It emits events to both the legacy EventBus (for backwards compatibility)
// and the new typed Broker for the modern chat system.
type ChatManager struct {
	agents    map[string]*Agent
	sessionID string
	apiKey    string
	mu        sync.RWMutex

	// New chat infrastructure
	broker     *broker.Broker
	messageSvc chatservice.MessageService

	// Legacy event bus (for backwards compatibility during migration)
	legacyBus *events.EventBus

	// Health monitoring
	healthChecks map[string]context.CancelFunc
	healthMu     sync.Mutex

	// Current streaming messages per agent
	streamingMsgs map[string]string // agentID -> messageID
	streamMu      sync.RWMutex

	// Tool execution context cache for diff generation
	executionContexts map[string]*ToolExecutionContext
	contextMu         sync.RWMutex
}

// NewChatManager creates a new chat-aware agent manager.
func NewChatManager(
	sessionID, apiKey string,
	broker *broker.Broker,
	messageSvc chatservice.MessageService,
	legacyBus *events.EventBus,
) *ChatManager {
	return &ChatManager{
		agents:            make(map[string]*Agent),
		sessionID:         sessionID,
		apiKey:            apiKey,
		broker:            broker,
		messageSvc:        messageSvc,
		legacyBus:         legacyBus,
		healthChecks:      make(map[string]context.CancelFunc),
		streamingMsgs:     make(map[string]string),
		executionContexts: make(map[string]*ToolExecutionContext),
	}
}

// storeExecutionContext stores a tool execution context for later retrieval
func (m *ChatManager) storeExecutionContext(ctx *ToolExecutionContext) {
	m.contextMu.Lock()
	defer m.contextMu.Unlock()
	m.executionContexts[ctx.ToolCallID] = ctx
}

// getExecutionContext retrieves a tool execution context by tool call ID
func (m *ChatManager) getExecutionContext(toolCallID string) (*ToolExecutionContext, bool) {
	m.contextMu.RLock()
	defer m.contextMu.RUnlock()
	ctx, exists := m.executionContexts[toolCallID]
	return ctx, exists
}

// cleanupExecutionContext removes a tool execution context from cache
func (m *ChatManager) cleanupExecutionContext(toolCallID string) {
	m.contextMu.Lock()
	defer m.contextMu.Unlock()
	delete(m.executionContexts, toolCallID)
}

// extractFilePathsFromArgs extracts file paths from Write tool arguments
func (m *ChatManager) extractFilePathsFromArgs(toolName string, args map[string]interface{}) []string {
	var filePaths []string

	// Currently only handle Write tool
	if toolName != "Write" {
		return filePaths
	}

	// Extract file_path parameter
	if filePath, ok := args["file_path"].(string); ok && filePath != "" {
		filePaths = append(filePaths, filePath)
	}

	return filePaths
}

// captureToolExecutionContext captures file state before tool execution
func (m *ChatManager) captureToolExecutionContext(toolCallID, toolName string, args map[string]interface{}) {
	// Extract file paths from tool arguments
	filePaths := m.extractFilePathsFromArgs(toolName, args)
	if len(filePaths) == 0 {
		return // No files to track
	}

	// Create execution context
	execCtx := NewToolExecutionContext(toolCallID, toolName)

	// Capture file content for each path
	for _, filePath := range filePaths {
		if err := execCtx.AddFilePath(filePath); err != nil {
			// Log error but continue with other files
			// Note: Could add structured logging here
		}
	}

	// Store context for later retrieval
	m.storeExecutionContext(execCtx)
}

// generateDiffForToolResult generates a diff for Write tool results
func (m *ChatManager) generateDiffForToolResult(toolCallID string) *diff.FileDiff {
	// Retrieve execution context
	execCtx, exists := m.getExecutionContext(toolCallID)
	if !exists || execCtx.ToolName != "Write" {
		return nil // Only generate diffs for Write tools
	}

	// We expect exactly one file path for Write tool
	if len(execCtx.FilePaths) != 1 {
		return nil
	}

	filePath := execCtx.FilePaths[0]
	originalContent, exists := execCtx.GetOriginalContent(filePath)
	if !exists {
		return nil
	}

	// Read current file content after tool execution
	var newContent string
	if content, err := os.ReadFile(filePath); err == nil {
		newContent = string(content)
	} else {
		// File might have been deleted or is inaccessible
		newContent = ""
	}

	// Generate diff using the diff package
	generator := diff.NewGenerator()
	fileDiff, err := generator.GenerateUnifiedDiff(originalContent, newContent, filePath)
	if err != nil {
		// Log error but don't fail the tool result processing
		return nil
	}

	// Only return diff if there are actual changes
	if fileDiff.UnifiedDiff == "" && originalContent == newContent {
		return nil
	}

	return fileDiff
}

// SpawnAgent creates and starts a new agent subprocess.
// Note: MCP tools removed - agents use built-in tools only and communicate via files
func (m *ChatManager) SpawnAgent(ctx context.Context, cfg *types.Agent) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.agents[cfg.ID]; exists {
		return nil, fmt.Errorf("agent %s already spawned", cfg.ID)
	}

	agent := NewAgent(cfg)

	if err := agent.Start(ctx, m.apiKey, cfg.ID); err != nil {
		return nil, fmt.Errorf("failed to start agent %s: %w", cfg.ID, err)
	}

	m.agents[cfg.ID] = agent
	m.startHealthMonitoring(ctx, agent)

	return agent, nil
}

// StartTurn executes a single turn for the given agent using the new chat infrastructure.
func (m *ChatManager) StartTurn(ctx context.Context, agentID, query string, turnNumber int) (*TurnResult, error) {
	m.mu.RLock()
	agent, exists := m.agents[agentID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	if !agent.IsHealthy() {
		return nil, fmt.Errorf("agent %s is not healthy", agentID)
	}

	start := time.Now()

	// Emit turn started event
	m.emitTurnStarted(agentID, turnNumber)

	// Start a streaming message for this turn
	msg, err := m.messageSvc.StartStream(ctx, m.sessionID, agentID, turnNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to start stream: %w", err)
	}

	// Track streaming message
	m.streamMu.Lock()
	m.streamingMsgs[agentID] = msg.ID
	m.streamMu.Unlock()

	// Track last error for recovery decisions
	var lastQueryError error

	// Retry loop for handling timeouts and broken pipes
	for attempt := 0; attempt <= maxTurnRetries; attempt++ {
		if attempt > 0 {
			// Restart client on recoverable errors (broken pipes, etc.) or if client is nil
			if IsRecoverableError(lastQueryError) || agent.GetClient() == nil {
				_ = agent.Restart(context.Background()) // Use fresh context
			}

			// Brief delay before retry
			time.Sleep(turnRetryDelay)
		}

		// Get client inside the loop (may have been recreated)
		client := agent.GetClient()
		if client == nil {
			lastQueryError = fmt.Errorf("agent client is nil")
			if attempt < maxTurnRetries {
				continue
			}
			m.finishStream(ctx, msg.ID, conversation.FinishReasonError, 0, 0)
			return nil, fmt.Errorf("agent %s SDK client not initialized after %d retries", agentID, maxTurnRetries)
		}

		// Create timeout context (5 minute default)
		turnCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

		// Inject turn number and agent ID into context for tool handlers
		turnCtx = context.WithValue(turnCtx, "turn", turnNumber)
		turnCtx = context.WithValue(turnCtx, "agent_id", agentID)

		// Send query to agent via SDK
		if err := client.Query(turnCtx, query); err != nil {
			cancel()
			lastQueryError = err

			// Check if this is a recoverable error (broken pipe, etc.)
			if IsRecoverableError(err) && attempt < maxTurnRetries {
				continue
			}

			m.finishStream(turnCtx, msg.ID, conversation.FinishReasonError, 0, 0)
			return nil, fmt.Errorf("failed to send query to agent: %w", err)
		}

		// Receive and collect response messages
		msgChan, errChan := client.ReceiveMessages(turnCtx)

		var totalInputTokens, totalOutputTokens, toolCalls int
		var responseText strings.Builder
		var lastError error
		var receivedResult bool
		completed := false
		shouldRetry := false

	messageLoop:
		for {
			select {
			case <-turnCtx.Done():
				// Send interrupt to stop the current operation and clean up (with timeout)
				interruptClientWithTimeout(client)
				// Check if we should retry
				if attempt < maxTurnRetries {
					shouldRetry = true
					break messageLoop
				}
				// No more retries
				m.finishStream(turnCtx, msg.ID, conversation.FinishReasonError, totalInputTokens, totalOutputTokens)
				cancel()
				if lastError != nil {
					return nil, fmt.Errorf("turn timeout after %d retries: %w", maxTurnRetries, lastError)
				}
				return nil, fmt.Errorf("turn timeout after %d retries", maxTurnRetries)

			case err := <-errChan:
				if err != nil {
					lastError = err
					// Continue to drain messages - don't exit immediately
				}

			case sdkMsg, ok := <-msgChan:
				if !ok {
					// Channel truly closed - check if we have an error
					if lastError != nil {
						m.finishStream(turnCtx, msg.ID, conversation.FinishReasonError, totalInputTokens, totalOutputTokens)
						cancel()
						return nil, lastError
					}
					// Normal completion (channel closed without result message)
					completed = true
					break messageLoop
				}

				// Skip nil messages - SDK may send these during normal operation
				if sdkMsg == nil {
					continue
				}

				// Capture session ID for resume capability
				if sessionIDProvider, ok := sdkMsg.(interface{ SessionID() string }); ok {
					if sid := sessionIDProvider.SessionID(); sid != "" {
						agent.SetSessionID(sid)
					}
				}

				msgType := sdkMsg.Type()

				switch msgType {
				case "assistant":
					if assistantMsg, ok := sdkMsg.(*claude.SDKAssistantMessage); ok {
						for _, block := range assistantMsg.Message.Content {
							switch content := block.(type) {
							case claude.TextContentBlock:
								if content.Text != "" {
									// Append to streaming message
									m.messageSvc.AppendDelta(turnCtx, msg.ID, content.Text)
									responseText.WriteString(content.Text)

									// Also emit to legacy bus
									if m.legacyBus != nil {
										m.legacyBus.Publish(events.NewAssistantMessage(
											agentID,
											m.sessionID,
											turnNumber,
											content.Text,
										))
									}
								}

							case claude.ToolUseContentBlock:
								toolCalls++
								displayName := strings.TrimPrefix(content.Name, "mcp__collaboration__")

								// Extract tool arguments from JSONValue (json.RawMessage)
								var toolArgs map[string]interface{}
								if len(content.Input) > 0 {
									if err := json.Unmarshal(content.Input, &toolArgs); err != nil {
										// If unmarshal fails, store raw as string
										toolArgs = map[string]interface{}{"_raw": string(content.Input)}
									}
								}

								// Capture file state before tool execution for diff generation
								m.captureToolExecutionContext(content.ID, displayName, toolArgs)

								// Add tool call part to message
								toolCallPart := conversation.NewToolCallPart(content.ID, displayName, toolArgs)
								m.messageSvc.AddPart(turnCtx, msg.ID, toolCallPart)

								// Also emit to legacy bus
								if m.legacyBus != nil {
									m.legacyBus.Publish(events.NewToolInvoked(
										displayName,
										agentID,
										m.sessionID,
										turnNumber,
										toolArgs,
									))
								}
							}
						}

						// Extract token usage if available
						usage := assistantMsg.Message.Usage
						if usage.InputTokens > 0 || usage.OutputTokens > 0 {
							totalInputTokens += usage.InputTokens
							totalOutputTokens += usage.OutputTokens
						}
					}

				case "user":
					// User messages contain tool results
					if userMsg, ok := sdkMsg.(*claude.SDKUserMessage); ok {
						for _, block := range userMsg.Message.Content {
							if toolResult, ok := block.(claude.ToolResultContentBlock); ok {
								// Extract tool result content
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

								// Generate diff if this was a Write tool execution
								fileDiff := m.generateDiffForToolResult(toolResult.ToolUseID)

								// Add tool result part to message (with or without diff)
								var toolResultPart conversation.ToolResultPart
								if fileDiff != nil {
									toolResultPart = conversation.NewToolResultPartWithDiff(toolResult.ToolUseID, contentStr, toolResult.IsError, fileDiff)
								} else {
									toolResultPart = conversation.NewToolResultPart(toolResult.ToolUseID, contentStr, toolResult.IsError)
								}
								m.messageSvc.AddPart(turnCtx, msg.ID, toolResultPart)

								// Clean up execution context
								m.cleanupExecutionContext(toolResult.ToolUseID)
							}
						}
					}

				case "result":
					// Result message signals explicit completion
					receivedResult = true

					// Try to extract final token usage from result
					if resultMsg, ok := sdkMsg.(*claude.SDKResultMessage); ok {
						usage := resultMsg.Usage
						if usage.InputTokens > 0 || usage.OutputTokens > 0 {
							totalInputTokens = usage.InputTokens
							totalOutputTokens = usage.OutputTokens
						}
					}

					completed = true
					break messageLoop
				}
			}
		}

		cancel()

		if completed {
			duration := time.Since(start)

			// Determine finish reason
			finishReason := conversation.FinishReasonStop
			if toolCalls > 0 {
				finishReason = conversation.FinishReasonToolUse
			}
			if lastError != nil {
				finishReason = conversation.FinishReasonError
			}

			// Finish the streaming message
			m.finishStream(ctx, msg.ID, finishReason, totalInputTokens, totalOutputTokens)

			// Clear streaming message tracking
			m.streamMu.Lock()
			delete(m.streamingMsgs, agentID)
			m.streamMu.Unlock()

			// Estimate token usage if SDK didn't provide it
			totalTokens := totalInputTokens + totalOutputTokens
			if totalTokens == 0 {
				totalTokens = len(query)/4 + responseText.Len()/4
				totalInputTokens = len(query) / 4
				totalOutputTokens = responseText.Len() / 4
			}

			// Calculate cost
			cost := CalculateCostDetailed(totalInputTokens, totalOutputTokens, agent.Model)

			result := &TurnResult{
				AgentID:      agentID,
				TurnNumber:   turnNumber,
				Duration:     duration,
				TokensUsed:   totalTokens,
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
				Cost:         cost,
				Response:     responseText.String(),
				ToolCalls:    toolCalls,
				Success:      lastError == nil,
				Error:        "",
			}

			if lastError != nil {
				result.Error = lastError.Error()
			}

			// Emit turn completed event
			m.emitTurnCompleted(agentID, turnNumber, totalInputTokens, totalOutputTokens, cost, toolCalls, duration)

			// Log completion status
			if !receivedResult && lastError == nil {
				// Channel closed without explicit result - this is OK but worth noting
				// The SDK may close the channel after all content is sent
			}

			return result, lastError
		}

		if !shouldRetry {
			return nil, fmt.Errorf("turn failed unexpectedly")
		}
	}

	return nil, fmt.Errorf("turn failed after %d retries", maxTurnRetries)
}

// finishStream finishes a streaming message.
func (m *ChatManager) finishStream(ctx context.Context, messageID string, reason conversation.FinishReason, inputTokens, outputTokens int) {
	if m.messageSvc != nil {
		m.messageSvc.FinishStream(ctx, messageID, reason, inputTokens, outputTokens)
	}
}

// emitTurnStarted emits turn started events to both systems.
func (m *ChatManager) emitTurnStarted(agentID string, turnNumber int) {
	// Emit to new broker
	if m.broker != nil {
		m.broker.Publish(broker.NewTurnStartedEvent(m.sessionID, agentID, turnNumber))
	}

	// Emit to legacy bus
	if m.legacyBus != nil {
		turn := &types.Turn{
			Number:  turnNumber,
			AgentID: agentID,
			Status:  types.TurnInProgress,
		}
		m.legacyBus.Publish(events.NewTurnStarted(turn, agentID, m.sessionID))
	}
}

// emitTurnCompleted emits turn completed events to both systems.
func (m *ChatManager) emitTurnCompleted(agentID string, turnNumber, inputTokens, outputTokens int, cost float64, toolCalls int, duration time.Duration) {
	// Emit to new broker
	if m.broker != nil {
		m.broker.Publish(broker.NewTurnCompletedEvent(
			m.sessionID,
			agentID,
			turnNumber,
			inputTokens,
			outputTokens,
			cost,
			toolCalls,
			duration.Milliseconds(),
		))
	}

	// Emit to legacy bus
	if m.legacyBus != nil {
		turn := &types.Turn{
			Number:     turnNumber,
			AgentID:    agentID,
			Status:     types.TurnCompleted,
			TokensUsed: inputTokens + outputTokens,
			Cost:       cost,
			DurationMs: duration.Milliseconds(),
			ToolCalls:  toolCalls,
		}
		m.legacyBus.Publish(events.NewTurnCompleted(turn, m.sessionID))
	}
}

// GetCurrentStreamingMessage returns the current streaming message ID for an agent.
func (m *ChatManager) GetCurrentStreamingMessage(agentID string) (string, bool) {
	m.streamMu.RLock()
	defer m.streamMu.RUnlock()
	msgID, ok := m.streamingMsgs[agentID]
	return msgID, ok
}

// startHealthMonitoring begins periodic health checks for an agent.
func (m *ChatManager) startHealthMonitoring(ctx context.Context, agent *Agent) {
	healthCtx, cancel := context.WithCancel(ctx)

	m.healthMu.Lock()
	m.healthChecks[agent.ID] = cancel
	m.healthMu.Unlock()

	go m.monitorHealth(healthCtx, agent)
}

// monitorHealth runs periodic health checks on an agent.
func (m *ChatManager) monitorHealth(ctx context.Context, agent *Agent) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastHealthy := agent.IsHealthy()
	consecutiveFailures := 0
	const maxConsecutiveFailures = 3

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			client := agent.GetClient()
			isAlive := client != nil

			if !isAlive {
				consecutiveFailures++
			} else {
				consecutiveFailures = 0
			}

			// Mark unhealthy after consecutive failures
			if consecutiveFailures >= maxConsecutiveFailures {
				agent.SetHealth(false)
			} else {
				agent.SetHealth(isAlive)
			}

			currentHealth := agent.IsHealthy()

			if lastHealthy && !currentHealth {
				// Agent crashed or client closed - emit to both systems
				if m.broker != nil {
					errSession := &conversation.Session{ID: m.sessionID}
					m.broker.Publish(broker.NewSessionErrorEvent(errSession, fmt.Errorf("agent %s connection degraded", agent.ID)))
				}
				if m.legacyBus != nil {
					m.legacyBus.Publish(events.NewSessionError(
						&types.Session{ID: m.sessionID},
						fmt.Sprintf("Agent %s connection degraded", agent.ID),
					))
				}
			}

			lastHealthy = currentHealth
		}
	}
}

// GetAgent returns an agent by ID.
func (m *ChatManager) GetAgent(agentID string) (*Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// Shutdown gracefully stops all agents.
func (m *ChatManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop all health monitors
	m.healthMu.Lock()
	for _, cancel := range m.healthChecks {
		cancel()
	}
	m.healthMu.Unlock()

	// Stop all agents
	var errs []error
	for id, agent := range m.agents {
		if err := agent.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop agent %s: %w", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}
