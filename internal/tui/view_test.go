package tui

import (
	"testing"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestView_Initial(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.width = 0
	model.height = 0

	view := model.View()

	// Should show initialization message
	assert.Contains(t, view, "Initializing TUI")
}

func TestView_Standard80x24(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.width = 80
	model.height = 24

	view := model.View()

	// Verify all major sections present
	assert.Contains(t, view, "Session:")
	assert.Contains(t, view, "Turn History")
	assert.NotEmpty(t, view)

	// View should not be empty
	assert.Greater(t, len(view), 100, "View should contain substantial content")
}

func TestView_Components(t *testing.T) {
	tmpDir := t.TempDir()
	session := mockSession(tmpDir)
	model := NewModel(session, mockEventBus())
	model.width = 120
	model.height = 30

	view := model.View()

	// Header should contain session info
	assert.Contains(t, view, "Session:")
	assert.Contains(t, view, session.TaskName)

	// Turn history should be visible
	assert.Contains(t, view, "Turn History")

	// Status bar should be included
	// Note: Status bar content depends on component implementation
	assert.NotEmpty(t, view)
}

func TestView_DifferentSizes(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"80x24 standard", 80, 24},
		{"120x30 large", 120, 30},
		{"60x20 small", 60, 20},
		{"100x40 tall", 100, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.width = tt.width
			model.height = tt.height

			view := model.View()

			// Should render without panic
			assert.NotEmpty(t, view)
			assert.Contains(t, view, "Session:")
		})
	}
}

func TestView_SplitPanes(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.width = 80
	model.height = 24

	view := model.View()

	// Both panes should be visible
	assert.Contains(t, view, "Turn History") // Left pane title
	// Right pane should show file viewer content
	assert.NotEmpty(t, view)
}

func TestView_PaneActivation(t *testing.T) {
	tmpDir := t.TempDir()
	model := NewModel(mockSession(tmpDir), mockEventBus())
	model.width = 80
	model.height = 24

	// Test with turns pane active
	model.selectedPane = "turns"
	viewTurns := model.View()
	assert.NotEmpty(t, viewTurns)

	// Test with file pane active
	model.selectedPane = "file"
	viewFile := model.View()
	assert.NotEmpty(t, viewFile)

	// Views should be different (different border styles for active pane)
	// This is a basic check - detailed styling is tested in component tests
	assert.NotEqual(t, viewTurns, viewFile)
}

func TestView_WithContent(t *testing.T) {
	tmpDir := t.TempDir()
	session := mockSession(tmpDir)
	model := NewModel(session, mockEventBus())
	model.width = 100
	model.height = 30
	model.fileContent = "# Test File\n\nThis is test content."

	view := model.View()

	// Verify view renders without errors
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Session:")
}

func TestView_NoTurns(t *testing.T) {
	tmpDir := t.TempDir()
	session := mockSession(tmpDir)
	session.TurnHistory = []types.Turn{} // Empty turn history

	model := NewModel(session, mockEventBus())
	model.width = 80
	model.height = 24

	view := model.View()

	// Should render gracefully with no turns
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Turn History")
}

func TestView_ManyTurns(t *testing.T) {
	tmpDir := t.TempDir()
	session := mockSession(tmpDir)

	// Add many turns
	for i := 2; i <= 50; i++ {
		session.TurnHistory = append(session.TurnHistory, types.Turn{
			Number:  i,
			AgentID: "agent1",
			Status:  types.TurnCompleted,
		})
	}

	model := NewModel(session, mockEventBus())
	model.width = 80
	model.height = 24

	view := model.View()

	// Should render without issue
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Turn History")
}

func TestView_Snapshot(t *testing.T) {
	tmpDir := t.TempDir()
	session := mockSession(tmpDir)
	model := NewModel(session, mockEventBus())
	model.width = 80
	model.height = 24
	model.fileContent = "# Shared Context\n\nTest content here."

	view := model.View()

	// Snapshot assertions - verify key elements are present
	assert.Contains(t, view, "Session:", "Header should contain session label")
	assert.Contains(t, view, session.TaskName, "Header should contain task name")
	assert.Contains(t, view, "Turn History", "Should show turn history pane")

	// Basic structure validation
	assert.NotEmpty(t, view, "View should not be empty")
	assert.Greater(t, len(view), 200, "View should have substantial content")
}

func TestView_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		panics bool
	}{
		{"zero dimensions", 0, 0, false},
		{"zero width", 0, 24, false},
		{"zero height", 80, 0, false},
		{"very small", 20, 10, false},
		{"very large", 500, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			model := NewModel(mockSession(tmpDir), mockEventBus())
			model.width = tt.width
			model.height = tt.height

			if tt.panics {
				assert.Panics(t, func() {
					model.View()
				})
			} else {
				assert.NotPanics(t, func() {
					view := model.View()
					assert.NotEmpty(t, view)
				})
			}
		})
	}
}
