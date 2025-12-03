package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/tasks"
	"github.com/dphaener/shiki-cli/pkg/types"
)

func TestTaskProgressFileFormatConsistency(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid tasks.md file
	tasksContent := `### WP01: Authentication Setup
**Priority**: P0
**Goal**: Set up user authentication system

#### Subtasks
- [ ] T001: Create user model
- [ ] T002: Implement login endpoint
- [ ] T003: Add password hashing

### WP02: Frontend Integration
**Priority**: P1
**Goal**: Connect frontend to authentication

#### Subtasks
- [ ] T004: Create login form
- [ ] T005: Handle authentication state
`
	tasksFile := filepath.Join(tempDir, "tasks.md")
	err = os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write tasks.md: %v", err)
	}

	// Create a workflow session
	session := &types.WorkflowSession{
		ID:         "integration-test",
		FeatureDir: tempDir,
		TasksFile:  tasksFile,
	}

	// Create event bus
	eventBus := events.NewEventBus(10)
	defer eventBus.Shutdown()

	// Create and initialize orchestrator
	orchestrator := NewImplementOrchestrator(session, "test-api-key", eventBus)
	err = orchestrator.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}
	defer orchestrator.Stop()

	// Verify task progress is enabled
	if !orchestrator.IsTaskProgressEnabled() {
		t.Fatal("Task progress should be enabled")
	}

	// Check that progress file was created
	progressFile := filepath.Join(tempDir, "task-progress.md")
	if _, err := os.Stat(progressFile); os.IsNotExist(err) {
		t.Fatal("Progress file should be created")
	}

	// Read the progress file content
	content, err := os.ReadFile(progressFile)
	if err != nil {
		t.Fatalf("Failed to read progress file: %v", err)
	}

	// Parse the content using the tasks parser
	progressManager := tasks.NewProgressFileManager(progressFile)
	err = progressManager.Validate()
	if err != nil {
		t.Errorf("Progress file validation failed: %v", err)
		t.Logf("Progress file content:\n%s", string(content))
	}

	// Test that the file can be loaded into a new ProgressTracker
	parser := tasks.NewTasksParser()
	taskStructure, err := parser.ParseTasksFile(tasksFile)
	if err != nil {
		t.Fatalf("Failed to re-parse tasks file: %v", err)
	}

	newTracker := tasks.NewProgressTracker(taskStructure, progressFile)
	if newTracker == nil {
		t.Error("Failed to create new progress tracker from saved file")
	}

	// Verify all expected tasks are present
	progress := orchestrator.GetTaskProgress()
	if progress == nil {
		t.Fatal("Progress tracker should not be nil")
	}

	// Check we have the expected number of tasks
	workPackages := progress.GetWorkPackages()
	if len(workPackages) != 2 { // 2 work packages
		t.Errorf("Expected 2 work packages, got %d", len(workPackages))
	}

	// Verify work package structure
	wp1Found := false
	wp2Found := false
	totalTasks := 0

	for _, wp := range workPackages {
		if wp.ID == "WP01" {
			wp1Found = true
			if wp.Name != "Authentication Setup" {
				t.Errorf("WP01 name mismatch: expected 'Authentication Setup', got '%s'", wp.Name)
			}
			totalTasks += len(wp.Tasks)
		} else if wp.ID == "WP02" {
			wp2Found = true
			if wp.Name != "Frontend Integration" {
				t.Errorf("WP02 name mismatch: expected 'Frontend Integration', got '%s'", wp.Name)
			}
			totalTasks += len(wp.Tasks)
		}
	}

	if !wp1Found {
		t.Error("WP01 not found in progress data")
	}
	if !wp2Found {
		t.Error("WP02 not found in progress data")
	}

	if totalTasks != 5 { // 3 + 2 tasks
		t.Errorf("Expected 5 total tasks, got %d", totalTasks)
	}
}

func TestEventBridgeIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "event_integration_test")
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
		ID:         "event-test",
		FeatureDir: tempDir,
		TasksFile:  tasksFile,
	}

	// Create event bus
	eventBus := events.NewEventBus(10)
	defer eventBus.Shutdown()

	// Create and initialize orchestrator
	orchestrator := NewImplementOrchestrator(session, "test-api-key", eventBus)
	err = orchestrator.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}
	defer orchestrator.Stop()

	// Start task progress tracking
	err = orchestrator.StartTaskProgress()
	if err != nil {
		t.Fatalf("Failed to start task progress: %v", err)
	}

	// Simulate tool invocation event
	toolEvent := events.NewToolInvoked("Edit", "agent-1", "event-test", 1, map[string]interface{}{
		"file_path": "test.go",
		"content":   "Starting work on T001: Test task 1",
	})

	// Publish the event
	eventBus.Publish(toolEvent)

	// Give some time for event processing
	time.Sleep(100 * time.Millisecond)

	// Simulate assistant response event
	responseEvent := events.NewAssistantResponse("event-test", "orchestrator", "agent-1", "conv-1",
		"I have completed T001: Test task 1 successfully", 50, 1000, "claude-3", nil)

	// Publish the event
	eventBus.Publish(responseEvent)

	// Give some time for event processing
	time.Sleep(100 * time.Millisecond)

	// Stop task progress to force save
	err = orchestrator.StopTaskProgress()
	if err != nil {
		t.Errorf("Failed to stop task progress: %v", err)
	}

	// Verify that the progress file exists and has been updated
	progressFile := filepath.Join(tempDir, "task-progress.md")
	content, err := os.ReadFile(progressFile)
	if err != nil {
		t.Fatalf("Failed to read progress file: %v", err)
	}

	// The file should contain the initial structure
	contentStr := string(content)
	if contentStr == "" {
		t.Error("Progress file should not be empty after event processing")
	}

	// Validate the file format is still correct
	progressManager := tasks.NewProgressFileManager(progressFile)
	err = progressManager.Validate()
	if err != nil {
		t.Errorf("Progress file validation failed after event processing: %v", err)
		t.Logf("Progress file content:\n%s", contentStr)
	}
}

func TestErrorRecoveryScenarios(t *testing.T) {
	tests := []struct {
		name        string
		tasksContent string
		expectError bool
		description string
	}{
		{
			name: "empty_tasks_file",
			tasksContent: "",
			expectError: true,
			description: "Empty tasks file should disable task progress",
		},
		{
			name: "invalid_task_format",
			tasksContent: `### Invalid
This is not a valid tasks file format
- Invalid task line without proper format
`,
			expectError: true,
			description: "Invalid task format should disable task progress",
		},
		{
			name: "no_work_packages",
			tasksContent: `# Just a title
Some content without work packages.
`,
			expectError: true,
			description: "No work packages should disable task progress",
		},
		{
			name: "valid_minimal",
			tasksContent: `### WP01: Minimal Package
**Priority**: P0
**Goal**: Minimal test

#### Subtasks
- [ ] T001: One task
`,
			expectError: false,
			description: "Minimal valid format should enable task progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "error_recovery_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create tasks.md file with test content
			tasksFile := filepath.Join(tempDir, "tasks.md")
			err = os.WriteFile(tasksFile, []byte(tt.tasksContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write tasks.md: %v", err)
			}

			// Create a workflow session
			session := &types.WorkflowSession{
				ID:         "error-test-" + tt.name,
				FeatureDir: tempDir,
				TasksFile:  tasksFile,
			}

			// Create event bus
			eventBus := events.NewEventBus(10)
			defer eventBus.Shutdown()

			// Create and initialize orchestrator
			orchestrator := NewImplementOrchestrator(session, "test-api-key", eventBus)
			err = orchestrator.Initialize()
			if err != nil {
				t.Fatalf("Failed to initialize orchestrator: %v", err)
			}
			defer orchestrator.Stop()

			// Check if task progress is enabled/disabled as expected
			isEnabled := orchestrator.IsTaskProgressEnabled()
			if tt.expectError && isEnabled {
				t.Errorf("%s: expected task progress to be disabled due to %s", tt.name, tt.description)
			} else if !tt.expectError && !isEnabled {
				t.Errorf("%s: expected task progress to be enabled for %s", tt.name, tt.description)
			}
		})
	}
}

