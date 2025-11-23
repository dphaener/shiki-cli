package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
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

	// Start agent process
	if err := agent.Start(ctx, mcpEndpoint, m.apiKey); err != nil {
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

	// Create timeout context (5 minute default)
	turnCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	start := time.Now()

	// TODO: Implement actual SDK communication
	// For now, this is a placeholder that simulates agent execution
	// In production, this would use the claude-agent-sdk-go client

	// Simulate agent thinking time
	select {
	case <-turnCtx.Done():
		return nil, fmt.Errorf("turn timeout exceeded")
	case <-time.After(100 * time.Millisecond):
		// Simulated completion
	}

	duration := time.Since(start)

	// Collect metrics (placeholder values)
	result := &TurnResult{
		AgentID:    agentID,
		TurnNumber: turnNumber,
		Duration:   duration,
		TokensUsed: 1000, // TODO: Get from SDK
		Cost:       0.0,  // Will be calculated
		Response:   fmt.Sprintf("Agent %s response to: %s", agentID, query),
		ToolCalls:  0,
		Success:    true,
	}

	// Calculate cost
	result.Cost = CalculateCost(result.TokensUsed, agent.Model)

	return result, nil
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
			// Check if process is still alive
			isAlive := m.checkProcessAlive(agent.GetPID())

			// Update agent health status
			agent.SetHealth(isAlive)

			// Emit event if health changed
			if lastHealthy && !isAlive {
				// Agent crashed
				m.eventBus.Publish(events.NewSessionError(
					&types.Session{ID: m.sessionID},
					fmt.Sprintf("Agent %s (PID %d) crashed", agent.ID, agent.GetPID()),
				))
			}

			lastHealthy = isAlive
		}
	}
}

// checkProcessAlive checks if a process with given PID is still running
func (m *Manager) checkProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 doesn't actually send a signal, just checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
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
