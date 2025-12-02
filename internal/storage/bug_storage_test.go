package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dphaener/shiki-cli/pkg/types"
)

func TestSetupBugSession(t *testing.T) {
	// For this test, we'll manually create the directory structure
	// since we're testing the core functionality
	tempDir := t.TempDir()

	// Create bugs directory manually for testing
	bugsDir := filepath.Join(tempDir, "bugs")
	err := os.MkdirAll(bugsDir, 0o755)
	require.NoError(t, err)

	// Override config.GetDataDir during test by setting environment
	t.Setenv("XDG_DATA_HOME", tempDir)

	cfg := BugSetupConfig{
		Title:       "Login fails on mobile",
		Description: "Users cannot login when accessing the app from mobile devices",
	}

	result, err := SetupBugSession(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result structure
	assert.NotEmpty(t, result.BugID)
	assert.NotEmpty(t, result.BugDir)
	assert.NotEmpty(t, result.PlanFile)
	assert.NotEmpty(t, result.TasksFile)

	// Verify directory was created
	assert.DirExists(t, result.BugDir)

	// Verify plan file was created
	assert.FileExists(t, result.PlanFile)

	// Verify plan file content
	content, err := os.ReadFile(result.PlanFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), cfg.Title)
	assert.Contains(t, string(content), cfg.Description)
	assert.Contains(t, string(content), "bug_id:")
	assert.Contains(t, string(content), "# Bug Fix Plan:")
}

func TestSaveAndLoadBugSession(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create initial session
	cfg := BugSetupConfig{
		Title:       "API timeout issues",
		Description: "API requests are timing out under heavy load",
	}

	setupResult, err := SetupBugSession(cfg)
	require.NoError(t, err)

	// Create a bug session
	session := &types.BugSession{
		ID:           setupResult.BugID,
		Title:        cfg.Title,
		Description:  cfg.Description,
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     setupResult.PlanFile,
		TasksFile:    setupResult.TasksFile,
		BugDir:       setupResult.BugDir,
		Checkpoints:  make(map[types.BugPhase]*types.PhaseCheckpoint),
		CurrentPlan:  "# Test plan content",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       types.SessionRunning,
	}

	// Save session
	err = SaveBugSession(session)
	require.NoError(t, err)

	// Verify metadata file was created
	metaPath := GetBugMetaPath(setupResult.BugID)
	assert.FileExists(t, metaPath)

	// Load session
	loadedSession, err := LoadBugSession(setupResult.BugID)
	require.NoError(t, err)
	require.NotNil(t, loadedSession)

	// Verify loaded session matches
	assert.Equal(t, session.ID, loadedSession.ID)
	assert.Equal(t, session.Title, loadedSession.Title)
	assert.Equal(t, session.Description, loadedSession.Description)
	assert.Equal(t, session.CurrentPhase, loadedSession.CurrentPhase)
	assert.Equal(t, session.PlanFile, loadedSession.PlanFile)
	assert.Equal(t, session.TasksFile, loadedSession.TasksFile)
	assert.Equal(t, session.BugDir, loadedSession.BugDir)
	assert.Equal(t, session.Status, loadedSession.Status)
}

func TestBugExists(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	bugID := "test-bug-123"

	// Initially should not exist
	assert.False(t, BugExists(bugID))

	// Create a bug session
	cfg := BugSetupConfig{
		Title:       "Test bug",
		Description: "Test description",
	}
	setupResult, err := SetupBugSession(cfg)
	require.NoError(t, err)

	session := &types.BugSession{
		ID:           setupResult.BugID,
		Title:        cfg.Title,
		Description:  cfg.Description,
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     setupResult.PlanFile,
		TasksFile:    setupResult.TasksFile,
		BugDir:       setupResult.BugDir,
		Checkpoints:  make(map[types.BugPhase]*types.PhaseCheckpoint),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       types.SessionRunning,
	}

	err = SaveBugSession(session)
	require.NoError(t, err)

	// Now should exist
	assert.True(t, BugExists(setupResult.BugID))
}

