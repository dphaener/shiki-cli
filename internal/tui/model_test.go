package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventBus creates a mock EventBus for testing
func mockEventBus() *events.EventBus {
	return events.NewEventBus(100)
}

// mockSession creates a mock session for testing
func mockSession(workspaceDir string) *types.Session {
	now := time.Now()
	session := &types.Session{
		ID:           "test-session-123",
		TaskName:     "Test Task",
		WorkspaceDir: workspaceDir,
		Status:       types.SessionRunning,
		CreatedAt:    now,
		StartedAt:    &now,
		CurrentTurn:  1,
		MaxTurns:     10,
		TotalTokens:  1000,
		TotalCost:    0.05,
		TurnHistory: []types.Turn{
			{
				Number:     1,
				AgentID:    "agent1",
				Status:     types.TurnCompleted,
				TokensUsed: 500,
				Cost:       0.025,
				DurationMs: 1500,
			},
		},
	}
	return session
}

func TestNewModel(t *testing.T) {
	// Create temporary workspace with some files
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "shared_context.md"), []byte("test"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "notes.md"), []byte("notes"), 0600))

	bus := mockEventBus()
	session := mockSession(tmpDir)

	model := NewModel(session, bus)

	// Verify basic initialization
	assert.Equal(t, session, model.session)
	assert.Equal(t, tmpDir, model.workspaceDir)
	assert.Equal(t, session.TurnHistory, model.turnHistory)
	assert.NotNil(t, model.eventSub)

	// Verify UI state
	assert.Equal(t, "turns", model.selectedPane)
	assert.Equal(t, 0, model.selectedTurn) // Should select last turn (index 0 for 1 turn)
	assert.False(t, model.paused)
	assert.Nil(t, model.err)

	// Verify file list
	assert.Contains(t, model.fileList, "shared_context.md")
	assert.Contains(t, model.fileList, "notes.md")

	// Verify currentFile is set to shared_context.md
	assert.Equal(t, "shared_context.md", model.currentFile)
}

func TestNewModel_NoSharedContext(t *testing.T) {
	// Create temporary workspace without shared_context.md
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "notes.md"), []byte("notes"), 0600))

	bus := mockEventBus()
	session := mockSession(tmpDir)

	model := NewModel(session, bus)

	// Should default to first file in list
	assert.Equal(t, "notes.md", model.currentFile)
}

func TestNewModel_EmptyWorkspace(t *testing.T) {
	// Create empty workspace
	tmpDir := t.TempDir()

	bus := mockEventBus()
	session := mockSession(tmpDir)

	model := NewModel(session, bus)

	// Should handle empty workspace gracefully
	assert.Empty(t, model.fileList)
	assert.Equal(t, "shared_context.md", model.currentFile) // Still set default even if it doesn't exist
}

func TestNewModel_EventSubscription(t *testing.T) {
	tmpDir := t.TempDir()
	bus := mockEventBus()
	session := mockSession(tmpDir)

	model := NewModel(session, bus)

	// Verify event subscription was created
	assert.NotNil(t, model.eventSub)
	// Subscriber has private ID field, so we just verify it's not nil
}

func TestGetWorkspaceFiles(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(dir string)
		expectedFiles []string
	}{
		{
			name: "single file",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "test.md"), []byte("test"), 0600)
			},
			expectedFiles: []string{"test.md"},
		},
		{
			name: "multiple files",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0600)
				os.WriteFile(filepath.Join(dir, "b.md"), []byte("b"), 0600)
			},
			expectedFiles: []string{"a.md", "b.md"},
		},
		{
			name: "nested directory",
			setup: func(dir string) {
				os.Mkdir(filepath.Join(dir, "subdir"), 0700)
				os.WriteFile(filepath.Join(dir, "root.md"), []byte("root"), 0600)
				os.WriteFile(filepath.Join(dir, "subdir", "nested.md"), []byte("nested"), 0600)
			},
			expectedFiles: []string{"root.md", "subdir/nested.md"},
		},
		{
			name: "hidden files excluded",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "visible.md"), []byte("visible"), 0600)
				os.WriteFile(filepath.Join(dir, ".hidden"), []byte("hidden"), 0600)
			},
			expectedFiles: []string{"visible.md"},
		},
		{
			name:          "empty directory",
			setup:         func(dir string) {},
			expectedFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(tmpDir)

			files := getWorkspaceFiles(tmpDir)

			if len(tt.expectedFiles) == 0 {
				assert.Empty(t, files)
			} else {
				for _, expected := range tt.expectedFiles {
					assert.Contains(t, files, expected)
				}
				assert.Len(t, files, len(tt.expectedFiles))
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"first item", []string{"x", "y", "z"}, "x", true},
		{"last item", []string{"x", "y", "z"}, "z", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected int
	}{
		{"found at index 1", []string{"a", "b", "c"}, "b", 1},
		{"found at index 0", []string{"a", "b", "c"}, "a", 0},
		{"found at index 2", []string{"a", "b", "c"}, "c", 2},
		{"not found returns 0", []string{"a", "b", "c"}, "d", 0},
		{"empty slice returns 0", []string{}, "a", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadFileContent(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name            string
		filename        string
		content         string
		expectError     bool
		expectedContent string
	}{
		{
			name:            "valid file",
			filename:        "test.md",
			content:         "# Test Content\n\nHello, world!",
			expectError:     false,
			expectedContent: "# Test Content\n\nHello, world!",
		},
		{
			name:        "nonexistent file",
			filename:    "missing.md",
			expectError: true,
		},
		{
			name:            "empty file",
			filename:        "empty.md",
			content:         "",
			expectError:     false,
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.content != "" || !tt.expectError {
				err := os.WriteFile(filepath.Join(tmpDir, tt.filename), []byte(tt.content), 0600)
				require.NoError(t, err)
			}

			// Execute command
			cmd := loadFileContent(tmpDir, tt.filename)
			msg := cmd()

			// Verify result
			fileMsg, ok := msg.(fileContentMsg)
			require.True(t, ok)

			if tt.expectError {
				assert.Contains(t, fileMsg.content, "Error reading file")
			} else {
				assert.Equal(t, tt.expectedContent, fileMsg.content)
				assert.Equal(t, tt.filename, fileMsg.filename)
			}
		})
	}
}

func TestLoadFileContent_EmptyFilename(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := loadFileContent(tmpDir, "")
	msg := cmd()

	fileMsg, ok := msg.(fileContentMsg)
	require.True(t, ok)
	assert.Equal(t, "No file selected", fileMsg.content)
}
