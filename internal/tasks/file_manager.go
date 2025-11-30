package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darinhaener/collab/internal/storage"
)

// ProgressFileManager handles reading and writing task-progress.md files
type ProgressFileManager struct {
	filePath string
	parser   *TasksParser
}

// NewProgressFileManager creates a new progress file manager
func NewProgressFileManager(progressFilePath string) *ProgressFileManager {
	return &ProgressFileManager{
		filePath: progressFilePath,
		parser:   NewTasksParser(),
	}
}

// WriteProgressFile writes the current progress to task-progress.md in markdown format
func (pfm *ProgressFileManager) WriteProgressFile(tracker *ProgressTracker) error {
	content := pfm.generateProgressMarkdown(tracker)

	// Use atomic write to ensure file consistency
	return storage.AtomicWriteString(pfm.filePath, content, 0644)
}

// ReadProgressFile reads and parses an existing task-progress.md file
func (pfm *ProgressFileManager) ReadProgressFile() (*ProgressTracker, error) {
	content, err := os.ReadFile(pfm.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read progress file: %w", err)
	}

	return pfm.parseProgressMarkdown(string(content))
}

// CreateInitialProgressFile creates the initial task-progress.md from a tasks.md file
func (pfm *ProgressFileManager) CreateInitialProgressFile(tasksFilePath string) error {
	// Parse the tasks.md file
	taskStructure, err := pfm.parser.ParseTasksFile(tasksFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse tasks file: %w", err)
	}

	// Create progress tracker with initial state
	tracker := NewProgressTracker(taskStructure, pfm.filePath)

	// Write initial progress file
	return pfm.WriteProgressFile(tracker)
}

// generateProgressMarkdown creates the markdown content for task-progress.md
func (pfm *ProgressFileManager) generateProgressMarkdown(tracker *ProgressTracker) string {
	var builder strings.Builder

	progress := tracker.GetProgress()
	workPackages := tracker.GetWorkPackages()

	// Header
	builder.WriteString("# Implementation Progress\n\n")

	// Overall status
	overallStatus := pfm.determineOverallStatus(progress)
	builder.WriteString(fmt.Sprintf("**Status**: %s\n", overallStatus))
	builder.WriteString(fmt.Sprintf("**Updated**: %s\n", progress.LastUpdated.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("**Progress**: %d/%d tasks completed (%.1f%%)\n\n",
		progress.CompletedTasks, progress.TotalTasks, progress.OverallProgress))

	// Work packages
	for _, wp := range workPackages {
		builder.WriteString(fmt.Sprintf("## Work Package %s: %s (%s)\n", wp.ID, wp.Name, wp.Status))

		// Work package metadata
		if wp.StartedAt != nil {
			builder.WriteString(fmt.Sprintf("**Started**: %s\n", wp.StartedAt.Format(time.RFC3339)))
		}
		if wp.CompletedAt != nil {
			builder.WriteString(fmt.Sprintf("**Completed**: %s\n", wp.CompletedAt.Format(time.RFC3339)))
		}
		if wp.Goal != "" {
			builder.WriteString(fmt.Sprintf("**Goal**: %s\n", wp.Goal))
		}
		builder.WriteString("\n")

		// Tasks
		for _, task := range wp.Tasks {
			checkbox := pfm.getTaskCheckbox(task.Status)
			line := fmt.Sprintf("- %s %s: %s", checkbox, task.ID, task.Description)

			// Add status and timing information
			if task.Status != TaskStatePending {
				line += fmt.Sprintf(" (%s", task.Status)
				if task.StartedAt != nil && task.Status == TaskStateInProgress {
					line += fmt.Sprintf(" since %s", task.StartedAt.Format("15:04:05"))
				}
				if task.CompletedAt != nil {
					line += fmt.Sprintf(" at %s", task.CompletedAt.Format("15:04:05"))
				}
				line += ")"
			}

			builder.WriteString(line + "\n")

			// Add block reason if present
			if task.BlockReason != "" && (task.Status == TaskStateBlocked || task.Status == TaskStateSkipped) {
				builder.WriteString(fmt.Sprintf("  - **Reason**: %s\n", task.BlockReason))
			}

			// Add output if present
			if task.Output != "" {
				outputLines := strings.Split(task.Output, "\n")
				for _, outputLine := range outputLines {
					if strings.TrimSpace(outputLine) != "" {
						builder.WriteString(fmt.Sprintf("  - %s\n", outputLine))
					}
				}
			}
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// parseProgressMarkdown parses task-progress.md content back into a ProgressTracker
func (pfm *ProgressFileManager) parseProgressMarkdown(content string) (*ProgressTracker, error) {
	// This is a simplified parser - in a production system, you might want a more robust implementation
	// For now, this returns an error as parsing existing progress files is complex
	// The system is designed to create new progress files from tasks.md
	return nil, fmt.Errorf("parsing existing progress files not yet implemented - create new from tasks.md")
}

// getTaskCheckbox returns the appropriate checkbox marker for a task status
func (pfm *ProgressFileManager) getTaskCheckbox(status TaskProgressState) string {
	switch status {
	case TaskStateCompleted:
		return "[x]"
	case TaskStateInProgress:
		return "[~]" // Using tilde to indicate in progress
	case TaskStateBlocked:
		return "[!]" // Using exclamation to indicate blocked
	case TaskStateSkipped:
		return "[s]" // Using 's' to indicate skipped
	default:
		return "[ ]" // Pending
	}
}

// determineOverallStatus determines the overall implementation status
func (pfm *ProgressFileManager) determineOverallStatus(progress ProgressSummary) string {
	if progress.TotalTasks == 0 {
		return "not_started"
	}

	if progress.CompletedTasks+progress.SkippedTasks == progress.TotalTasks {
		return "completed"
	}

	if progress.BlockedTasks > 0 && progress.InProgressTasks == 0 {
		return "blocked"
	}

	if progress.InProgressTasks > 0 {
		return "in_progress"
	}

	return "pending"
}

// Exists checks if the progress file exists
func (pfm *ProgressFileManager) Exists() bool {
	_, err := os.Stat(pfm.filePath)
	return err == nil
}

// GetFilePath returns the file path being managed
func (pfm *ProgressFileManager) GetFilePath() string {
	return pfm.filePath
}

// Validate checks if the progress file is valid and readable
func (pfm *ProgressFileManager) Validate() error {
	if !pfm.Exists() {
		return fmt.Errorf("progress file does not exist: %s", pfm.filePath)
	}

	content, err := os.ReadFile(pfm.filePath)
	if err != nil {
		return fmt.Errorf("cannot read progress file: %w", err)
	}

	// Basic validation - check if it looks like a progress file
	contentStr := string(content)
	if !strings.Contains(contentStr, "# Implementation Progress") {
		return fmt.Errorf("file does not appear to be a valid progress file")
	}

	return nil
}

// GetProgressFileDir returns the directory where the progress file is located
func (pfm *ProgressFileManager) GetProgressFileDir() string {
	return filepath.Dir(pfm.filePath)
}

// BackupProgressFile creates a backup of the current progress file
func (pfm *ProgressFileManager) BackupProgressFile() (string, error) {
	if !pfm.Exists() {
		return "", fmt.Errorf("progress file does not exist to backup")
	}

	backupPath := pfm.filePath + ".backup." + time.Now().Format("20060102-150405")

	content, err := os.ReadFile(pfm.filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read progress file for backup: %w", err)
	}

	err = storage.AtomicWrite(backupPath, content, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}