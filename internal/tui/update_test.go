package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdate_KeyboardQuit(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())

	// Press 'q' to quit
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Should return quit command
	assert.NotNil(t, cmd)
	assert.IsType(t, Model{}, updatedModel)
}

func TestUpdate_KeyboardEscape(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())

	// Press 'esc' to quit
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should return quit command
	assert.NotNil(t, cmd)
	assert.IsType(t, Model{}, updatedModel)
}

func TestUpdate_KeyboardCtrlC(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.paused = false

	// Press Ctrl+C to pause
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	m := updatedModel.(Model)
	assert.True(t, m.paused)
	assert.NotNil(t, cmd)
}

func TestUpdate_KeyboardUpDown(t *testing.T) {
	tmpDir := t.TempDir()
	session := mockSession(tmpDir)

	model := NewModel(session, mockEventBus())
	agentID := model.activeAgents[0]

	// Set initial scroll offset
	model.agentOutputs[agentID].ScrollOffset = 5

	// Press down arrow - should increase scroll offset
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := updatedModel.(Model)
	assert.Equal(t, 6, m.agentOutputs[agentID].ScrollOffset)

	// Press up arrow - should decrease scroll offset
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updatedModel.(Model)
	assert.Equal(t, 5, m.agentOutputs[agentID].ScrollOffset)

	// Try to go above 0
	m.agentOutputs[agentID].ScrollOffset = 0
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updatedModel.(Model)
	assert.Equal(t, 0, m.agentOutputs[agentID].ScrollOffset) // Should stay at 0
}

func TestUpdate_KeyboardViKeys(t *testing.T) {
	tmpDir := t.TempDir()
	session := mockSession(tmpDir)

	model := NewModel(session, mockEventBus())
	agentID := model.activeAgents[0]

	// Set initial scroll offset
	model.agentOutputs[agentID].ScrollOffset = 3

	// Press 'j' (down) - should increase scroll offset
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m := updatedModel.(Model)
	assert.Equal(t, 4, m.agentOutputs[agentID].ScrollOffset)

	// Press 'k' (up) - should decrease scroll offset
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updatedModel.(Model)
	assert.Equal(t, 3, m.agentOutputs[agentID].ScrollOffset)
}

func TestUpdate_KeyboardLeftRight(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())

	// Should have 2 agents
	assert.Equal(t, 2, len(model.activeAgents))
	assert.Equal(t, 0, model.selectedAgent) // Start at agent 0

	// Press right arrow - should switch to agent 1
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	m := updatedModel.(Model)
	assert.Equal(t, 1, m.selectedAgent)

	// Press left arrow - should switch back to agent 0
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updatedModel.(Model)
	assert.Equal(t, 0, m.selectedAgent)
}

func TestUpdate_KeyboardViPaneSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())

	// Should have 2 agents
	assert.Equal(t, 2, len(model.activeAgents))
	assert.Equal(t, 0, model.selectedAgent) // Start at agent 0
	assert.Equal(t, ViewModeAgents, model.viewMode)

	// Press 'l' (right) - should switch to agent 1
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m := updatedModel.(Model)
	assert.Equal(t, 1, m.selectedAgent)

	// Press 'h' (left) - in agent view, should switch back to agent 0
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updatedModel.(Model)
	assert.Equal(t, 0, m.selectedAgent)
	assert.Equal(t, ViewModeAgents, m.viewMode) // Should still be in agent view
}

func TestUpdate_KeyboardTab(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.md"), []byte("1"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.md"), []byte("2"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file3.md"), []byte("3"), 0600))

	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.selectedFile = 0
	assert.Len(t, model.fileList, 3)

	// Press tab to cycle forward
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	m := updatedModel.(Model)
	assert.Equal(t, 1, m.selectedFile)
	assert.NotNil(t, cmd) // Should load file content

	// Press tab again
	updatedModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(Model)
	assert.Equal(t, 2, m.selectedFile)

	// Press tab to wrap around
	updatedModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(Model)
	assert.Equal(t, 0, m.selectedFile)
}

