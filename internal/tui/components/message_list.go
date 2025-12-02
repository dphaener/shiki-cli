package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/conversation"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
)

// MessageListState holds all information needed to render a message list pane.
type MessageListState struct {
	ParticipantID   string
	ParticipantName string
	Messages        []*conversation.Message
	Status          conversation.SessionStatus
	CurrentTurn     int
	TotalCost       float64
	TotalTokens     int
	ScrollOffset    int
	AutoScroll      bool // When true, automatically scroll to show latest content
	IsStreaming     bool // True when actively receiving content
}

// Message list styles using Sekkei Design System theme
var (
	messageListHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				Foreground(theme.Primary)

	messageListMetadataStyle = lipgloss.NewStyle().
					Padding(0, 1).
					Foreground(theme.TextMuted)

	statusStreamingIcon = lipgloss.NewStyle().
				Foreground(theme.Info).
				Bold(true).
				Render("⋯")

	statusActiveIcon = lipgloss.NewStyle().
				Foreground(theme.Info).
				Bold(true).
				Render("●")

	statusIdleIconMsg = lipgloss.NewStyle().
				Foreground(theme.Success).
				Render("✓")

	statusErrorIconMsg = lipgloss.NewStyle().
				Foreground(theme.Error).
				Bold(true).
				Render("✗")

	statusPausedIcon = lipgloss.NewStyle().
				Foreground(theme.Warning).
				Render("⏸")

	messageSeparatorStyle = lipgloss.NewStyle().
				Foreground(theme.Slate700)

	streamingBufferStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(theme.Slate200).
				Background(theme.BgMedium)
)

// RenderMessageList renders a participant's message list pane.
func RenderMessageList(state *MessageListState, width, height int, isActive bool) string {
	return RenderMessageListWithChainManager(state, width, height, isActive, nil)
}

// RenderMessageListWithChainManager renders a participant's message list pane with tool chain awareness.
func RenderMessageListWithChainManager(state *MessageListState, width, height int, isActive bool, chainManager *ToolChainManager) string {
	// Determine border style based on active state
	borderStyle := paneBorderStyle
	if isActive {
		borderStyle = selectedPaneBorderStyle
	}

	// Build header with participant info
	header := buildMessageListHeader(state)

	// Build message content
	contentHeight := height - 7 // Account for borders, header (3 lines), and padding
	content := buildMessageListContentWithChainManager(state, width-4, contentHeight, chainManager)

	// Combine header and content
	paneContent := lipgloss.JoinVertical(lipgloss.Left, header, content)

	// Apply border and sizing
	pane := borderStyle.
		Width(width - 2).
		Height(height - 1).
		Render(paneContent)

	return pane
}

// buildMessageListHeader creates the header section with participant metadata.
func buildMessageListHeader(state *MessageListState) string {
	// Participant name (truncate if too long)
	name := state.ParticipantName
	if name == "" {
		name = state.ParticipantID
	}
	if len(name) > 20 {
		name = name[:20] + "..."
	}
	nameLine := messageListHeaderStyle.Render(fmt.Sprintf("Agent: %s", name))

	// Status and turn count
	statusIcon := statusIdleIconMsg
	statusText := "Idle"

	if state.IsStreaming {
		statusIcon = statusStreamingIcon
		statusText = "Streaming"
	} else {
		switch state.Status {
		case conversation.SessionStatusRunning:
			statusIcon = statusActiveIcon
			statusText = "Running"
		case conversation.SessionStatusPaused:
			statusIcon = statusPausedIcon
			statusText = "Paused"
		case conversation.SessionStatusCompleted:
			statusIcon = statusIdleIconMsg
			statusText = "Completed"
		case conversation.SessionStatusError:
			statusIcon = statusErrorIconMsg
			statusText = "Error"
		default:
			statusIcon = statusIdleIconMsg
			statusText = "Idle"
		}
	}

	statusLine := messageListMetadataStyle.Render(
		fmt.Sprintf("%s %s | Turn %d", statusIcon, statusText, state.CurrentTurn),
	)

	// Cost and tokens
	costLine := messageListMetadataStyle.Render(
		fmt.Sprintf("$%.4f | %d tokens", state.TotalCost, state.TotalTokens),
	)

	return lipgloss.JoinVertical(lipgloss.Left, nameLine, statusLine, costLine)
}

