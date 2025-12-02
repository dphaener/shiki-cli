package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// maxInputHeight is the fixed height of the input container (textarea grows upward within it)
const maxInputHeight = 3

// errorMessageHorizontalPadding accounts for left/right padding and margins in error messages
const errorMessageHorizontalPadding = 8

// errorMessageStylePadding accounts for additional padding from errorMessageStyle (Padding(1, 2))
const errorMessageStylePadding = 4

// ChatView manages the chat interface for specification collaboration
type ChatView struct {
	viewport         viewport.Model
	textarea         textarea.Model
	messages         []types.ChatMessage
	width            int
	height           int
	ready            bool
	lastMessageCount int
	// contentDirty tracks whether messages have changed since last render.
	// Safe to use without synchronization due to Bubble Tea's single-threaded model.
	contentDirty        bool
	isStreaming         bool           // Track if we're currently receiving a streaming message
	showingLoadingState bool           // Track if we're showing loading state
	loadingMessage      string         // Message to display while loading
	spinner             spinner.Model  // Animated spinner for loading state
	currentPhase        string         // Current workflow phase for context
}

// NewChatView creates a new chat view
func NewChatView(width, height int) ChatView {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Prompt = "> "
	ta.CharLimit = 10000 // Increased for multi-line support
	ta.SetWidth(width - 4)
	ta.SetHeight(maxInputHeight) // Always show 3 lines for consistent UI
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(true)
	ta.KeyMap.InsertNewline.SetKeys("ctrl+j") // Bind to ctrl+j (what Shift+Enter sends via Ghostty keybind)
	ta.KeyMap.DeleteBeforeCursor.SetEnabled(false) // Disable so Ctrl+U can be handled by model for full clear

	vp := viewport.New(width-4, height-4)
	vp.YPosition = 0

	// Initialize spinner with MiniDot style
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	return ChatView{
		viewport: vp,
		textarea: ta,
		messages: []types.ChatMessage{},
		width:    width,
		height:   height,
		ready:    false,
		spinner:  s,
	}
}

// Init implements tea.Model
func (c ChatView) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, c.spinner.Tick)
}

// Update implements tea.Model
func (c ChatView) Update(msg tea.Msg) (ChatView, tea.Cmd) {
	var (
		vpCmd      tea.Cmd
		taCmd      tea.Cmd
		spinnerCmd tea.Cmd
	)

	c.viewport, vpCmd = c.viewport.Update(msg)
	c.textarea, taCmd = c.textarea.Update(msg)

	// Always update spinner to keep tick chain alive
	c.spinner, spinnerCmd = c.spinner.Update(msg)

	// Adjust textarea height based on content for multi-line support
	c.adjustTextareaHeight()

	return c, tea.Batch(vpCmd, taCmd, spinnerCmd)
}

