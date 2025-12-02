package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// Styles using Sekkei Design System theme
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Text).
			Background(theme.PrimaryDark).
			Padding(0, 1)

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(theme.Info).
				Bold(true)

	statusPausedStyle = lipgloss.NewStyle().
				Foreground(theme.Warning).
				Bold(true)

	statusCompletedStyle = lipgloss.NewStyle().
				Foreground(theme.Success).
				Bold(true)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(theme.Error).
				Bold(true)
)

// RenderHeader renders the top session header
func RenderHeader(session *types.Session, width int) string {
	if session == nil {
		return ""
	}

	// Calculate session duration
	var duration time.Duration
	if session.CompletedAt != nil {
		duration = session.CompletedAt.Sub(session.CreatedAt)
	} else if session.StartedAt != nil {
		duration = time.Since(*session.StartedAt)
	}

	// Format header components
	sessionID := fmt.Sprintf("Session: %s", session.ID[:8])
	taskName := fmt.Sprintf("Task: %s", session.TaskName)
	status := fmt.Sprintf("Status: %s", formatStatus(string(session.Status)))
	durationStr := fmt.Sprintf("Duration: %s", formatDuration(duration))
	cost := fmt.Sprintf("Cost: $%.4f", session.TotalCost)
	turns := fmt.Sprintf("Turn: %d/%d", session.CurrentTurn, session.MaxTurns)

	// Build header with proper spacing
	leftSection := lipgloss.JoinHorizontal(lipgloss.Left,
		sessionID,
		" │ ",
		taskName,
	)

	rightSection := lipgloss.JoinHorizontal(lipgloss.Left,
		status,
		" │ ",
		turns,
		" │ ",
		durationStr,
		" │ ",
		cost,
	)

	// Calculate spacing to fill width
	usedWidth := lipgloss.Width(leftSection) + lipgloss.Width(rightSection)
	spacing := width - usedWidth - 4 // Account for padding
	if spacing < 1 {
		spacing = 1
	}

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		leftSection,
		lipgloss.NewStyle().Width(spacing).Render(""),
		rightSection,
	)

	return headerStyle.Width(width).Render(header)
}

// formatStatus returns styled status string
func formatStatus(status string) string {
	var style lipgloss.Style
	switch status {
	case "running":
		style = statusRunningStyle
	case "paused":
		style = statusPausedStyle
	case "completed":
		style = statusCompletedStyle
	case "error", "incomplete":
		style = statusErrorStyle
	default:
		style = lipgloss.NewStyle()
	}
	return style.Render(status)
}

// formatDuration returns human-readable duration
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
