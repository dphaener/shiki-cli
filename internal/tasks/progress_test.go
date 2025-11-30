package tasks

import (
	"testing"
	"time"
)

func createTestTaskStructure() *TaskStructure {
	return &TaskStructure{
		WorkPackages: []WorkPackage{
			{
				ID:       "WP01",
				Name:     "Infrastructure",
				Priority: "P0",
				Goal:     "Setup basic infrastructure",
				Tasks: []Task{
					{ID: "T001", Description: "Initialize project"},
					{ID: "T002", Description: "Setup database"},
					{ID: "T003", Description: "Create API endpoints"},
				},
			},
			{
				ID:           "WP02",
				Name:         "Features",
				Priority:     "P1",
				Goal:         "Implement core features",
				Dependencies: []string{"WP01"},
				Tasks: []Task{
					{ID: "T004", Description: "User authentication"},
					{ID: "T005", Description: "Data processing"},
				},
			},
		},
		TaskMap: make(map[string]*Task),
		WPMap:   make(map[string]*WorkPackage),
	}
}

func TestProgressTracker_Initialization(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	// Check that all work packages are initialized
	workPackages := tracker.GetWorkPackages()
	if len(workPackages) != 2 {
		t.Errorf("expected 2 work packages, got %d", len(workPackages))
	}

	// Check initial state
	for _, wp := range workPackages {
		if wp.Status != TaskStatePending {
			t.Errorf("expected work package %s to be pending, got %s", wp.ID, wp.Status)
		}

		for _, task := range wp.Tasks {
			if task.Status != TaskStatePending {
				t.Errorf("expected task %s to be pending, got %s", task.ID, task.Status)
			}
		}
	}
}

