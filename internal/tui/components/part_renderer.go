package components

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/conversation"
)

// Part rendering styles
var (
	textPartStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("252"))

	toolCallStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("39"))

	toolCallNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39"))

	toolCallArgsStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	toolCallArgKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("147"))

	toolCallArgValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("222"))

	toolResultStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("244"))

	toolResultErrorStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(lipgloss.Color("196"))

	reasoningStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("241")).
			Italic(true)

	toolStatusPendingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Render("◦")

	toolStatusRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Render("●")

	toolStatusCompletedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("42")).
					Render("✓")

	toolStatusFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true).
				Render("✗")
)

// RenderPart renders a single message part to a string.
func RenderPart(part conversation.Part, width int) string {
	switch p := part.(type) {
	case conversation.TextPart:
		return renderTextPart(p, width)
	case conversation.ToolCallPart:
		return renderToolCallPart(p, width)
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
		Foreground(lipgloss.Color("39")).
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

	result := prefix + " " + lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n             " + lines[i]
	}

	return style.Render(result)
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
