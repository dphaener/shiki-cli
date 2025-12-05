package components

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tasks"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
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
	layout  PreviewLayout
	tasks   []ImplementTask
	phase   types.WorkflowPhase
	width   int
	height  int
	ready   bool
	summary string

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
	// Initialize with a default implement phase - will be updated via SetPhase
	defaultPhase := types.WorkflowPhaseImplement
	layout := NewPreviewLayout(defaultPhase)
	layout.SetSize(width, height)

	return ImplementPreview{
		layout: layout,
		tasks:  []ImplementTask{},
		phase:  defaultPhase,
		width:  width,
		height: height,
		ready:  false,
	}
}

// Init initializes the preview component
func (p ImplementPreview) Init() tea.Cmd {
	return nil
}

// Update handles messages for the preview
func (p ImplementPreview) Update(msg tea.Msg) (ImplementPreview, tea.Cmd) {
	var cmd tea.Cmd
	viewport := p.layout.GetViewport()
	*viewport, cmd = viewport.Update(msg)
	return p, cmd
}

// View renders the preview
func (p ImplementPreview) View() string {
	if !p.ready {
		return implementPreviewPlaceholderStyle.Render("Implementation progress will appear here...")
	}

	if len(p.tasks) == 0 {
		placeholder := implementPreviewPlaceholderStyle.Render("No tasks loaded yet.\nTasks will appear as implementation begins.")
		p.layout.SetContent(placeholder)
	}

	return p.layout.View()
}

// SetSize updates the preview dimensions
func (p *ImplementPreview) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.layout.SetSize(width, height)
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

// SetImplementPhase updates the current workflow phase
func (p *ImplementPreview) SetImplementPhase(phase types.WorkflowPhase) {
	p.phase = phase
	p.layout.SetPhase(phase)
}

// GetPhase returns the current phase for header display
func (p ImplementPreview) GetPhase() types.PreviewPhase {
	return p.phase
}

// SetPhase sets the current phase for header display (PreviewComponent interface)
func (p *ImplementPreview) SetPhase(phase types.PreviewPhase) {
	if workflowPhase, ok := phase.(types.WorkflowPhase); ok {
		p.SetImplementPhase(workflowPhase)
	}
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

	p.layout.SetContent(strings.Join(parts, "\n"))
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
		p.layout.SetContent(strings.Join(parts, "\n"))
		return
	}

	if len(p.workPackages) == 0 {
		parts = append(parts, implementPreviewPlaceholderStyle.Render("No progress data available"))
		p.layout.SetContent(strings.Join(parts, "\n"))
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

	p.layout.SetContent(strings.Join(parts, "\n"))
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

// parseProgressContent parses task-progress.md file content to extract work packages.
// This parser handles the markdown format generated by the task implementation system.
//
// Expected format:
//
//	# Implementation Progress
//	**Status**: <status>
//	**Updated**: <timestamp>
//	**Progress**: <progress info>
//
//	## Work Package WP01: Name (status)
//	**Started**: <timestamp>
//	**Completed**: <timestamp>  // optional
//	**Goal**: <description>
//
//	- [x] T001: Task description (completed at HH:MM:SS)
//	- [~] T002: Task description (in_progress since HH:MM:SS)
//	- [!] T003: Task description (blocked: reason)
//	- [s] T004: Task description (skipped: reason)
//	- [ ] T005: Task description
//
// The parser is designed to be fault-tolerant and will skip malformed entries
// while continuing to process valid ones.
func (p *ImplementPreview) parseProgressContent(content string) ([]tasks.WorkPackageProgress, error) {
	if strings.TrimSpace(content) == "" {
		return []tasks.WorkPackageProgress{}, nil
	}

	lines := strings.Split(content, "\n")
	var workPackages []tasks.WorkPackageProgress
	var currentWP *tasks.WorkPackageProgress

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and non-relevant lines
		if line == "" || (!strings.HasPrefix(line, "##") && !strings.HasPrefix(line, "- [")) {
			continue
		}

		// Check for work package header
		if strings.HasPrefix(line, "## Work Package ") {
			// If we have a current work package, save it
			if currentWP != nil {
				workPackages = append(workPackages, *currentWP)
			}

			// Parse new work package header
			wp, nextIndex, err := parseWorkPackageHeader(lines, i)
			if err != nil {
				// Log error but continue parsing
				continue
			}
			currentWP = wp
			i = nextIndex - 1 // -1 because the loop will increment
			continue
		}

		// Check for task line
		if strings.HasPrefix(line, "- [") && currentWP != nil {
			task, err := parseTaskLine(line)
			if err != nil {
				// Log error but continue parsing
				continue
			}
			currentWP.Tasks = append(currentWP.Tasks, *task)
		}
	}

	// Don't forget the last work package
	if currentWP != nil {
		workPackages = append(workPackages, *currentWP)
	}

	return workPackages, nil
}

