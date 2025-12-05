package components

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data fixtures for bug preview testing
const (
	testPlanContent = `# Bug Fix Plan

## Problem Analysis
The application crashes when user clicks the submit button.

## Solution Approach
1. Add input validation
2. Fix null pointer exception
3. Add error handling

## Implementation Steps
1. Analyze crash logs
2. Implement validation
3. Add tests
`

	testTasksContent = `# Bug Fix Tasks

## Task 1: Analyze Crash Logs
- Examine stack traces
- Identify root cause

## Task 2: Implement Input Validation
- Add form validation
- Test edge cases

## Task 3: Fix Null Pointer Exception
- Add null checks
- Update error handling
`

	testTaskProgressContent = `# Task Progress

## Current Status
Working on Task 2: Implement Input Validation

## Completed Tasks
- [x] Task 1: Analyze Crash Logs

## In Progress
- [~] Task 2: Implement Input Validation

## Pending
- [ ] Task 3: Fix Null Pointer Exception
`

	testContentWithFrontmatter = `---
title: Test Bug Plan
status: draft
---
# Bug Fix Plan

This is the actual content after frontmatter.
`
)

// TestBugPreview_SetFiles tests the SetFiles method with three parameters
func TestBugPreview_SetFiles(t *testing.T) {
	preview := NewBugPreview(80, 24)

	planFile := "/tmp/bug-plan.md"
	tasksFile := "/tmp/bug-tasks.md"
	taskProgressFile := "/tmp/task-progress.md"

	preview.SetFiles(planFile, tasksFile, taskProgressFile)

	assert.Equal(t, planFile, preview.planFile)
	assert.Equal(t, tasksFile, preview.tasksFile)
	assert.Equal(t, taskProgressFile, preview.taskProgressFile)
}

// TestBugPreview_GetTaskProgressFilePath tests the path resolution helper
func TestBugPreview_GetTaskProgressFilePath(t *testing.T) {
	preview := NewBugPreview(80, 24)

	baseDir := "/tmp/bug-123"
	expected := filepath.Join(baseDir, "task-progress.md")
	actual := preview.getTaskProgressFilePath(baseDir)

	assert.Equal(t, expected, actual)
}

