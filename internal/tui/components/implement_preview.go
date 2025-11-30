package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tasks"
	"github.com/darinhaener/collab/internal/tui/theme"
)

// TaskStatus represents the status of an implementation task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusComplete   TaskStatus = "complete"
	TaskStatusFailed     TaskStatus = "failed"
)

// ImplementTask represents a task in the implementation preview
type ImplementTask struct {
	ID          string
	Description string
	Status      TaskStatus
	Output      string
}

// ImplementPreview displays implementation progress with task status
type ImplementPreview struct {
	viewport         viewport.Model
	tasks            []ImplementTask
	width            int
	height           int
	ready            bool
	summary          string

	// Task progress tracking
	progressTracker  *tasks.ProgressTracker
	progressManager  *tasks.ProgressFileManager
	workPackages     []tasks.WorkPackageProgress
	lastRefresh      time.Time
	progressFilePath string
	loadingError     error
}

// NewImplementPreview creates a new implement preview component
func NewImplementPreview(width, height int) ImplementPreview {
	vp := viewport.New(width-4, height-4)
	vp.YPosition = 0

	return ImplementPreview{
		viewport: vp,
		tasks:    []ImplementTask{},
		width:    width,
		height:   height,
		ready:    false,
	}
}

// Init initializes the preview component
func (p ImplementPreview) Init() tea.Cmd {
	return nil
}

// Update handles messages for the preview
func (p ImplementPreview) Update(msg tea.Msg) (ImplementPreview, tea.Cmd) {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// View renders the preview
func (p ImplementPreview) View() string {
	if !p.ready {
		return implementPreviewPlaceholderStyle.Render("Implementation progress will appear here...")
	}

	if len(p.tasks) == 0 {
		return implementPreviewPlaceholderStyle.Render("No tasks loaded yet.\nTasks will appear as implementation begins.")
	}

	return p.viewport.View()
}

// SetSize updates the preview dimensions
func (p *ImplementPreview) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = width
	p.viewport.Height = height
	p.ready = true

	// Re-render content with new size
	p.renderContent()
}

// SetTasks updates the tasks list
func (p *ImplementPreview) SetTasks(tasks []ImplementTask) {
	p.tasks = tasks
	if p.ready {
		p.renderContent()
	}
}

// SetSummary updates the summary text
func (p *ImplementPreview) SetSummary(summary string) {
	p.summary = summary
	if p.ready {
		p.renderContent()
	}
}

// SetContent implements the PreviewComponent interface.
// For ImplementPreview, this sets the summary text.
func (p *ImplementPreview) SetContent(content string) {
	p.SetSummary(content)
}

// UpdateTaskStatus updates the status of a specific task
func (p *ImplementPreview) UpdateTaskStatus(taskID string, status TaskStatus, output string) {
	for i, task := range p.tasks {
		if task.ID == taskID {
			p.tasks[i].Status = status
			if output != "" {
				p.tasks[i].Output = output
			}
			break
		}
	}
	if p.ready {
		p.renderContent()
	}
}

// renderContent renders the implementation progress view
func (p *ImplementPreview) renderContent() {
	var parts []string

	// Summary section
	if p.summary != "" {
		parts = append(parts, implementPreviewSummaryStyle.Render("Summary"))
		parts = append(parts, implementPreviewTextStyle.Render(p.summary))
		parts = append(parts, "")
	}

	// Progress overview
	completed := 0
	inProgress := 0
	failed := 0
	for _, task := range p.tasks {
		switch task.Status {
		case TaskStatusComplete:
			completed++
		case TaskStatusInProgress:
			inProgress++
		case TaskStatusFailed:
			failed++
		}
	}

	if len(p.tasks) > 0 {
		progressLine := fmt.Sprintf("Progress: %d/%d complete", completed, len(p.tasks))
		if inProgress > 0 {
			progressLine += fmt.Sprintf(" | %d in progress", inProgress)
		}
		if failed > 0 {
			progressLine += fmt.Sprintf(" | %d failed", failed)
		}
		parts = append(parts, implementPreviewProgressStyle.Render(progressLine))
		parts = append(parts, "")
	}

	// Tasks list
	parts = append(parts, implementPreviewH2Style.Render("Tasks"))
	for _, task := range p.tasks {
		taskLine := p.renderTask(task)
		parts = append(parts, taskLine)
	}

	p.viewport.SetContent(strings.Join(parts, "\n"))
}

// renderTask renders a single task with status indicator
func (p *ImplementPreview) renderTask(task ImplementTask) string {
	var icon string
	var style lipgloss.Style

	switch task.Status {
	case TaskStatusComplete:
		icon = "✓"
		style = implementPreviewCompleteStyle
	case TaskStatusInProgress:
		icon = "●"
		style = implementPreviewInProgressStyle
	case TaskStatusFailed:
		icon = "✗"
		style = implementPreviewFailedStyle
	default:
		icon = "○"
		style = implementPreviewPendingStyle
	}

	line := fmt.Sprintf("%s %s: %s", icon, task.ID, task.Description)
	rendered := style.Render(line)

	// Add output if present
	if task.Output != "" {
		outputLines := strings.Split(task.Output, "\n")
		for _, ol := range outputLines {
			rendered += "\n" + implementPreviewOutputStyle.Render("  "+ol)
		}
	}

	return rendered
}

