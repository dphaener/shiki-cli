package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/darinhaener/collab/internal/broker"
	"github.com/darinhaener/collab/internal/chatservice"
	"github.com/darinhaener/collab/internal/conversation"
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
}

// NewChatManager creates a new chat-aware agent manager.
func NewChatManager(
	sessionID, apiKey string,
	broker *broker.Broker,
	messageSvc chatservice.MessageService,
	legacyBus *events.EventBus,
) *ChatManager {
	return &ChatManager{
		agents:        make(map[string]*Agent),
		sessionID:     sessionID,
		apiKey:        apiKey,
		broker:        broker,
		messageSvc:    messageSvc,
		legacyBus:     legacyBus,
		healthChecks:  make(map[string]context.CancelFunc),
		streamingMsgs: make(map[string]string),
	}
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
