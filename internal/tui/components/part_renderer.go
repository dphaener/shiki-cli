package components

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/conversation"
	"github.com/darinhaener/collab/internal/diff"
	"github.com/darinhaener/collab/internal/tui/theme"
)

// Part rendering styles using Sekkei Design System theme
var (
	textPartStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(theme.Slate200)

	toolCallStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(theme.Info)

	toolCallNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Info)

	toolCallArgsStyle = lipgloss.NewStyle().
				Foreground(theme.TextMuted)

	toolCallArgKeyStyle = lipgloss.NewStyle().
				Foreground(theme.Emerald400)

	toolCallArgValueStyle = lipgloss.NewStyle().
				Foreground(theme.Amber500)

	toolResultStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(theme.TextMuted)

	toolResultErrorStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(theme.Error)

	reasoningStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(theme.TextDim).
			Italic(true)

	toolStatusPendingStyle = lipgloss.NewStyle().
				Foreground(theme.TextMuted).
				Render("◦")

	toolStatusRunningStyle = lipgloss.NewStyle().
				Foreground(theme.Info).
				Bold(true).
				Render("●")

	toolStatusCompletedStyle = lipgloss.NewStyle().
					Foreground(theme.Success).
					Render("✓")

	toolStatusFailedStyle = lipgloss.NewStyle().
				Foreground(theme.Error).
				Bold(true).
				Render("✗")
)

// RenderPart renders a single message part to a string.
func RenderPart(part conversation.Part, width int) string {
	return RenderPartWithChainManager(part, width, nil)
}

// RenderPartWithChainManager renders a single message part with tool chain awareness.
func RenderPartWithChainManager(part conversation.Part, width int, chainManager *ToolChainManager) string {
	switch p := part.(type) {
	case conversation.TextPart:
		return renderTextPart(p, width)
	case conversation.ToolCallPart:
		return renderToolCallPartWithChains(p, width, chainManager)
	case conversation.ToolResultPart:
		return renderToolResultPart(p, width)
	case conversation.ReasoningPart:
		return renderReasoningPart(p, width)
	default:
		return fmt.Sprintf("[Unknown part type: %s]", part.Type())
	}
}

// renderTextPart renders text content with markdown.
func renderTextPart(part conversation.TextPart, width int) string {
	if part.Content == "" {
		return ""
	}

	renderer := GetMarkdownRenderer()
	return renderer.RenderWithStyle(part.Content, width-4, textPartStyle)
}

// renderToolCallPart renders a tool call with status, name, and arguments.
func renderToolCallPart(part conversation.ToolCallPart, width int) string {
	// Status indicator
	statusIcon := toolStatusPendingStyle
	switch part.Status {
	case conversation.ToolCallPending:
		statusIcon = toolStatusPendingStyle
	case conversation.ToolCallRunning:
		statusIcon = toolStatusRunningStyle
	case conversation.ToolCallCompleted:
		statusIcon = toolStatusCompletedStyle
	case conversation.ToolCallFailed:
		statusIcon = toolStatusFailedStyle
	}

	// Tool icon and name
	toolIcon := lipgloss.NewStyle().
		Foreground(theme.Info).
		Bold(true).
		Render("▶")

	toolName := toolCallNameStyle.Render(truncateString(part.ToolName, width-20))

	// Build the header line
	headerLine := fmt.Sprintf("  %s %s %s", statusIcon, toolIcon, toolName)

	// Format arguments if present
	if len(part.Input) > 0 {
		argsLines := formatToolArgs(part.Input, width-6)
		if argsLines != "" {
			return headerLine + "\n" + argsLines
		}
	}

	return headerLine
}

// renderToolCallPartWithChains renders a tool call with chain awareness and smart rollup.
func renderToolCallPartWithChains(part conversation.ToolCallPart, width int, chainManager *ToolChainManager) string {
	// If no chain manager, fall back to regular rendering
	if chainManager == nil {
		return renderToolCallPart(part, width)
	}

	// Check if this tool is part of a chain
	chain := chainManager.GetChainByToolID(part.ID)
	if chain == nil {
		return renderToolCallPart(part, width)
	}

	// If this tool is replaced in its chain, don't render it (it's been rolled up)
	if part.IsReplaced {
		return "" // Tool is hidden by rollup behavior
	}

	// This is the current tool in the chain - render with enhanced state
	return renderActiveChainTool(chain, part, width)
}