// LoadProgressFromFile loads task progress from a task-progress.md file
func (p *ImplementPreview) LoadProgressFromFile(progressFilePath string) error {
	p.progressFilePath = progressFilePath

	// Create progress file manager
	p.progressManager = tasks.NewProgressFileManager(progressFilePath)

	// Check if progress file exists
	if !p.progressManager.Exists() {
		// Try to create initial progress file from tasks.md
		tasksFilePath := filepath.Join(filepath.Dir(progressFilePath), "tasks.md")
		if err := p.progressManager.CreateInitialProgressFile(tasksFilePath); err != nil {
			p.loadingError = fmt.Errorf("failed to create initial progress file: %w", err)
			return p.loadingError
		}
	}

	// Load existing progress
	return p.RefreshFromProgressFile()
}

// RefreshFromProgressFile reloads progress from the file
func (p *ImplementPreview) RefreshFromProgressFile() error {
	if p.progressManager == nil {
		return fmt.Errorf("progress manager not initialized")
	}

	// Validate progress file first
	if err := p.progressManager.Validate(); err != nil {
		p.loadingError = fmt.Errorf("progress file validation failed: %w", err)
		return p.loadingError
	}

	// For now, we'll parse the progress file content directly
	// In a full implementation, we'd use the ProgressTracker to load state
	content, err := readProgressFileContent(p.progressFilePath)
	if err != nil {
		p.loadingError = fmt.Errorf("failed to read progress file: %w", err)
		return p.loadingError
	}

	// Parse the content and extract work packages
	workPackages, err := p.parseProgressContent(content)
	if err != nil {
		p.loadingError = fmt.Errorf("failed to parse progress content: %w", err)
		return p.loadingError
	}

	p.workPackages = workPackages
	p.lastRefresh = time.Now()
	p.loadingError = nil

	// Update display
	if p.ready {
		p.renderProgressContent()
	}

	return nil
}

// SetWorkPackages updates the work packages display
func (p *ImplementPreview) SetWorkPackages(workPackages []tasks.WorkPackageProgress) {
	p.workPackages = workPackages
	p.lastRefresh = time.Now()
	if p.ready {
		p.renderProgressContent()
	}
}

// UpdateWorkPackageStatus updates the status of a specific work package
func (p *ImplementPreview) UpdateWorkPackageStatus(wpID string, status tasks.TaskProgressState) {
	for i := range p.workPackages {
		if p.workPackages[i].ID == wpID {
			p.workPackages[i].Status = status
			break
		}
	}
	if p.ready {
		p.renderProgressContent()
	}
}

// GetProgressFilePath returns the current progress file path
func (p *ImplementPreview) GetProgressFilePath() string {
	return p.progressFilePath
}

// GetLastRefresh returns the last refresh time
func (p *ImplementPreview) GetLastRefresh() time.Time {
	return p.lastRefresh
}

// GetLoadingError returns any loading error
func (p *ImplementPreview) GetLoadingError() error {
	return p.loadingError
}

// renderProgressContent renders the work package progress view
func (p *ImplementPreview) renderProgressContent() {
	var parts []string

	// Header
	parts = append(parts, implementPreviewH2Style.Render("Implementation Progress"))

	if p.loadingError != nil {
		parts = append(parts, implementPreviewFailedStyle.Render("Error: "+p.loadingError.Error()))
		p.viewport.SetContent(strings.Join(parts, "\n"))
		return
	}

	if len(p.workPackages) == 0 {
		parts = append(parts, implementPreviewPlaceholderStyle.Render("No progress data available"))
		p.viewport.SetContent(strings.Join(parts, "\n"))
		return
	}

	// Overall progress
	totalTasks := 0
	completedTasks := 0
	inProgressTasks := 0
	blockedTasks := 0

	for _, wp := range p.workPackages {
		for _, task := range wp.Tasks {
			totalTasks++
			switch task.Status {
			case tasks.TaskStateCompleted, tasks.TaskStateSkipped:
				completedTasks++
			case tasks.TaskStateInProgress:
				inProgressTasks++
			case tasks.TaskStateBlocked:
				blockedTasks++
			}
		}
	}

	progressPercent := 0.0
	if totalTasks > 0 {
		progressPercent = float64(completedTasks) / float64(totalTasks) * 100
	}

	progressLine := fmt.Sprintf("Progress: %d/%d tasks (%.1f%%)", completedTasks, totalTasks, progressPercent)
	if inProgressTasks > 0 {
		progressLine += fmt.Sprintf(" | %d in progress", inProgressTasks)
	}
	if blockedTasks > 0 {
		progressLine += fmt.Sprintf(" | %d blocked", blockedTasks)
	}

	parts = append(parts, implementPreviewProgressStyle.Render(progressLine))
	parts = append(parts, "")

	// Last updated
	if !p.lastRefresh.IsZero() {
		parts = append(parts, implementPreviewTextStyle.Render("Updated: "+p.lastRefresh.Format("15:04:05")))
		parts = append(parts, "")
	}

	// Work packages
	for _, wp := range p.workPackages {
		wpLine := fmt.Sprintf("WP %s: %s", wp.ID, wp.Name)
		if wp.Status != tasks.TaskStatePending {
			wpLine += fmt.Sprintf(" (%s)", wp.Status)
		}

		parts = append(parts, implementPreviewH2Style.Render(wpLine))

		// Work package timing
		if wp.StartedAt != nil {
			parts = append(parts, implementPreviewTextStyle.Render("Started: "+wp.StartedAt.Format("15:04:05")))
		}
		if wp.CompletedAt != nil {
			parts = append(parts, implementPreviewTextStyle.Render("Completed: "+wp.CompletedAt.Format("15:04:05")))
		}

		// Tasks
		for _, task := range wp.Tasks {
			taskLine := p.renderProgressTask(task)
			parts = append(parts, taskLine)
		}

		parts = append(parts, "")
	}

	p.viewport.SetContent(strings.Join(parts, "\n"))
}

