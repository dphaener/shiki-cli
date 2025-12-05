package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// PlanPreview manages the plan preview pane
type PlanPreview struct {
	layout          PreviewLayout
	content         string
	renderedContent string // Cached rendered markdown content
	phase           types.PlanPhase
	width           int
	height          int
	visible         bool
	ready           bool
}

// NewPlanPreview creates a new plan preview
func NewPlanPreview(width, height int) PlanPreview {
	// Initialize with a default phase - will be updated via SetPhase
	defaultPhase := types.PlanPhaseInterrogation
	layout := NewPreviewLayout(defaultPhase)
	layout.SetSize(width, height)

	return PlanPreview{
		layout:  layout,
		phase:   defaultPhase,
		width:   width,
		height:  height,
		visible: true,
		ready:   true,
	}
}

// Init implements tea.Model
func (p PlanPreview) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (p PlanPreview) Update(msg tea.Msg) (PlanPreview, tea.Cmd) {
	var cmd tea.Cmd

	// Handle key events for scrolling explicitly
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			p.layout.GetViewport().LineUp(1)
			return p, nil
		case "down", "j":
			p.layout.GetViewport().LineDown(1)
			return p, nil
		case "pgup":
			p.layout.GetViewport().HalfViewUp()
			return p, nil
		case "pgdown":
			p.layout.GetViewport().HalfViewDown()
			return p, nil
		}
	}

	// Delegate other messages to viewport
	viewport := p.layout.GetViewport()
	*viewport, cmd = viewport.Update(msg)
	return p, cmd
}

// View implements tea.Model
func (p PlanPreview) View() string {
	if !p.ready {
		return "Initializing preview..."
	}

	if !p.visible {
		return planHiddenStyle.Render("Preview hidden (Tab to show)")
	}

	// Use layout view which handles header and viewport composition
	return p.layout.View()
}

// SetSize updates the preview dimensions
func (p *PlanPreview) SetSize(width, height int) {
	widthChanged := p.width != width
	p.width = width
	p.height = height
	p.layout.SetSize(width, height)

	// Re-render if width changed (affects markdown rendering) and we have content
	if widthChanged && p.content != "" {
		p.updateRenderedContent()
	}
}

// SetContent updates the plan content
func (p *PlanPreview) SetContent(content string) {
	if p.content != content {
		p.content = content
		p.updateRenderedContent()
	}
}

// updateRenderedContent re-renders the content and updates the viewport
func (p *PlanPreview) updateRenderedContent() {
	rendered := p.renderContent()
	p.renderedContent = rendered

	// Wrap content to viewport width to ensure proper line breaks
	// The viewport counts lines by \n, so content that wraps visually
	// without \n characters will break scrolling
	viewport := p.layout.GetViewport()
	wrappedContent := lipgloss.NewStyle().Width(viewport.Width).Render(rendered)
	p.layout.SetContent(wrappedContent)
}

// SetPlanPhase updates the current workflow phase
func (p *PlanPreview) SetPlanPhase(phase types.PlanPhase) {
	p.phase = phase
	p.layout.SetPhase(phase)
}

// GetPhase returns the current phase for header display
func (p PlanPreview) GetPhase() types.PreviewPhase {
	return p.phase
}

// SetPhase sets the current phase for header display (PreviewComponent interface)
func (p *PlanPreview) SetPhase(phase types.PreviewPhase) {
	if planPhase, ok := phase.(types.PlanPhase); ok {
		p.SetPlanPhase(planPhase)
	}
}

// Toggle toggles the preview visibility
func (p *PlanPreview) Toggle() {
	p.visible = !p.visible
}

// IsVisible returns whether the preview is visible
func (p *PlanPreview) IsVisible() bool {
	return p.visible
}


// renderContent renders the plan content with markdown
func (p *PlanPreview) renderContent() string {
	if p.content == "" {
		return p.renderEmptyState()
	}

	// Calculate render width with minimum validation
	// Width might be 0 before first WindowSizeMsg
	renderWidth := p.width - 8
	if renderWidth < 40 {
		renderWidth = 40 // Minimum width for readable markdown
	}

	// Use shared markdown renderer with proper width handling
	renderer := GetMarkdownRenderer()
	rendered, err := renderer.Render(p.content, renderWidth)
	if err != nil {
		// Fallback to plain content if rendering fails
		rendered = p.content
	}

	return rendered
}

// renderEmptyState renders the empty state message
func (p *PlanPreview) renderEmptyState() string {
	var message string
	switch p.phase {
	case types.PlanPhaseInterrogation:
		message = "Plan will appear here after interrogation phase..."
	case types.PlanPhaseResearch:
		message = "Researching requirements..."
	case types.PlanPhaseDesign:
		message = "Generating implementation plan..."
	case types.PlanPhaseComplete:
		message = "Implementation plan complete!"
	default:
		message = "Waiting for plan..."
	}

	return planEmptyStyle.Render(message)
}


// Styles for plan preview using Sekkei Design System theme
var (
	planEmptyStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Padding(2, 2)

	planHiddenStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Align(lipgloss.Center).
			Padding(2, 2)
)