// renderActiveChainTool renders the currently active tool in a chain with enhanced state info.
func renderActiveChainTool(chain *ToolChain, part conversation.ToolCallPart, width int) string {
	// Get current tool execution state from chain
	var currentExec *ToolExecution
	for _, tool := range chain.Tools {
		if tool.ToolCallID == part.ID {
			currentExec = tool
			break
		}
	}

	if currentExec == nil {
		// Fallback to regular rendering if we can't find execution state
		return renderToolCallPart(part, width)
	}

	// Enhanced status indicator based on execution state
	statusIcon := getEnhancedStatusIcon(currentExec.State)

	// Tool icon and name
	toolIcon := lipgloss.NewStyle().
		Foreground(theme.Info).
		Bold(true).
		Render("▶")

	toolName := toolCallNameStyle.Render(truncateString(part.ToolName, width-20))

	// Show resource info if this is part of a chain with replaced tools
	resourceInfo := ""
	if len(chain.ReplacedTools) > 0 {
		displayName := chain.Resource
		if len(displayName) > 30 {
			displayName = "..." + displayName[len(displayName)-27:]
		}
		resourceInfo = toolCallArgsStyle.Render(fmt.Sprintf(" [chain: %s]", displayName))
	}

	// Build the header line
	headerLine := fmt.Sprintf("  %s %s %s%s", statusIcon, toolIcon, toolName, resourceInfo)

	// Format arguments if present (but shorter for chain tools to save space)
	if len(part.Input) > 0 {
		argsLines := formatToolArgsCompact(part.Input, width-6, len(chain.ReplacedTools) > 0)
		if argsLines != "" {
			return headerLine + "\n" + argsLines
		}
	}

	return headerLine
}

// getEnhancedStatusIcon returns appropriate status icon for tool execution state.
func getEnhancedStatusIcon(state ExecutionState) string {
	switch state {
	case ExecutionStatePending:
		return toolStatusPendingStyle
	case ExecutionStateRunning:
		return toolStatusRunningStyle // Could add spinner animation here
	case ExecutionStateCompleted:
		return toolStatusCompletedStyle
	case ExecutionStateError:
		return toolStatusFailedStyle
	case ExecutionStateCancelled:
		return lipgloss.NewStyle().Foreground(theme.Warning).Render("⊗")
	default:
		return toolStatusPendingStyle
	}
}

// formatToolArgsCompact formats tool arguments in a more compact form for chains.
func formatToolArgsCompact(args map[string]interface{}, width int, isChainTool bool) string {
	if len(args) == 0 {
		return ""
	}

	maxArgs := 3 // Fewer args for chain tools to save space
	if !isChainTool {
		maxArgs = 5
	}

	var lines []string
	maxKeyLen := 12 // Shorter for compact display

	argCount := 0
	for key, value := range args {
		if key == "_raw" {
			continue // Skip internal raw data
		}

		if argCount >= maxArgs {
			remaining := len(args) - maxArgs
			if remaining > 0 {
				lines = append(lines, toolCallArgsStyle.Render(
					fmt.Sprintf("      ... and %d more", remaining),
				))
			}
			break
		}

		keyStr := truncateString(key, maxKeyLen)
		valueStr := formatArgValue(value, width-maxKeyLen-8)

		// Pad key for alignment
		paddedKey := fmt.Sprintf("%-*s", maxKeyLen, keyStr)

		line := fmt.Sprintf("      %s: %s",
			toolCallArgKeyStyle.Render(paddedKey),
			toolCallArgValueStyle.Render(valueStr),
		)
		lines = append(lines, line)
		argCount++
	}

	return strings.Join(lines, "\n")
}

// formatToolArgs formats tool arguments for display.
func formatToolArgs(args map[string]interface{}, width int) string {
	if len(args) == 0 {
		return ""
	}

	var lines []string
	maxKeyLen := 0

	// Find max key length for alignment
	for k := range args {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}
	if maxKeyLen > 15 {
		maxKeyLen = 15
	}

	// Format each argument
	for key, value := range args {
		if key == "_raw" {
			continue // Skip internal raw data
		}

		keyStr := truncateString(key, maxKeyLen)
		valueStr := formatArgValue(value, width-maxKeyLen-8)

		// Pad key for alignment
		paddedKey := fmt.Sprintf("%-*s", maxKeyLen, keyStr)

		line := fmt.Sprintf("      %s: %s",
			toolCallArgKeyStyle.Render(paddedKey),
			toolCallArgValueStyle.Render(valueStr),
		)
		lines = append(lines, line)

		// Limit to 5 args to avoid overwhelming display
		if len(lines) >= 5 {
			remaining := len(args) - 5
			if remaining > 0 {
				lines = append(lines, toolCallArgsStyle.Render(
					fmt.Sprintf("      ... and %d more arguments", remaining),
				))
			}
			break
		}
	}

	return strings.Join(lines, "\n")
}

