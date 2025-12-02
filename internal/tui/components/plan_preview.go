package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// PlanPreview manages the plan preview pane
type PlanPreview struct {
	viewport        viewport.Model
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
	vp := viewport.New(width-4, height-4)
	vp.YPosition = 0

	return PlanPreview{
		viewport: vp,
		width:    width,
		height:   height,
		visible:  true,
		ready:    true,
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
			p.viewport.LineUp(1)
			return p, nil
		case "down", "j":
			p.viewport.LineDown(1)
			return p, nil
		case "pgup":
			p.viewport.HalfViewUp()
			return p, nil
		case "pgdown":
			p.viewport.HalfViewDown()
			return p, nil
		}
	}

	// Delegate other messages to viewport
	p.viewport, cmd = p.viewport.Update(msg)
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

	// Add header with phase indicator
	header := p.renderHeader()

	// Use cached viewport content (set by SetContent/SetSize)
	// Don't call SetContent here - it resets scroll position
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		p.viewport.View(),
	)
}

// SetSize updates the preview dimensions
func (p *PlanPreview) SetSize(width, height int) {
	widthChanged := p.width != width
	p.width = width
	p.height = height
	p.viewport.Width = width - 4
	p.viewport.Height = height - 4

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
	wrappedContent := lipgloss.NewStyle().Width(p.viewport.Width).Render(rendered)
	p.viewport.SetContent(wrappedContent)
}

// SetPhase updates the current workflow phase
func (p *PlanPreview) SetPhase(phase types.PlanPhase) {
	p.phase = phase
}

// Toggle toggles the preview visibility
func (p *PlanPreview) Toggle() {
	p.visible = !p.visible
}

// IsVisible returns whether the preview is visible
func (p *PlanPreview) IsVisible() bool {
	return p.visible
}

// renderHeader renders the preview header with phase info
func (p *PlanPreview) renderHeader() string {
	phaseIcon := p.getPhaseIcon()
	phaseText := p.getPhaseText()

	header := planHeaderStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			phaseIcon,
			" ",
			phaseText,
		),
	)

	separator := planSeparatorStyle.Render(strings.Repeat("─", p.width-4))

	return lipgloss.JoinVertical(lipgloss.Left, header, separator)
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

// getPhaseIcon returns an icon for the current phase
func (p *PlanPreview) getPhaseIcon() string {
	switch p.phase {
	case types.PlanPhaseInterrogation:
		return planPhaseInterrogationIcon.Render("💬")
	case types.PlanPhaseResearch:
		return planPhaseResearchIcon.Render("🔍")
	case types.PlanPhaseDesign:
		return planPhaseDesignIcon.Render("📐")
	case types.PlanPhaseComplete:
		return planPhaseCompleteIcon.Render("✅")
	default:
		return ""
	}
}

// getPhaseText returns text for the current phase
func (p *PlanPreview) getPhaseText() string {
	switch p.phase {
	case types.PlanPhaseInterrogation:
		return "Planning Interrogation"
	case types.PlanPhaseResearch:
		return "Research Phase"
	case types.PlanPhaseDesign:
		return "Design Phase"
	case types.PlanPhaseComplete:
		return "Plan Complete"
	default:
		return "Plan Preview"
	}
}

// Styles for plan preview using Sekkei Design System theme
var (
	planHeaderStyle = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1)

	planSeparatorStyle = lipgloss.NewStyle().
				Foreground(theme.Border)

	planEmptyStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Padding(2, 2)

	planHiddenStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Align(lipgloss.Center).
			Padding(2, 2)

	planPhaseInterrogationIcon = lipgloss.NewStyle().
					Foreground(theme.Info)

	planPhaseResearchIcon = lipgloss.NewStyle().
				Foreground(theme.Warning)

	planPhaseDesignIcon = lipgloss.NewStyle().
				Foreground(theme.Success)

	planPhaseCompleteIcon = lipgloss.NewStyle().
				Foreground(theme.Success)
)
