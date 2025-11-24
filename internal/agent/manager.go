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

// SpawnAgent creates and starts a new agent subprocess with MCP tools
func (m *Manager) SpawnAgent(ctx context.Context, cfg *types.Agent, mcpTools []claude.McpTool) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if agent already exists
	if _, exists := m.agents[cfg.ID]; exists {
		return nil, fmt.Errorf("agent %s already spawned", cfg.ID)
	}

	// Create agent instance
	agent := NewAgent(cfg)

	// Start agent process with SDK and MCP tools
	if err := agent.Start(ctx, mcpTools, m.apiKey, cfg.ID); err != nil {
		return nil, fmt.Errorf("failed to start agent %s: %w", cfg.ID, err)
	}

	// Register agent
	m.agents[cfg.ID] = agent

	// Start health monitoring
	m.startHealthMonitoring(ctx, agent)

	return agent, nil
}

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

	// Create timeout context (5 minute default)
	turnCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	start := time.Now()

	// Send query to agent via SDK
	if err := client.Query(turnCtx, query); err != nil {
		return nil, fmt.Errorf("failed to send query to agent: %w", err)
	}

	// Receive and collect response messages
	msgChan, errChan := client.ReceiveMessages(turnCtx)

	var totalInputTokens, totalOutputTokens, toolCalls int
	var responseText string
	var lastError error

	for {
		select {
		case <-turnCtx.Done():
			if lastError != nil {
				return nil, fmt.Errorf("turn timeout or cancelled: %w", lastError)
			}
			return nil, fmt.Errorf("turn timeout exceeded")

		case err := <-errChan:
			if err != nil {
				lastError = err
				// Continue to drain messages
			}

		case msg := <-msgChan:
			if msg == nil {
				// Query completed
				goto done
			}

			msgType := msg.Type()

			// Handle different message types
			switch msgType {
			case "assistant":
				// Assistant messages may contain tool use or text
				msgStr := fmt.Sprintf("%+v", msg)

				// Check if this message contains a tool use
				if strings.Contains(msgStr, "Type:tool_use") {
					toolCalls++
					// Try to extract tool name from the message string
					// Format: Name:ToolName
					if idx := strings.Index(msgStr, "Name:"); idx != -1 {
						nameStart := idx + 5 // len("Name:")
						nameEnd := strings.Index(msgStr[nameStart:], " ")
						if nameEnd == -1 {
							nameEnd = strings.Index(msgStr[nameStart:], "}")
						}
						if nameEnd > 0 {
							toolName := msgStr[nameStart : nameStart+nameEnd]
							// Strip mcp__ prefix for cleaner display
							displayName := strings.TrimPrefix(toolName, "mcp__collaboration__")
							fmt.Printf("  → %s using tool: %s\n", agentID, displayName)
						}
					}
				}

				// Extract and display text content from assistant messages
				if strings.Contains(msgStr, "Type:text") {
					// Try to extract text content
					// Format: Text:... (content until next field)
					if idx := strings.Index(msgStr, "Text:"); idx != -1 {
						textStart := idx + 5 // len("Text:")
						// Find the end - look for "}]" which marks end of content block
						textEnd := strings.Index(msgStr[textStart:], "}]")
						if textEnd > 0 && textEnd < 200 { // Only show first ~200 chars
							text := msgStr[textStart : textStart+textEnd]
							text = strings.TrimSpace(text)
							if len(text) > 0 {
								// Truncate if too long
								if len(text) > 150 {
									text = text[:150] + "..."
								}
								fmt.Printf("  💬 %s: %s\n", agentID, text)
							}
						}
					}
				}

				// Accumulate response text
				responseText += fmt.Sprintf("%v", msg)

			case "result":
				// Result message signals completion
				goto done
			}

			// TODO: Extract token usage from SDK messages
			// The SDK should provide usage info in result messages
			// For now, we'll use estimates
		}
	}

done:
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
