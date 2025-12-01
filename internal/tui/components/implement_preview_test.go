package components

import (
	"fmt"
	"testing"
	"time"

	"github.com/darinhaener/collab/internal/tasks"
	"github.com/stretchr/testify/assert"
)

// Test data fixtures for markdown content parsing
const (
	// Valid markdown content with multiple work packages
	validMarkdownContent = `# Implementation Progress

**Status**: In Progress
**Updated**: 2025-11-30T10:15:30Z
**Progress**: 3/8 tasks completed (37.5%)

## Work Package WP01: Authentication Setup (in_progress)

**Started**: 2025-11-30T10:00:00Z
**Goal**: Implement user authentication system with JWT tokens

- [x] T001: Create user model (completed at 10:05:00)
- [~] T002: Implement login endpoint (in_progress since 10:10:00)
- [ ] T003: Add password hashing
- [!] T004: Setup JWT middleware (blocked: missing JWT library)

## Work Package WP02: Database Schema (completed)

**Started**: 2025-11-30T09:45:00Z
**Completed**: 2025-11-30T10:00:00Z
**Goal**: Set up database tables and migrations

- [x] T005: Create migration files (completed at 09:50:00)
- [x] T006: Run database setup (completed at 09:55:00)
- [s] T007: Add test data (skipped: not needed for MVP)

## Work Package WP03: API Endpoints (pending)

**Goal**: Create REST API endpoints for user management

- [ ] T008: Create user registration endpoint
`

	// Empty markdown content
	emptyMarkdownContent = ""

	// Malformed markdown content
	malformedMarkdownContent = `# Implementation Progress

**Status**: In Progress
**Updated**: invalid-timestamp

## Work Package INVALID_ID: Missing Status

**Goal**: This is missing proper format
- [x] INVALID_TASK: No description
- [?] T002: Invalid checkbox status
`

	// Partial markdown content
	partialMarkdownContent = `# Implementation Progress

## Work Package WP01: Only Basic Info (in_progress)

- [x] T001: Simple task
`
)

// Helper functions for creating test data

// createTestWorkPackageProgress creates a WorkPackageProgress for testing
func createTestWorkPackageProgress(id, name string, status tasks.TaskProgressState) tasks.WorkPackageProgress {
	now := time.Now()
	wp := tasks.WorkPackageProgress{
		ID:       id,
		Name:     name,
		Priority: "High",
		Status:   status,
		Goal:     "Test goal for " + name,
		Tasks:    []tasks.TaskProgress{},
	}

	if status != tasks.TaskStatePending {
		wp.StartedAt = &now
	}
	if status == tasks.TaskStateCompleted {
		wp.CompletedAt = &now
	}

	return wp
}

// createTestTaskProgress creates a TaskProgress for testing
func createTestTaskProgress(id, description string, status tasks.TaskProgressState) tasks.TaskProgress {
	now := time.Now()
	task := tasks.TaskProgress{
		ID:          id,
		Description: description,
		Status:      status,
	}

	if status != tasks.TaskStatePending {
		task.StartedAt = &now
	}
	if status == tasks.TaskStateCompleted {
		task.CompletedAt = &now
	}

	return task
}

// createTestWorkPackagesWithTasks creates a set of test work packages with tasks
func createTestWorkPackagesWithTasks() []tasks.WorkPackageProgress {
	wp1 := createTestWorkPackageProgress("WP01", "Authentication Setup", tasks.TaskStateInProgress)
	wp1.Tasks = []tasks.TaskProgress{
		createTestTaskProgress("T001", "Create user model", tasks.TaskStateCompleted),
		createTestTaskProgress("T002", "Implement login endpoint", tasks.TaskStateInProgress),
		createTestTaskProgress("T003", "Add password hashing", tasks.TaskStatePending),
	}

	wp2 := createTestWorkPackageProgress("WP02", "Database Schema", tasks.TaskStateCompleted)
	wp2.Tasks = []tasks.TaskProgress{
		createTestTaskProgress("T004", "Create migration files", tasks.TaskStateCompleted),
		createTestTaskProgress("T005", "Run database setup", tasks.TaskStateCompleted),
	}

	return []tasks.WorkPackageProgress{wp1, wp2}
}

