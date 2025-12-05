package components

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// PreviewLayout provides a standardized layout for preview components.
// It manages a header and viewport, automatically calculating proper dimensions
// to ensure the header is always visible and the viewport content is positioned correctly.
type PreviewLayout struct {
	header   PreviewHeader
	viewport viewport.Model
	width    int
	height   int
}

// NewPreviewLayout creates a new preview layout with the given phase.
func NewPreviewLayout(phase types.PreviewPhase) PreviewLayout {
	header := NewPreviewHeader(phase, 0)
	vp := viewport.New(0, 0)
	vp.YPosition = 0

	return PreviewLayout{
		header:   header,
		viewport: vp,
		width:    0,
		height:   0,
	}
}

// SetSize updates the layout dimensions and recalculates viewport size.
// This is the critical fix that ensures headers are always visible by
// dynamically calculating header height using lipgloss.Height().
func (pl *PreviewLayout) SetSize(width, height int) {
	pl.width = width
	pl.height = height
	pl.header.SetWidth(width)

	// Critical fix: Dynamic header height calculation
	// This replaces the fixed height calculations that caused header clipping
	headerHeight := lipgloss.Height(pl.header.Render())

	// Calculate viewport dimensions with proper padding
	pl.viewport.Width = width - 4
	pl.viewport.Height = height - headerHeight - 4

	// Ensure minimum viewport height to prevent negative heights
	if pl.viewport.Height < 1 {
		pl.viewport.Height = 1
	}
}

// View renders the complete layout with header and viewport.
func (pl PreviewLayout) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		pl.header.Render(),
		pl.viewport.View(),
	)
}

// SetPhase updates the layout's phase and refreshes header sizing.
func (pl *PreviewLayout) SetPhase(phase types.PreviewPhase) {
	pl.header.SetPhase(phase)
	// Recalculate layout in case the new phase affects header height
	pl.SetSize(pl.width, pl.height)
}

// GetPhase returns the current phase.
func (pl PreviewLayout) GetPhase() types.PreviewPhase {
	return pl.header.GetPhase()
}

// GetViewport returns a pointer to the viewport for content management.
func (pl *PreviewLayout) GetViewport() *viewport.Model {
	return &pl.viewport
}

// SetContent sets the viewport content.
func (pl *PreviewLayout) SetContent(content string) {
	pl.viewport.SetContent(content)
}

// GetWidth returns the current layout width.
func (pl PreviewLayout) GetWidth() int {
	return pl.width
}

// GetHeight returns the current layout height.
func (pl PreviewLayout) GetHeight() int {
	return pl.height
}

// GetViewportHeight returns the calculated viewport height.
// This is useful for components that need to know the available content area.
func (pl PreviewLayout) GetViewportHeight() int {
	return pl.viewport.Height
}

// GetHeaderHeight returns the calculated header height.
// This is useful for debugging and layout calculations.
func (pl PreviewLayout) GetHeaderHeight() int {
	return lipgloss.Height(pl.header.Render())
}