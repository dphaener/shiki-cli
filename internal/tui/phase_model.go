package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/orchestrator"
	"github.com/dphaener/shiki-cli/internal/tui/components"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// PhaseType identifies the type of phase for configuration
type PhaseType string

const (
	PhasePlan      PhaseType = "plan"
	PhaseTasks     PhaseType = "tasks"
	PhaseImplement PhaseType = "implement"
	PhaseSpecify   PhaseType = "specify"
	PhaseBug       PhaseType = "bug"
)

// PhaseModelConfig provides configuration for a PhaseModel instance.
// Note: PaneType, ChatPane, PreviewPane are defined in specify_model.go
type PhaseModelConfig struct {
	// PhaseType identifies the phase (plan, tasks, implement, etc.)
	PhaseType PhaseType

	// Title shown in the header
	Title string

	// SubtitleFunc generates the subtitle dynamically
	SubtitleFunc func() string

	// PreviewTitle is the title shown on the preview pane
	PreviewTitle string

	// InitialPromptFunc generates the initial prompt to send to the agent
	InitialPromptFunc func() string

	// RefreshPreviewFunc reads content from file and updates preview
	// Called after tool_use events and on completion
	RefreshPreviewFunc func()

	// OnToolUseFunc is called when a tool_use event is received
	// Can be used for phase-specific side effects
	OnToolUseFunc func(toolName string, args map[string]interface{})

	// GetChatHistoryFunc returns the chat history for the session
	GetChatHistoryFunc func() []types.ChatMessage

	// AppendChatHistoryFunc adds a message to the session's chat history
	AppendChatHistoryFunc func(msg types.ChatMessage)

	// LoadingStateLabel is shown in the chat view while waiting for AI
	LoadingStateLabel string
}

// PhaseModel is a generic model for phase-based TUI workflows.
// It handles the common split-pane chat+preview pattern with streaming AI responses.
type PhaseModel struct {
	// Configuration
	config PhaseModelConfig

	// Dependencies
	orchestrator orchestrator.PhaseOrchestrator
	eventBus     *events.EventBus
	preview      PreviewUpdater

	// UI Components
	chatView components.ChatView

	// Layout state
	width       int
	height      int
	ready       bool
	quitting    bool
	activePane  PaneType
	layoutCache LayoutCache

	// AI interaction state
	waitingForAI     bool
	responseBuffer   string
	activeUpdateChan <-chan orchestrator.MessageUpdate
	activeErrorChan  <-chan error

	// Error state
	err error

	// Interrupt tracking
	interruptRequested bool
	lastInterruptTime  time.Time
	lastEscTime        time.Time
}

// NewPhaseModel creates a new phase model with the given configuration.
func NewPhaseModel(
	config PhaseModelConfig,
	orch orchestrator.PhaseOrchestrator,
	preview PreviewUpdater,
	eventBus *events.EventBus,
) *PhaseModel {
	chatView := components.NewChatView(80, 24)

	return &PhaseModel{
		config:       config,
		orchestrator: orch,
		preview:      preview,
		eventBus:     eventBus,
		chatView:     chatView,
		activePane:   ChatPane,
	}
}

// Init implements tea.Model
func (m *PhaseModel) Init() tea.Cmd {
	return tea.Batch(
		m.chatView.Init(),
		m.preview.Init(),
	)
}

