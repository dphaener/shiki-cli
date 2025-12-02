package agent

import (
	"context"
	"testing"

	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	cfg := &types.Agent{
		ID:           "test_agent",
		Name:         "Test Agent",
		Role:         "Tester",
		SystemPrompt: "You are a test agent",
		Model:        "claude-sonnet-4",
		WorkspaceDir: "/tmp/test",
	}

	agent := NewAgent(cfg)

	assert.Equal(t, "test_agent", agent.ID)
	assert.Equal(t, "Test Agent", agent.Name)
	assert.Equal(t, "Tester", agent.Role)
	assert.Equal(t, "You are a test agent", agent.SystemPrompt)
	assert.Equal(t, "claude-sonnet-4", agent.Model)
	assert.Equal(t, "/tmp/test", agent.WorkspaceDir)
	assert.False(t, agent.IsHealthy(), "New agent should not be healthy until started")
	assert.Nil(t, agent.GetClient(), "New agent should have nil client")
}

func TestAgent_IsHealthy(t *testing.T) {
	agent := NewAgent(&types.Agent{
		ID:    "test",
		Name:  "Test",
		Model: "claude-sonnet-4",
	})

	// Initially not healthy
	assert.False(t, agent.IsHealthy())

	// Set healthy
	agent.SetHealth(true)
	assert.False(t, agent.IsHealthy(), "Should be false because client is nil")

	// Note: We can't easily test with a real client without mocking the SDK
	// The SDK integration tests would require actual Claude API access
}

func TestAgent_SetHealth(t *testing.T) {
	agent := NewAgent(&types.Agent{
		ID:    "test",
		Name:  "Test",
		Model: "claude-sonnet-4",
	})

	agent.SetHealth(true)
	// Even though we set health to true, IsHealthy returns false because client is nil
	assert.False(t, agent.IsHealthy())

	agent.SetHealth(false)
	assert.False(t, agent.IsHealthy())
}

func TestAgent_Stop_WhenNotStarted(t *testing.T) {
	agent := NewAgent(&types.Agent{
		ID:    "test",
		Name:  "Test",
		Model: "claude-sonnet-4",
	})

	// Stopping an agent that was never started should not error
	err := agent.Stop()
	require.NoError(t, err)
}

func TestAgent_GetClient(t *testing.T) {
	agent := NewAgent(&types.Agent{
		ID:    "test",
		Name:  "Test",
		Model: "claude-sonnet-4",
	})

	// Client should be nil before Start
	assert.Nil(t, agent.GetClient())
}

// TestAgent_Start_Integration would test actual SDK integration
// This requires:
// - ANTHROPIC_API_KEY environment variable
// - Actual Claude CLI in PATH
// - Running MCP server
// Skipping for unit tests, should be in integration test suite
func TestAgent_Start_Integration(t *testing.T) {
	t.Skip("Integration test - requires Claude CLI and API key")

	cfg := &types.Agent{
		ID:           "integration_test",
		Name:         "Integration Test Agent",
		Role:         "Tester",
		SystemPrompt: "You are a test agent",
		Model:        "claude-sonnet-4",
		WorkspaceDir: t.TempDir(),
	}

	agent := NewAgent(cfg)

	ctx := context.Background()
	apiKey := "test-api-key"

	err := agent.Start(ctx, apiKey, cfg.ID)
	require.NoError(t, err)

	defer func() {
		err := agent.Stop()
		assert.NoError(t, err)
	}()

	assert.True(t, agent.IsHealthy())
	assert.NotNil(t, agent.GetClient())
}
