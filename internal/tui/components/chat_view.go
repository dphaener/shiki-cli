package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/theme"
	"github.com/darinhaener/collab/pkg/types"
)

// maxInputHeight is the fixed height of the input container (textarea grows upward within it)
const maxInputHeight = 5

// ChatView manages the chat interface for specification collaboration
type ChatView struct {
	viewport         viewport.Model
	textarea         textarea.Model
	messages         []types.ChatMessage
	width            int
	height           int
	ready            bool
	lastMessageCount int
	contentDirty     bool
	isStreaming      bool // Track if we're currently receiving a streaming message
}

// NewChatView creates a new chat view
func NewChatView(width, height int) ChatView {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Prompt = "> "
	ta.CharLimit = 2000
	ta.SetWidth(width - 4)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(width-4, height-4)
	vp.YPosition = 0

	return ChatView{
		viewport: vp,
		textarea: ta,
		messages: []types.ChatMessage{},
		width:    width,
		height:   height,
		ready:    false,
	}
}

// Init implements tea.Model
func (c ChatView) Init() tea.Cmd {
	return textarea.Blink
}

// Update implements tea.Model
func (c ChatView) Update(msg tea.Msg) (ChatView, tea.Cmd) {
	var (
		vpCmd tea.Cmd
		taCmd tea.Cmd
	)

	c.viewport, vpCmd = c.viewport.Update(msg)
	c.textarea, taCmd = c.textarea.Update(msg)

	return c, tea.Batch(vpCmd, taCmd)
}

// View implements tea.Model
func (c ChatView) View() string {
	if !c.ready {
		return "Initializing chat..."
	}

	// Only update viewport content if messages have changed
	if c.contentDirty {
		chatContent := c.renderMessages()
		c.viewport.SetContent(chatContent)
		c.contentDirty = false
	}

	// Wrap textarea in a bottom-aligned container so input grows upward
	inputContainer := lipgloss.NewStyle().
		Width(c.width - 4).
		Height(maxInputHeight).
		AlignVertical(lipgloss.Bottom).
		Render(c.textarea.View())

	// Build the complete view
	return lipgloss.JoinVertical(
		lipgloss.Left,
		c.viewport.View(),
		chatInputSeparator.Render(strings.Repeat("─", c.width-4)),
		inputContainer,
	)
}

// SetSize updates the chat view dimensions
func (c *ChatView) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.viewport.Width = width - 4
	c.textarea.SetWidth(width - 4)

	// Fixed input container height, textarea can grow within it
	// Viewport height = total - separator (1) - input container (maxInputHeight)
	c.viewport.Height = height - 1 - maxInputHeight
	if c.viewport.Height < 3 {
		c.viewport.Height = 3
	}
	c.textarea.SetHeight(maxInputHeight)

	// Focus textarea on first resize
	if !c.ready {
		c.ready = true
		// Clear BEFORE focusing to avoid capturing any buffered terminal responses
		c.textarea.SetValue("")
		c.textarea.Focus()
		// Clear AGAIN after focusing to catch any responses that arrived during focus
		c.textarea.SetValue("")
	}
}

// AddMessage adds a new message to the chat history
func (c *ChatView) AddMessage(msg types.ChatMessage) {
	c.messages = append(c.messages, msg)
	c.contentDirty = true

	// Update viewport content immediately for new messages
	chatContent := c.renderMessages()
	c.viewport.SetContent(chatContent)

	// Always scroll to bottom for new content during active conversation
	c.viewport.GotoBottom()
	c.contentDirty = false
}

// AddToolUse adds a tool use indicator to the chat
func (c *ChatView) AddToolUse(msg types.ChatMessage) {
	c.messages = append(c.messages, msg)
	c.contentDirty = true

	// Update viewport content immediately for new messages
	chatContent := c.renderMessages()
	c.viewport.SetContent(chatContent)

	// Always scroll to bottom for new content
	c.viewport.GotoBottom()
	c.contentDirty = false
}

// SetMessages replaces all messages
func (c *ChatView) SetMessages(messages []types.ChatMessage) {
	c.messages = messages
	c.contentDirty = true

	// Update viewport content immediately
	chatContent := c.renderMessages()
	c.viewport.SetContent(chatContent)
	c.viewport.GotoBottom()
	c.contentDirty = false
}

// UpdateLastMessage updates the last message's content (for streaming)
func (c *ChatView) UpdateLastMessage(content string) {
	if len(c.messages) == 0 {
		return
	}

	// Update the last message
	c.messages[len(c.messages)-1].Content = content
	c.contentDirty = true

	// Update viewport content immediately
	chatContent := c.renderMessages()
	c.viewport.SetContent(chatContent)

	// Always scroll to bottom when streaming to follow the content
	c.viewport.GotoBottom()
	c.contentDirty = false
}

