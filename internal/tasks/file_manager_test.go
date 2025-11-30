package tasks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgressFileManager_GenerateProgressMarkdown(t *testing.T) {
	// Create test data
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	// Update some task statuses to test different states
	tracker.UpdateTaskStatus("T001", TaskStateInProgress, "")
	tracker.UpdateTaskStatus("T002", TaskStateCompleted, "")
	tracker.UpdateTaskStatus("T003", TaskStateBlocked, "Missing credentials")

	pfm := NewProgressFileManager("/tmp/progress.md")
	content := pfm.generateProgressMarkdown(tracker)

	// Check that content contains expected sections
	expectedSections := []string{
		"# Implementation Progress",
		"**Status**:",
		"**Updated**:",
		"**Progress**:",
		"## Work Package WP01:",
		"## Work Package WP02:",
		"- [~] T001: Initialize project (in_progress",
		"- [x] T002: Setup database (completed",
		"- [!] T003: Create API endpoints (blocked",
		"**Reason**: Missing credentials",
	}

	for _, expected := range expectedSections {
		if !strings.Contains(content, expected) {
			t.Errorf("expected content to contain '%s', but it didn't. Content:\n%s", expected, content)
		}
	}
}

func TestProgressFileManager_GetTaskCheckbox(t *testing.T) {
	pfm := NewProgressFileManager("/tmp/progress.md")

	testCases := []struct {
		status   TaskProgressState
		expected string
	}{
		{TaskStatePending, "[ ]"},
		{TaskStateInProgress, "[~]"},
		{TaskStateCompleted, "[x]"},
		{TaskStateBlocked, "[!]"},
		{TaskStateSkipped, "[s]"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.status), func(t *testing.T) {
			result := pfm.getTaskCheckbox(tc.status)
			if result != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestProgressFileManager_DetermineOverallStatus(t *testing.T) {
	pfm := NewProgressFileManager("/tmp/progress.md")

	testCases := []struct {
		name     string
		progress ProgressSummary
		expected string
	}{
		{
			name: "not started",
			progress: ProgressSummary{
				TotalTasks:      0,
				CompletedTasks:  0,
				InProgressTasks: 0,
			},
			expected: "not_started",
		},
		{
			name: "completed",
			progress: ProgressSummary{
				TotalTasks:      5,
				CompletedTasks:  5,
				InProgressTasks: 0,
				BlockedTasks:    0,
			},
			expected: "completed",
		},
		{
			name: "in progress",
			progress: ProgressSummary{
				TotalTasks:      5,
				CompletedTasks:  2,
				InProgressTasks: 1,
				BlockedTasks:    0,
			},
			expected: "in_progress",
		},
		{
			name: "blocked",
			progress: ProgressSummary{
				TotalTasks:      5,
				CompletedTasks:  2,
				InProgressTasks: 0,
				BlockedTasks:    1,
			},
			expected: "blocked",
		},
		{
			name: "pending",
			progress: ProgressSummary{
				TotalTasks:      5,
				CompletedTasks:  0,
				InProgressTasks: 0,
				BlockedTasks:    0,
			},
			expected: "pending",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pfm.determineOverallStatus(tc.progress)
			if result != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestProgressFileManager_WriteAndRead(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	progressFile := filepath.Join(tempDir, "progress.md")

	// Create test data
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, progressFile)

	// Update some task statuses
	tracker.UpdateTaskStatus("T001", TaskStateCompleted, "")
	tracker.SetTaskOutput("T001", "Successfully completed initialization")

	pfm := NewProgressFileManager(progressFile)

	// Test writing
	err := pfm.WriteProgressFile(tracker)
	if err != nil {
		t.Fatalf("failed to write progress file: %v", err)
	}

	// Check that file exists
	if !pfm.Exists() {
		t.Fatal("progress file should exist after writing")
	}

	// Test validation
	err = pfm.Validate()
	if err != nil {
		t.Errorf("progress file should be valid: %v", err)
	}

	// Read and check content
	content, err := os.ReadFile(progressFile)
	if err != nil {
		t.Fatalf("failed to read progress file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "# Implementation Progress") {
		t.Error("progress file should contain header")
	}
	if !strings.Contains(contentStr, "[x] T001: Initialize project") {
		t.Error("progress file should show completed task")
	}
}

func TestProgressFileManager_CreateInitialProgressFile(t *testing.T) {
	// Create temporary files
	tempDir := t.TempDir()
	tasksFile := filepath.Join(tempDir, "tasks.md")
	progressFile := filepath.Join(tempDir, "progress.md")

	// Create a test tasks.md file
	tasksContent := `# Implementation Tasks

## Work Packages

### WP01: Core Infrastructure
**Priority**: P0
**Goal**: Create the foundational components
**Dependencies**: None

#### Subtasks
- [ ] T001: Initialize project structure
- [ ] T002: Setup database
- [ ] T003: Create API endpoints
`

	err := os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test tasks file: %v", err)
	}

	pfm := NewProgressFileManager(progressFile)

	// Test creating initial progress file
	err = pfm.CreateInitialProgressFile(tasksFile)
	if err != nil {
		t.Fatalf("failed to create initial progress file: %v", err)
	}

	// Verify the file was created
	if !pfm.Exists() {
		t.Fatal("progress file should exist after creation")
	}

	// Verify content
	content, err := os.ReadFile(progressFile)
	if err != nil {
		t.Fatalf("failed to read created progress file: %v", err)
	}

	contentStr := string(content)
	expectedContent := []string{
		"# Implementation Progress",
		"WP01: Core Infrastructure",
		"[ ] T001: Initialize project structure",
		"[ ] T002: Setup database",
		"[ ] T003: Create API endpoints",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected content to contain '%s', content:\n%s", expected, contentStr)
		}
	}
}

func TestProgressFileManager_BackupProgressFile(t *testing.T) {
	tempDir := t.TempDir()
	progressFile := filepath.Join(tempDir, "progress.md")

	// Create initial progress file
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, progressFile)
	pfm := NewProgressFileManager(progressFile)

	err := pfm.WriteProgressFile(tracker)
	if err != nil {
		t.Fatalf("failed to write initial progress file: %v", err)
	}

	// Test backup
	backupPath, err := pfm.BackupProgressFile()
	if err != nil {
		t.Fatalf("failed to backup progress file: %v", err)
	}

	// Check that backup was created
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("backup file should exist")
	}

	// Check that backup contains same content
	originalContent, err := os.ReadFile(progressFile)
	if err != nil {
		t.Fatalf("failed to read original file: %v", err)
	}

	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}

	if string(originalContent) != string(backupContent) {
		t.Error("backup content should match original content")
	}

	// Test backup of non-existent file
	pfm2 := NewProgressFileManager("/non/existent/path/progress.md")
	_, err = pfm2.BackupProgressFile()
	if err == nil {
		t.Error("should get error when backing up non-existent file")
	}
}

func TestProgressFileManager_Validation(t *testing.T) {
	tempDir := t.TempDir()
	progressFile := filepath.Join(tempDir, "progress.md")
	pfm := NewProgressFileManager(progressFile)

	// Test validation of non-existent file
	err := pfm.Validate()
	if err == nil {
		t.Error("should get error when validating non-existent file")
	}

	// Create invalid file
	err = os.WriteFile(progressFile, []byte("This is not a valid progress file"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	err = pfm.Validate()
	if err == nil {
		t.Error("should get error when validating invalid progress file")
	}

	// Create valid file
	validContent := "# Implementation Progress\n\nSome content here"
	err = os.WriteFile(progressFile, []byte(validContent), 0644)
	if err != nil {
		t.Fatalf("failed to write valid test file: %v", err)
	}

	err = pfm.Validate()
	if err != nil {
		t.Errorf("should not get error when validating valid progress file: %v", err)
	}
}

func TestProgressFileManager_FilePaths(t *testing.T) {
	progressFile := "/path/to/progress.md"
	pfm := NewProgressFileManager(progressFile)

	if pfm.GetFilePath() != progressFile {
		t.Errorf("expected file path '%s', got '%s'", progressFile, pfm.GetFilePath())
	}

	expectedDir := "/path/to"
	if pfm.GetProgressFileDir() != expectedDir {
		t.Errorf("expected dir '%s', got '%s'", expectedDir, pfm.GetProgressFileDir())
	}
}

func TestProgressFileManager_MarkdownGeneration_WithTimestamps(t *testing.T) {
	taskStructure := createTestTaskStructure()
	tracker := NewProgressTracker(taskStructure, "/tmp/progress.md")

	// Manually set timestamps to test formatting
	tracker.UpdateTaskStatus("T001", TaskStateInProgress, "")
	// Update the tracker's internal state to have specific timestamps
	tracker.UpdateTaskStatus("T001", TaskStateCompleted, "")

	pfm := NewProgressFileManager("/tmp/progress.md")
	content := pfm.generateProgressMarkdown(tracker)

	// Check that timestamps are included in the output
	if !strings.Contains(content, "completed at") {
		t.Error("expected content to include completion timestamp")
	}

	// Check that the content includes proper formatting
	if !strings.Contains(content, "[x] T001:") {
		t.Error("expected content to show completed task with checkbox")
	}
}