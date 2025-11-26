package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/internal/tui/components"
	"github.com/darinhaener/collab/internal/tui/theme"
	"github.com/darinhaener/collab/pkg/types"
)

// PaneType represents which pane is currently active
type PaneType int

const (
	ChatPane PaneType = iota
	PreviewPane
)

// SpecifyModel manages the specify mode TUI
type SpecifyModel struct {
	session           *types.SpecifySession
	orchestrator      *orchestrator.SpecifyOrchestrator
	eventBus          *events.EventBus
	chatView          components.ChatView
	specPreview       components.SpecPreview
	width             int
	height            int
	ready             bool
	quitting          bool
	waitingForAI      bool
	err               error
	responseBuffer    string // Buffer for accumulating assistant response
	activeUpdateChan  <-chan orchestrator.MessageUpdate
	activeErrorChan   <-chan error
	activePane        PaneType // Which pane is currently active
}

// NewSpecifyModel creates a new specify model
func NewSpecifyModel(session *types.SpecifySession, eventBus *events.EventBus) SpecifyModel {
	// Initialize with default sizes - will be updated on first WindowSizeMsg
	chatView := components.NewChatView(80, 24)
	specPreview := components.NewSpecPreview(80, 24)

	return SpecifyModel{
		session:     session,
		eventBus:    eventBus,
		chatView:    chatView,
		specPreview: specPreview,
		ready:       false,
		activePane:  ChatPane, // Start with chat pane active
	}
}

// Init implements tea.Model
func (m SpecifyModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen, // Enter alt screen FIRST to isolate from terminal
		m.chatView.Init(),
		m.specPreview.Init(),
		initializeAgent(m.session, m.eventBus),
	)
}

// initializeAgent initializes the AI agent
func initializeAgent(session *types.SpecifySession, eventBus *events.EventBus) tea.Cmd {
	return func() tea.Msg {
		apiKey := orchestrator.GetAPIKey()
		orch := orchestrator.NewSpecifyOrchestrator(session, apiKey, eventBus)

		if err := orch.Initialize(); err != nil {
			return AgentErrorMsg{Err: err}
		}

		return AgentInitializedMsg{Orchestrator: orch}
	}
}

// clearInputAfterDelay returns a cmd that clears input after a brief delay
func clearInputAfterDelay() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return clearInputMsg{}
	})
}

// clearInputMsg signals to clear the input field
type clearInputMsg struct{}