func TestListActiveBugs(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Initially should be empty
	bugs, err := ListActiveBugs()
	require.NoError(t, err)
	assert.Empty(t, bugs)

	// Create first bug (active)
	cfg1 := BugSetupConfig{
		Title:       "Bug 1",
		Description: "First bug",
	}
	result1, err := SetupBugSession(cfg1)
	require.NoError(t, err)

	session1 := &types.BugSession{
		ID:           result1.BugID,
		Title:        cfg1.Title,
		Description:  cfg1.Description,
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     result1.PlanFile,
		TasksFile:    result1.TasksFile,
		BugDir:       result1.BugDir,
		Checkpoints:  make(map[types.BugPhase]*types.PhaseCheckpoint),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       types.SessionRunning,
	}

	err = SaveBugSession(session1)
	require.NoError(t, err)

	// Create second bug (completed - should not be in active list)
	cfg2 := BugSetupConfig{
		Title:       "Bug 2",
		Description: "Second bug",
	}
	result2, err := SetupBugSession(cfg2)
	require.NoError(t, err)

	completedTime := time.Now()
	session2 := &types.BugSession{
		ID:           result2.BugID,
		Title:        cfg2.Title,
		Description:  cfg2.Description,
		CurrentPhase: types.BugPhaseComplete,
		PlanFile:     result2.PlanFile,
		TasksFile:    result2.TasksFile,
		BugDir:       result2.BugDir,
		Checkpoints:  make(map[types.BugPhase]*types.PhaseCheckpoint),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CompletedAt:  &completedTime,
		Status:       types.SessionCompleted,
	}

	err = SaveBugSession(session2)
	require.NoError(t, err)
	t.Logf("Saved session2 with ID: %s", session2.ID)

	// Should only return active bugs
	activeBugs, err := ListActiveBugs()
	require.NoError(t, err)

	// Debug: Check what we actually got
	t.Logf("Found %d active bugs", len(activeBugs))
	for i, bug := range activeBugs {
		t.Logf("Bug %d: ID=%s, Phase=%s", i, bug.ID, bug.CurrentPhase)
	}

	require.Len(t, activeBugs, 1, "Expected 1 active bug, but got %d", len(activeBugs))
	assert.Equal(t, session1.ID, activeBugs[0].ID)
	assert.Equal(t, session1.Title, activeBugs[0].Title)
}

func TestSaveBugCheckpoint(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create a bug session
	cfg := BugSetupConfig{
		Title:       "Test bug",
		Description: "Test description",
	}
	setupResult, err := SetupBugSession(cfg)
	require.NoError(t, err)

	session := &types.BugSession{
		ID:           setupResult.BugID,
		Title:        cfg.Title,
		Description:  cfg.Description,
		CurrentPhase: types.BugPhaseTasks,
		PlanFile:     setupResult.PlanFile,
		TasksFile:    setupResult.TasksFile,
		BugDir:       setupResult.BugDir,
		Checkpoints:  make(map[types.BugPhase]*types.PhaseCheckpoint),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       types.SessionRunning,
	}

	// Save checkpoint
	chatHistory := []types.ChatMessage{
		{
			Role:      "user",
			Content:   "Test message",
			Timestamp: time.Now(),
		},
	}

	err = SaveBugCheckpoint(session, types.BugPhasePlan, chatHistory, "bug-plan.md")
	require.NoError(t, err)

	// Verify checkpoint was saved
	checkpoint := GetBugCheckpoint(session, types.BugPhasePlan)
	require.NotNil(t, checkpoint)
	assert.Equal(t, types.WorkflowPhase(types.BugPhasePlan), checkpoint.Phase)
	assert.Equal(t, "bug-plan.md", checkpoint.ArtifactRef)
	assert.Len(t, checkpoint.ChatHistory, 1)
	assert.Equal(t, "Test message", checkpoint.ChatHistory[0].Content)
}

