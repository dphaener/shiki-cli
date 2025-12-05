package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// TasksPreview displays a preview of the generated tasks.md file
type TasksPreview struct {
	layout  PreviewLayout
	content string
	phase   types.WorkflowPhase
	width   int
	height  int
	ready   bool
}

// NewTasksPreview creates a new tasks preview component
func NewTasksPreview(width, height int) TasksPreview {
	// Initialize with a default task phase - will be updated via SetPhase
	defaultPhase := types.WorkflowPhaseTasks
	layout := NewPreviewLayout(defaultPhase)
	layout.SetSize(width, height)

	return TasksPreview{
		layout: layout,
		phase:  defaultPhase,
		width:  width,
		height: height,
		ready:  false,
	}
}

// Init initializes the preview component
func (p TasksPreview) Init() tea.Cmd {
	return nil
}

// Update handles messages for the preview
func (p TasksPreview) Update(msg tea.Msg) (TasksPreview, tea.Cmd) {
	var cmd tea.Cmd
	viewport := p.layout.GetViewport()
	*viewport, cmd = viewport.Update(msg)
	return p, cmd
}

// View renders the preview
func (p TasksPreview) View() string {
	if !p.ready {
		return tasksPreviewPlaceholderStyle.Render("Tasks will appear here once generated...")
	}

	if p.content == "" {
		placeholder := tasksPreviewPlaceholderStyle.Render("No tasks generated yet.\nThe AI will create tasks.md based on the spec and plan.")
		p.layout.SetContent(placeholder)
	}

	return p.layout.View()
}

// SetSize updates the preview dimensions
func (p *TasksPreview) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.layout.SetSize(width, height)
	p.ready = true

	// Re-render content with new size
	if p.content != "" {
		p.renderContent()
	}
}

// SetContent updates the tasks content
func (p *TasksPreview) SetContent(content string) {
	p.content = content
	if p.ready {
		p.renderContent()
	}
}

// SetTaskPhase updates the current workflow phase
func (p *TasksPreview) SetTaskPhase(phase types.WorkflowPhase) {
	p.phase = phase
	p.layout.SetPhase(phase)
}

// GetPhase returns the current phase for header display
func (p TasksPreview) GetPhase() types.PreviewPhase {
	return p.phase
}

// SetPhase sets the current phase for header display (PreviewComponent interface)
func (p *TasksPreview) SetPhase(phase types.PreviewPhase) {
	if workflowPhase, ok := phase.(types.WorkflowPhase); ok {
		p.SetTaskPhase(workflowPhase)
	}
}

// renderContent renders the markdown content
func (p *TasksPreview) renderContent() {
	// Simple markdown rendering
	viewport := p.layout.GetViewport()
	rendered := renderTasksMarkdown(p.content, viewport.Width)
	p.layout.SetContent(rendered)
	// Scroll to bottom to show latest content
	viewport.GotoBottom()
}

// renderTasksMarkdown renders tasks markdown with basic formatting
func renderTasksMarkdown(content string, width int) string {
	lines := strings.Split(content, "\n")
	var rendered []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "# ") {
			// H1
			text := strings.TrimPrefix(trimmed, "# ")
			rendered = append(rendered, tasksPreviewH1Style.Render(text))
		} else if strings.HasPrefix(trimmed, "## ") {
			// H2
			text := strings.TrimPrefix(trimmed, "## ")
			rendered = append(rendered, tasksPreviewH2Style.Render(text))
		} else if strings.HasPrefix(trimmed, "### ") {
			// H3
			text := strings.TrimPrefix(trimmed, "### ")
			rendered = append(rendered, tasksPreviewH3Style.Render(text))
		} else if strings.HasPrefix(trimmed, "- [ ]") {
			// Unchecked checkbox
			text := strings.TrimPrefix(trimmed, "- [ ]")
			rendered = append(rendered, tasksPreviewCheckboxStyle.Render("☐"+text))
		} else if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
			// Checked checkbox
			text := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- [x]"), "- [X]")
			rendered = append(rendered, tasksPreviewCheckedStyle.Render("☑"+text))
		} else if strings.HasPrefix(trimmed, "- ") {
			// List item
			rendered = append(rendered, tasksPreviewListStyle.Render(line))
		} else if strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**") {
			// Bold text
			text := strings.Trim(trimmed, "*")
			rendered = append(rendered, tasksPreviewBoldStyle.Render(text))
		} else {
			rendered = append(rendered, tasksPreviewTextStyle.Render(line))
		}
	}

	return strings.Join(rendered, "\n")
}

// Styles for tasks preview
var (
	tasksPreviewPlaceholderStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Italic(true).
		Align(lipgloss.Center)

	tasksPreviewH1Style = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		MarginBottom(1)

	tasksPreviewH2Style = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	tasksPreviewH3Style = lipgloss.NewStyle().
		Foreground(theme.Text).
		Bold(true)

	tasksPreviewTextStyle = lipgloss.NewStyle().
		Foreground(theme.Text)

	tasksPreviewListStyle = lipgloss.NewStyle().
		Foreground(theme.Text).
		PaddingLeft(2)

	tasksPreviewCheckboxStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted)

	tasksPreviewCheckedStyle = lipgloss.NewStyle().
		Foreground(theme.Success)

	tasksPreviewBoldStyle = lipgloss.NewStyle().
		Foreground(theme.Text).
		Bold(true)
)