// Test parsing function with valid markdown content
func TestParseProgressContent_ValidMarkdown(t *testing.T) {
	preview := NewImplementPreview(80, 30)

	// Test with valid markdown content - should parse correctly
	result, err := preview.parseProgressContent(validMarkdownContent)

	assert.NoError(t, err, "Valid markdown should parse without error")
	assert.Len(t, result, 3, "Should parse 3 work packages")

	// Verify first work package
	wp1 := result[0]
	assert.Equal(t, "WP01", wp1.ID)
	assert.Equal(t, "Authentication Setup", wp1.Name)
	assert.Equal(t, tasks.TaskStateInProgress, wp1.Status)
	assert.Len(t, wp1.Tasks, 4, "WP01 should have 4 tasks")

	// Verify first task
	task1 := wp1.Tasks[0]
	assert.Equal(t, "T001", task1.ID)
	assert.Equal(t, "Create user model", task1.Description)
	assert.Equal(t, tasks.TaskStateCompleted, task1.Status)
	assert.NotNil(t, task1.CompletedAt, "Completed task should have completion time")

	// Verify second task (in progress)
	task2 := wp1.Tasks[1]
	assert.Equal(t, "T002", task2.ID)
	assert.Equal(t, tasks.TaskStateInProgress, task2.Status)
	assert.NotNil(t, task2.StartedAt, "In progress task should have start time")

	// Verify blocked task
	task4 := wp1.Tasks[3]
	assert.Equal(t, "T004", task4.ID)
	assert.Equal(t, tasks.TaskStateBlocked, task4.Status)
	assert.Equal(t, "missing JWT library", task4.BlockReason)

	// Verify second work package (completed)
	wp2 := result[1]
	assert.Equal(t, "WP02", wp2.ID)
	assert.Equal(t, "Database Schema", wp2.Name)
	assert.Equal(t, tasks.TaskStateCompleted, wp2.Status)

	// Verify skipped task
	task7 := wp2.Tasks[2]
	assert.Equal(t, "T007", task7.ID)
	assert.Equal(t, tasks.TaskStateSkipped, task7.Status)

	// Verify third work package (pending)
	wp3 := result[2]
	assert.Equal(t, "WP03", wp3.ID)
	assert.Equal(t, tasks.TaskStatePending, wp3.Status)
}

func TestParseProgressContent_EmptyContent(t *testing.T) {
	preview := NewImplementPreview(80, 30)

	// Test with empty content
	result, err := preview.parseProgressContent("")

	assert.NoError(t, err, "Empty content should not return error")
	assert.Empty(t, result, "Empty content should return empty array")
}

func TestParseProgressContent_MalformedMarkdown(t *testing.T) {
	preview := NewImplementPreview(80, 30)

	// Test with malformed markdown - should handle gracefully
	result, err := preview.parseProgressContent(malformedMarkdownContent)

	assert.NoError(t, err, "Should handle malformed markdown gracefully")
	// Should return valid slice (may be empty), but not crash
	// Note: the function should gracefully skip malformed entries
	// and return whatever it could successfully parse
	if result != nil {
		assert.IsType(t, []tasks.WorkPackageProgress{}, result, "Should return correct type")
	}
}

func TestParseProgressContent_PartialData(t *testing.T) {
	preview := NewImplementPreview(80, 30)

	// Test with partial markdown content
	result, err := preview.parseProgressContent(partialMarkdownContent)

	assert.NoError(t, err, "Should handle partial data gracefully")
	assert.Len(t, result, 1, "Should parse the one work package")

	wp := result[0]
	assert.Equal(t, "WP01", wp.ID)
	assert.Equal(t, "Only Basic Info", wp.Name)
	assert.Equal(t, tasks.TaskStateInProgress, wp.Status)
	assert.Len(t, wp.Tasks, 1, "Should have one task")
}