// TestBugPreview_RefreshFromFiles_PlanPhase tests file loading for plan phase
func TestBugPreview_RefreshFromFiles_PlanPhase(t *testing.T) {
	// Create temporary directory and file
	tempDir, err := os.MkdirTemp("", "bug-preview-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	planFile := filepath.Join(tempDir, "bug-plan.md")
	err = os.WriteFile(planFile, []byte(testPlanContent), 0644)
	require.NoError(t, err)

	// Test plan phase
	preview := NewBugPreview(80, 24)
	preview.SetFiles(planFile, "", "")
	preview.SetBugPhase(types.BugPhasePlan)
	preview.SetSize(80, 24) // Make it ready

	preview.RefreshFromFiles()

	assert.Contains(t, preview.content, "Bug Fix Plan")
	assert.Contains(t, preview.content, "Problem Analysis")
}

// TestBugPreview_RefreshFromFiles_TasksPhase tests file loading for tasks phase
func TestBugPreview_RefreshFromFiles_TasksPhase(t *testing.T) {
	// Create temporary directory and file
	tempDir, err := os.MkdirTemp("", "bug-preview-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tasksFile := filepath.Join(tempDir, "bug-tasks.md")
	err = os.WriteFile(tasksFile, []byte(testTasksContent), 0644)
	require.NoError(t, err)

	// Test tasks phase
	preview := NewBugPreview(80, 24)
	preview.SetFiles("", tasksFile, "")
	preview.SetBugPhase(types.BugPhaseTasks)
	preview.SetSize(80, 24) // Make it ready

	preview.RefreshFromFiles()

	assert.Contains(t, preview.content, "Bug Fix Tasks")
	assert.Contains(t, preview.content, "Task 1: Analyze Crash Logs")
}

// TestBugPreview_RefreshFromFiles_ImplementPhase tests file loading for implement phase
func TestBugPreview_RefreshFromFiles_ImplementPhase(t *testing.T) {
	// Create temporary directory and file
	tempDir, err := os.MkdirTemp("", "bug-preview-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	taskProgressFile := filepath.Join(tempDir, "task-progress.md")
	err = os.WriteFile(taskProgressFile, []byte(testTaskProgressContent), 0644)
	require.NoError(t, err)

	// Test implement phase
	preview := NewBugPreview(80, 24)
	preview.SetFiles("", "", taskProgressFile)
	preview.SetBugPhase(types.BugPhaseImplement)
	preview.SetSize(80, 24) // Make it ready

	preview.RefreshFromFiles()

	assert.Contains(t, preview.content, "Task Progress")
	assert.Contains(t, preview.content, "Current Status")
}

// TestBugPreview_RefreshFromFiles_MissingFile tests error handling for missing files
func TestBugPreview_RefreshFromFiles_MissingFile(t *testing.T) {
	preview := NewBugPreview(80, 24)
	preview.SetFiles("/nonexistent/plan.md", "", "")
	preview.SetBugPhase(types.BugPhasePlan)
	preview.SetSize(80, 24) // Make it ready

	preview.RefreshFromFiles()

	// Should set empty content for missing files (graceful handling)
	assert.Empty(t, preview.content)
}

// TestBugPreview_RefreshFromFiles_EmptyFile tests error handling for empty files
func TestBugPreview_RefreshFromFiles_EmptyFile(t *testing.T) {
	// Create temporary directory and empty file
	tempDir, err := os.MkdirTemp("", "bug-preview-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	emptyFile := filepath.Join(tempDir, "empty.md")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	require.NoError(t, err)

	preview := NewBugPreview(80, 24)
	preview.SetFiles(emptyFile, "", "")
	preview.SetBugPhase(types.BugPhasePlan)
	preview.SetSize(80, 24) // Make it ready

	preview.RefreshFromFiles()

	// Should show appropriate message for empty files
	assert.Contains(t, preview.content, "plan file is empty")
	assert.Contains(t, preview.content, "in progress")
}

// TestBugPreview_RefreshFromFiles_PermissionDenied tests error handling for permission issues
func TestBugPreview_RefreshFromFiles_PermissionDenied(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Create temporary directory and file with no read permissions
	tempDir, err := os.MkdirTemp("", "bug-preview-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	restrictedFile := filepath.Join(tempDir, "restricted.md")
	err = os.WriteFile(restrictedFile, []byte(testPlanContent), 0644)
	require.NoError(t, err)

	// Remove read permissions
	err = os.Chmod(restrictedFile, 0000)
	require.NoError(t, err)

	preview := NewBugPreview(80, 24)
	preview.SetFiles(restrictedFile, "", "")
	preview.SetBugPhase(types.BugPhasePlan)
	preview.SetSize(80, 24) // Make it ready

	preview.RefreshFromFiles()

	// Should show permission error message
	assert.Contains(t, preview.content, "Permission denied")
	assert.Contains(t, preview.content, "check file permissions")
}

// TestBugPreview_StripFrontmatter tests YAML frontmatter removal
func TestBugPreview_StripFrontmatter(t *testing.T) {
	// Create temporary directory and file with frontmatter
	tempDir, err := os.MkdirTemp("", "bug-preview-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	frontmatterFile := filepath.Join(tempDir, "with-frontmatter.md")
	err = os.WriteFile(frontmatterFile, []byte(testContentWithFrontmatter), 0644)
	require.NoError(t, err)

	preview := NewBugPreview(80, 24)
	preview.SetFiles(frontmatterFile, "", "")
	preview.SetBugPhase(types.BugPhasePlan)
	preview.SetSize(80, 24) // Make it ready

	preview.RefreshFromFiles()

	// Should strip frontmatter and only show content
	assert.Contains(t, preview.content, "Bug Fix Plan")
	assert.Contains(t, preview.content, "actual content after frontmatter")
	assert.NotContains(t, preview.content, "title: Test Bug Plan")
	assert.NotContains(t, preview.content, "status: draft")
	assert.NotContains(t, preview.content, "---")
}

// TestBugPreview_GetPlaceholderText tests placeholder text for different phases
func TestBugPreview_GetPlaceholderText(t *testing.T) {
	preview := NewBugPreview(80, 24)

	// Test plan phase placeholder
	preview.SetBugPhase(types.BugPhasePlan)
	placeholder := preview.getPlaceholderText()
	assert.Contains(t, placeholder, "plan will appear")

	// Test tasks phase placeholder
	preview.SetBugPhase(types.BugPhaseTasks)
	placeholder = preview.getPlaceholderText()
	assert.Contains(t, placeholder, "task breakdown will appear")

	// Test implement phase placeholder with no task progress file
	preview.SetBugPhase(types.BugPhaseImplement)
	preview.taskProgressFile = ""
	placeholder = preview.getPlaceholderText()
	assert.Contains(t, placeholder, "Task progress will appear here once implementation begins")

	// Test implement phase placeholder with task progress file
	preview.taskProgressFile = "/tmp/task-progress.md"
	placeholder = preview.getPlaceholderText()
	assert.Contains(t, placeholder, "Implementation progress will appear")
}

// TestBugPreview_ValidateFiles tests the file validation method
func TestBugPreview_ValidateFiles(t *testing.T) {
	// Create temporary directory with some files
	tempDir, err := os.MkdirTemp("", "bug-preview-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	existingFile := filepath.Join(tempDir, "existing.md")
	err = os.WriteFile(existingFile, []byte("content"), 0644)
	require.NoError(t, err)

	preview := NewBugPreview(80, 24)

	// Test with mix of existing and non-existing files
	preview.SetFiles(existingFile, "/nonexistent.md", "")

	// Should not return error (graceful handling)
	err = preview.validateFiles()
	assert.NoError(t, err)
}

// TestBugPreview_PhaseIntegration tests phase switching with different file types
func TestBugPreview_PhaseIntegration(t *testing.T) {
	// Create temporary directory and files
	tempDir, err := os.MkdirTemp("", "bug-preview-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	planFile := filepath.Join(tempDir, "plan.md")
	tasksFile := filepath.Join(tempDir, "tasks.md")
	progressFile := filepath.Join(tempDir, "progress.md")

	err = os.WriteFile(planFile, []byte(testPlanContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(tasksFile, []byte(testTasksContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(progressFile, []byte(testTaskProgressContent), 0644)
	require.NoError(t, err)

	preview := NewBugPreview(80, 24)
	preview.SetFiles(planFile, tasksFile, progressFile)
	preview.SetSize(80, 24) // Make it ready

	// Test plan phase
	preview.SetBugPhase(types.BugPhasePlan)
	preview.RefreshFromFiles()
	assert.Contains(t, preview.content, "Bug Fix Plan")

	// Test tasks phase
	preview.SetBugPhase(types.BugPhaseTasks)
	preview.RefreshFromFiles()
	assert.Contains(t, preview.content, "Bug Fix Tasks")

	// Test implement phase
	preview.SetBugPhase(types.BugPhaseImplement)
	preview.RefreshFromFiles()
	assert.Contains(t, preview.content, "Task Progress")
}