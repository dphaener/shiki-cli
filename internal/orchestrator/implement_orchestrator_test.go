package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
)

func TestImplementOrchestrator_TaskProgressEnabled(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "implement_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid tasks.md file
	tasksContent := `### WP01: Test Work Package
**Priority**: P0
**Goal**: Test goal

#### Subtasks
- [ ] T001: Test task 1
- [ ] T002: Test task 2
`
	tasksFile := filepath.Join(tempDir, "tasks.md")
	err = os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write tasks.md: %v", err)
	}

	// Create a workflow session
	session := &types.WorkflowSession{
		ID:         "test-session",
		FeatureDir: tempDir,
		TasksFile:  tasksFile,
	}

	// Create event bus
	eventBus := events.NewEventBus(10)
	defer eventBus.Shutdown()

	// Create orchestrator
	orchestrator := NewImplementOrchestrator(session, "test-api-key", eventBus)

	// Initially task progress should be disabled
	if orchestrator.IsTaskProgressEnabled() {
		t.Error("Task progress should be disabled before initialization")
	}

	// Initialize orchestrator
	err = orchestrator.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// After initialization, task progress should be enabled
	if !orchestrator.IsTaskProgressEnabled() {
		t.Error("Task progress should be enabled after initialization with valid tasks file")
	}

	// Check that progress tracker is available
	tracker := orchestrator.GetTaskProgress()
	if tracker == nil {
		t.Error("Progress tracker should be available when task progress is enabled")
	}

	// Check that progress file was created
	progressFile := filepath.Join(tempDir, "task-progress.md")
	if _, err := os.Stat(progressFile); os.IsNotExist(err) {
		t.Error("Progress file should be created during initialization")
	}

	// Clean up
	err = orchestrator.Stop()
	if err != nil {
		t.Errorf("Failed to stop orchestrator: %v", err)
	}
}

func TestImplementOrchestrator_TaskProgressDisabledWithoutTasks(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "implement_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a workflow session without tasks.md
	session := &types.WorkflowSession{
		ID:         "test-session",
		FeatureDir: tempDir,
		TasksFile:  filepath.Join(tempDir, "nonexistent-tasks.md"),
	}

	// Create event bus
	eventBus := events.NewEventBus(10)
	defer eventBus.Shutdown()

	// Create orchestrator
	orchestrator := NewImplementOrchestrator(session, "test-api-key", eventBus)

	// Initialize orchestrator
	err = orchestrator.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// Task progress should be disabled due to missing tasks file
	if orchestrator.IsTaskProgressEnabled() {
		t.Error("Task progress should be disabled when tasks.md doesn't exist")
	}

	// Progress tracker should be nil
	tracker := orchestrator.GetTaskProgress()
	if tracker != nil {
		t.Error("Progress tracker should be nil when task progress is disabled")
	}

	// Clean up
	err = orchestrator.Stop()
	if err != nil {
		t.Errorf("Failed to stop orchestrator: %v", err)
	}
}

func TestImplementOrchestrator_TaskProgressLifecycle(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "implement_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid tasks.md file
	tasksContent := `### WP01: Test Work Package
**Priority**: P0
**Goal**: Test goal

#### Subtasks
- [ ] T001: Test task 1
`
	tasksFile := filepath.Join(tempDir, "tasks.md")
	err = os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write tasks.md: %v", err)
	}

	// Create a workflow session
	session := &types.WorkflowSession{
		ID:         "test-session",
		FeatureDir: tempDir,
		TasksFile:  tasksFile,
	}

	// Create event bus
	eventBus := events.NewEventBus(10)
	defer eventBus.Shutdown()

	// Create orchestrator
	orchestrator := NewImplementOrchestrator(session, "test-api-key", eventBus)

	// Initialize orchestrator
	err = orchestrator.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// Task progress should be enabled
	if !orchestrator.IsTaskProgressEnabled() {
		t.Fatal("Task progress should be enabled")
	}

	// Start task progress
	err = orchestrator.StartTaskProgress()
	if err != nil {
		t.Errorf("Failed to start task progress: %v", err)
	}

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Stop task progress
	err = orchestrator.StopTaskProgress()
	if err != nil {
		t.Errorf("Failed to stop task progress: %v", err)
	}

	// Clean up
	err = orchestrator.Stop()
	if err != nil {
		t.Errorf("Failed to stop orchestrator: %v", err)
	}
}

func TestImplementOrchestrator_InitializePreservesBaseCapabilities(t *testing.T) {
	// Create a workflow session
	session := &types.WorkflowSession{
		ID: "test-session",
	}

	// Create event bus
	eventBus := events.NewEventBus(10)
	defer eventBus.Shutdown()

	// Create orchestrator
	orchestrator := NewImplementOrchestrator(session, "test-api-key", eventBus)

	// Check that it implements PhaseOrchestrator interface
	var _ PhaseOrchestrator = orchestrator

	// Test basic orchestrator methods are available
	if orchestrator.GetSession() != session {
		t.Error("GetSession should return the original session")
	}

	// Initialize should not fail even without task progress
	err := orchestrator.Initialize()
	if err != nil {
		t.Errorf("Initialize should not fail: %v", err)
	}

	// Clean up
	err = orchestrator.Stop()
	if err != nil {
		t.Errorf("Failed to stop orchestrator: %v", err)
	}
}