// Test the overall preview functionality with test data
func TestImplementPreview_SetWorkPackages(t *testing.T) {
	preview := NewImplementPreview(80, 30)
	preview.SetSize(80, 30) // Make it ready

	testWorkPackages := createTestWorkPackagesWithTasks()

	// Set work packages
	preview.SetWorkPackages(testWorkPackages)

	// Verify work packages are stored
	assert.Equal(t, testWorkPackages, preview.workPackages)
	assert.False(t, preview.GetLastRefresh().IsZero(), "Last refresh time should be set")
}

func TestImplementPreview_UpdateWorkPackageStatus(t *testing.T) {
	preview := NewImplementPreview(80, 30)
	preview.SetSize(80, 30) // Make it ready

	testWorkPackages := createTestWorkPackagesWithTasks()
	preview.SetWorkPackages(testWorkPackages)

	// Update work package status
	preview.UpdateWorkPackageStatus("WP01", tasks.TaskStateCompleted)

	// Verify status was updated
	for _, wp := range preview.workPackages {
		if wp.ID == "WP01" {
			assert.Equal(t, tasks.TaskStateCompleted, wp.Status)
			break
		}
	}
}

func TestImplementPreview_RenderProgressContent(t *testing.T) {
	preview := NewImplementPreview(80, 30)
	preview.SetSize(80, 30) // Make it ready

	testWorkPackages := createTestWorkPackagesWithTasks()
	preview.SetWorkPackages(testWorkPackages)

	// Directly call renderProgressContent to test the progress rendering
	preview.renderProgressContent()

	// Get the viewport content
	content := preview.viewport.View()

	assert.NotEmpty(t, content, "Content should render")
	assert.Contains(t, content, "Implementation Progress", "Should contain header")
}

func TestImplementPreview_RenderProgressTask(t *testing.T) {
	preview := NewImplementPreview(80, 30)

	testCases := []struct {
		name          string
		task          tasks.TaskProgress
		expectedIcon  string
		shouldContain []string
	}{
		{
			name:          "completed task",
			task:          createTestTaskProgress("T001", "Test task", tasks.TaskStateCompleted),
			expectedIcon:  "✓",
			shouldContain: []string{"✓", "T001", "Test task"},
		},
		{
			name:          "in progress task",
			task:          createTestTaskProgress("T002", "Progress task", tasks.TaskStateInProgress),
			expectedIcon:  "●",
			shouldContain: []string{"●", "T002", "Progress task"},
		},
		{
			name:          "blocked task",
			task:          createTestTaskProgress("T003", "Blocked task", tasks.TaskStateBlocked),
			expectedIcon:  "!",
			shouldContain: []string{"!", "T003", "Blocked task"},
		},
		{
			name:          "pending task",
			task:          createTestTaskProgress("T004", "Pending task", tasks.TaskStatePending),
			expectedIcon:  "○",
			shouldContain: []string{"○", "T004", "Pending task"},
		},
		{
			name:          "skipped task",
			task:          createTestTaskProgress("T005", "Skipped task", tasks.TaskStateSkipped),
			expectedIcon:  "~",
			shouldContain: []string{"~", "T005", "Skipped task"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := preview.renderProgressTask(tc.task)

			assert.NotEmpty(t, result, "Should render task")
			for _, expected := range tc.shouldContain {
				assert.Contains(t, result, expected, "Should contain expected text")
			}
		})
	}
}