// Update implements tea.Model
func (m *PhaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case PhaseAgentReadyMsg:
		// Agent initialized - start the conversation if no history
		if len(m.config.GetChatHistoryFunc()) == 0 {
			return m, m.startConversation()
		}
		return m, nil

	case PhaseAgentErrorMsg:
		m.chatView.AddError(msg.Err)
		m.waitingForAI = false
		m.chatView.ClearLoadingState()
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		m.err = msg.Err
		return m, nil

	case phaseChannelsReadyMsg:
		m.activeUpdateChan = msg.updateChan
		m.activeErrorChan = msg.errorChan
		return m, waitForPhaseUpdate(m.activeUpdateChan, m.activeErrorChan)

	case phaseUpdateMsg:
		return m.handleStreamUpdate(msg.update)

	case phaseStreamCompleteMsg:
		return m.handleStreamComplete()

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	// Route to active pane
	if m.activePane == ChatPane {
		var chatCmd tea.Cmd
		m.chatView, chatCmd = m.chatView.Update(msg)
		cmds = append(cmds, chatCmd)
	} else {
		cmd := m.preview.UpdatePreview(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleStreamUpdate processes streaming updates from the orchestrator
func (m *PhaseModel) handleStreamUpdate(update orchestrator.MessageUpdate) (tea.Model, tea.Cmd) {
	switch update.Type {
	case "text":
		m.responseBuffer += update.Content
		m.chatView.AddOrUpdateAssistantMessage(m.responseBuffer, true)

	case "tool_use":
		toolMsg := types.ChatMessage{
			Role:      "tool",
			Content:   update.Content,
			Timestamp: time.Now(),
			Args:      update.Args,
		}
		m.chatView.AddToolUse(toolMsg)

		// Call phase-specific tool use handler
		if m.config.OnToolUseFunc != nil {
			m.config.OnToolUseFunc(update.Content, update.Args)
		}

		// Refresh preview from file
		if m.config.RefreshPreviewFunc != nil {
			m.config.RefreshPreviewFunc()
		}

	case "tool_error":
		m.chatView.AddError(fmt.Errorf("Tool failed: %s", update.Content))

	case "message_done":
		if len(m.responseBuffer) > 0 {
			assistantMsg := types.ChatMessage{
				Role:      "assistant",
				Content:   m.responseBuffer,
				Timestamp: time.Now(),
			}
			m.config.AppendChatHistoryFunc(assistantMsg)
			m.responseBuffer = ""
		}
		m.chatView.FinishStreaming()

	case "complete":
		if len(m.responseBuffer) > 0 {
			assistantMsg := types.ChatMessage{
				Role:      "assistant",
				Content:   m.responseBuffer,
				Timestamp: time.Now(),
			}
			m.config.AppendChatHistoryFunc(assistantMsg)
			m.responseBuffer = ""
		}
		m.chatView.FinishStreaming()
		if m.config.RefreshPreviewFunc != nil {
			m.config.RefreshPreviewFunc()
		}
		m.waitingForAI = false
		m.chatView.ClearLoadingState()
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		return m, nil
	}

	// Continue listening
	return m, waitForPhaseUpdate(m.activeUpdateChan, m.activeErrorChan)
}

// handleStreamComplete handles stream completion
func (m *PhaseModel) handleStreamComplete() (tea.Model, tea.Cmd) {
	if len(m.responseBuffer) > 0 {
		assistantMsg := types.ChatMessage{
			Role:      "assistant",
			Content:   m.responseBuffer,
			Timestamp: time.Now(),
		}
		m.config.AppendChatHistoryFunc(assistantMsg)
		m.responseBuffer = ""
	}
	m.chatView.FinishStreaming()
	if m.config.RefreshPreviewFunc != nil {
		m.config.RefreshPreviewFunc()
	}
	m.waitingForAI = false
	m.chatView.ClearLoadingState()
	m.activeUpdateChan = nil
	m.activeErrorChan = nil
	return m, nil
}

// handleWindowSize handles window resize events
func (m *PhaseModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	m.width = msg.Width
	m.height = msg.Height
	m.layoutCache.Dirty = true
	m.updateLayoutCache()

	availableHeight := msg.Height - m.layoutCache.HeaderHeight - m.layoutCache.FooterHeight
	if availableHeight < 5 {
		availableHeight = 5
	}

	chatWidth := int(float64(msg.Width) * 0.6)
	previewWidth := msg.Width - chatWidth

	m.chatView.SetSize(chatWidth-4, availableHeight-4)
	m.preview.SetSize(previewWidth-4, availableHeight-4)

	if !m.ready {
		m.ready = true
		if len(m.config.GetChatHistoryFunc()) == 0 {
			m.chatView.AddWelcomeMessage(m.config.Title, true)
		}
		m.chatView.ClearInput()
		// Focus the chat input so user can type immediately
		cmd := m.chatView.Focus()
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress handles keyboard input
func (m *PhaseModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		if m.orchestrator != nil {
			_ = m.orchestrator.Stop()
		}
		return m, tea.Quit

	case tea.KeyEsc:
		now := time.Now()
		if m.waitingForAI && m.orchestrator != nil {
			m.interruptRequested = true
			m.lastInterruptTime = now
			m.waitingForAI = false
			m.chatView.ClearLoadingState()
			m.activeUpdateChan = nil
			m.activeErrorChan = nil

			m.chatView.AddMessage(types.ChatMessage{
				Role:      "assistant",
				Content:   "Interrupting...",
				Timestamp: now,
			})

			if m.activePane == ChatPane {
				cmd := m.chatView.Focus()
				cmds = append(cmds, cmd)
			}

			go func() {
				_ = m.orchestrator.Interrupt()
			}()
		} else if m.activePane == ChatPane {
			// Double-ESC clears input
			if now.Sub(m.lastEscTime) < 2*time.Second {
				m.chatView.ClearInput()
				m.lastEscTime = time.Time{}
			} else {
				m.lastEscTime = now
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyTab:
		if m.activePane == ChatPane {
			m.activePane = PreviewPane
			m.chatView.Blur()
		} else {
			m.activePane = ChatPane
			cmd := m.chatView.Focus()
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyEnter:
		if m.activePane == ChatPane {
			input := m.chatView.GetInput()
			if input != "" {
				return m, m.sendMessage(input)
			}
		}
	}

	// Pass key events to the active pane for regular typing
	if m.activePane == ChatPane {
		var chatCmd tea.Cmd
		m.chatView, chatCmd = m.chatView.Update(msg)
		cmds = append(cmds, chatCmd)
	} else {
		cmd := m.preview.UpdatePreview(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *PhaseModel) View() string {
	if m.err != nil && !m.ready {
		return phaseErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return phaseQuitStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	m.layoutCache.HeaderHeight = lipgloss.Height(header)
	m.layoutCache.FooterHeight = lipgloss.Height(footer)

	availableHeight := m.height - m.layoutCache.HeaderHeight - m.layoutCache.FooterHeight

	chatWidth := int(float64(m.width) * 0.6)
	previewWidth := m.width - chatWidth

	chatPane := m.renderPane("Chat", m.chatView.View(), chatWidth, availableHeight, m.activePane == ChatPane)
	previewPane := m.renderPane(m.config.PreviewTitle, m.preview.View(), previewWidth, availableHeight, m.activePane == PreviewPane)

	content := lipgloss.JoinHorizontal(lipgloss.Top, chatPane, previewPane)

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// ViewContent returns just the content panes without header/footer
func (m *PhaseModel) ViewContent() string {
	if m.err != nil && !m.ready {
		return phaseErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return phaseQuitStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing..."
	}

	m.updateLayoutCache()
	availableHeight := m.height - m.layoutCache.HeaderHeight - m.layoutCache.FooterHeight

	chatWidth := int(float64(m.width) * 0.6)
	previewWidth := m.width - chatWidth

	chatPane := m.renderPane("Chat", m.chatView.View(), chatWidth, availableHeight, m.activePane == ChatPane)
	previewPane := m.renderPane(m.config.PreviewTitle, m.preview.View(), previewWidth, availableHeight, m.activePane == PreviewPane)

	return lipgloss.JoinHorizontal(lipgloss.Top, chatPane, previewPane)
}

// renderHeader renders the TUI header
func (m *PhaseModel) renderHeader() string {
	subtitle := ""
	if m.config.SubtitleFunc != nil {
		subtitle = m.config.SubtitleFunc()
	}

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		phaseTitleStyle.Render(m.config.Title),
		phaseSubtitleStyle.Render(subtitle),
	)

	return phaseHeaderBoxStyle.Width(m.width).Render(headerContent)
}

// renderFooter renders the TUI footer
func (m *PhaseModel) renderFooter() string {
	var shortcuts []string

	if m.activePane == ChatPane {
		shortcuts = []string{
			"Enter: Send",
			"Shift+Enter: New Line",
			"Esc×2: Clear",
			"Tab: Switch to Preview",
			"↑/↓: Scroll",
			"Esc: Interrupt",
			"Ctrl+C: Quit",
		}
	} else {
		shortcuts = []string{
			"Tab: Switch to Chat",
			"↑/↓: Scroll",
			"Ctrl+C: Quit",
		}
	}

	footer := phaseFooterStyle.Render(strings.Join(shortcuts, " • "))
	return phaseFooterBoxStyle.Width(m.width).Render(footer)
}

// renderPane renders a pane with border and title
func (m *PhaseModel) renderPane(title, content string, width, height int, active bool) string {
	borderStyle := phasePaneBorderStyle
	if active {
		borderStyle = phaseSelectedPaneBorderStyle
	}

	paneStyle := borderStyle.
		MaxWidth(width).
		MaxHeight(height).
		Width(width - 2).
		Height(height - 2)

	return paneStyle.Render(content)
}

// startConversation starts the initial conversation
func (m *PhaseModel) startConversation() tea.Cmd {
	if m.config.InitialPromptFunc == nil {
		return nil
	}

	initialPrompt := m.config.InitialPromptFunc()

	msg := types.ChatMessage{
		Role:      "user",
		Content:   initialPrompt,
		Timestamp: time.Now(),
	}
	m.config.AppendChatHistoryFunc(msg)
	m.chatView.AddMessage(msg)
	m.waitingForAI = true
	m.chatView.SetLoadingState(string(m.config.PhaseType))

	m.responseBuffer = ""
	return startPhaseStreaming(m.orchestrator, initialPrompt)
}

// sendMessage sends a user message
func (m *PhaseModel) sendMessage(input string) tea.Cmd {
	msg := types.ChatMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	}
	m.config.AppendChatHistoryFunc(msg)
	m.chatView.AddMessage(msg)
	m.chatView.ClearInput()
	m.waitingForAI = true
	m.chatView.SetLoadingState(string(m.config.PhaseType))

	m.responseBuffer = ""
	return startPhaseStreaming(m.orchestrator, input)
}

// updateLayoutCache updates cached layout dimensions
func (m *PhaseModel) updateLayoutCache() {
	if m.layoutCache.Dirty || m.width != m.layoutCache.LastWidth || m.height != m.layoutCache.LastHeight {
		m.layoutCache.HeaderHeight = lipgloss.Height(m.renderHeader())
		m.layoutCache.FooterHeight = lipgloss.Height(m.renderFooter())
		m.layoutCache.LastWidth = m.width
		m.layoutCache.LastHeight = m.height
		m.layoutCache.Dirty = false
	}
}

// Streaming helpers

func startPhaseStreaming(orch orchestrator.PhaseOrchestrator, input string) tea.Cmd {
	return func() tea.Msg {
		updateChan, errorChan := orch.SendMessage(input)
		return phaseChannelsReadyMsg{
			updateChan: updateChan,
			errorChan:  errorChan,
		}
	}
}

func waitForPhaseUpdate(updateChan <-chan orchestrator.MessageUpdate, errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case update, ok := <-updateChan:
			if !ok {
				return phaseStreamCompleteMsg{}
			}
			return phaseUpdateMsg{update: update}
		case err, ok := <-errorChan:
			if ok && err != nil {
				return PhaseAgentErrorMsg{Err: err}
			}
			return phaseStreamCompleteMsg{}
		}
	}
}

// Message types for PhaseModel

// PhaseAgentReadyMsg signals the orchestrator is initialized and ready
type PhaseAgentReadyMsg struct{}

// PhaseAgentErrorMsg represents an error from the orchestrator
type PhaseAgentErrorMsg struct {
	Err error
}

type phaseChannelsReadyMsg struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

type phaseUpdateMsg struct {
	update orchestrator.MessageUpdate
}

type phaseStreamCompleteMsg struct{}

// Styles for PhaseModel using Sekkei Design System theme
var (
	phaseTitleStyle = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1)

	phaseSubtitleStyle = lipgloss.NewStyle().
				Foreground(theme.TextMuted).
				Padding(0, 1)

	phaseHeaderBoxStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(theme.Border)

	phaseFooterStyle = lipgloss.NewStyle().
				Foreground(theme.TextMuted).
				Padding(0, 1)

	phaseFooterBoxStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderTop(true).
				BorderForeground(theme.Border)

	phasePaneBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(theme.Border).
				Padding(1)

	phaseSelectedPaneBorderStyle = lipgloss.NewStyle().
					BorderStyle(lipgloss.RoundedBorder()).
					BorderForeground(theme.BorderActive).
					Padding(1)

	phaseErrorStyle = lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true).
			Padding(1)

	phaseQuitStyle = lipgloss.NewStyle().
			Foreground(theme.Success).
			Bold(true).
			Padding(1)
)
