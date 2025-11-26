package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/theme"
	"github.com/darinhaener/collab/pkg/types"
)

// Styles for turn list using Sekkei Design System theme
var (
	paneTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			Padding(0, 1)

	paneBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border)

	selectedPaneBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.BorderActive)

	turnStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedTurnStyle = turnStyle.Copy().
				Background(theme.Border).
				Bold(true)

	turnNumberStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Width(4)

	turnAgentStyle = lipgloss.NewStyle().
			Bold(true).
			Width(15)

	turnStatusRunningStyle = lipgloss.NewStyle().
				Foreground(theme.Info).
				Bold(true)

	turnStatusCompletedStyle = lipgloss.NewStyle().
					Foreground(theme.Success)

	turnStatusErrorStyle = lipgloss.NewStyle().
				Foreground(theme.Error).
				Bold(true)

	turnDurationStyle = lipgloss.NewStyle().
				Foreground(theme.TextMuted)

	turnCostStyle = lipgloss.NewStyle().
			Foreground(theme.Warning)
)

// RenderTurnList renders the turn history list pane
func RenderTurnList(turns []types.Turn, selectedTurn int, width, height int, isActive bool) string {
	// Title
	title := paneTitleStyle.Render("Turn History")

	// Determine border style based on active state
	borderStyle := paneBorderStyle
	if isActive {
		borderStyle = selectedPaneBorderStyle
	}

	// Build turn list
	var turnLines []string

	if len(turns) == 0 {
		turnLines = append(turnLines, "  No turns yet...")
	} else {
		// Calculate visible range (simple scrolling)
		visibleHeight := height - 4 // Account for borders and title
		startIdx := 0
		endIdx := len(turns)

		if len(turns) > visibleHeight {
			// Center selected turn in viewport
			startIdx = selectedTurn - visibleHeight/2
			if startIdx < 0 {
				startIdx = 0
			}
			endIdx = startIdx + visibleHeight
			if endIdx > len(turns) {
				endIdx = len(turns)
				startIdx = endIdx - visibleHeight
				if startIdx < 0 {
					startIdx = 0
				}
			}
		}

		for i := startIdx; i < endIdx; i++ {
			turn := turns[i]
			isSelected := i == selectedTurn

			turnLine := formatTurnLine(turn, isSelected, width-4)
			turnLines = append(turnLines, turnLine)
		}
	}

	// Join lines
	content := strings.Join(turnLines, "\n")

	// Apply border and sizing
	pane := borderStyle.
		Width(width - 2).
		Height(height - 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, content))

	return pane
}

// formatTurnLine formats a single turn entry
func formatTurnLine(turn types.Turn, isSelected bool, width int) string {
	// Turn number
	turnNum := turnNumberStyle.Render(fmt.Sprintf("T%02d", turn.Number))

	// Agent ID
	agentID := turn.AgentID
	if len(agentID) > 12 {
		agentID = agentID[:12]
	}
	agent := turnAgentStyle.Render(agentID)

	// Status with color
	var status string
	switch turn.Status {
	case types.TurnInProgress:
		status = turnStatusRunningStyle.Render("●")
	case types.TurnCompleted:
		status = turnStatusCompletedStyle.Render("✓")
	case types.TurnError:
		status = turnStatusErrorStyle.Render("✗")
	case types.TurnTimeout:
		status = turnStatusErrorStyle.Render("⏱")
	default:
		status = " "
	}

	// Duration
	durationMS := turn.DurationMs
	var durationStr string
	if durationMS < 1000 {
		durationStr = fmt.Sprintf("%dms", durationMS)
	} else {
		durationStr = fmt.Sprintf("%.1fs", float64(durationMS)/1000.0)
	}
	duration := turnDurationStyle.Render(fmt.Sprintf("%6s", durationStr))

	// Cost
	costStr := fmt.Sprintf("$%.3f", turn.Cost)
	cost := turnCostStyle.Render(fmt.Sprintf("%7s", costStr))

	// Combine elements
	line := fmt.Sprintf("%s %s %s %s %s",
		turnNum,
		agent,
		status,
		duration,
		cost,
	)

	// Apply selection style
	if isSelected {
		return selectedTurnStyle.Width(width).Render(line)
	}
	return turnStyle.Width(width).Render(line)
}