// formatArgValue formats a single argument value.
func formatArgValue(value interface{}, maxLen int) string {
	var str string

	switch v := value.(type) {
	case string:
		str = v
	case bool:
		if v {
			str = "true"
		} else {
			str = "false"
		}
	case float64:
		if v == float64(int(v)) {
			str = fmt.Sprintf("%d", int(v))
		} else {
			str = fmt.Sprintf("%.2f", v)
		}
	case int:
		str = fmt.Sprintf("%d", v)
	case []interface{}:
		if len(v) == 0 {
			str = "[]"
		} else {
			str = fmt.Sprintf("[%d items]", len(v))
		}
	case map[string]interface{}:
		if len(v) == 0 {
			str = "{}"
		} else {
			str = fmt.Sprintf("{%d fields}", len(v))
		}
	case nil:
		str = "null"
	default:
		// Try JSON marshaling for other types
		if b, err := json.Marshal(v); err == nil {
			str = string(b)
		} else {
			str = fmt.Sprintf("%v", v)
		}
	}

	// Handle multiline strings - show first line only
	if idx := strings.Index(str, "\n"); idx != -1 {
		str = str[:idx] + "..."
	}

	return truncateString(str, maxLen)
}

// renderToolResultPart renders the result of a tool execution.
func renderToolResultPart(part conversation.ToolResultPart, width int) string {
	var result strings.Builder

	// Render the main tool result content
	style := toolResultStyle
	prefix := "    └─ Result:"
	if part.IsError {
		style = toolResultErrorStyle
		prefix = "    └─ Error:"
	}

	// Truncate long results
	content := part.Content
	if len(content) > 200 {
		content = content[:200] + "..."
	}

	// Handle multiline - indent continuation
	lines := strings.Split(content, "\n")
	if len(lines) > 3 {
		lines = lines[:3]
		lines = append(lines, "...")
	}

	toolResult := prefix + " " + lines[0]
	for i := 1; i < len(lines); i++ {
		toolResult += "\n             " + lines[i]
	}

	result.WriteString(style.Render(toolResult))

	// Render diff if present
	if part.Diff != nil && !part.IsError {
		diffContent := renderDiff(part.Diff, width)
		if diffContent != "" {
			result.WriteString("\n")
			result.WriteString(diffContent)
		}
	}

	return result.String()
}

// renderDiff renders a file diff with colors and formatting
func renderDiff(fileDiff *diff.FileDiff, width int) string {
	if fileDiff == nil {
		return ""
	}

	renderer := diff.NewRenderer()

	// Adjust width for diff content (leave some margin)
	diffWidth := width - 8 // Account for indentation
	if diffWidth < 40 {
		diffWidth = 40 // Minimum reasonable width
	}

	// Set reasonable limits for TUI display
	renderer.SetMaxLines(50) // Limit to 50 lines for chat display
	renderer.SetMaxWidth(diffWidth)

	// Format the diff for terminal display
	formattedDiff := renderer.FormatDiffForTerminal(fileDiff, diffWidth)

	if formattedDiff == "" {
		return ""
	}

	// Apply line wrapping if needed
	wrappedDiff := renderer.ApplyLineWrapping(formattedDiff, diffWidth)

	// Add some padding around the diff content
	diffStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1)

	return diffStyle.Render(wrappedDiff)
}

// renderReasoningPart renders thinking/reasoning content.
func renderReasoningPart(part conversation.ReasoningPart, width int) string {
	if part.Content == "" {
		return ""
	}

	// Truncate long reasoning
	content := part.Content
	if len(content) > 300 {
		content = content[:300] + "..."
	}

	prefix := reasoningStyle.Render("  [thinking] ")
	return prefix + reasoningStyle.Render(truncateString(content, width-15))
}

// truncateString truncates a string to the specified length with ellipsis if needed.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
