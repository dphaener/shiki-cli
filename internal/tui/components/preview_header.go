package components

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// PreviewHeader renders a standardized header for preview components.
// It displays the phase icon and text with a separator line below.
type PreviewHeader struct {
	phase types.PreviewPhase
	width int
}

// NewPreviewHeader creates a new preview header with the given phase and width.
func NewPreviewHeader(phase types.PreviewPhase, width int) PreviewHeader {
	return PreviewHeader{
		phase: phase,
		width: width,
	}
}

// Render renders the header with the phase icon, text, and separator line.
// Returns a string containing the complete header layout.
func (ph PreviewHeader) Render() string {
	icon := ph.phase.GetIcon()
	text := ph.phase.GetText()

	header := previewHeaderStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Left, icon, " ", text),
	)

	// Calculate separator width, ensuring it's never negative
	separatorWidth := int(math.Max(0, float64(ph.width-4)))
	separator := previewSeparatorStyle.Render(strings.Repeat("─", separatorWidth))

	return lipgloss.JoinVertical(lipgloss.Left, header, separator)
}

// SetPhase updates the header's phase.
func (ph *PreviewHeader) SetPhase(phase types.PreviewPhase) {
	ph.phase = phase
}

// SetWidth updates the header's width, affecting the separator line length.
func (ph *PreviewHeader) SetWidth(width int) {
	ph.width = width
}

// GetPhase returns the current phase.
func (ph PreviewHeader) GetPhase() types.PreviewPhase {
	return ph.phase
}

// Consolidated header styles for all preview components.
// These replace the duplicate styles from individual preview files.
var (
	// previewHeaderStyle styles the phase icon and text portion of the header.
	previewHeaderStyle = lipgloss.NewStyle().
				Foreground(theme.Primary).
				Bold(true).
				Padding(0, 1)

	// previewSeparatorStyle styles the horizontal separator line below the header.
	previewSeparatorStyle = lipgloss.NewStyle().
				Foreground(theme.Border)
)