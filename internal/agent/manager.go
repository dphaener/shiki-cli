package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// SpawnAgent creates and starts a new agent subprocess
func (m *Manager) SpawnAgent(ctx context.Context, cfg *types.Agent, mcpEndpoint string) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if agent already exists
	if _, exists := m.agents[cfg.ID]; exists {
		return nil, fmt.Errorf("agent %s already spawned", cfg.ID)
	}

	// Create agent instance
	agent := NewAgent(cfg)

	// Parse MCP endpoint to extract command and args
	// For now, assume mcpEndpoint is in format "command args..."
	// TODO: Properly parse STDIO endpoint format
	mcpCmd := "collab-mcp-server" // This should be the actual MCP server binary
	mcpArgs := []string{}         // Any args for the MCP server

	// Start agent process with SDK
	if err := agent.Start(ctx, mcpCmd, mcpArgs, m.apiKey); err != nil {
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

			// Handle different message types to collect metrics
			// This is a simplified version - full implementation would handle all message types
			switch msg.Type() {
			case "text_delta", "text":
				// Accumulate response text (simplified)
				responseText += fmt.Sprintf("%v", msg)
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