func TestProgressTracker_UpdateTaskStatus(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	// Test updating task to in progress
	err := tracker.UpdateTaskStatus("T001", TaskStateInProgress, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	task, exists := tracker.GetTask("T001")
	if !exists {
		t.Fatal("task T001 should exist")
	}
	if task.Status != TaskStateInProgress {
		t.Errorf("expected task status to be in_progress, got %s", task.Status)
	}
	if task.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}

	// Test updating task to completed
	err = tracker.UpdateTaskStatus("T001", TaskStateCompleted, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	task, exists = tracker.GetTask("T001")
	if !exists {
		t.Fatal("task T001 should exist")
	}
	if task.Status != TaskStateCompleted {
		t.Errorf("expected task status to be completed, got %s", task.Status)
	}
	if task.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestProgressTracker_UpdateNonExistentTask(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	err := tracker.UpdateTaskStatus("T999", TaskStateCompleted, "")
	if err == nil {
		t.Error("expected error when updating non-existent task")
	}
}

func TestProgressTracker_WorkPackageStatusAggregation(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	// Complete all tasks in WP01
	tracker.UpdateTaskStatus("T001", TaskStateCompleted, "")
	tracker.UpdateTaskStatus("T002", TaskStateCompleted, "")
	tracker.UpdateTaskStatus("T003", TaskStateCompleted, "")

	wp, exists := tracker.GetWorkPackage("WP01")
	if !exists {
		t.Fatal("work package WP01 should exist")
	}
	if wp.Status != TaskStateCompleted {
		t.Errorf("expected work package status to be completed, got %s", wp.Status)
	}
	if wp.CompletedAt == nil {
		t.Error("expected work package CompletedAt to be set")
	}
}

func TestProgressTracker_BlockedTask(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	reason := "Missing API credentials"
	err := tracker.UpdateTaskStatus("T001", TaskStateBlocked, reason)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	task, exists := tracker.GetTask("T001")
	if !exists {
		t.Fatal("task T001 should exist")
	}
	if task.Status != TaskStateBlocked {
		t.Errorf("expected task status to be blocked, got %s", task.Status)
	}
	if task.BlockReason != reason {
		t.Errorf("expected block reason '%s', got '%s'", reason, task.BlockReason)
	}
}

func TestProgressTracker_SkippedTask(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	reason := "Feature not needed anymore"
	err := tracker.MarkTaskAsSkipped("T001", reason)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	task, exists := tracker.GetTask("T001")
	if !exists {
		t.Fatal("task T001 should exist")
	}
	if task.Status != TaskStateSkipped {
		t.Errorf("expected task status to be skipped, got %s", task.Status)
	}
	if task.BlockReason != reason {
		t.Errorf("expected skip reason '%s', got '%s'", reason, task.BlockReason)
	}
	if task.CompletedAt == nil {
		t.Error("expected CompletedAt to be set for skipped task")
	}
}

func TestProgressTracker_GetProgress(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	// Initial state
	progress := tracker.GetProgress()
	if progress.TotalTasks != 5 {
		t.Errorf("expected 5 total tasks, got %d", progress.TotalTasks)
	}
	if progress.CompletedTasks != 0 {
		t.Errorf("expected 0 completed tasks, got %d", progress.CompletedTasks)
	}
	if progress.OverallProgress != 0 {
		t.Errorf("expected 0%% progress, got %.2f%%", progress.OverallProgress)
	}

	// Complete some tasks
	tracker.UpdateTaskStatus("T001", TaskStateCompleted, "")
	tracker.UpdateTaskStatus("T002", TaskStateInProgress, "")
	tracker.UpdateTaskStatus("T003", TaskStateBlocked, "reason")

	progress = tracker.GetProgress()
	if progress.CompletedTasks != 1 {
		t.Errorf("expected 1 completed task, got %d", progress.CompletedTasks)
	}
	if progress.InProgressTasks != 1 {
		t.Errorf("expected 1 in progress task, got %d", progress.InProgressTasks)
	}
	if progress.BlockedTasks != 1 {
		t.Errorf("expected 1 blocked task, got %d", progress.BlockedTasks)
	}
	if progress.OverallProgress != 20.0 {
		t.Errorf("expected 20%% progress, got %.2f%%", progress.OverallProgress)
	}
}

func TestProgressTracker_AddDiscoveredTask(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	// Add a new task to existing work package
	err := tracker.AddDiscoveredTask("WP01", "T999", "Newly discovered task")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	wp, exists := tracker.GetWorkPackage("WP01")
	if !exists {
		t.Fatal("work package WP01 should exist")
	}

	// Should now have 4 tasks instead of 3
	if len(wp.Tasks) != 4 {
		t.Errorf("expected 4 tasks in WP01, got %d", len(wp.Tasks))
	}

	// Check the new task
	task, exists := tracker.GetTask("T999")
	if !exists {
		t.Fatal("newly added task T999 should exist")
	}
	if task.Description != "Newly discovered task" {
		t.Errorf("expected description 'Newly discovered task', got '%s'", task.Description)
	}
	if task.Status != TaskStatePending {
		t.Errorf("expected new task to be pending, got %s", task.Status)
	}

	// Test adding task to non-existent work package
	err = tracker.AddDiscoveredTask("WP999", "T888", "Another task")
	if err == nil {
		t.Error("expected error when adding task to non-existent work package")
	}

	// Test adding duplicate task
	err = tracker.AddDiscoveredTask("WP01", "T999", "Duplicate task")
	if err == nil {
		t.Error("expected error when adding duplicate task")
	}
}

func TestProgressTracker_SetTaskOutput(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	output := "Task completed successfully with output"
	err := tracker.SetTaskOutput("T001", output)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	task, exists := tracker.GetTask("T001")
	if !exists {
		t.Fatal("task T001 should exist")
	}
	if task.Output != output {
		t.Errorf("expected output '%s', got '%s'", output, task.Output)
	}

	// Test setting output for non-existent task
	err = tracker.SetTaskOutput("T999", "output")
	if err == nil {
		t.Error("expected error when setting output for non-existent task")
	}
}

func TestTaskProgressState_IsValid(t *testing.T) {
	testCases := []struct {
		state   TaskProgressState
		isValid bool
	}{
		{TaskStatePending, true},
		{TaskStateInProgress, true},
		{TaskStateCompleted, true},
		{TaskStateBlocked, true},
		{TaskStateSkipped, true},
		{TaskProgressState("invalid"), false},
		{TaskProgressState(""), false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.state), func(t *testing.T) {
			if tc.state.IsValid() != tc.isValid {
				t.Errorf("expected IsValid() to be %v for state %s", tc.isValid, tc.state)
			}
		})
	}
}

func TestProgressTracker_ConcurrentAccess(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	// Test concurrent updates
	done := make(chan bool, 3)

	// Goroutine 1: Update task statuses
	go func() {
		for i := 0; i < 10; i++ {
			tracker.UpdateTaskStatus("T001", TaskStateInProgress, "")
			tracker.UpdateTaskStatus("T001", TaskStateCompleted, "")
		}
		done <- true
	}()

	// Goroutine 2: Get progress
	go func() {
		for i := 0; i < 10; i++ {
			tracker.GetProgress()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 3: Get work packages
	go func() {
		for i := 0; i < 10; i++ {
			tracker.GetWorkPackages()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// If we get here without a race condition, the test passes
}

func TestWorkPackageProgress_StatusTransitions(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	wpID := "WP01"

	// Start with pending
	wp, _ := tracker.GetWorkPackage(wpID)
	if wp.Status != TaskStatePending {
		t.Errorf("expected initial status to be pending, got %s", wp.Status)
	}

	// Start first task - should make WP in progress
	tracker.UpdateTaskStatus("T001", TaskStateInProgress, "")
	wp, _ = tracker.GetWorkPackage(wpID)
	if wp.Status != TaskStateInProgress {
		t.Errorf("expected WP status to be in_progress after starting task, got %s", wp.Status)
	}
	if wp.StartedAt == nil {
		t.Error("expected WP StartedAt to be set")
	}

	// Block the only active task - should make WP blocked
	tracker.UpdateTaskStatus("T001", TaskStateBlocked, "test reason")
	wp, _ = tracker.GetWorkPackage(wpID)
	if wp.Status != TaskStateBlocked {
		t.Errorf("expected WP status to be blocked, got %s", wp.Status)
	}

	// Complete all tasks - should make WP completed
	tracker.UpdateTaskStatus("T001", TaskStateCompleted, "")
	tracker.UpdateTaskStatus("T002", TaskStateCompleted, "")
	tracker.UpdateTaskStatus("T003", TaskStateCompleted, "")
	wp, _ = tracker.GetWorkPackage(wpID)
	if wp.Status != TaskStateCompleted {
		t.Errorf("expected WP status to be completed, got %s", wp.Status)
	}
	if wp.CompletedAt == nil {
		t.Error("expected WP CompletedAt to be set")
	}
}