// renderProgressTask renders a single task with progress status
func (p *ImplementPreview) renderProgressTask(task tasks.TaskProgress) string {
	var icon string
	var style lipgloss.Style

	switch task.Status {
	case tasks.TaskStateCompleted:
		icon = "✓"
		style = implementPreviewCompleteStyle
	case tasks.TaskStateInProgress:
		icon = "●"
		style = implementPreviewInProgressStyle
	case tasks.TaskStateBlocked:
		icon = "!"
		style = implementPreviewFailedStyle
	case tasks.TaskStateSkipped:
		icon = "~"
		style = implementPreviewPendingStyle
	default:
		icon = "○"
		style = implementPreviewPendingStyle
	}

	line := fmt.Sprintf("  %s %s: %s", icon, task.ID, task.Description)

	// Add timing information
	if task.Status != tasks.TaskStatePending {
		if task.StartedAt != nil && task.Status == tasks.TaskStateInProgress {
			line += fmt.Sprintf(" (since %s)", task.StartedAt.Format("15:04"))
		} else if task.CompletedAt != nil {
			line += fmt.Sprintf(" (at %s)", task.CompletedAt.Format("15:04"))
		}
	}

	rendered := style.Render(line)

	// Add block reason if present
	if task.BlockReason != "" && (task.Status == tasks.TaskStateBlocked || task.Status == tasks.TaskStateSkipped) {
		rendered += "\n" + implementPreviewOutputStyle.Render("    Reason: "+task.BlockReason)
	}

	// Add output if present
	if task.Output != "" {
		outputLines := strings.Split(task.Output, "\n")
		for _, ol := range outputLines {
			if strings.TrimSpace(ol) != "" {
				rendered += "\n" + implementPreviewOutputStyle.Render("    "+ol)
			}
		}
	}

	return rendered
}

// convertTaskProgressState converts tasks.TaskProgressState to local TaskStatus
func (p *ImplementPreview) convertTaskProgressState(state tasks.TaskProgressState) TaskStatus {
	switch state {
	case tasks.TaskStateCompleted:
		return TaskStatusComplete
	case tasks.TaskStateInProgress:
		return TaskStatusInProgress
	case tasks.TaskStateBlocked:
		return TaskStatusFailed
	default:
		return TaskStatusPending
	}
}

// parseProgressContent parses progress file content to extract work packages
// This is a simplified parser for the preview component
func (p *ImplementPreview) parseProgressContent(content string) ([]tasks.WorkPackageProgress, error) {
	// For now, return empty slice - in a full implementation this would parse the markdown
	// The actual parsing would be done by the ProgressTracker
	return []tasks.WorkPackageProgress{}, nil
}

// Helper function to read progress file content
func readProgressFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// Styles for implement preview
var (
	implementPreviewPlaceholderStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Italic(true).
		Align(lipgloss.Center)

	implementPreviewSummaryStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		MarginBottom(1)

	implementPreviewH2Style = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	implementPreviewTextStyle = lipgloss.NewStyle().
		Foreground(theme.Text)

	implementPreviewProgressStyle = lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true)

	implementPreviewCompleteStyle = lipgloss.NewStyle().
		Foreground(theme.Success)

	implementPreviewInProgressStyle = lipgloss.NewStyle().
		Foreground(theme.Warning)

	implementPreviewPendingStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted)

	implementPreviewFailedStyle = lipgloss.NewStyle().
		Foreground(theme.Error)

	implementPreviewOutputStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Italic(true)
)
