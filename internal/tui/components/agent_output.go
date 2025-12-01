package components

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/theme"
	"github.com/darinhaener/collab/pkg/types"
)

// AgentOutputState and related types are deprecated.
// Use MessageListState and conversation.Message with Parts instead.
//
// Migration guide:
//   - Replace AgentOutputState with MessageListState
//   - Replace OutputEntry with conversation.Message containing Parts
//   - Use RenderMessageList() instead of RenderAgentOutput()
//   - Use RenderPart() for individual part rendering

// OutputEntryType identifies the type of output entry.
//
// Deprecated: Use conversation.PartType instead.
type OutputEntryType string

const (
	OutputTypeToolUse   OutputEntryType = "tool_use"
	OutputTypeMessage   OutputEntryType = "assistant_message"
)

// OutputEntry represents a single line in the agent output log
type OutputEntry struct {
	Type       OutputEntryType
	Content    string
	Timestamp  string
	IsReplaced bool                   // True if this entry has been replaced by a later tool in the same turn
	Turn       int                    // Turn number this entry belongs to
	ToolCallID string                 // Unique ID for tool calls (empty for messages)
	Args       map[string]interface{} // Tool arguments for rich display
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
	AutoScroll   bool // When true, automatically scroll to show latest content
}

// MarkPreviousToolsReplaced marks all tool entries in the current turn as replaced.
// This is called when a new tool starts to implement the rollup behavior.
func (s *AgentOutputState) MarkPreviousToolsReplaced(currentTurn int) {
	for i := range s.Outputs {
		if s.Outputs[i].Type == OutputTypeToolUse && s.Outputs[i].Turn == currentTurn {
			s.Outputs[i].IsReplaced = true
		}
	}
}

// ResetScrollState ensures clean scroll state for agent output
func (state *AgentOutputState) ResetScrollState() {
	state.AutoScroll = true
	state.ScrollOffset = 0
}

// ValidateScrollState checks and fixes inconsistent scroll state
func (state *AgentOutputState) ValidateScrollState(totalLines, height int) {
	// Ensure scroll offset is within valid bounds
	maxOffset := totalLines - height
	if maxOffset < 0 {
		maxOffset = 0
	}

	if state.ScrollOffset < 0 {
		state.ScrollOffset = 0
	} else if state.ScrollOffset > maxOffset {
		state.ScrollOffset = maxOffset
	}

	// Re-enable auto-scroll if we're at the bottom
	if state.ScrollOffset >= maxOffset {
		state.AutoScroll = true
	}
}

// Styles for agent output using Sekkei Design System theme
var (
	agentHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				Foreground(theme.Primary)

	agentMetadataStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(theme.TextMuted)

	toolUseStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(theme.Info)

	assistantMessageStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(theme.Slate200)

	statusRunningIcon = lipgloss.NewStyle().
				Foreground(theme.Info).
				Bold(true).
				Render("●")

	statusIdleIcon = lipgloss.NewStyle().
			Foreground(theme.Success).
			Render("✓")

	statusErrorIcon = lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true).
			Render("✗")
)