func TestImplementPreview_ConvertTaskProgressState(t *testing.T) {
	preview := NewImplementPreview(80, 30)

	testCases := []struct {
		input    tasks.TaskProgressState
		expected TaskStatus
	}{
		{tasks.TaskStateCompleted, TaskStatusComplete},
		{tasks.TaskStateInProgress, TaskStatusInProgress},
		{tasks.TaskStateBlocked, TaskStatusFailed},
		{tasks.TaskStatePending, TaskStatusPending},
		{tasks.TaskStateSkipped, TaskStatusPending},
	}

	for _, tc := range testCases {
		t.Run(string(tc.input), func(t *testing.T) {
			result := preview.convertTaskProgressState(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestImplementPreview_GettersAndSetters(t *testing.T) {
	preview := NewImplementPreview(80, 30)

	// Test progress file path
	testPath := "/test/path/progress.md"
	preview.progressFilePath = testPath
	assert.Equal(t, testPath, preview.GetProgressFilePath())

	// Test loading error
	preview.loadingError = assert.AnError
	assert.Equal(t, assert.AnError, preview.GetLoadingError())

	// Test last refresh is initially zero
	assert.True(t, preview.GetLastRefresh().IsZero())

	// After setting work packages, last refresh should be set
	preview.SetWorkPackages([]tasks.WorkPackageProgress{})
	assert.False(t, preview.GetLastRefresh().IsZero())
}

// Test helper functions individually

func TestParseWorkPackageHeader(t *testing.T) {
	testCases := []struct {
		name           string
		lines          []string
		startIndex     int
		expectedID     string
		expectedName   string
		expectedStatus tasks.TaskProgressState
		shouldFail     bool
	}{
		{
			name: "valid header with metadata",
			lines: []string{
				"## Work Package WP01: Authentication Setup (in_progress)",
				"**Started**: 2025-11-30T10:00:00Z",
				"**Goal**: Implement user authentication",
				"",
			},
			startIndex:     0,
			expectedID:     "WP01",
			expectedName:   "Authentication Setup",
			expectedStatus: tasks.TaskStateInProgress,
			shouldFail:     false,
		},
		{
			name: "valid header without metadata",
			lines: []string{
				"## Work Package WP02: Database Schema (completed)",
			},
			startIndex:     0,
			expectedID:     "WP02",
			expectedName:   "Database Schema",
			expectedStatus: tasks.TaskStateCompleted,
			shouldFail:     false,
		},
		{
			name: "invalid header format",
			lines: []string{
				"## Invalid Header",
			},
			startIndex: 0,
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, nextIndex, err := parseWorkPackageHeader(tc.lines, tc.startIndex)

			if tc.shouldFail {
				assert.Error(t, err, "Should return error for invalid format")
				return
			}

			assert.NoError(t, err, "Should not return error for valid format")
			assert.NotNil(t, result, "Should return work package")
			assert.Equal(t, tc.expectedID, result.ID)
			assert.Equal(t, tc.expectedName, result.Name)
			assert.Equal(t, tc.expectedStatus, result.Status)
			assert.Greater(t, nextIndex, tc.startIndex, "Should advance index")
		})
	}
}

func TestParseTaskLine(t *testing.T) {
	testCases := []struct {
		name           string
		line           string
		expectedID     string
		expectedDesc   string
		expectedStatus tasks.TaskProgressState
		shouldFail     bool
		hasTimestamp   bool
		hasBlockReason bool
	}{
		{
			name:           "completed task",
			line:           "- [x] T001: Create user model (completed at 10:15:00)",
			expectedID:     "T001",
			expectedDesc:   "Create user model",
			expectedStatus: tasks.TaskStateCompleted,
			hasTimestamp:   true,
		},
		{
			name:           "in progress task",
			line:           "- [~] T002: Implement login endpoint (in_progress since 10:20:00)",
			expectedID:     "T002",
			expectedDesc:   "Implement login endpoint",
			expectedStatus: tasks.TaskStateInProgress,
			hasTimestamp:   true,
		},
		{
			name:           "blocked task",
			line:           "- [!] T003: Setup JWT middleware (blocked: missing JWT library)",
			expectedID:     "T003",
			expectedDesc:   "Setup JWT middleware",
			expectedStatus: tasks.TaskStateBlocked,
			hasBlockReason: true,
		},
		{
			name:           "skipped task",
			line:           "- [s] T004: Add test data (skipped: not needed for MVP)",
			expectedID:     "T004",
			expectedDesc:   "Add test data",
			expectedStatus: tasks.TaskStateSkipped,
		},
		{
			name:           "pending task",
			line:           "- [ ] T005: Add password hashing",
			expectedID:     "T005",
			expectedDesc:   "Add password hashing",
			expectedStatus: tasks.TaskStatePending,
		},
		{
			name:       "invalid task format",
			line:       "- Invalid task line",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseTaskLine(tc.line)

			if tc.shouldFail {
				assert.Error(t, err, "Should return error for invalid format")
				return
			}

			assert.NoError(t, err, "Should not return error for valid format")
			assert.NotNil(t, result, "Should return task")
			assert.Equal(t, tc.expectedID, result.ID)
			assert.Equal(t, tc.expectedDesc, result.Description)
			assert.Equal(t, tc.expectedStatus, result.Status)

			if tc.hasTimestamp {
				if tc.expectedStatus == tasks.TaskStateCompleted {
					assert.NotNil(t, result.CompletedAt, "Completed task should have completion time")
				} else if tc.expectedStatus == tasks.TaskStateInProgress {
					assert.NotNil(t, result.StartedAt, "In progress task should have start time")
				}
			}

			if tc.hasBlockReason {
				assert.NotEmpty(t, result.BlockReason, "Blocked task should have block reason")
			}
		})
	}
}

func TestParseTaskProgressStateFromCheckbox(t *testing.T) {
	testCases := []struct {
		checkbox string
		expected tasks.TaskProgressState
	}{
		{"x", tasks.TaskStateCompleted},
		{"~", tasks.TaskStateInProgress},
		{"!", tasks.TaskStateBlocked},
		{"s", tasks.TaskStateSkipped},
		{" ", tasks.TaskStatePending},
		{"?", tasks.TaskStatePending}, // unknown defaults to pending
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("checkbox_%s", tc.checkbox), func(t *testing.T) {
			result := parseTaskProgressStateFromCheckbox(tc.checkbox)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseTaskProgressState(t *testing.T) {
	testCases := []struct {
		input    string
		expected tasks.TaskProgressState
	}{
		{"completed", tasks.TaskStateCompleted},
		{"in_progress", tasks.TaskStateInProgress},
		{"blocked", tasks.TaskStateBlocked},
		{"skipped", tasks.TaskStateSkipped},
		{"pending", tasks.TaskStatePending},
		{"unknown", tasks.TaskStatePending},       // unknown defaults to pending
		{" completed ", tasks.TaskStateCompleted}, // should trim whitespace
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parseTaskProgressState(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseMetadataLine(t *testing.T) {
	testCases := []struct {
		name          string
		line          string
		expectedKey   string
		expectedValue string
		shouldFail    bool
	}{
		{
			name:          "started metadata",
			line:          "**Started**: 2025-11-30T10:00:00Z",
			expectedKey:   "Started",
			expectedValue: "2025-11-30T10:00:00Z",
		},
		{
			name:          "goal metadata",
			line:          "**Goal**: Implement user authentication",
			expectedKey:   "Goal",
			expectedValue: "Implement user authentication",
		},
		{
			name:       "invalid metadata format",
			line:       "Not a metadata line",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key, value, err := parseMetadataLine(tc.line)

			if tc.shouldFail {
				assert.Error(t, err, "Should return error for invalid format")
				return
			}

			assert.NoError(t, err, "Should not return error for valid format")
			assert.Equal(t, tc.expectedKey, key)
			assert.Equal(t, tc.expectedValue, value)
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		shouldFail bool
	}{
		{
			name:  "ISO 8601 with Z",
			input: "2025-11-30T10:00:00Z",
		},
		{
			name:  "ISO 8601 with timezone",
			input: "2025-11-30T10:00:00-08:00",
		},
		{
			name:  "time only HH:MM:SS",
			input: "10:15:30",
		},
		{
			name:  "time only HH:MM",
			input: "10:15",
		},
		{
			name:       "invalid format",
			input:      "invalid-timestamp",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseTimestamp(tc.input)

			if tc.shouldFail {
				assert.Error(t, err, "Should return error for invalid format")
				assert.True(t, result.IsZero(), "Should return zero time on error")
				return
			}

			assert.NoError(t, err, "Should not return error for valid format")
			assert.False(t, result.IsZero(), "Should return valid time")
		})
	}
}

// Integration tests

func TestImplementPreview_FullWorkflow(t *testing.T) {
	preview := NewImplementPreview(80, 30)
	preview.SetSize(80, 30) // Make it ready

	// Test the full workflow: parsing -> setting work packages -> rendering
	result, err := preview.parseProgressContent(validMarkdownContent)
	assert.NoError(t, err, "Should parse content successfully")
	assert.Len(t, result, 3, "Should parse all work packages")

	// Set the parsed work packages
	preview.SetWorkPackages(result)

	// Verify internal state
	assert.Equal(t, result, preview.workPackages, "Work packages should be set correctly")
	assert.False(t, preview.GetLastRefresh().IsZero(), "Last refresh should be set")

	// Test rendering
	preview.renderProgressContent()
	content := preview.viewport.View()

	assert.NotEmpty(t, content, "Should render content")
	assert.Contains(t, content, "Implementation Progress", "Should contain header")
	assert.Contains(t, content, "WP01", "Should contain work package ID")
	assert.Contains(t, content, "Authentication Setup", "Should contain work package name")
	assert.Contains(t, content, "T001", "Should contain task ID")
	assert.Contains(t, content, "Create user model", "Should contain task description")

	// Test progress calculation
	assert.Contains(t, content, "Progress:", "Should show progress summary")
	assert.Contains(t, content, "Updated:", "Should show last updated time")
}

func TestImplementPreview_ErrorRecovery(t *testing.T) {
	preview := NewImplementPreview(80, 30)
	preview.SetSize(80, 30)

	// Test with malformed content - should not crash
	result, err := preview.parseProgressContent(malformedMarkdownContent)
	assert.NoError(t, err, "Should handle malformed content gracefully")

	preview.SetWorkPackages(result)
	preview.renderProgressContent()

	// Should render without crashing, even with no/minimal data
	content := preview.viewport.View()
	assert.NotEmpty(t, content, "Should render something even with malformed input")
}

func TestImplementPreview_EmptyStateHandling(t *testing.T) {
	preview := NewImplementPreview(80, 30)
	preview.SetSize(80, 30)

	// Test with empty content
	result, err := preview.parseProgressContent("")
	assert.NoError(t, err, "Should handle empty content")
	assert.Empty(t, result, "Should return empty array")

	preview.SetWorkPackages(result)
	preview.renderProgressContent()

	content := preview.viewport.View()
	assert.NotEmpty(t, content, "Should render empty state")
	assert.Contains(t, content, "No progress data available", "Should show appropriate message")
}

func TestImplementPreview_ProgressUpdates(t *testing.T) {
	preview := NewImplementPreview(80, 30)
	preview.SetSize(80, 30)

	// Start with initial data
	result, err := preview.parseProgressContent(validMarkdownContent)
	assert.NoError(t, err)
	preview.SetWorkPackages(result)

	// Simulate a work package status update
	preview.UpdateWorkPackageStatus("WP01", tasks.TaskStateCompleted)

	// Verify the update
	found := false
	for _, wp := range preview.workPackages {
		if wp.ID == "WP01" {
			assert.Equal(t, tasks.TaskStateCompleted, wp.Status, "Work package status should be updated")
			found = true
			break
		}
	}
	assert.True(t, found, "Should find the work package")

	// Verify rendering reflects the change
	preview.renderProgressContent()
	content := preview.viewport.View()
	assert.Contains(t, content, "WP01", "Should still show work package")

	// Note: The specific status display format depends on implementation
	// but the content should be updated
	assert.NotEmpty(t, content, "Should render updated content")
}