// buildMessageListContent creates the scrollable message content.
func buildMessageListContent(state *MessageListState, width, height int) string {
	return buildMessageListContentWithChainManager(state, width, height, nil)
}

// buildMessageListContentWithChainManager creates the scrollable message content with tool chain awareness.
func buildMessageListContentWithChainManager(state *MessageListState, width, height int, chainManager *ToolChainManager) string {
	if len(state.Messages) == 0 && !state.IsStreaming {
		return messageListMetadataStyle.Render("  Waiting for activity...")
	}

	// Render all messages and their parts
	var allRenderedLines []string
	for i, msg := range state.Messages {
		// Add separator between messages (except first)
		if i > 0 {
			separator := messageSeparatorStyle.Render(strings.Repeat("─", width-2))
			allRenderedLines = append(allRenderedLines, separator)
		}

		// Render message parts with chain awareness
		msgLines := renderMessageWithChainManager(msg, width, chainManager)
		allRenderedLines = append(allRenderedLines, msgLines...)
	}

	// If streaming, show the current buffer
	if state.IsStreaming && len(state.Messages) > 0 {
		lastMsg := state.Messages[len(state.Messages)-1]
		if lastMsg.IsStreaming && lastMsg.StreamBuffer != "" {
			bufferLines := renderStreamBuffer(lastMsg.StreamBuffer, width)
			allRenderedLines = append(allRenderedLines, bufferLines...)
		}
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
	var visibleLines []string
	if startIdx < totalLines && endIdx <= totalLines {
		visibleLines = allRenderedLines[startIdx:endIdx]
	}

	// Show scroll indicator if there's more content above
	var lines []string
	if startIdx > 0 {
		indicator := messageListMetadataStyle.Render(
			fmt.Sprintf("  ↑ %d earlier lines", startIdx),
		)
		lines = append(lines, indicator)
		if len(visibleLines) > 0 {
			visibleLines = visibleLines[1:]
		}
	}

	lines = append(lines, visibleLines...)

	// Show scroll indicator if there's more content below
	if endIdx < totalLines {
		indicator := messageListMetadataStyle.Render(
			fmt.Sprintf("  ↓ %d more lines", totalLines-endIdx),
		)
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

// renderMessage renders all parts of a message.
func renderMessage(msg *conversation.Message, width int) []string {
	return renderMessageWithChainManager(msg, width, nil)
}

// renderMessageWithChainManager renders all parts of a message with tool chain awareness.
func renderMessageWithChainManager(msg *conversation.Message, width int, chainManager *ToolChainManager) []string {
	var lines []string

	for _, part := range msg.Parts {
		rendered := RenderPartWithChainManager(part, width, chainManager)
		if rendered != "" {
			partLines := strings.Split(rendered, "\n")
			lines = append(lines, partLines...)
		}
	}

	return lines
}

// renderStreamBuffer renders the current streaming buffer.
func renderStreamBuffer(buffer string, width int) []string {
	if buffer == "" {
		return nil
	}

	// Show streaming indicator and buffered content
	indicator := statusStreamingIcon + " "
	renderer := GetMarkdownRenderer()
	rendered := renderer.RenderWithStyle(buffer, width-6, streamingBufferStyle)

	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		lines[0] = "  " + indicator + lines[0]
	}

	return lines
}

// ConvertAgentOutputToMessageList converts legacy AgentOutputState to MessageListState.
// This is a bridge function for gradual migration.
func ConvertAgentOutputToMessageList(state *AgentOutputState) *MessageListState {
	// Create a temporary message to hold all the output entries
	msg := conversation.NewMessage("", state.AgentID, conversation.RoleAssistant, state.CurrentTurn)

	for _, entry := range state.Outputs {
		switch entry.Type {
		case OutputTypeToolUse:
			msg.AddPart(conversation.NewToolCallPart("", entry.Content, nil))
		case OutputTypeMessage:
			msg.AddPart(conversation.NewTextPart(entry.Content))
		}
	}

	return &MessageListState{
		ParticipantID:   state.AgentID,
		ParticipantName: state.AgentID,
		Messages:        []*conversation.Message{msg},
		CurrentTurn:     state.CurrentTurn,
		TotalCost:       state.TotalCost,
		TotalTokens:     state.TotalTokens,
		ScrollOffset:    state.ScrollOffset,
		AutoScroll:      state.AutoScroll,
		IsStreaming:     state.Status == "in_progress",
	}
}
