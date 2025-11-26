package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// Note: claude import kept for SDK message types in StartTurn

// Manager handles agent lifecycle, health monitoring, and turn execution
type Manager struct {
	agents    map[string]*Agent
	eventBus  *events.EventBus
	sessionID string
	apiKey    string
	mu        sync.RWMutex

	// Health monitoring
	healthChecks map[string]context.CancelFunc
	healthMu     sync.Mutex
}

// NewManager creates a new agent manager
func NewManager(eventBus *events.EventBus, sessionID, apiKey string) *Manager {
	return &Manager{
		agents:       make(map[string]*Agent),
		eventBus:     eventBus,
		sessionID:    sessionID,
		apiKey:       apiKey,
		healthChecks: make(map[string]context.CancelFunc),
	}
}

// SpawnAgent creates and starts a new agent subprocess
// Note: MCP tools removed - agents use built-in tools only and communicate via files
func (m *Manager) SpawnAgent(ctx context.Context, cfg *types.Agent) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if agent already exists
	if _, exists := m.agents[cfg.ID]; exists {
		return nil, fmt.Errorf("agent %s already spawned", cfg.ID)
	}

	// Create agent instance
	agent := NewAgent(cfg)

	// Start agent process with SDK (built-in tools only)
	if err := agent.Start(ctx, m.apiKey, cfg.ID); err != nil {
		return nil, fmt.Errorf("failed to start agent %s: %w", cfg.ID, err)
	}

	// Register agent
	m.agents[cfg.ID] = agent

	// Start health monitoring
	m.startHealthMonitoring(ctx, agent)

	return agent, nil
}

// maxTurnRetries is the maximum number of times to retry a turn on timeout
const maxTurnRetries = 3

// turnRetryDelay is the delay between retry attempts
const turnRetryDelay = 2 * time.Second