func TestUpdate_KeyboardShiftTab(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.md"), []byte("1"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.md"), []byte("2"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file3.md"), []byte("3"), 0600))

	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.selectedFile = 0

	// Press shift+tab to cycle backward (should wrap to last file)
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m := updatedModel.(Model)
	assert.Equal(t, 2, m.selectedFile)
	assert.NotNil(t, cmd)
}

func TestUpdate_KeyboardPageUpDown(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.selectedPane = "file"
	model.scrollOffset = 10

	// Press PgUp
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m := updatedModel.(Model)
	assert.Equal(t, 0, m.scrollOffset) // Should go to 0 (10 - 10)

	// Press PgDn
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updatedModel.(Model)
	assert.Equal(t, 10, m.scrollOffset) // Should go to 10 (0 + 10)
}

func TestUpdate_KeyboardHomeEnd(t *testing.T) {
	tmpDir := t.TempDir()
	session := mockSession(tmpDir)
	// Add multiple turns
	for i := 2; i <= 10; i++ {
		session.TurnHistory = append(session.TurnHistory, types.Turn{
			Number:  i,
			AgentID: "agent1",
			Status:  types.TurnCompleted,
		})
	}

	model := NewModel(session, mockEventBus())
	model.selectedPane = "turns"
	model.selectedTurn = 5

	// Press Home
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyHome})
	m := updatedModel.(Model)
	assert.Equal(t, 0, m.selectedTurn)
	assert.Equal(t, 0, m.scrollOffset)

	// Press End
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = updatedModel.(Model)
	assert.Equal(t, 9, m.selectedTurn) // 10 turns, index 9
	assert.Equal(t, 9, m.scrollOffset)
}

func TestUpdate_EventTurnStarted(t *testing.T) {
	tmpDir := t.TempDir()
	bus := mockEventBus()
	model := NewModel(mockSession(tmpDir), bus)
	initialTurns := len(model.turnHistory)

	// Create TurnStarted event
	newTurn := &types.Turn{
		Number:  2,
		AgentID: "agent2",
		Status:  types.TurnInProgress,
	}

	event := events.Event{
		Type: types.EventTurnStarted,
		Payload: events.TurnStartedPayload{
			Turn: newTurn,
		},
	}

	// Handle event
	updatedModel, cmd := model.handleEvent(event)
	m := updatedModel.(Model)

	assert.Len(t, m.turnHistory, initialTurns+1)
	assert.Equal(t, 2, m.turnHistory[len(m.turnHistory)-1].Number)
	assert.Equal(t, initialTurns, m.selectedTurn) // Should auto-select new turn
	assert.NotNil(t, cmd) // Should continue listening for events
}

func TestUpdate_EventTurnCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	bus := mockEventBus()
	session := mockSession(tmpDir)
	session.TurnHistory = []types.Turn{
		{Number: 1, AgentID: "agent1", Status: types.TurnInProgress, TokensUsed: 0, Cost: 0},
	}

	model := NewModel(session, bus)
	model.session.TotalCost = 0
	model.session.TotalTokens = 0

	// Create TurnCompleted event
	completedTurn := &types.Turn{
		Number:     1,
		AgentID:    "agent1",
		Status:     types.TurnCompleted,
		TokensUsed: 1000,
		Cost:       0.05,
		DurationMs: 2000,
	}

	event := events.Event{
		Type: types.EventTurnCompleted,
		Payload: events.TurnCompletedPayload{
			Turn: completedTurn,
		},
	}

	// Handle event
	updatedModel, cmd := model.handleEvent(event)
	m := updatedModel.(Model)

	assert.Equal(t, types.TurnCompleted, m.turnHistory[0].Status)
	assert.Equal(t, 1000, m.turnHistory[0].TokensUsed)
	assert.Equal(t, 0.05, m.turnHistory[0].Cost)
	assert.Equal(t, int64(2000), m.turnHistory[0].DurationMs)
	assert.Equal(t, 1000, m.session.TotalTokens)
	assert.Equal(t, 0.05, m.session.TotalCost)
	assert.NotNil(t, cmd)
}