// parseWorkPackageHeader parses a work package header and its associated metadata.
//
// Parses lines starting with "## Work Package ID: Name (status)" followed by
// optional metadata lines like "**Started**: timestamp" and "**Goal**: description.
//
// Returns the parsed WorkPackageProgress, the index of the next line to process,
// and any parsing error. The parser stops at empty lines or when it encounters
// task lines or other section headers.
func parseWorkPackageHeader(lines []string, startIndex int) (*tasks.WorkPackageProgress, int, error) {
	if startIndex >= len(lines) {
		return nil, startIndex, fmt.Errorf("invalid start index")
	}

	headerLine := strings.TrimSpace(lines[startIndex])

	// Parse line like: "## Work Package WP01: Authentication Setup (in_progress)"
	headerRegex := regexp.MustCompile(`## Work Package ([A-Z0-9]+): (.+?) \((.+)\)`)
	matches := headerRegex.FindStringSubmatch(headerLine)

	if len(matches) != 4 {
		return nil, startIndex + 1, fmt.Errorf("invalid work package header format: %s", headerLine)
	}

	wp := &tasks.WorkPackageProgress{
		ID:     matches[1],
		Name:   matches[2],
		Status: parseTaskProgressState(matches[3]),
		Tasks:  []tasks.TaskProgress{},
	}

	// Parse metadata lines that follow
	nextIndex := startIndex + 1
	for nextIndex < len(lines) {
		line := strings.TrimSpace(lines[nextIndex])

		// Stop when we hit an empty line or another section
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "- [") {
			break
		}

		// Parse metadata like "**Started**: 2025-11-30T10:00:00Z"
		key, value, err := parseMetadataLine(line)
		if err != nil {
			nextIndex++
			continue
		}

		switch key {
		case "Started":
			if timestamp, err := parseTimestamp(value); err == nil {
				wp.StartedAt = &timestamp
			}
		case "Completed":
			if timestamp, err := parseTimestamp(value); err == nil {
				wp.CompletedAt = &timestamp
			}
		case "Goal":
			wp.Goal = value
		}

		nextIndex++
	}

	return wp, nextIndex, nil
}

// parseTaskLine parses a task line from markdown checkbox format.
//
// Supported formats:
//   - [x] T001: Task description (completed at HH:MM:SS)
//   - [~] T002: Task description (in_progress since HH:MM:SS)
//   - [!] T003: Task description (blocked: reason)
//   - [s] T004: Task description (skipped: reason)
//   - [ ] T005: Task description
//
// Checkbox states:
//
//	[x] = completed, [~] = in_progress, [!] = blocked, [s] = skipped, [ ] = pending
//
// The optional metadata in parentheses provides timing information or block reasons.
func parseTaskLine(line string) (*tasks.TaskProgress, error) {
	// Parse line like: "- [x] T001: Create user model (completed at 10:15:00)"
	// or: "- [~] T002: Implement login endpoint (in_progress since 10:20:00)"
	taskRegex := regexp.MustCompile(`- \[(.)\] ([A-Z0-9]+): (.+?)(?:\s*\((.+)\))?$`)
	matches := taskRegex.FindStringSubmatch(line)

	if len(matches) < 4 {
		return nil, fmt.Errorf("invalid task line format: %s", line)
	}

	checkbox := matches[1]
	taskID := matches[2]
	description := matches[3]
	metadata := ""
	if len(matches) > 4 {
		metadata = matches[4]
	}

	// Convert checkbox to TaskProgressState
	status := parseTaskProgressStateFromCheckbox(checkbox)

	task := &tasks.TaskProgress{
		ID:          taskID,
		Description: description,
		Status:      status,
	}

	// Parse metadata if present
	if metadata != "" {
		parseTaskMetadata(task, metadata)
	}

	return task, nil
}