// RenderAgentOutput renders an agent's conversation output pane
func RenderAgentOutput(state *AgentOutputState, width, height int, isActive bool) string {
	// Determine border style based on active state
	borderStyle := paneBorderStyle
	if isActive {
		borderStyle = selectedPaneBorderStyle
	}

	// Build header with agent info
	header := buildAgentHeader(*state)

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
func buildAgentContent(state *AgentOutputState, width, height int) string {
	if len(state.Outputs) == 0 {
		return agentMetadataStyle.Render("  Waiting for agent activity...")
	}

	// First, render all entries with blank lines between each entry
	var allRenderedLines []string
	firstEntry := true
	for _, entry := range state.Outputs {
		// Skip entries that have been replaced by later tools in the same turn
		if entry.IsReplaced {
			continue
		}

		// Add blank line between entries (except before first entry)
		if !firstEntry {
			allRenderedLines = append(allRenderedLines, "")
		}

		rendered := formatOutputEntry(entry, width)
		// Split rendered output by newlines since markdown/formatting can create multiple lines
		entryLines := strings.Split(rendered, "\n")
		allRenderedLines = append(allRenderedLines, entryLines...)
		firstEntry = false
	}

	totalLines := len(allRenderedLines)

	// Calculate visible range
	var startIdx int
	if state.AutoScroll {
		// Auto-scroll mode: always show the bottom
		if totalLines > height {
			startIdx = totalLines - height
		} else {
			startIdx = 0
		}
	} else {
		// Manual scroll mode: use the scroll offset
		startIdx = state.ScrollOffset
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx > totalLines-height && totalLines > height {
			startIdx = totalLines - height
		}

		// Re-enable autoscroll if user has scrolled to the bottom
		if totalLines <= height || startIdx >= totalLines-height {
			state.AutoScroll = true
		}
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
		// Format tool use with rich parameter display
		return formatToolUseRich(entry.Content, entry.Args, width)
	case OutputTypeMessage:
		// Format: Full text, word-wrapped if needed
		return formatMessage(entry.Content, width)
	default:
		return entry.Content
	}
}

// formatToolUse formats a tool use entry with structured display
func formatToolUse(toolName string, width int) string {
	return formatToolUseRich(toolName, nil, width)
}

// formatToolUseRich formats a tool use entry with rich information from args
// Format matches Charmbracelet Crush style: ▶ ToolName mainArg key: value
func formatToolUseRich(toolName string, args map[string]interface{}, width int) string {
	// Tool icon
	toolIcon := lipgloss.NewStyle().
		Foreground(theme.Info).
		Bold(true).
		Render("▶")

	// Tool name
	toolNameStyled := lipgloss.NewStyle().
		Foreground(theme.Info).
		Bold(true).
		Render(toolName)

	// Build parameter string based on tool type
	params := buildToolParams(toolName, args, width-len(toolName)-10)

	if params != "" {
		paramsStyled := lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Render(params)
		return fmt.Sprintf("  %s %s %s", toolIcon, toolNameStyled, paramsStyled)
	}

	return fmt.Sprintf("  %s %s", toolIcon, toolNameStyled)
}

// buildToolParams builds the parameter string for a tool call (Crush-style)
func buildToolParams(toolName string, args map[string]interface{}, maxWidth int) string {
	if args == nil {
		return ""
	}

	var parts []string

	switch toolName {
	case "Read":
		if filePath, ok := args["file_path"].(string); ok {
			parts = append(parts, prettyPath(filePath))
		}
		if limit, ok := args["limit"].(float64); ok && limit > 0 {
			parts = append(parts, fmt.Sprintf("limit: %d", int(limit)))
		}
		if offset, ok := args["offset"].(float64); ok && offset > 0 {
			parts = append(parts, fmt.Sprintf("offset: %d", int(offset)))
		}

	case "Write":
		if filePath, ok := args["file_path"].(string); ok {
			parts = append(parts, prettyPath(filePath))
		}

	case "Edit":
		if filePath, ok := args["file_path"].(string); ok {
			parts = append(parts, prettyPath(filePath))
		}

	case "Glob":
		if pattern, ok := args["pattern"].(string); ok {
			parts = append(parts, pattern)
		}
		if path, ok := args["path"].(string); ok && path != "" {
			parts = append(parts, fmt.Sprintf("path: %s", prettyPath(path)))
		}

	case "Grep":
		if pattern, ok := args["pattern"].(string); ok {
			// Truncate long patterns
			if len(pattern) > 30 {
				pattern = pattern[:27] + "..."
			}
			parts = append(parts, fmt.Sprintf("\"%s\"", pattern))
		}
		if path, ok := args["path"].(string); ok && path != "" {
			parts = append(parts, fmt.Sprintf("path: %s", prettyPath(path)))
		}

	case "Bash":
		// Prefer description if available
		if desc, ok := args["description"].(string); ok && desc != "" {
			parts = append(parts, desc)
		} else if cmd, ok := args["command"].(string); ok {
			// Clean up command for display
			cmd = strings.ReplaceAll(cmd, "\n", " ")
			cmd = strings.ReplaceAll(cmd, "\t", " ")
			if len(cmd) > 60 {
				cmd = cmd[:57] + "..."
			}
			parts = append(parts, cmd)
		}
		if bg, ok := args["run_in_background"].(bool); ok && bg {
			parts = append(parts, "[background]")
		}

	case "WebFetch":
		if url, ok := args["url"].(string); ok {
			if len(url) > 50 {
				url = url[:47] + "..."
			}
			parts = append(parts, url)
		}

	case "WebSearch":
		if query, ok := args["query"].(string); ok {
			if len(query) > 40 {
				query = query[:37] + "..."
			}
			parts = append(parts, fmt.Sprintf("\"%s\"", query))
		}

	case "Task":
		if desc, ok := args["description"].(string); ok {
			parts = append(parts, desc)
		}
		if agentType, ok := args["subagent_type"].(string); ok {
			parts = append(parts, fmt.Sprintf("agent: %s", agentType))
		}

	case "TodoWrite":
		if todos, ok := args["todos"].([]interface{}); ok {
			parts = append(parts, fmt.Sprintf("%d items", len(todos)))
		}

	default:
		// For unknown tools, try to extract common parameter names
		for _, key := range []string{"file_path", "path", "pattern", "command", "query", "url"} {
			if val, ok := args[key].(string); ok && val != "" {
				if len(val) > 40 {
					val = val[:37] + "..."
				}
				parts = append(parts, val)
				break
			}
		}
	}

	result := strings.Join(parts, " ")
	if len(result) > maxWidth && maxWidth > 3 {
		result = result[:maxWidth-3] + "..."
	}
	return result
}

// prettyPath formats a file path for display, similar to Crush's fsext.PrettyPath
func prettyPath(path string) string {
	if path == "" {
		return ""
	}

	// Get home directory for ~ substitution
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}

	// If still too long, show last few path components
	if len(path) > 50 {
		parts := strings.Split(path, "/")
		if len(parts) > 3 {
			path = ".../" + strings.Join(parts[len(parts)-3:], "/")
		}
	}

	return path
}

// formatMessage formats assistant message text with markdown rendering
func formatMessage(text string, width int) string {
	if text == "" {
		return ""
	}

	// Use shared markdown renderer with proper width handling
	renderer := GetMarkdownRenderer()
	return renderer.RenderWithStyle(text, width-4, assistantMessageStyle)
}