// StartTurn executes a single turn for the given agent
func (m *Manager) StartTurn(ctx context.Context, agentID, query string, turnNumber int) (*TurnResult, error) {
	m.mu.RLock()
	agent, exists := m.agents[agentID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	if !agent.IsHealthy() {
		return nil, fmt.Errorf("agent %s is not healthy", agentID)
	}

	client := agent.GetClient()
	if client == nil {
		return nil, fmt.Errorf("agent %s SDK client not initialized", agentID)
	}

	start := time.Now()

	// Retry loop for handling timeouts
	for attempt := 0; attempt <= maxTurnRetries; attempt++ {
		if attempt > 0 {
			// Brief delay before retry
			time.Sleep(turnRetryDelay)
		}

		// Create timeout context (5 minute default)
		turnCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

		// Inject turn number and agent ID into context for tool handlers
		turnCtx = context.WithValue(turnCtx, "turn", turnNumber)
		turnCtx = context.WithValue(turnCtx, "agent_id", agentID)

		// Send query to agent via SDK
		if err := client.Query(turnCtx, query); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to send query to agent: %w", err)
		}

		// Receive and collect response messages
		msgChan, errChan := client.ReceiveMessages(turnCtx)

		var totalInputTokens, totalOutputTokens, toolCalls int
		var responseText string
		var lastError error
		completed := false
		shouldRetry := false

	messageLoop:
		for {
			select {
			case <-turnCtx.Done():
				// Send interrupt to stop the current operation and clean up
				if interruptErr := client.Interrupt(context.Background()); interruptErr != nil {
					_ = interruptErr
				}
				// Check if we should retry
				if attempt < maxTurnRetries {
					shouldRetry = true
					break messageLoop
				}
				// No more retries
				cancel()
				if lastError != nil {
					return nil, fmt.Errorf("turn timeout after %d retries: %w", maxTurnRetries, lastError)
				}
				return nil, fmt.Errorf("turn timeout after %d retries", maxTurnRetries)

			case err := <-errChan:
				if err != nil {
					lastError = err
					// Continue to drain messages
				}

			case msg, ok := <-msgChan:
				if !ok {
					// Channel truly closed - query completed
					completed = true
					break messageLoop
				}

				// Skip nil messages - SDK may send these during normal operation
				if msg == nil {
					continue
				}

				msgType := msg.Type()

				// Handle different message types
				switch msgType {
				case "assistant":
					// Type assert to SDKAssistantMessage to access content
					if assistantMsg, ok := msg.(*claude.SDKAssistantMessage); ok {
						// Process each content block in the message
						for _, block := range assistantMsg.Message.Content {
							switch content := block.(type) {
							case claude.TextContentBlock:
								// Extract and emit text content for TUI display
								if content.Text != "" {
									m.eventBus.Publish(events.NewAssistantMessage(
										agentID,
										m.sessionID,
										turnNumber,
										content.Text,
									))
									responseText += content.Text
								}

							case claude.ToolUseContentBlock:
								toolCalls++
								// Strip mcp__ prefix for cleaner display
								displayName := strings.TrimPrefix(content.Name, "mcp__collaboration__")

								// Emit ToolInvoked event for TUI
								m.eventBus.Publish(events.NewToolInvoked(
									displayName,
									agentID,
									m.sessionID,
									turnNumber,
									nil, // We could parse content.Input if needed
								))
							}
						}
					} else {
						// Fallback: if type assertion fails, use string representation
						msgStr := fmt.Sprintf("%v", msg)
						m.eventBus.Publish(events.NewAssistantMessage(
							agentID,
							m.sessionID,
							turnNumber,
							msgStr,
						))
						responseText += msgStr
					}

				case "result":
					// Result message signals completion
					completed = true
					break messageLoop
				}
			}
		}

		cancel()

		if completed {
			duration := time.Since(start)

			// Estimate token usage (this should come from SDK result messages)
			totalTokens := totalInputTokens + totalOutputTokens
			if totalTokens == 0 {
				// Fallback estimate if SDK doesn't provide usage
				totalTokens = len(query)/4 + len(responseText)/4
				totalInputTokens = len(query) / 4
				totalOutputTokens = len(responseText) / 4
			}

			// Calculate cost using detailed breakdown
			cost := CalculateCostDetailed(totalInputTokens, totalOutputTokens, agent.Model)

			result := &TurnResult{
				AgentID:      agentID,
				TurnNumber:   turnNumber,
				Duration:     duration,
				TokensUsed:   totalTokens,
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
				Cost:         cost,
				Response:     responseText,
				ToolCalls:    toolCalls,
				Success:      lastError == nil,
				Error:        "",
			}

			if lastError != nil {
				result.Error = lastError.Error()
			}

			return result, lastError
		}

		if !shouldRetry {
			return nil, fmt.Errorf("turn failed unexpectedly")
		}
	}

	return nil, fmt.Errorf("turn failed after %d retries", maxTurnRetries)
}

// startHealthMonitoring begins periodic health checks for an agent
func (m *Manager) startHealthMonitoring(ctx context.Context, agent *Agent) {
	healthCtx, cancel := context.WithCancel(ctx)

	m.healthMu.Lock()
	m.healthChecks[agent.ID] = cancel
	m.healthMu.Unlock()

	go m.monitorHealth(healthCtx, agent)
}

// monitorHealth runs periodic health checks on an agent
func (m *Manager) monitorHealth(ctx context.Context, agent *Agent) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastHealthy := agent.IsHealthy()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if SDK client is still functional
			// We check this by verifying the client is non-nil and healthy
			isAlive := agent.GetClient() != nil

			// Update agent health status
			agent.SetHealth(isAlive)

			// Emit event if health changed
			if lastHealthy && !isAlive {
				// Agent crashed or client closed
				m.eventBus.Publish(events.NewSessionError(
					&types.Session{ID: m.sessionID},
					fmt.Sprintf("Agent %s SDK client failed", agent.ID),
				))
			}

			lastHealthy = isAlive
		}
	}
}


// GetAgent returns an agent by ID
func (m *Manager) GetAgent(agentID string) (*Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// Shutdown gracefully stops all agents
func (m *Manager) Shutdown() error {
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
