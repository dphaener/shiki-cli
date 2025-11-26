package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/pkg/types"
)

// SpecPreview manages the specification preview pane
type SpecPreview struct {
	viewport         viewport.Model
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
	vp := viewport.New(width-4, height-4)
	vp.YPosition = 0

	return SpecPreview{
		viewport: vp,
		width:    width,
		height:   height,
		visible:  true,
		ready:    true,
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
			s.viewport.LineUp(1)
			return s, nil
		case "down", "j":
			s.viewport.LineDown(1)
			return s, nil
		case "pgup":
			s.viewport.HalfViewUp()
			return s, nil
		case "pgdown":
			s.viewport.HalfViewDown()
			return s, nil
		}
	}

	// Delegate other messages to viewport
	s.viewport, cmd = s.viewport.Update(msg)
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

	// Add header with phase indicator
	header := s.renderHeader()

	// Use cached viewport content (set by SetContent/SetSize)
	// Don't call SetContent here - it resets scroll position
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		s.viewport.View(),
	)
}

// SetSize updates the preview dimensions
func (s *SpecPreview) SetSize(width, height int) {
	widthChanged := s.width != width
	s.width = width
	s.height = height
	s.viewport.Width = width - 4
	s.viewport.Height = height - 4

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
	wrappedContent := lipgloss.NewStyle().Width(s.viewport.Width).Render(rendered)
	s.viewport.SetContent(wrappedContent)
}

// SetPhase updates the current workflow phase
func (s *SpecPreview) SetPhase(phase types.SpecifyPhase) {
	s.phase = phase
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

// renderHeader renders the preview header with phase info
func (s *SpecPreview) renderHeader() string {
	phaseIcon := s.getPhaseIcon()
	phaseText := s.getPhaseText()

	header := specHeaderStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			phaseIcon,
			" ",
			phaseText,
		),
	)

	separator := specSeparatorStyle.Render(strings.Repeat("─", s.width-4))

	return lipgloss.JoinVertical(lipgloss.Left, header, separator)
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

// getPhaseIcon returns an icon for the current phase
func (s *SpecPreview) getPhaseIcon() string {
	switch s.phase {
	case types.PhaseDiscovery:
		return phaseDiscoveryIcon.Render("🔍")
	case types.PhaseGeneration:
		return phaseGenerationIcon.Render("✍️")
	case types.PhaseValidation:
		return phaseValidationIcon.Render("✓")
	case types.PhaseComplete:
		return phaseCompleteIcon.Render("✅")
	default:
		return ""
	}
}

// getPhaseText returns text for the current phase
func (s *SpecPreview) getPhaseText() string {
	switch s.phase {
	case types.PhaseDiscovery:
		return "Discovery Phase"
	case types.PhaseGeneration:
		return "Generating Specification"
	case types.PhaseValidation:
		return "Validating Specification"
	case types.PhaseComplete:
		return "Specification Complete"
	default:
		return "Specification Preview"
	}
}

// Styles for spec preview
var (
	specHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1)

	specSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	specEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Padding(2, 2)

	specHiddenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Align(lipgloss.Center).
			Padding(2, 2)

	phaseDiscoveryIcon = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39"))

	phaseGenerationIcon = lipgloss.NewStyle().
				Foreground(lipgloss.Color("226"))

	phaseValidationIcon = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))

	phaseCompleteIcon = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))

	validationHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true).
				Padding(1, 0, 0, 0)

	validationIssueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))
)
