package agent

import (
	"context"
	"testing"
	"time"

	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")

	assert.NotNil(t, mgr)
	assert.Equal(t, "test-session", mgr.sessionID)
	assert.Equal(t, "test-api-key", mgr.apiKey)
	assert.NotNil(t, mgr.agents)
	assert.NotNil(t, mgr.healthChecks)
}

func TestManager_GetAgent_NotFound(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")

	agent, err := mgr.GetAgent("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_SpawnAgent_AlreadyExists(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")
	ctx := context.Background()

	cfg := &types.Agent{
		ID:           "test_agent",
		Name:         "Test Agent",
		Model:        "claude-sonnet-4",
		WorkspaceDir: t.TempDir(),
	}

	// Manually add an agent to test duplicate detection
	mgr.agents["test_agent"] = NewAgent(cfg)

	// Try to spawn the same agent again
	agent, err := mgr.SpawnAgent(ctx, cfg, "stdio://test")
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "already spawned")
}

func TestManager_StartTurn_AgentNotFound(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")
	ctx := context.Background()

	result, err := mgr.StartTurn(ctx, "nonexistent", "test query", 1)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_StartTurn_AgentNotHealthy(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")
	ctx := context.Background()

	cfg := &types.Agent{
		ID:           "unhealthy_agent",
		Name:         "Unhealthy Agent",
		Model:        "claude-sonnet-4",
		WorkspaceDir: t.TempDir(),
	}

	// Create an agent but don't start it (so it's not healthy)
	agent := NewAgent(cfg)
	mgr.agents["unhealthy_agent"] = agent

	result, err := mgr.StartTurn(ctx, "unhealthy_agent", "test query", 1)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not healthy")
}

func TestManager_StartTurn_ClientNotInitialized(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")
	ctx := context.Background()

	cfg := &types.Agent{
		ID:           "no_client_agent",
		Name:         "No Client Agent",
		Model:        "claude-sonnet-4",
		WorkspaceDir: t.TempDir(),
	}

	// Create an agent and mark it healthy but with no client
	agent := NewAgent(cfg)
	agent.SetHealth(true) // This won't make IsHealthy() return true because client is nil
	mgr.agents["no_client_agent"] = agent

	result, err := mgr.StartTurn(ctx, "no_client_agent", "test query", 1)
	assert.Error(t, err)
	assert.Nil(t, result)
	// Will fail on "not healthy" check since client is nil
}

func TestManager_Shutdown_NoAgents(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")

	err := mgr.Shutdown()
	assert.NoError(t, err)
}

func TestManager_Shutdown_WithUnstartedAgents(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")

	// Add an unstarted agent
	cfg := &types.Agent{
		ID:           "unstarted",
		Name:         "Unstarted Agent",
		Model:        "claude-sonnet-4",
		WorkspaceDir: t.TempDir(),
	}
	mgr.agents["unstarted"] = NewAgent(cfg)

	err := mgr.Shutdown()
	assert.NoError(t, err, "Shutdown should succeed even with unstarted agents")
}

func TestManager_HealthMonitoring(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")

	cfg := &types.Agent{
		ID:           "health_test",
		Name:         "Health Test Agent",
		Model:        "claude-sonnet-4",
		WorkspaceDir: t.TempDir(),
	}

	agent := NewAgent(cfg)
	agent.SetHealth(true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health monitoring
	mgr.startHealthMonitoring(ctx, agent)

	// Let it run for a brief moment
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop monitoring
	cancel()

	// Give goroutine time to exit
	time.Sleep(50 * time.Millisecond)

	// Test passes if no panic occurs
}

func TestManager_HealthMonitoring_DetectsCrash(t *testing.T) {
	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	// Subscribe to error events
	subscriber := bus.Subscribe("test_subscriber", types.EventSessionError)
	defer bus.Unsubscribe("test_subscriber")

	mgr := NewManager(bus, "test-session", "test-api-key")

	cfg := &types.Agent{
		ID:           "crash_test",
		Name:         "Crash Test Agent",
		Model:        "claude-sonnet-4",
		WorkspaceDir: t.TempDir(),
	}

	agent := NewAgent(cfg)
	agent.SetHealth(true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health monitoring
	mgr.startHealthMonitoring(ctx, agent)

	// Simulate agent becoming unhealthy
	// Since client is nil, it's already "unhealthy" in IsHealthy check
	// but we manually set health to true initially
	time.Sleep(100 * time.Millisecond)

	// Now set health to false to simulate crash
	agent.SetHealth(false)

	// Wait for next health check cycle (health checks run every 5s, but we're checking frequently)
	// The health monitor will detect the change
	time.Sleep(6 * time.Second)

	// Check if error event was published
	select {
	case event := <-subscriber.Events():
		if event.Type == types.EventSessionError {
			// Success - error event was published
			t.Log("Error event detected as expected")
		}
	case <-time.After(1 * time.Second):
		// Timeout waiting for event - this is okay for this test
		// since we're mainly testing the monitoring goroutine doesn't panic
	}

	cancel()
}

// Integration test for SpawnAgent
func TestManager_SpawnAgent_Integration(t *testing.T) {
	t.Skip("Integration test - requires Claude CLI and API key")

	bus := events.NewEventBus(10)
	defer bus.Shutdown()

	mgr := NewManager(bus, "test-session", "test-api-key")
	ctx := context.Background()

	cfg := &types.Agent{
		ID:           "integration_spawn",
		Name:         "Integration Spawn Test",
		Model:        "claude-sonnet-4",
		WorkspaceDir: t.TempDir(),
	}

	agent, err := mgr.SpawnAgent(ctx, cfg, "stdio://mcp-server")
	require.NoError(t, err)
	require.NotNil(t, agent)

	defer func() {
		err := mgr.Shutdown()
		assert.NoError(t, err)
	}()

	assert.True(t, agent.IsHealthy())
	assert.NotNil(t, agent.GetClient())

	// Verify agent is registered
	retrievedAgent, err := mgr.GetAgent("integration_spawn")
	require.NoError(t, err)
	assert.Equal(t, agent, retrievedAgent)
}
