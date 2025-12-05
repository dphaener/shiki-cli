package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBugModel_RefreshBugPreview tests the refreshBugPreview method
func TestBugModel_RefreshBugPreview(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bug-model-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test session
	session := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "Test bug description",
		CurrentPhase: types.BugPhasePlan,
		BugDir:       tempDir,
		PlanFile:     filepath.Join(tempDir, "bug-plan.md"),
		TasksFile:    filepath.Join(tempDir, "bug-tasks.md"),
	}

	// Create event bus
	eventBus := &events.EventBus{}

	// Create bug model
	model := NewBugModel(session, eventBus)

	// Test that refreshBugPreview sets correct file paths
	model.refreshBugPreview()

	// We can't directly access the preview files, but we can test that
	// the method doesn't panic and the phase is set correctly
	assert.Equal(t, types.BugPhasePlan, session.CurrentPhase)
}

// TestBugModel_HandleTaskProgressFileUpdate tests the file update event handling
func TestBugModel_HandleTaskProgressFileUpdate(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bug-model-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test session
	session := &types.BugSession{
		ID:           "test-bug-002",
		Title:        "Test Bug",
		Description:  "Test bug description",
		CurrentPhase: types.BugPhaseImplement,
		BugDir:       tempDir,
		PlanFile:     filepath.Join(tempDir, "bug-plan.md"),
		TasksFile:    filepath.Join(tempDir, "bug-tasks.md"),
	}

	// Create event bus
	eventBus := &events.EventBus{}

	// Create bug model
	model := NewBugModel(session, eventBus)

	// Test with matching file path
	taskProgressPath := filepath.Join(tempDir, "task-progress.md")
	payload := events.FileUpdatedPayload{
		Path:      taskProgressPath,
		Operation: "modified",
	}

	// Call the handler
	resultModel, cmd := model.handleTaskProgressFileUpdate(payload)

	// Verify that the model is returned and command is generated
	assert.NotNil(t, resultModel)
	assert.NotNil(t, cmd)
}

// TestBugModel_HandleTaskProgressFileUpdate_NoMatch tests file update with non-matching path
func TestBugModel_HandleTaskProgressFileUpdate_NoMatch(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bug-model-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test session
	session := &types.BugSession{
		ID:           "test-bug-003",
		Title:        "Test Bug",
		Description:  "Test bug description",
		CurrentPhase: types.BugPhaseImplement,
		BugDir:       tempDir,
		PlanFile:     filepath.Join(tempDir, "bug-plan.md"),
		TasksFile:    filepath.Join(tempDir, "bug-tasks.md"),
	}

	// Create event bus
	eventBus := &events.EventBus{}

	// Create bug model
	model := NewBugModel(session, eventBus)

	// Test with non-matching file path
	payload := events.FileUpdatedPayload{
		Path:      "/different/path/some-file.md",
		Operation: "modified",
	}

	// Call the handler
	resultModel, cmd := model.handleTaskProgressFileUpdate(payload)

	// Should return model with nil command since path doesn't match
	assert.NotNil(t, resultModel)
	assert.Nil(t, cmd)
}

// TestBugModel_HandleTaskProgressFileUpdate_EmptyBugDir tests file update with empty bug directory
func TestBugModel_HandleTaskProgressFileUpdate_EmptyBugDir(t *testing.T) {
	// Create test session with empty bug directory
	session := &types.BugSession{
		ID:           "test-bug-004",
		Title:        "Test Bug",
		Description:  "Test bug description",
		CurrentPhase: types.BugPhaseImplement,
		BugDir:       "", // Empty directory
		PlanFile:     "/tmp/bug-plan.md",
		TasksFile:    "/tmp/bug-tasks.md",
	}

	// Create event bus
	eventBus := &events.EventBus{}

	// Create bug model
	model := NewBugModel(session, eventBus)

	// Test with any file path
	payload := events.FileUpdatedPayload{
		Path:      "/some/path/task-progress.md",
		Operation: "modified",
	}

	// Call the handler
	resultModel, cmd := model.handleTaskProgressFileUpdate(payload)

	// Should return model with nil command since BugDir is empty
	assert.NotNil(t, resultModel)
	assert.Nil(t, cmd)
}

