package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
)

// Styles for status bar using Sekkei Design System theme
var (
	statusBarStyle = lipgloss.NewStyle().
			Background(theme.PrimaryDark).
			Foreground(theme.Text).
			Padding(0, 1)

	statusBarKeyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Text).
				Background(theme.Border).
				Padding(0, 1)
)

// RenderStatusBar renders the bottom status bar with keybindings
func RenderStatusBar(agentCount int, selectedAgentIndex int, width int, recentTool string) string {
	// Agent count
	agents := fmt.Sprintf("Agents: %d", agentCount)

	// Current agent indicator
	agentIndicator := fmt.Sprintf("Agent: %d/%d", selectedAgentIndex+1, agentCount)

	// Recent tool activity
	toolActivity := ""
	if recentTool != "" {
		toolActivity = fmt.Sprintf("→ %s", recentTool)
	}

	// Keybindings help
	keys := []string{
		"↑/↓:Scroll",
		"←/→:Switch Agent",
		"q:Quit",
	}

	keysStr := lipgloss.JoinHorizontal(lipgloss.Left,
		statusBarKeyStyle.Render(keys[0]),
		" ",
		statusBarKeyStyle.Render(keys[1]),
		" ",
		statusBarKeyStyle.Render(keys[2]),
	)

	// Build status bar layout
	leftParts := []string{agents, " │ ", agentIndicator}
	if toolActivity != "" {
		leftParts = append(leftParts, " │ ", toolActivity)
	}
	leftSection := lipgloss.JoinHorizontal(lipgloss.Left, leftParts...)

	// Calculate spacing
	usedWidth := lipgloss.Width(leftSection) + lipgloss.Width(keysStr)
	spacing := width - usedWidth - 4
	if spacing < 1 {
		spacing = 1
	}

	statusBar := lipgloss.JoinHorizontal(lipgloss.Left,
		leftSection,
		lipgloss.NewStyle().Width(spacing).Render(""),
		keysStr,
	)

	return statusBarStyle.Width(width).Render(statusBar)
}