// View implements tea.Model
func (c ChatView) View() string {
	if !c.ready {
		return "Initializing chat..."
	}

	// Update viewport content if messages have changed or loading state is active
	// When loading state is active, always update to refresh spinner animation
	if c.contentDirty || c.showingLoadingState {
		chatContent := c.renderMessages()
		c.viewport.SetContent(chatContent)
		// Scroll to bottom when loading so user can see the spinner
		if c.showingLoadingState {
			c.viewport.GotoBottom()
		}
		// Only clear contentDirty if we're not loading (to ensure spinner keeps animating)
		if !c.showingLoadingState {
			c.contentDirty = false
		}
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
	// Note: Don't clear loading state here - it should persist until the agent turn is completely done
	// The loading state will be cleared when the turn completes (on "complete" message)

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
	if len(c.messages) == 0 && !c.showingLoadingState {
		return chatEmptyStyle.Render("No messages yet. Start the conversation!")
	}

	var rendered []string
	for _, msg := range c.messages {
		rendered = append(rendered, c.renderMessage(msg))
	}

	// Add loading state message if active
	if c.showingLoadingState {
		loadingStyle := lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true).
			Padding(1, 2)
		loadingMsg := loadingStyle.Render(c.spinner.View() + " " + c.loadingMessage)
		rendered = append(rendered, loadingMsg)
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
		// Tool use messages: styled with icon, tool name, and rich parameters
		toolIcon := lipgloss.NewStyle().
			Foreground(theme.Info).
			Bold(true).
			Render("▶")

		toolName := lipgloss.NewStyle().
			Foreground(theme.Info).
			Bold(true).
			Render(msg.Content)

		// Build rich parameter display
		params := buildToolParams(msg.Content, msg.Args, c.width-20)
		var paramsStyled string
		if params != "" {
			paramsStyled = lipgloss.NewStyle().
				Foreground(theme.TextMuted).
				Render(params)
		}

		if paramsStyled != "" {
			return lipgloss.NewStyle().
				Padding(0, 2).
				Render(fmt.Sprintf("%s %s %s", toolIcon, toolName, paramsStyled))
		}
		return lipgloss.NewStyle().
			Padding(0, 2).
			Render(fmt.Sprintf("%s %s", toolIcon, toolName))

	case "error":
		// Error messages: styled with warning icon and red text
		errorIcon := lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true).
			Render("⚠")

		header := errorHeaderStyle.Render(fmt.Sprintf("%s Error (%s)", errorIcon, timestamp))

		// Apply text wrapping to error content using existing infrastructure
		contentWidth := c.width - errorMessageHorizontalPadding

		// Use fallback word wrap for plain text error messages to maintain styling
		wrappedContent := wordWrap(msg.Content, contentWidth-errorMessageStylePadding)
		content := errorMessageStyle.Render(wrappedContent)

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
// title is the phase title (e.g., "Collab Specify: Feature Name" or "Collab Bug Fix: Bug Name")
// hasDescription indicates whether the user provided a description as an argument
func (c *ChatView) AddWelcomeMessage(title string, hasDescription bool) {
	var welcome string

	// Detect phase type from title
	if strings.Contains(title, "Bug Fix") {
		welcome = fmt.Sprintf(`# Welcome to Bug Fix

I'll help you fix this bug: **%s**

Please describe the bug so I can help you resolve it.`, strings.TrimPrefix(title, "Collab Bug Fix: "))
	} else if strings.Contains(title, "Plan") {
		welcome = fmt.Sprintf(`# Welcome to Implementation Planning

I'll help you create an implementation plan for: **%s**`, strings.TrimPrefix(title, "Collab Plan: "))
	} else if strings.Contains(title, "Tasks") {
		welcome = fmt.Sprintf(`# Welcome to Task Generation

I'll help you break down the implementation into tasks for: **%s**`, strings.TrimPrefix(title, "Collab Tasks: "))
	} else if strings.Contains(title, "Implement") {
		welcome = fmt.Sprintf(`# Welcome to Implementation

I'll help you implement: **%s**`, strings.TrimPrefix(title, "Collab Implement: "))
	} else if hasDescription {
		welcome = fmt.Sprintf(`# Welcome to Feature Specification

I'll help you create a comprehensive specification for: **%s**`, strings.TrimPrefix(title, "Collab Specify: "))
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

// SetLoadingState displays a contextual loading message in the chat area.
// The phase parameter should be one of: "specify", "plan", "tasks", "implement".
func (c *ChatView) SetLoadingState(phase string) {
	c.showingLoadingState = true
	c.currentPhase = phase

	// Get last user message for context
	lastUserMsg := c.getLastUserMessage()

	// Generate contextual loading message
	c.loadingMessage = GenerateLoadingMessage(lastUserMsg, phase)
}

// getLastUserMessage returns the content of the most recent user message.
func (c *ChatView) getLastUserMessage() string {
	for i := len(c.messages) - 1; i >= 0; i-- {
		if c.messages[i].Role == "user" {
			return c.messages[i].Content
		}
	}
	return ""
}

// ClearLoadingState removes the loading message from the chat area
func (c *ChatView) ClearLoadingState() {
	c.showingLoadingState = false
	c.loadingMessage = ""
	c.contentDirty = true // Trigger re-render to remove loading message
}

// IsShowingLoading returns whether the chat is currently showing a loading state
func (c *ChatView) IsShowingLoading() bool {
	return c.showingLoadingState
}

// adjustTextareaHeight dynamically adjusts the textarea height based on content
func (c *ChatView) adjustTextareaHeight() {
	// Always use maxInputHeight to show consistent 3-line input area
	// The textarea component shows the prompt (>) on each line
	if c.textarea.Height() != maxInputHeight {
		c.textarea.SetHeight(maxInputHeight)
	}
}

// ResetScrollState completely reinitializes the viewport state
func (c *ChatView) ResetScrollState() {
	// Calculate viewport height same as SetSize method
	viewportHeight := c.height - 1 - maxInputHeight
	if viewportHeight < 3 {
		viewportHeight = 3
	}

	// Recreate viewport with current dimensions to clear internal state
	c.viewport = viewport.New(c.width-4, viewportHeight)
	c.viewport.YPosition = 0

	// Refresh content and scroll to bottom
	if len(c.messages) > 0 {
		chatContent := c.renderMessages()
		c.viewport.SetContent(chatContent)
		c.viewport.GotoBottom()
	}

	// Reset streaming and loading states
	c.isStreaming = false
	c.showingLoadingState = false
	c.contentDirty = false
}

// ResetToBottom forces scroll to bottom and re-enables auto-scroll behavior
func (c *ChatView) ResetToBottom() {
	c.viewport.GotoBottom()
	c.contentDirty = true
}
