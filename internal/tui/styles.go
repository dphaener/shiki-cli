package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/theme"
)

// Color aliases using Sekkei Design System theme
var (
	colorPrimary   = theme.Primary
	colorSecondary = theme.Border
	colorSuccess   = theme.Success
	colorError     = theme.Error
	colorWarning   = theme.Warning
	colorInfo      = theme.Info
	colorText      = theme.Text
	colorMuted     = theme.TextMuted
)

// Header styles
var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 1)

	HeaderKeyStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	HeaderValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText)
)

// Pane styles
var (
	PaneTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	PaneBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSecondary)

	SelectedPaneBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary)
)

// Turn list styles
var (
	TurnStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedTurnStyle = TurnStyle.Copy().
				Background(colorSecondary).
				Bold(true)

	TurnNumberStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(4)

	TurnAgentStyle = lipgloss.NewStyle().
			Bold(true).
			Width(15)

	TurnStatusRunningStyle = lipgloss.NewStyle().
				Foreground(colorInfo).
				Bold(true)

	TurnStatusCompletedStyle = lipgloss.NewStyle().
					Foreground(colorSuccess)

	TurnStatusErrorStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true)

	TurnDurationStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	TurnCostStyle = lipgloss.NewStyle().
			Foreground(colorWarning)
)

// File viewer styles
var (
	FileNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	FileContentStyle = lipgloss.NewStyle().
				Padding(1)

	FileLineNumberStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Width(4).
				Align(lipgloss.Right)

	FilePathStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
)

// Status bar styles
var (
	StatusBarStyle = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(colorText).
			Padding(0, 1)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText).
				Background(colorSecondary).
				Padding(0, 1)

	StatusBarValueStyle = lipgloss.NewStyle().
				Foreground(colorText)
)

// Session status styles
var (
	StatusRunningStyle = lipgloss.NewStyle().
				Foreground(colorInfo).
				Bold(true)

	StatusPausedStyle = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true)

	StatusCompletedStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true)
)

// GetStatusStyle returns the appropriate style for a status string (exported)
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return StatusRunningStyle
	case "paused":
		return StatusPausedStyle
	case "completed":
		return StatusCompletedStyle
	case "error", "incomplete":
		return StatusErrorStyle
	default:
		return lipgloss.NewStyle()
	}
}
