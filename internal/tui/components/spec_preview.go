package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// SpecPreview manages the specification preview pane
type SpecPreview struct {
	layout           PreviewLayout
	content          string
	renderedContent  string // Cached rendered markdown content
	validationIssues []string
	phase            types.SpecifyPhase
	width            int
	height           int
	visible          bool
	ready            bool
}

// NewSpecPreview creates a new spec preview
func NewSpecPreview(width, height int) SpecPreview {
	// Initialize with a default phase - will be updated via SetPhase
	defaultPhase := types.PhaseDiscovery
	layout := NewPreviewLayout(defaultPhase)
	layout.SetSize(width, height)

	return SpecPreview{
		layout:  layout,
		phase:   defaultPhase,
		width:   width,
		height:  height,
		visible: true,
		ready:   true,
	}
}

// Init implements tea.Model
func (s SpecPreview) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (s SpecPreview) Update(msg tea.Msg) (SpecPreview, tea.Cmd) {
	var cmd tea.Cmd

	// Handle key events for scrolling explicitly
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			s.layout.GetViewport().LineUp(1)
			return s, nil
		case "down", "j":
			s.layout.GetViewport().LineDown(1)
			return s, nil
		case "pgup":
			s.layout.GetViewport().HalfViewUp()
			return s, nil
		case "pgdown":
			s.layout.GetViewport().HalfViewDown()
			return s, nil
		}
	}

	// Delegate other messages to viewport
	viewport := s.layout.GetViewport()
	*viewport, cmd = viewport.Update(msg)
	return s, cmd
}

// View implements tea.Model
func (s SpecPreview) View() string {
	if !s.ready {
		return "Initializing preview..."
	}

	if !s.visible {
		return specHiddenStyle.Render("Preview hidden (Tab to show)")
	}

	// Use layout view which handles header and viewport composition
	return s.layout.View()
}

// SetSize updates the preview dimensions
func (s *SpecPreview) SetSize(width, height int) {
	widthChanged := s.width != width
	s.width = width
	s.height = height
	s.layout.SetSize(width, height)

	// Re-render if width changed (affects markdown rendering) and we have content
	if widthChanged && s.content != "" {
		s.updateRenderedContent()
	}
}

// SetContent updates the spec content
func (s *SpecPreview) SetContent(content string) {
	if s.content != content {
		s.content = content
		s.updateRenderedContent()
	}
}

// updateRenderedContent re-renders the content and updates the viewport
func (s *SpecPreview) updateRenderedContent() {
	rendered := s.renderContent()
	s.renderedContent = rendered

	// Wrap content to viewport width to ensure proper line breaks
	// The viewport counts lines by \n, so content that wraps visually
	// without \n characters will break scrolling
	viewport := s.layout.GetViewport()
	wrappedContent := lipgloss.NewStyle().Width(viewport.Width).Render(rendered)
	s.layout.SetContent(wrappedContent)
}

// SetSpecPhase updates the current workflow phase
func (s *SpecPreview) SetSpecPhase(phase types.SpecifyPhase) {
	s.phase = phase
	s.layout.SetPhase(phase)
}

// GetPhase returns the current phase for header display
func (s SpecPreview) GetPhase() types.PreviewPhase {
	return s.phase
}

// SetPhase sets the current phase for header display (PreviewComponent interface)
func (s *SpecPreview) SetPhase(phase types.PreviewPhase) {
	if specPhase, ok := phase.(types.SpecifyPhase); ok {
		s.SetSpecPhase(specPhase)
	}
}

// SetValidationIssues updates the validation issues
func (s *SpecPreview) SetValidationIssues(issues []string) {
	s.validationIssues = issues
}

// Toggle toggles the preview visibility
func (s *SpecPreview) Toggle() {
	s.visible = !s.visible
}

// IsVisible returns whether the preview is visible
func (s *SpecPreview) IsVisible() bool {
	return s.visible
}


// renderContent renders the spec content with markdown
func (s *SpecPreview) renderContent() string {
	if s.content == "" {
		return s.renderEmptyState()
	}

	// Calculate render width with minimum validation
	// Width might be 0 before first WindowSizeMsg
	renderWidth := s.width - 8
	if renderWidth < 40 {
		renderWidth = 40 // Minimum width for readable markdown
	}

	// Use shared markdown renderer with proper width handling
	renderer := GetMarkdownRenderer()
	rendered, err := renderer.Render(s.content, renderWidth)
	if err != nil {
		// Fallback to plain content if rendering fails
		rendered = s.content
	}

	// Add validation issues if present
	if len(s.validationIssues) > 0 && s.phase == types.PhaseValidation {
		issuesSection := s.renderValidationIssues()
		rendered = lipgloss.JoinVertical(lipgloss.Left, rendered, "", issuesSection)
	}

	return rendered
}

// renderEmptyState renders the empty state message
func (s *SpecPreview) renderEmptyState() string {
	var message string
	switch s.phase {
	case types.PhaseDiscovery:
		message = "Specification will appear here after discovery phase..."
	case types.PhaseGeneration:
		message = "Generating specification..."
	case types.PhaseValidation:
		message = "Validating specification..."
	case types.PhaseComplete:
		message = "Specification complete!"
	default:
		message = "Waiting for specification..."
	}

	return specEmptyStyle.Render(message)
}

// renderValidationIssues renders validation issues as a list
func (s *SpecPreview) renderValidationIssues() string {
	var issues []string
	issues = append(issues, validationHeaderStyle.Render("⚠ Validation Issues:"))

	for i, issue := range s.validationIssues {
		issues = append(issues, validationIssueStyle.Render(
			lipgloss.JoinHorizontal(lipgloss.Left, "  ", string(rune(i+1))+".", " ", issue),
		))
	}

	return strings.Join(issues, "\n")
}


// Styles for spec preview using Sekkei Design System theme
var (
	specEmptyStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Padding(2, 2)

	specHiddenStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Align(lipgloss.Center).
			Padding(2, 2)

	validationHeaderStyle = lipgloss.NewStyle().
				Foreground(theme.Warning).
				Bold(true).
				Padding(1, 0, 0, 0)

	validationIssueStyle = lipgloss.NewStyle().
				Foreground(theme.Warning)
)
