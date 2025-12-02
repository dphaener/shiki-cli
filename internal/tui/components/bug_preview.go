package components

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// BugPreview displays bug fix artifacts (plan, tasks, implementation progress)
type BugPreview struct {
	viewport    viewport.Model
	width       int
	height      int
	ready       bool
	content     string
	phase       types.BugPhase
	planFile    string
	tasksFile   string
	mdRenderer  *glamour.TermRenderer
}

// NewBugPreview creates a new bug preview component
func NewBugPreview(width, height int) BugPreview {
	vp := viewport.New(width-4, height-4)
	vp.YPosition = 0

	// Create markdown renderer
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-8),
	)

	return BugPreview{
		viewport:   vp,
		width:      width,
		height:     height,
		ready:      false,
		phase:      types.BugPhasePlan,
		mdRenderer: renderer,
	}
}

// Init initializes the preview component
func (p BugPreview) Init() tea.Cmd {
	return nil
}

// Update handles messages for the preview
func (p BugPreview) Update(msg tea.Msg) (BugPreview, tea.Cmd) {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// View renders the preview
func (p BugPreview) View() string {
	if !p.ready {
		return bugPreviewPlaceholderStyle.Render("Bug fix artifacts will appear here as you work...")
	}

	if p.content == "" {
		return bugPreviewPlaceholderStyle.Render(p.getPlaceholderText())
	}

	return p.viewport.View()
}

// getPlaceholderText returns phase-appropriate placeholder text
func (p BugPreview) getPlaceholderText() string {
	switch p.phase {
	case types.BugPhasePlan, types.BugPhaseApprovePlan:
		return "The bug fix plan will appear here once the AI creates it..."
	case types.BugPhaseTasks, types.BugPhaseApproveTasks:
		return "The task breakdown will appear here once the AI creates it..."
	case types.BugPhaseImplement:
		return "Implementation progress will appear here..."
	default:
		return "Bug fix artifacts will appear here..."
	}
}

// SetSize updates the preview dimensions
func (p *BugPreview) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = width
	p.viewport.Height = height
	p.ready = true

	// Update markdown renderer with new width
	p.mdRenderer, _ = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)

	// Re-render content with new size
	if p.content != "" {
		p.renderContent()
	}
}

// SetContent updates the preview content
func (p *BugPreview) SetContent(content string) {
	p.content = content
	if p.ready {
		p.renderContent()
	}
}

// SetPhase updates the current phase
func (p *BugPreview) SetPhase(phase types.BugPhase) {
	p.phase = phase
}

// SetFiles sets the file paths for plan and tasks
func (p *BugPreview) SetFiles(planFile, tasksFile string) {
	p.planFile = planFile
	p.tasksFile = tasksFile
}

// RefreshFromFiles reads content from the appropriate file based on phase
func (p *BugPreview) RefreshFromFiles() {
	var filePath string
	switch p.phase {
	case types.BugPhasePlan, types.BugPhaseApprovePlan:
		filePath = p.planFile
	case types.BugPhaseTasks, types.BugPhaseApproveTasks, types.BugPhaseImplement:
		filePath = p.tasksFile
	}

	if filePath == "" {
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return // File may not exist yet
	}

	// Strip YAML frontmatter if present
	cleanContent := stripBugFrontmatter(string(content))
	p.SetContent(cleanContent)
}

// stripBugFrontmatter removes YAML frontmatter from markdown content
func stripBugFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2])
	}
	return content
}

// renderContent renders the markdown content to the viewport
func (p *BugPreview) renderContent() {
	if p.content == "" {
		p.viewport.SetContent(p.getPlaceholderText())
		return
	}

	// Add phase header
	header := p.getPhaseHeader()

	// Render markdown
	rendered := p.content
	if p.mdRenderer != nil {
		if out, err := p.mdRenderer.Render(p.content); err == nil {
			rendered = out
		}
	}

	fullContent := header + rendered
	p.viewport.SetContent(fullContent)
}

// getPhaseHeader returns a header string for the current phase
func (p BugPreview) getPhaseHeader() string {
	var title string
	switch p.phase {
	case types.BugPhasePlan:
		title = "📋 Bug Fix Plan"
	case types.BugPhaseApprovePlan:
		title = "📋 Bug Fix Plan (Review)"
	case types.BugPhaseTasks:
		title = "📝 Implementation Tasks"
	case types.BugPhaseApproveTasks:
		title = "📝 Implementation Tasks (Review)"
	case types.BugPhaseImplement:
		title = "🔧 Implementation Progress"
	case types.BugPhaseComplete:
		title = "✅ Bug Fix Complete"
	default:
		title = "🐛 Bug Fix"
	}

	return bugPreviewHeaderStyle.Render(title) + "\n\n"
}

// GetPhaseStatus returns a status string for display
func (p BugPreview) GetPhaseStatus() string {
	switch p.phase {
	case types.BugPhasePlan:
		return "Planning"
	case types.BugPhaseApprovePlan:
		return "Review Plan"
	case types.BugPhaseTasks:
		return "Creating Tasks"
	case types.BugPhaseApproveTasks:
		return "Review Tasks"
	case types.BugPhaseImplement:
		return "Implementing"
	case types.BugPhaseComplete:
		return "Complete"
	default:
		return string(p.phase)
	}
}

// Styles for bug preview
var (
	bugPreviewPlaceholderStyle = lipgloss.NewStyle().
					Foreground(theme.TextMuted).
					Italic(true).
					Align(lipgloss.Center)

	bugPreviewHeaderStyle = lipgloss.NewStyle().
				Foreground(theme.Primary).
				Bold(true).
				MarginBottom(1)

	bugPreviewPhaseStyle = lipgloss.NewStyle().
				Foreground(theme.Info).
				Bold(true)

	bugPreviewStatusStyle = lipgloss.NewStyle().
				Foreground(theme.Success)
)

// Helper function to format phase for display
func FormatBugPhase(phase types.BugPhase) string {
	switch phase {
	case types.BugPhasePlan:
		return "Plan"
	case types.BugPhaseApprovePlan:
		return "Approve Plan"
	case types.BugPhaseTasks:
		return "Tasks"
	case types.BugPhaseApproveTasks:
		return "Approve Tasks"
	case types.BugPhaseImplement:
		return "Implement"
	case types.BugPhaseComplete:
		return "Complete"
	default:
		return string(phase)
	}
}