// parseTaskProgressStateFromCheckbox converts a checkbox marker to TaskProgressState.
// Supports: x=completed, ~=in_progress, !=blocked, s=skipped, (space)=pending
func parseTaskProgressStateFromCheckbox(checkbox string) tasks.TaskProgressState {
	switch checkbox {
	case "x":
		return tasks.TaskStateCompleted
	case "~":
		return tasks.TaskStateInProgress
	case "!":
		return tasks.TaskStateBlocked
	case "s":
		return tasks.TaskStateSkipped
	default:
		return tasks.TaskStatePending
	}
}

// parseTaskMetadata parses timing and status metadata from task lines.
// Handles formats like "completed at HH:MM:SS", "in_progress since HH:MM:SS",
// and "blocked: reason" to populate task timing and block information.
func parseTaskMetadata(task *tasks.TaskProgress, metadata string) {
	// Parse metadata like "completed at 10:15:00" or "in_progress since 10:20:00"
	if strings.Contains(metadata, "completed at") {
		timeStr := strings.TrimSpace(strings.Split(metadata, "completed at")[1])
		if timestamp, err := parseTimestamp(timeStr); err == nil {
			task.CompletedAt = &timestamp
		}
	} else if strings.Contains(metadata, "since") {
		timeStr := strings.TrimSpace(strings.Split(metadata, "since")[1])
		if timestamp, err := parseTimestamp(timeStr); err == nil {
			task.StartedAt = &timestamp
		}
	} else if strings.Contains(metadata, "blocked:") {
		reason := strings.TrimSpace(strings.Split(metadata, "blocked:")[1])
		task.BlockReason = reason
	}
}

// parseTaskProgressState converts string status to TaskProgressState enum.
// Handles status strings from work package headers with trimming and defaults to pending.
func parseTaskProgressState(statusStr string) tasks.TaskProgressState {
	switch strings.TrimSpace(statusStr) {
	case "completed":
		return tasks.TaskStateCompleted
	case "in_progress":
		return tasks.TaskStateInProgress
	case "blocked":
		return tasks.TaskStateBlocked
	case "skipped":
		return tasks.TaskStateSkipped
	default:
		return tasks.TaskStatePending
	}
}

// parseMetadataLine parses metadata lines with bold markdown format.
// Extracts key-value pairs from lines like "**Started**: 2025-11-30T10:00:00Z".
func parseMetadataLine(line string) (key, value string, err error) {
	metadataRegex := regexp.MustCompile(`\*\*(.+?)\*\*:\s*(.+)`)
	matches := metadataRegex.FindStringSubmatch(line)

	if len(matches) != 3 {
		return "", "", fmt.Errorf("invalid metadata format: %s", line)
	}

	return matches[1], matches[2], nil
}

// parseTimestamp parses timestamps in various formats used in the progress file.
// Supports ISO 8601 formats and time-only formats (HH:MM:SS, HH:MM).
// For time-only formats, assumes today's date.
func parseTimestamp(timeStr string) (time.Time, error) {
	timeStr = strings.TrimSpace(timeStr)

	// Common timestamp formats to try
	formats := []string{
		"2006-01-02T15:04:05Z",      // ISO 8601 with Z
		"2006-01-02T15:04:05-07:00", // ISO 8601 with timezone
		"15:04:05",                  // Time only (use today's date)
		"15:04",                     // Hour:minute only
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			// For time-only formats, use today's date
			if format == "15:04:05" || format == "15:04" {
				now := time.Now()
				return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location()), nil
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timeStr)
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