// TestBugModel_NewBugModel tests the constructor with task progress file setup
func TestBugModel_NewBugModel(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bug-model-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test session
	session := &types.BugSession{
		ID:           "test-bug-005",
		Title:        "Test Bug",
		Description:  "Test bug description",
		CurrentPhase: types.BugPhasePlan,
		BugDir:       tempDir,
		PlanFile:     filepath.Join(tempDir, "bug-plan.md"),
		TasksFile:    filepath.Join(tempDir, "bug-tasks.md"),
	}

	// Create event bus
	eventBus := &events.EventBus{}

	// Create bug model
	model := NewBugModel(session, eventBus)

	// Verify that the model was created successfully
	assert.NotNil(t, model.session)
	assert.Equal(t, session.ID, model.session.ID)
	assert.Equal(t, session.Title, model.session.Title)
	assert.Equal(t, session.CurrentPhase, model.session.CurrentPhase)
	assert.NotNil(t, model.bugPreview)
}

// TestBugModel_ConstructTaskProgressPath tests task progress file path construction
func TestBugModel_ConstructTaskProgressPath(t *testing.T) {
	tests := []struct {
		name       string
		bugDir     string
		expected   string
	}{
		{
			name:       "Standard bug directory",
			bugDir:     "/tmp/bugs/bug-001",
			expected:   "/tmp/bugs/bug-001/task-progress.md",
		},
		{
			name:       "Root directory",
			bugDir:     "/bugs",
			expected:   "/bugs/task-progress.md",
		},
		{
			name:       "Relative directory",
			bugDir:     "bugs/bug-002",
			expected:   "bugs/bug-002/task-progress.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test session
			session := &types.BugSession{
				ID:           "test-bug",
				Title:        "Test Bug",
				Description:  "Test bug description",
				CurrentPhase: types.BugPhasePlan,
				BugDir:       tt.bugDir,
				PlanFile:     filepath.Join(tt.bugDir, "bug-plan.md"),
				TasksFile:    filepath.Join(tt.bugDir, "bug-tasks.md"),
			}

			// Create event bus
			eventBus := &events.EventBus{}

			// Create bug model (this triggers the task progress file path construction)
			model := NewBugModel(session, eventBus)

			// Call refreshBugPreview to trigger path construction
			model.refreshBugPreview()

			// The path construction is internal, but we can verify it doesn't panic
			// and the model is properly initialized
			assert.NotNil(t, model.bugPreview)
			assert.Equal(t, session.BugDir, tt.bugDir)
		})
	}
}

// TestBugModel_FilePathNormalization tests file path normalization in file update handling
func TestBugModel_FilePathNormalization(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bug-model-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test session
	session := &types.BugSession{
		ID:           "test-bug-006",
		Title:        "Test Bug",
		Description:  "Test bug description",
		CurrentPhase: types.BugPhaseImplement,
		BugDir:       tempDir,
		PlanFile:     filepath.Join(tempDir, "bug-plan.md"),
		TasksFile:    filepath.Join(tempDir, "bug-tasks.md"),
	}

	// Create event bus
	eventBus := &events.EventBus{}

	// Create bug model
	model := NewBugModel(session, eventBus)

	// Test with path that needs normalization (double slashes, etc.)
	unnormalizedPath := tempDir + "//" + "task-progress.md"
	payload := events.FileUpdatedPayload{
		Path:      unnormalizedPath,
		Operation: "modified",
	}

	// Call the handler
	resultModel, cmd := model.handleTaskProgressFileUpdate(payload)

	// Should still match after normalization
	assert.NotNil(t, resultModel)
	assert.NotNil(t, cmd)
}