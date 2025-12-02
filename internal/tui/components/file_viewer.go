package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
)

// Styles for file viewer using Sekkei Design System theme
var (
	fileNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			Padding(0, 1)

	fileLineNumberStyle = lipgloss.NewStyle().
				Foreground(theme.TextMuted).
				Width(4).
				Align(lipgloss.Right)
)

// RenderFileViewer renders the file content viewer pane
func RenderFileViewer(content, filename string, width, height int, scrollOffset int, isActive bool) string {
	// Title with filename
	title := fileNameStyle.Render(fmt.Sprintf("File: %s", filename))

	// Determine border style based on active state
	borderStyle := paneBorderStyle
	if isActive {
		borderStyle = selectedPaneBorderStyle
	}

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Calculate visible range
	visibleHeight := height - 4 // Account for borders and title
	startLine := scrollOffset
	endLine := scrollOffset + visibleHeight

	if startLine < 0 {
		startLine = 0
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > len(lines) {
		startLine = len(lines)
	}

	// Build visible content with line numbers
	var visibleLines []string
	for i := startLine; i < endLine; i++ {
		lineNum := fileLineNumberStyle.Render(fmt.Sprintf("%4d", i+1))
		line := lines[i]

		// Truncate long lines to fit width
		maxLineWidth := width - 10 // Account for line number and padding
		if len(line) > maxLineWidth {
			line = line[:maxLineWidth-3] + "..."
		}

		visibleLines = append(visibleLines, lineNum+" "+line)
	}

	// Add scroll indicator if content is scrolled
	var scrollIndicator string
	if len(lines) > visibleHeight {
		scrollPct := float64(startLine) / float64(len(lines)-visibleHeight) * 100
		scrollIndicator = fmt.Sprintf(" (%.0f%%)", scrollPct)
	}

	// Join visible lines
	fileContent := strings.Join(visibleLines, "\n")

	// Build final pane
	titleWithScroll := title + scrollIndicator
	pane := borderStyle.
		Width(width - 2).
		Height(height - 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleWithScroll, fileContent))

	return pane
}
