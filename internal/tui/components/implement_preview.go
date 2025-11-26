package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	viewport  viewport.Model
	tasks     []ImplementTask
	width     int
	height    int
	ready     bool
	summary   string
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