// Update implements tea.Model
func (m SpecifyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case AgentInitializedMsg:
		m.orchestrator = msg.Orchestrator
		// Handle initialization based on whether description was provided
		if len(m.session.ChatHistory) == 0 {
			if m.session.FeatureDesc != "" {
				// WITH DESCRIPTION: Auto-start discovery with the feature description
				return m, m.autoStartDiscovery()
			} else {
				// NO DESCRIPTION: Let AI initiate the conversation
				return m, m.letAIInitiate()
			}
		}
		return m, nil

	case AgentErrorMsg:
		// Show error inline in chat instead of replacing entire view
		m.chatView.AddError(msg.Err)
		m.waitingForAI = false
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		// Also store for potential full-screen error display
		m.err = msg.Err
		return m, nil

	case AgentResponseMsg:
		// Add assistant message to chat
		assistantMsg := types.ChatMessage{
			Role:      "assistant",
			Content:   msg.Content,
			Timestamp: time.Now(),
		}
		m.session.ChatHistory = append(m.session.ChatHistory, assistantMsg)
		m.chatView.AddMessage(assistantMsg)
		m.waitingForAI = false
		return m, nil

	case AgentStreamMsg:
		// Stream partial response (for future streaming support)
		// For now, we'll wait for the complete message
		return m, nil

	case AgentToolUseMsg:
		// Add tool use indicator to chat
		toolMsg := types.ChatMessage{
			Role:      "tool",
			Content:   msg.ToolName,
			Timestamp: time.Now(),
		}
		m.chatView.AddToolUse(toolMsg)
		return m, nil

	case agentChannelsReady:
		// Store channels and start listening for updates
		m.activeUpdateChan = msg.updateChan
		m.activeErrorChan = msg.errorChan
		return m, waitForNextUpdate(m.activeUpdateChan, m.activeErrorChan)

	case agentUpdate:
		// Process the update
		switch msg.update.Type {
		case "text":
			// Accumulate text
			m.responseBuffer += msg.update.Content
			// Stream the text to the chat view in real-time
			m.chatView.AddOrUpdateAssistantMessage(m.responseBuffer, true)

		case "tool_use":
			// Show tool use immediately
			toolMsg := types.ChatMessage{
				Role:      "tool",
				Content:   msg.update.Content,
				Timestamp: time.Now(),
			}
			m.chatView.AddToolUse(toolMsg)

			// Refresh preview from file - agent may have written spec
			m.refreshSpecPreviewFromFile()

		case "tool_error":
			// Show tool error to user
			m.chatView.AddError(fmt.Errorf("Tool failed: %s", msg.update.Content))

		case "message_done":
			// One assistant message is complete, but conversation continues
			// Finalize current message and prepare for next one
			if len(m.responseBuffer) > 0 {
				assistantMsg := types.ChatMessage{
					Role:      "assistant",
					Content:   m.responseBuffer,
					Timestamp: time.Now(),
				}
				m.session.ChatHistory = append(m.session.ChatHistory, assistantMsg)

				// Check if this response contains spec content
				m.updateSpecPreviewFromMessage(m.responseBuffer)

				m.responseBuffer = ""
			}
			// Mark streaming as finished so next text starts a NEW message
			m.chatView.FinishStreaming()
			// Continue listening - don't stop!

		case "complete":
			// Final response - add to session history and update spec
			if len(m.responseBuffer) > 0 {
				assistantMsg := types.ChatMessage{
					Role:      "assistant",
					Content:   m.responseBuffer,
					Timestamp: time.Now(),
				}
				m.session.ChatHistory = append(m.session.ChatHistory, assistantMsg)

				// Update phase indicators based on message content
				m.updateSpecPreviewFromMessage(m.responseBuffer)

				m.responseBuffer = ""
			}
			// Mark streaming as finished
			m.chatView.FinishStreaming()
			// Refresh preview from file to show actual spec content
			m.refreshSpecPreviewFromFile()
			m.waitingForAI = false
			m.activeUpdateChan = nil
			m.activeErrorChan = nil
			return m, nil
		}

		// Continue listening for more updates
		return m, waitForNextUpdate(m.activeUpdateChan, m.activeErrorChan)

	case agentStreamComplete:
		// Stream ended - finalize response if we have one
		if len(m.responseBuffer) > 0 {
			assistantMsg := types.ChatMessage{
				Role:      "assistant",
				Content:   m.responseBuffer,
				Timestamp: time.Now(),
			}
			m.session.ChatHistory = append(m.session.ChatHistory, assistantMsg)

			// Update phase indicators based on message content
			m.updateSpecPreviewFromMessage(m.responseBuffer)

			m.responseBuffer = ""
		}
		// Mark streaming as finished
		m.chatView.FinishStreaming()
		// Refresh preview from file to show actual spec content
		m.refreshSpecPreviewFromFile()
		m.waitingForAI = false
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		return m, nil

	case clearInputMsg:
		// Clear input field to remove any late-arriving escape sequences
		m.chatView.ClearInput()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Let lipgloss calculate header/footer height dynamically
		headerHeight := lipgloss.Height(m.renderHeader())
		footerHeight := lipgloss.Height(m.renderFooter())

		// Available height for content
		availableHeight := msg.Height - headerHeight - footerHeight
		if availableHeight < 5 {
			availableHeight = 5 // Minimum
		}

		// Split width: 60% chat, 40% preview
		chatWidth := int(float64(msg.Width) * 0.6)
		previewWidth := msg.Width - chatWidth

		// Account for pane borders (2) + padding (2) = 4 total for both width and height
		m.chatView.SetSize(chatWidth-4, availableHeight-4)
		m.specPreview.SetSize(previewWidth-4, availableHeight-4)

		if !m.ready {
			m.ready = true
			// Only add welcome message if we haven't auto-started discovery
			if len(m.session.ChatHistory) == 0 {
				hasDescription := m.session.FeatureDesc != ""
				m.chatView.AddWelcomeMessage(m.session.FriendlyName, hasDescription)
			}
			// Clear any escape sequences that leaked into textarea
			m.chatView.ClearInput()
			// Schedule one more clear after a brief delay to catch any late terminal responses
			cmds = append(cmds, clearInputAfterDelay())
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			// Cleanup orchestrator
			if m.orchestrator != nil {
				_ = m.orchestrator.Stop()
			}
			return m, tea.Quit

		case "tab":
			// Switch between panes
			if m.activePane == ChatPane {
				m.activePane = PreviewPane
				// Blur chat textarea when switching away
				m.chatView.Blur()
			} else {
				m.activePane = ChatPane
				// Focus chat textarea when switching to it
				cmd := m.chatView.Focus()
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case "enter":
			// Only send message if chat pane is active
			if m.activePane == ChatPane {
				input := m.chatView.GetInput()
				if input != "" {
					return m, m.sendMessage(input)
				}
			}
		}
	}

	// Route keyboard events to the active pane
	if m.activePane == ChatPane {
		// Update chat view (handles viewport scrolling and textarea input)
		var chatCmd tea.Cmd
		m.chatView, chatCmd = m.chatView.Update(msg)
		cmds = append(cmds, chatCmd)
	} else {
		// Update spec preview (handles viewport scrolling)
		var previewCmd tea.Cmd
		m.specPreview, previewCmd = m.specPreview.Update(msg)
		cmds = append(cmds, previewCmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m SpecifyModel) View() string {
	// Only show full-screen error if not ready yet (critical init failure)
	// Once ready, errors are shown inline in the chat
	if m.err != nil && !m.ready {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return quitMessageStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing..."
	}

	// Render components
	header := m.renderHeader()
	footer := m.renderFooter()

	// Calculate available height dynamically
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	availableHeight := m.height - headerHeight - footerHeight

	// Split panes: 60% chat, 40% preview
	chatWidth := int(float64(m.width) * 0.6)
	previewWidth := m.width - chatWidth

	// Render panes with proper sizing
	chatPane := m.renderPane(
		"Chat",
		m.chatView.View(),
		chatWidth,
		availableHeight,
		m.activePane == ChatPane, // Active if ChatPane is selected
	)

	previewPane := m.renderPane(
		"Specification Preview",
		m.specPreview.View(),
		previewWidth,
		availableHeight,
		m.activePane == PreviewPane, // Active if PreviewPane is selected
	)

	// Join panes horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, chatPane, previewPane)

	// Join everything vertically and let lipgloss handle sizing
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderHeader renders the TUI header
func (m *SpecifyModel) renderHeader() string {
	title := fmt.Sprintf("Collab Specify: %s", m.session.FriendlyName)
	subtitle := fmt.Sprintf("Feature #%03d - %s - Phase: %s",
		m.session.FeatureNumber,
		m.session.Slug,
		m.session.Phase,
	)

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		subtitleStyle.Render(subtitle),
	)

	return headerBoxStyle.Width(m.width).Render(headerContent)
}

// renderFooter renders the TUI footer with keyboard shortcuts
func (m *SpecifyModel) renderFooter() string {
	var shortcuts []string

	if m.activePane == ChatPane {
		shortcuts = []string{
			"Enter: Send",
			"Tab: Switch to Preview",
			"↑/↓/PgUp/PgDn: Scroll",
			"Ctrl+C/Esc: Quit",
		}
	} else {
		shortcuts = []string{
			"Tab: Switch to Chat",
			"↑/↓/PgUp/PgDn: Scroll",
			"Ctrl+C/Esc: Quit",
		}
	}

	var footer string
	if m.waitingForAI {
		footer = footerStyle.Render(strings.Join(shortcuts, " • ") + " | ⏳ Waiting for AI...")
	} else {
		footer = footerStyle.Render(strings.Join(shortcuts, " • "))
	}

	return footerBoxStyle.Width(m.width).Render(footer)
}

// renderPane renders a pane with border and title
func (m *SpecifyModel) renderPane(title, content string, width, height int, active bool) string {
	borderStyle := specifyPaneBorderStyle
	if active {
		borderStyle = specifySelectedPaneBorderStyle
	}

	// Use MaxWidth and MaxHeight instead of fixed Width/Height for better flexibility
	paneStyle := borderStyle.
		MaxWidth(width).
		MaxHeight(height).
		Width(width - 2).  // Account for borders
		Height(height - 2) // Account for borders

	return paneStyle.Render(content)
}

// startAgentStreaming initiates the agent query and returns channels
func startAgentStreaming(orch *orchestrator.SpecifyOrchestrator, input string) tea.Cmd {
	return func() tea.Msg {
		updateChan, errorChan := orch.SendMessage(input)
		return agentChannelsReady{
			updateChan: updateChan,
			errorChan:  errorChan,
		}
	}
}

// waitForNextUpdate waits for the next message from the agent
func waitForNextUpdate(updateChan <-chan orchestrator.MessageUpdate, errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case update, ok := <-updateChan:
			if !ok {
				// Channel closed
				return agentStreamComplete{}
			}
			return agentUpdate{update: update}

		case err, ok := <-errorChan:
			if ok && err != nil {
				return AgentErrorMsg{Err: err}
			}
			return agentStreamComplete{}
		}
	}
}