func TestConcurrentFileAccessAndErrorRecovery(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "concurrent_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid tasks.md file
	tasksContent := `### WP01: Concurrent Test Package
**Priority**: P0
**Goal**: Test concurrent access handling

#### Subtasks
- [ ] T001: Test task 1
- [ ] T002: Test task 2
`
	tasksFile := filepath.Join(tempDir, "tasks.md")
	err = os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write tasks.md: %v", err)
	}

	// Create a workflow session template for concurrent orchestrators
	sessionTemplate := &types.WorkflowSession{
		ID:         "concurrent-test",
		FeatureDir: tempDir,
		TasksFile:  tasksFile,
	}
	_ = sessionTemplate

	// Create event bus
	eventBus := events.NewEventBus(10)
	defer eventBus.Shutdown()

	// Test concurrent initialization of multiple orchestrators
	numOrchestrators := 3
	orchestrators := make([]*ImplementOrchestrator, numOrchestrators)
	errors := make(chan error, numOrchestrators)

	for i := 0; i < numOrchestrators; i++ {
		go func(index int) {
			// Each orchestrator gets a unique session ID but same feature dir
			testSession := &types.WorkflowSession{
				ID:         fmt.Sprintf("concurrent-test-%d", index),
				FeatureDir: tempDir,
				TasksFile:  tasksFile,
			}

			orch := NewImplementOrchestrator(testSession, "test-api-key", eventBus)
			orchestrators[index] = orch
			errors <- orch.Initialize()
		}(i)
	}

	// Wait for all initializations to complete
	successCount := 0
	for i := 0; i < numOrchestrators; i++ {
		err := <-errors
		if err == nil {
			successCount++
		} else {
			t.Logf("Orchestrator %d initialization failed (may be expected): %v", i, err)
		}
	}

	// At least one should succeed
	if successCount == 0 {
		t.Fatal("All orchestrators failed to initialize - at least one should succeed")
	}

	t.Logf("Successfully initialized %d out of %d orchestrators", successCount, numOrchestrators)

	// Clean up successfully initialized orchestrators
	for _, orch := range orchestrators {
		if orch != nil && orch.IsTaskProgressEnabled() {
			orch.Stop()
		}
	}
}

func TestEventBridgeErrorHandling(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "event_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid tasks.md file
	tasksContent := `### WP01: Error Test Package
**Priority**: P0
**Goal**: Test error handling

#### Subtasks
- [ ] T001: Test task
`
	tasksFile := filepath.Join(tempDir, "tasks.md")
	err = os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write tasks.md: %v", err)
	}

	// Create a workflow session
	session := &types.WorkflowSession{
		ID:         "error-test",
		FeatureDir: tempDir,
		TasksFile:  tasksFile,
	}

	// Create event bus
	eventBus := events.NewEventBus(10)
	defer eventBus.Shutdown()

	// Create and initialize orchestrator
	orchestrator := NewImplementOrchestrator(session, "test-api-key", eventBus)
	err = orchestrator.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}
	defer orchestrator.Stop()

	// Start task progress tracking
	err = orchestrator.StartTaskProgress()
	if err != nil {
		t.Fatalf("Failed to start task progress: %v", err)
	}

	// Test malformed events (should not crash)
	malformedEvent := events.Event{
		Type:      types.EventToolInvoked,
		SessionID: "", // Empty session ID
		Payload:   nil, // Nil payload
	}
	eventBus.Publish(malformedEvent)

	// Test events with wrong session ID (should be ignored)
	wrongSessionEvent := events.NewToolInvoked("Edit", "agent-1", "wrong-session", 1, map[string]interface{}{
		"file_path": "test.go",
	})
	eventBus.Publish(wrongSessionEvent)

	// Test events with malformed payload (should not crash)
	badPayloadEvent := events.Event{
		Type:      types.EventAssistantResponse,
		SessionID: session.ID,
		Payload:   "not a struct", // Wrong payload type
	}
	eventBus.Publish(badPayloadEvent)

	// Give some time for event processing
	time.Sleep(200 * time.Millisecond)

	// Stop task progress (should not hang)
	err = orchestrator.StopTaskProgress()
	if err != nil {
		t.Errorf("Failed to stop task progress: %v", err)
	}

	// Verify progress file still exists and is valid
	progressFile := filepath.Join(tempDir, "task-progress.md")
	if _, err := os.Stat(progressFile); err != nil {
		t.Errorf("Progress file should still exist after error handling: %v", err)
	}
}