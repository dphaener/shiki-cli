package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/components"
)

// View renders the complete TUI (Bubbletea View function)
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing TUI..."
	}

	// Render header (1 line + padding)
	header := components.RenderHeader(m.session, m.width)

	// Calculate pane dimensions
	// Layout: header (2-3 lines) + split panes (remaining) + status bar (1 line)
	headerHeight := lipgloss.Height(header)
	statusBarHeight := 1
	paneHeight := m.height - headerHeight - statusBarHeight

	// Split width for two panes (50/50 split)
	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	// Render left pane (turn history)
	leftPane := components.RenderTurnList(
		m.turnHistory,
		m.selectedTurn,
		leftWidth,
		paneHeight,
		m.selectedPane == "turns",
	)

	// Render right pane (file viewer)
	rightPane := components.RenderFileViewer(
		m.fileContent,
		m.currentFile,
		rightWidth,
		paneHeight,
		m.scrollOffset,
		m.selectedPane == "file",
	)

	// Join panes horizontally
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Render status bar
	statusBar := components.RenderStatusBar(
		len(m.fileList),
		m.selectedPane,
		m.width,
	)

	// Join all sections vertically
	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		panes,
		statusBar,
	)

	return view
}