// agentChannelsReady signals that streaming channels are ready
type agentChannelsReady struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

// agentUpdate carries a single update from the agent
type agentUpdate struct {
	update orchestrator.MessageUpdate
}

// agentStreamComplete signals that streaming is complete
type agentStreamComplete struct{}

// refreshSpecPreviewFromFile reads the spec file from disk and updates the preview
func (m *SpecifyModel) refreshSpecPreviewFromFile() {
	if m.session.SpecFile == "" {
		return
	}

	content, err := os.ReadFile(m.session.SpecFile)
	if err != nil {
		return // File may not exist yet
	}

	// Strip YAML frontmatter if present
	specContent := stripFrontmatter(string(content))
	if specContent == "" {
		return
	}

	m.specPreview.SetContent(specContent)
	m.session.CurrentSpec = specContent

	// Update phase based on content
	if containsSpecSections(specContent) {
		m.session.Phase = types.PhaseGeneration
		m.specPreview.SetPhase(types.PhaseGeneration)
	}
}

// stripFrontmatter removes YAML frontmatter from markdown content
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2])
	}
	return content
}

// updateSpecPreviewFromMessage updates phase indicators based on message content
// Note: Actual spec content is read from file via refreshSpecPreviewFromFile()
func (m *SpecifyModel) updateSpecPreviewFromMessage(content string) {
	// Only update phase indicators, not content (content comes from file)
	if m.session.Phase == types.PhaseDiscovery && containsSpecSections(content) {
		m.session.Phase = types.PhaseGeneration
		m.specPreview.SetPhase(types.PhaseGeneration)
	}
}