func TestUpdate_EventFileUpdated(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("original"), 0600))

	bus := mockEventBus()
	model := NewModel(mockSession(tmpDir), bus)
	model.currentFile = "test.md"

	// Update file content
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("updated"), 0600))

	// Create FileUpdated event
	event := events.Event{
		Type: types.EventFileUpdated,
		Payload: events.FileUpdatedPayload{
			Path: "test.md",
		},
	}

	// Handle event
	updatedModel, cmd := model.handleEvent(event)
	m := updatedModel.(Model)

	// Should trigger file reload
	assert.NotNil(t, cmd)
	assert.IsType(t, Model{}, m)
}

func TestUpdate_EventSessionCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	bus := mockEventBus()
	model := NewModel(mockSession(tmpDir), bus)
	model.session.Status = types.SessionRunning

	// Create SessionCompleted event
	event := events.Event{
		Type: types.EventSessionCompleted,
		Payload: events.SessionCompletedPayload{
			DeliverablePath: "deliverable.md",
		},
	}

	// Handle event
	updatedModel, cmd := model.handleEvent(event)
	m := updatedModel.(Model)

	assert.Equal(t, types.SessionCompleted, m.session.Status)
	assert.NotNil(t, m.session.CompletedAt)
	assert.Equal(t, "deliverable.md", m.session.DeliverablePath)
	assert.NotNil(t, cmd)
}

func TestUpdate_EventSessionPaused(t *testing.T) {
	tmpDir := t.TempDir()
	bus := mockEventBus()
	model := NewModel(mockSession(tmpDir), bus)
	model.paused = false

	// Create SessionPaused event
	event := events.Event{
		Type:    types.EventSessionPaused,
		Payload: events.SessionPausedPayload{},
	}

	// Handle event
	updatedModel, cmd := model.handleEvent(event)
	m := updatedModel.(Model)

	assert.Equal(t, types.SessionPaused, m.session.Status)
	assert.True(t, m.paused)
	assert.NotNil(t, cmd)
}

func TestUpdate_EventSessionError(t *testing.T) {
	tmpDir := t.TempDir()
	bus := mockEventBus()
	model := NewModel(mockSession(tmpDir), bus)

	// Create SessionError event
	event := events.Event{
		Type: types.EventSessionError,
		Payload: events.SessionErrorPayload{
			Error: "Test error",
		},
	}

	// Handle event
	updatedModel, cmd := model.handleEvent(event)
	m := updatedModel.(Model)

	assert.Equal(t, types.SessionError, m.session.Status)
	assert.NotNil(t, m.err)
	assert.Equal(t, "Test error", m.err.Error())
	assert.NotNil(t, cmd)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.width = 0
	model.height = 0

	// Send window size message
	updatedModel, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := updatedModel.(Model)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
	assert.Nil(t, cmd)
}

func TestUpdate_FileContentMsg(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.fileContent = ""
	model.currentFile = ""

	// Send file content message
	msg := fileContentMsg{
		content:  "# New Content",
		filename: "test.md",
	}

	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	assert.Equal(t, "# New Content", m.fileContent)
	assert.Equal(t, "test.md", m.currentFile)
	assert.NotNil(t, cmd) // Should continue waiting for events
}

func TestMaxMin(t *testing.T) {
	assert.Equal(t, 5, max(3, 5))
	assert.Equal(t, 5, max(5, 3))
	assert.Equal(t, -1, max(-1, -5))

	assert.Equal(t, 3, min(3, 5))
	assert.Equal(t, 3, min(5, 3))
	assert.Equal(t, -5, min(-1, -5))
}

func TestPauseError(t *testing.T) {
	err := &pauseError{msg: "test error"}
	assert.Equal(t, "test error", err.Error())
}
