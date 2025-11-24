package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/pkg/types"
)

// OutputEntryType identifies the type of output entry
type OutputEntryType string

const (
	OutputTypeToolUse   OutputEntryType = "tool_use"
	OutputTypeMessage   OutputEntryType = "assistant_message"
)

// OutputEntry represents a single line in the agent output log
type OutputEntry struct {
	Type      OutputEntryType
	Content   string
	Timestamp string
}

// AgentOutputState holds all information needed to render an agent's output pane
type AgentOutputState struct {
	AgentID      string
	Status       types.TurnStatus
	CurrentTurn  int
	TotalCost    float64
	TotalTokens  int
	Outputs      []OutputEntry
	ScrollOffset int
}

// Styles for agent output
var (
	agentHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				Foreground(lipgloss.Color("62"))

	agentMetadataStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(lipgloss.Color("244"))

	toolUseStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("39"))

	assistantMessageStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(lipgloss.Color("252"))

	statusRunningIcon = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Render("●")

	statusIdleIcon = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Render("✓")

	statusErrorIcon = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render("✗")

	// Markdown renderer for assistant messages
	markdownRenderer *glamour.TermRenderer
)

// getMarkdownRenderer returns a lazily-initialized markdown renderer
func getMarkdownRenderer(width int) *glamour.TermRenderer {
	if markdownRenderer == nil {
		var err error
		// Use a dark style optimized for terminals
		markdownRenderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			// If we can't create a renderer, return nil and fall back to plain text
			return nil
		}
	}
	return markdownRenderer
}

// RenderAgentOutput renders an agent's conversation output pane
func RenderAgentOutput(state AgentOutputState, width, height int, isActive bool) string {
	// Determine border style based on active state
	borderStyle := paneBorderStyle
	if isActive {
		borderStyle = selectedPaneBorderStyle
	}

	// Build header with agent info
	header := buildAgentHeader(state)

	// Build output content
	contentHeight := height - 7 // Account for borders, header (3 lines), and padding
	content := buildAgentContent(state, width-4, contentHeight)

	// Combine header and content
	paneContent := lipgloss.JoinVertical(lipgloss.Left, header, content)

	// Apply border and sizing
	pane := borderStyle.
		Width(width - 2).
		Height(height - 1).
		Render(paneContent)

	return pane
}

// buildAgentHeader creates the header section with agent metadata
func buildAgentHeader(state AgentOutputState) string {
	// Agent ID (truncate if too long)
	agentID := state.AgentID
	if len(agentID) > 20 {
		agentID = agentID[:20] + "..."
	}
	agentLine := agentHeaderStyle.Render(fmt.Sprintf("Agent: %s", agentID))

	// Status and turn count
	statusIcon := statusIdleIcon
	statusText := "Idle"
	switch state.Status {
	case types.TurnInProgress:
		statusIcon = statusRunningIcon
		statusText = "Running"
	case types.TurnError:
		statusIcon = statusErrorIcon
		statusText = "Error"
	case types.TurnTimeout:
		statusIcon = statusErrorIcon
		statusText = "Timeout"
	}

	statusLine := agentMetadataStyle.Render(
		fmt.Sprintf("%s %s | T%d", statusIcon, statusText, state.CurrentTurn),
	)

	// Cost and tokens
	costLine := agentMetadataStyle.Render(
		fmt.Sprintf("$%.3f | %d tokens", state.TotalCost, state.TotalTokens),
	)

	return lipgloss.JoinVertical(lipgloss.Left, agentLine, statusLine, costLine)
}

// buildAgentContent creates the scrollable output content
func buildAgentContent(state AgentOutputState, width, height int) string {
	if len(state.Outputs) == 0 {
		return agentMetadataStyle.Render("  Waiting for agent activity...")
	}

	// First, render all entries to get actual line counts
	var allRenderedLines []string
	for _, entry := range state.Outputs {
		rendered := formatOutputEntry(entry, width)
		// Split rendered output by newlines since markdown/formatting can create multiple lines
		entryLines := strings.Split(rendered, "\n")
		allRenderedLines = append(allRenderedLines, entryLines...)
	}

	totalLines := len(allRenderedLines)

	// Calculate visible range - auto-scroll to bottom
	startIdx := state.ScrollOffset
	if totalLines > height {
		// If scroll offset suggests we want to see the end, show last 'height' lines
		if startIdx >= totalLines - height {
			startIdx = totalLines - height
		}
	} else {
		// All content fits, start from beginning
		startIdx = 0
	}

	if startIdx < 0 {
		startIdx = 0
	}

	endIdx := startIdx + height
	if endIdx > totalLines {
		endIdx = totalLines
	}

	// Get visible lines
	visibleLines := allRenderedLines[startIdx:endIdx]

	// Show scroll indicator if there's more content above or below
	var lines []string
	if startIdx > 0 {
		indicator := agentMetadataStyle.Render(
			fmt.Sprintf("  (%d earlier lines...)", startIdx),
		)
		lines = append(lines, indicator)
		// Adjust visible lines to make room for indicator
		if len(visibleLines) > 0 {
			visibleLines = visibleLines[1:]
		}
	}

	lines = append(lines, visibleLines...)

	if endIdx < totalLines {
		indicator := agentMetadataStyle.Render(
			fmt.Sprintf("  (%d more lines...)", totalLines-endIdx),
		)
		// Adjust visible lines to make room for indicator
		if len(lines) >= height {
			lines = lines[:height-1]
		}
		lines = append(lines, indicator)
	}

	// Ensure we don't exceed height
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// formatOutputEntry formats a single output entry (tool use or message)
func formatOutputEntry(entry OutputEntry, width int) string {
	switch entry.Type {
	case OutputTypeToolUse:
		// Format tool use with better structure
		return formatToolUse(entry.Content, width)
	case OutputTypeMessage:
		// Format: Full text, word-wrapped if needed
		return formatMessage(entry.Content, width)
	default:
		return entry.Content
	}
}

// formatToolUse formats a tool use entry with structured display
func formatToolUse(toolName string, width int) string {
	// Create a styled tool use indicator
	toolIcon := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		Render("▶")

	toolLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render("Tool:")

	toolNameStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true).
		Render(truncateString(toolName, width-15))

	// Combine elements
	return fmt.Sprintf("  %s %s %s", toolIcon, toolLabel, toolNameStyled)
}

// formatMessage formats assistant message text with markdown rendering
func formatMessage(text string, width int) string {
	if text == "" {
		return ""
	}

	// Try to render as markdown
	renderer := getMarkdownRenderer(width - 4) // Account for padding
	if renderer != nil {
		rendered, err := renderer.Render(text)
		if err == nil {
			// Apply style and return rendered markdown
			return assistantMessageStyle.Render(strings.TrimSpace(rendered))
		}
	}

	// Fallback: Simple word wrapping if markdown rendering fails
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > width-2 {
			if currentLine != "" {
				lines = append(lines, assistantMessageStyle.Render(currentLine))
			}
			currentLine = word
		} else {
			currentLine = testLine
		}
	}

	if currentLine != "" {
		lines = append(lines, assistantMessageStyle.Render(currentLine))
	}

	return strings.Join(lines, "\n")
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
