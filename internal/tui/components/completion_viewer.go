package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/pkg/types"
)

// CompletionViewState holds all information needed to render the completion view
type CompletionViewState struct {
	Session            *types.Session
	DeliverableContent string
	DeliverablePath    string
	ScrollOffset       int
}

// Styles for completion view
var (
	completionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("42")).
				Padding(0, 1)

	completionSummaryStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(lipgloss.Color("252"))

	completionLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Bold(true)

	completionValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	completionPathStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Italic(true)

	completionKeybindingStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("244")).
					Padding(0, 1)

	completionDividerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("────────────────────────────────────────")
)

// RenderCompletionView renders the full-screen completion view
func RenderCompletionView(state CompletionViewState, width, height int) string {
	// Build sections
	title := buildCompletionTitle()
	summary := buildSessionSummary(state.Session, state.DeliverablePath, width)
	divider := completionDividerStyle

	// Calculate remaining height for deliverable content
	headerHeight := 10 // Title + summary + divider + spacing
	keybindingHeight := 2
	contentHeight := height - headerHeight - keybindingHeight

	// Render deliverable content
	content := buildDeliverableContent(state.DeliverableContent, width-4, contentHeight, state.ScrollOffset)

	// Build keybindings footer
	keybindings := buildKeybindings()

	// Combine all sections
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		summary,
		divider,
		content,
		keybindings,
	)

	return view
}

// buildCompletionTitle creates the title section
func buildCompletionTitle() string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true).
		Render("✓")

	title := completionTitleStyle.Render("Session Complete")

	return fmt.Sprintf("\n  %s %s\n", icon, title)
}

// buildSessionSummary creates the summary section with session stats
func buildSessionSummary(session *types.Session, deliverablePath string, width int) string {
	if session == nil {
		return completionSummaryStyle.Render("  No session data available")
	}

	// Calculate total stats
	totalCost := session.Agent1.TotalCost + session.Agent2.TotalCost
	totalTokens := session.Agent1.TotalTokens + session.Agent2.TotalTokens
	totalTurns := len(session.TurnHistory)

	// Calculate duration
	var duration string
	if session.CompletedAt != nil {
		d := session.CompletedAt.Sub(session.CreatedAt)
		if d.Seconds() < 60 {
			duration = fmt.Sprintf("%.0fs", d.Seconds())
		} else if d.Minutes() < 60 {
			duration = fmt.Sprintf("%.1fm", d.Minutes())
		} else {
			duration = fmt.Sprintf("%.1fh", d.Hours())
		}
	} else {
		duration = "N/A"
	}

	// Build summary lines
	var lines []string

	// Agents
	agentLine := fmt.Sprintf("  %s %s & %s",
		completionLabelStyle.Render("Agents:"),
		completionValueStyle.Render(truncateID(session.Agent1.ID)),
		completionValueStyle.Render(truncateID(session.Agent2.ID)),
	)
	lines = append(lines, agentLine)

	// Stats
	statsLine := fmt.Sprintf("  %s %d | %s $%.3f | %s %d",
		completionLabelStyle.Render("Turns:"),
		totalTurns,
		completionLabelStyle.Render("Cost:"),
		totalCost,
		completionLabelStyle.Render("Tokens:"),
		totalTokens,
	)
	lines = append(lines, statsLine)

	// Duration
	durationLine := fmt.Sprintf("  %s %s",
		completionLabelStyle.Render("Duration:"),
		completionValueStyle.Render(duration),
	)
	lines = append(lines, durationLine)

	// Deliverable path
	if deliverablePath != "" {
		pathLine := fmt.Sprintf("  %s %s",
			completionLabelStyle.Render("Deliverable:"),
			completionPathStyle.Render(deliverablePath),
		)
		lines = append(lines, pathLine)
	}

	return strings.Join(lines, "\n")
}

// buildDeliverableContent renders the deliverable markdown content
func buildDeliverableContent(content string, width, height, scrollOffset int) string {
	if content == "" {
		emptyMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Render("No deliverable content available")
		return fmt.Sprintf("\n  %s\n", emptyMsg)
	}

	// Try to render as markdown
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)

	var rendered string
	if err == nil {
		r, err := renderer.Render(content)
		if err == nil {
			rendered = r
		} else {
			// Fallback to plain text
			rendered = content
		}
	} else {
		// Fallback to plain text
		rendered = content
	}

	// Split into lines for scrolling
	lines := strings.Split(strings.TrimSpace(rendered), "\n")
	totalLines := len(lines)

	// Calculate visible range
	startIdx := scrollOffset
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= totalLines {
		startIdx = totalLines - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}

	endIdx := startIdx + height
	if endIdx > totalLines {
		endIdx = totalLines
	}

	// Get visible lines
	visibleLines := lines[startIdx:endIdx]

	// Add scroll indicators
	var output []string
	if startIdx > 0 {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Render(fmt.Sprintf("  ↑ %d earlier lines...", startIdx))
		output = append(output, indicator)
		// Adjust visible lines to make room for indicator
		if len(visibleLines) > 0 {
			visibleLines = visibleLines[1:]
		}
	}

	output = append(output, visibleLines...)

	if endIdx < totalLines {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Render(fmt.Sprintf("  ↓ %d more lines...", totalLines-endIdx))
		// Adjust visible lines to make room for indicator
		if len(output) >= height {
			output = output[:height-1]
		}
		output = append(output, indicator)
	}

	// Ensure we don't exceed height
	if len(output) > height {
		output = output[:height]
	}

	return "\n" + strings.Join(output, "\n")
}

// buildKeybindings creates the keybindings footer
func buildKeybindings() string {
	bindings := []string{
		"↑/↓: scroll",
		"h: view history",
		"q: exit",
	}

	bindingText := strings.Join(bindings, "  •  ")
	return completionKeybindingStyle.Render(bindingText)
}

// truncateID truncates an agent ID for display
func truncateID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12] + "..."
}
