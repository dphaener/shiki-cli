package storage

import (
	"testing"
	"time"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStatePersistence(t *testing.T) {
	wsDir := t.TempDir()
	now := time.Now()
	startedAt := now.Add(1 * time.Minute)

	original := &types.Session{
		ID:               "20251123-120000-abcd",
		TaskName:         "Test Collaboration",
		TaskTemplatePath: "/path/to/template.md",
		WorkspaceDir:     wsDir,
		Status:           types.SessionRunning,
		CreatedAt:        now,
		StartedAt:        &startedAt,
		CurrentTurn:      5,
		MaxTurns:         10,
		TotalCost:        1.23,
		TotalTokens:      5000,
		Agent1: types.Agent{
			ID:           "agent_1",
			Name:         "Architect",
			Role:         "Designer",
			SystemPrompt: "You are an architect",
			Model:        "claude-3-5-sonnet-20241022",
			WorkspaceDir: wsDir,
			MemoryFile:   "memory/agent_1_memory.md",
			TotalTurns:   3,
			TotalTokens:  2500,
			TotalCost:    0.62,
		},
		Agent2: types.Agent{
			ID:           "agent_2",
			Name:         "Reviewer",
			Role:         "Critic",
			SystemPrompt: "You are a reviewer",
			Model:        "claude-3-5-sonnet-20241022",
			WorkspaceDir: wsDir,
			MemoryFile:   "memory/agent_2_memory.md",
			TotalTurns:   2,
			TotalTokens:  2500,
			TotalCost:    0.61,
		},
		TurnHistory: []types.Turn{
			{
				Number:       0,
				AgentID:      "agent_1",
				StartedAt:    now,
				Status:       types.TurnCompleted,
				TokensUsed:   1000,
				Cost:         0.20,
				ToolCalls:    2,
				MessagesSent: 1,
			},
		},
	}

	// Save
	err := SaveSessionState(original)
	require.NoError(t, err)

	// Load
	loaded, err := LoadSessionState(wsDir)
	require.NoError(t, err)

	// Verify exact state restoration
	assert.Equal(t, original.ID, loaded.ID)
	assert.Equal(t, original.TaskName, loaded.TaskName)
	assert.Equal(t, original.Status, loaded.Status)
	assert.Equal(t, original.CurrentTurn, loaded.CurrentTurn)
	assert.Equal(t, original.MaxTurns, loaded.MaxTurns)
	assert.Equal(t, original.TotalCost, loaded.TotalCost)
	assert.Equal(t, original.TotalTokens, loaded.TotalTokens)
	assert.Equal(t, original.Agent1.Name, loaded.Agent1.Name)
	assert.Equal(t, original.Agent1.TotalTurns, loaded.Agent1.TotalTurns)
	assert.Equal(t, original.Agent2.Name, loaded.Agent2.Name)
	assert.Len(t, loaded.TurnHistory, 1)
	assert.Equal(t, original.TurnHistory[0].Number, loaded.TurnHistory[0].Number)
	assert.Equal(t, original.TurnHistory[0].TokensUsed, loaded.TurnHistory[0].TokensUsed)
}

func TestSessionStateVersion(t *testing.T) {
	wsDir := t.TempDir()

	session := &types.Session{
		ID:           "20251123-120000-version",
		WorkspaceDir: wsDir,
		Status:       types.SessionRunning,
		CreatedAt:    time.Now(),
	}

	// Save
	err := SaveSessionState(session)
	require.NoError(t, err)

	// Load and verify version is included
	loaded, err := LoadSessionState(wsDir)
	require.NoError(t, err)
	assert.Equal(t, session.ID, loaded.ID)
}

func TestLoadSessionStateMissing(t *testing.T) {
	wsDir := t.TempDir()

	_, err := LoadSessionState(wsDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read session state")
}

func TestSessionStateAtomicWrite(t *testing.T) {
	wsDir := t.TempDir()

	session := &types.Session{
		ID:           "20251123-120000-atomic",
		WorkspaceDir: wsDir,
		Status:       types.SessionRunning,
		CreatedAt:    time.Now(),
		CurrentTurn:  1,
	}

	// Save initial
	err := SaveSessionState(session)
	require.NoError(t, err)

	// Update and save again
	session.CurrentTurn = 2
	err = SaveSessionState(session)
	require.NoError(t, err)

	// Load and verify latest value
	loaded, err := LoadSessionState(wsDir)
	require.NoError(t, err)
	assert.Equal(t, 2, loaded.CurrentTurn)
}