func TestHasActiveBug(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Initially should have no active bugs
	hasActive, err := HasActiveBug()
	require.NoError(t, err)
	assert.False(t, hasActive)

	// Create an active bug
	cfg := BugSetupConfig{
		Title:       "Active bug",
		Description: "An active bug",
	}
	setupResult, err := SetupBugSession(cfg)
	require.NoError(t, err)

	session := &types.BugSession{
		ID:           setupResult.BugID,
		Title:        cfg.Title,
		Description:  cfg.Description,
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     setupResult.PlanFile,
		TasksFile:    setupResult.TasksFile,
		BugDir:       setupResult.BugDir,
		Checkpoints:  make(map[types.BugPhase]*types.PhaseCheckpoint),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       types.SessionRunning,
	}

	err = SaveBugSession(session)
	require.NoError(t, err)

	// Now should have an active bug
	hasActive, err = HasActiveBug()
	require.NoError(t, err)
	assert.True(t, hasActive)
}

func TestGetActiveBug(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Initially should have no active bugs
	_, err := GetActiveBug()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active bug workflows")

	// Create an active bug
	cfg := BugSetupConfig{
		Title:       "Active bug",
		Description: "An active bug",
	}
	setupResult, err := SetupBugSession(cfg)
	require.NoError(t, err)

	session := &types.BugSession{
		ID:           setupResult.BugID,
		Title:        cfg.Title,
		Description:  cfg.Description,
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     setupResult.PlanFile,
		TasksFile:    setupResult.TasksFile,
		BugDir:       setupResult.BugDir,
		Checkpoints:  make(map[types.BugPhase]*types.PhaseCheckpoint),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       types.SessionRunning,
	}

	err = SaveBugSession(session)
	require.NoError(t, err)

	// Should return the active bug
	activeBug, err := GetActiveBug()
	require.NoError(t, err)
	require.NotNil(t, activeBug)
	assert.Equal(t, session.ID, activeBug.ID)
	assert.Equal(t, session.Title, activeBug.Title)
}

func TestBugTasksFileExists(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	bugID := "test-bug-tasks"

	// Initially should not exist
	assert.False(t, BugTasksFileExists(bugID))

	// Create tasks file
	tasksPath := GetBugTasksPath(bugID)
	err := os.MkdirAll(filepath.Dir(tasksPath), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(tasksPath, []byte("# Tasks\n\n- Task 1\n- Task 2"), 0o644)
	require.NoError(t, err)

	// Now should exist
	assert.True(t, BugTasksFileExists(bugID))
}

func TestLoadBugTasksContent(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	bugID := "test-bug-tasks-content"
	expectedContent := "# Bug Tasks\n\n- Fix login validation\n- Update error messages\n- Add tests"

	// Create tasks file
	tasksPath := GetBugTasksPath(bugID)
	err := os.MkdirAll(filepath.Dir(tasksPath), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(tasksPath, []byte(expectedContent), 0o644)
	require.NoError(t, err)

	// Load content
	content, err := LoadBugTasksContent(bugID)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, content)

	// Test loading non-existent file
	_, err = LoadBugTasksContent("non-existent-bug")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read bug-tasks.md")
}

func TestGetBugPaths(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	bugID := "test-bug-paths"

	// Test path generation
	bugsDir := GetBugsDir()
	bugDir := GetBugDir(bugID)
	metaPath := GetBugMetaPath(bugID)
	planPath := GetBugPlanPath(bugID)
	tasksPath := GetBugTasksPath(bugID)

	// Verify paths
	assert.Equal(t, filepath.Join(tempDir, "collab-cli", "bugs"), bugsDir)
	assert.Equal(t, filepath.Join(tempDir, "collab-cli", "bugs", bugID), bugDir)
	assert.Equal(t, filepath.Join(tempDir, "collab-cli", "bugs", bugID, "bug-workflow.json"), metaPath)
	assert.Equal(t, filepath.Join(tempDir, "collab-cli", "bugs", bugID, "bug-plan.md"), planPath)
	assert.Equal(t, filepath.Join(tempDir, "collab-cli", "bugs", bugID, "bug-tasks.md"), tasksPath)
}