// AddOrUpdateAssistantMessage adds a new assistant message or updates the last one if it's streaming
func (c *ChatView) AddOrUpdateAssistantMessage(content string, streaming bool) {
	now := time.Now()

	if streaming {
		// Find the last assistant message (might not be the very last message if tools were used)
		lastAssistantIdx := -1
		for i := len(c.messages) - 1; i >= 0; i-- {
			if c.messages[i].Role == "assistant" {
				lastAssistantIdx = i
				break
			}
		}

		// If we found a recent assistant message and we're currently streaming, update it
		if c.isStreaming && lastAssistantIdx >= 0 {
			// Update the existing streaming message
			c.messages[lastAssistantIdx].Content = content
			c.contentDirty = true

			// Update viewport and scroll to bottom
			chatContent := c.renderMessages()
			c.viewport.SetContent(chatContent)
			c.viewport.GotoBottom()
			c.contentDirty = false
		} else {
			// Start a new streaming message
			c.isStreaming = true
			msg := types.ChatMessage{
				Role:      "assistant",
				Content:   content,
				Timestamp: now,
			}
			c.messages = append(c.messages, msg)
			c.contentDirty = true

			// Update viewport and scroll to bottom
			chatContent := c.renderMessages()
			c.viewport.SetContent(chatContent)
			c.viewport.GotoBottom()
			c.contentDirty = false
		}
	} else {
		// Not streaming - just add the message normally
		c.isStreaming = false
		msg := types.ChatMessage{
			Role:      "assistant",
			Content:   content,
			Timestamp: now,
		}
		c.AddMessage(msg)
	}
}

// FinishStreaming marks the end of a streaming message
func (c *ChatView) FinishStreaming() {
	c.isStreaming = false
}

// GetInput returns the current input text
func (c *ChatView) GetInput() string {
	return strings.TrimSpace(c.textarea.Value())
}

// ClearInput clears the input field
func (c *ChatView) ClearInput() {
	c.textarea.Reset()
}

// Focus focuses the input field
func (c *ChatView) Focus() tea.Cmd {
	return c.textarea.Focus()
}

// Blur removes focus from the input field
func (c *ChatView) Blur() {
	c.textarea.Blur()
}

// renderMessages renders all chat messages
func (c *ChatView) renderMessages() string {
	if len(c.messages) == 0 {
		return chatEmptyStyle.Render("No messages yet. Start the conversation!")
	}

	var rendered []string
	for _, msg := range c.messages {
		rendered = append(rendered, c.renderMessage(msg))
	}

	return strings.Join(rendered, "\n\n")
}

// renderMessage renders a single chat message
func (c *ChatView) renderMessage(msg types.ChatMessage) string {
	timestamp := msg.Timestamp.Format("15:04:05")

	switch msg.Role {
	case "user":
		// User messages: slate background with ">" prefix for visual distinction
		contentWidth := c.width - 8

		// Use shared markdown renderer for consistent text handling
		renderer := GetMarkdownRenderer()
		content := renderer.RenderWithStyle(msg.Content, contentWidth-4, userMessageStyle)

		// Add ">" prefix
		prefix := lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Render(">")

		return prefix + " " + content

	case "tool":
		// Tool use messages: styled with icon and tool name
		toolIcon := lipgloss.NewStyle().
			Foreground(theme.Info).
			Bold(true).
			Render("▶")

		toolLabel := lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Render("Tool:")

		toolName := lipgloss.NewStyle().
			Foreground(theme.Info).
			Bold(true).
			Render(msg.Content)

		timestampStyle := lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Render(fmt.Sprintf("(%s)", timestamp))

		return lipgloss.NewStyle().
			Padding(0, 2).
			Render(fmt.Sprintf("%s %s %s %s", toolIcon, toolLabel, toolName, timestampStyle))

	case "error":
		// Error messages: styled with warning icon and red text
		errorIcon := lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true).
			Render("⚠")

		header := errorHeaderStyle.Render(fmt.Sprintf("%s Error (%s)", errorIcon, timestamp))
		content := errorMessageStyle.Render(msg.Content)
		return lipgloss.JoinVertical(lipgloss.Left, header, content)

	default:
		// Assistant messages: no header, just content
		renderer := GetMarkdownRenderer()
		content := renderer.RenderWithStyle(msg.Content, c.width-8, assistantChatMessageStyle)

		return content
	}
}

// Styles for chat view using Sekkei Design System theme
var (
	userMessageStyle = lipgloss.NewStyle().
				Foreground(theme.Slate200).
				Background(theme.Slate700).
				Padding(0, 1)

	assistantChatMessageStyle = lipgloss.NewStyle().
					Foreground(theme.Slate200).
					Padding(0, 1)

	chatInputSeparator = lipgloss.NewStyle().
				Foreground(theme.Border)

	chatEmptyStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Padding(2, 2)

	errorHeaderStyle = lipgloss.NewStyle().
				Foreground(theme.Error).
				Bold(true).
				Padding(0, 1)

	errorMessageStyle = lipgloss.NewStyle().
				Foreground(theme.Error).
				Background(theme.ErrorLight).
				Padding(1, 2).
				MarginTop(0).
				MarginBottom(1)
)

// AddError adds an error message to the chat (displayed inline)
func (c *ChatView) AddError(err error) {
	errorMsg := types.ChatMessage{
		Role:      "error",
		Content:   err.Error(),
		Timestamp: time.Now(),
	}
	c.messages = append(c.messages, errorMsg)
	c.contentDirty = true

	// Update viewport content immediately
	chatContent := c.renderMessages()
	c.viewport.SetContent(chatContent)
	c.viewport.GotoBottom()
	c.contentDirty = false
}

// AddWelcomeMessage adds an initial welcome message from the assistant
// hasDescription indicates whether the user provided a feature description as an argument
func (c *ChatView) AddWelcomeMessage(featureName string, hasDescription bool) {
	var welcome string

	if hasDescription {
		welcome = fmt.Sprintf(`# Welcome to Feature Specification

I'll help you create a comprehensive specification for: **%s**`, featureName)
	} else {
		welcome = `# Welcome to Feature Specification

I'll help you create a comprehensive specification for your feature.`
	}

	c.AddMessage(types.ChatMessage{
		Role:      "assistant",
		Content:   welcome,
		Timestamp: time.Now(),
	})
}