// isSpecContent checks if content looks like a specification document
func isSpecContent(content string) bool {
	// Check for markdown headers and spec-like structure
	content = strings.ToLower(content)
	return strings.Contains(content, "##") &&
		(strings.Contains(content, "overview") ||
			strings.Contains(content, "requirements") ||
			strings.Contains(content, "success criteria"))
}

// containsSpecSections checks if content has multiple spec sections
func containsSpecSections(content string) bool {
	content = strings.ToLower(content)
	sectionCount := 0
	sections := []string{"overview", "requirements", "success criteria", "user scenarios", "assumptions"}
	for _, section := range sections {
		if strings.Contains(content, section) {
			sectionCount++
		}
	}
	return sectionCount >= 3
}

// autoStartDiscovery automatically starts the discovery process with the feature description
func (m *SpecifyModel) autoStartDiscovery() tea.Cmd {
	// Send the feature description as starting point for discovery
	initialPrompt := fmt.Sprintf("I want to create a feature: %s", m.session.FeatureDesc)

	// Add user message to chat
	msg := types.ChatMessage{
		Role:      "user",
		Content:   initialPrompt,
		Timestamp: time.Now(),
	}
	m.session.ChatHistory = append(m.session.ChatHistory, msg)
	m.chatView.AddMessage(msg)
	m.waitingForAI = true

	// Send to AI agent
	if m.orchestrator == nil {
		return func() tea.Msg {
			return AgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	// Reset response buffer and start streaming
	m.responseBuffer = ""
	return startAgentStreaming(m.orchestrator, initialPrompt)
}

// letAIInitiate triggers the AI to start the conversation when no description was provided
func (m *SpecifyModel) letAIInitiate() tea.Cmd {
	m.waitingForAI = true

	// Send to AI agent with a prompt that triggers the AI's greeting
	if m.orchestrator == nil {
		return func() tea.Msg {
			return AgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	// Reset response buffer and start streaming
	// Send an empty-ish message that signals the AI to initiate
	m.responseBuffer = ""
	return startAgentStreaming(m.orchestrator, "Hello, I'd like to specify a new feature.")
}

// sendMessage sends a user message and triggers AI response
func (m *SpecifyModel) sendMessage(input string) tea.Cmd {
	// Add user message to chat
	msg := types.ChatMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	}
	m.session.ChatHistory = append(m.session.ChatHistory, msg)
	m.chatView.AddMessage(msg)
	m.chatView.ClearInput()
	m.waitingForAI = true

	// Send to AI agent
	if m.orchestrator == nil {
		return func() tea.Msg {
			return AgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	// Reset response buffer and start streaming
	m.responseBuffer = ""
	return startAgentStreaming(m.orchestrator, input)
}

// Message types for agent communication

// AgentInitializedMsg indicates the agent is ready
type AgentInitializedMsg struct {
	Orchestrator *orchestrator.SpecifyOrchestrator
}

// AgentResponseMsg represents a complete AI agent response
type AgentResponseMsg struct {
	Content string
}

// AgentStreamMsg represents a streaming chunk from the AI agent
type AgentStreamMsg struct {
	Chunk string
}

// AgentToolUseMsg represents a tool invocation by the agent
type AgentToolUseMsg struct {
	ToolName string
}

// AgentErrorMsg represents an error from the AI agent
type AgentErrorMsg struct {
	Err error
}

// Styles for specify mode using Sekkei Design System theme
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1)

	headerBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(theme.Border)

	footerStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1)

	footerBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(theme.Border)

	specifyPaneBorderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(1)

	specifySelectedPaneBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(theme.BorderActive).
				Padding(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true).
			Padding(1)

	quitMessageStyle = lipgloss.NewStyle().
				Foreground(theme.Success).
				Bold(true).
				Padding(1)
)
