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

// ImplementPaneType represents which pane is currently active in implement mode
type ImplementPaneType int

const (
	ImplementChatPane ImplementPaneType = iota
	ImplementPreviewPane
)

// Keyboard interaction constants
const (
	// doubleEscapeTimeout is the time window for detecting double ESC key presses
	implementDoubleEscapeTimeout = 2 * time.Second
	// messageClearShortcut defines the keyboard combination for clearing messages
	implementMessageClearShortcut = "ctrl+u"
)

// ImplementModel manages the implementation TUI
type ImplementModel struct {
	session           *types.WorkflowSession
	orchestrator      *orchestrator.ImplementOrchestrator
	eventBus          *events.EventBus
	chatView          components.ChatView
	implementPreview  components.ImplementPreview
	width             int
	height            int
	ready             bool
	quitting          bool
	waitingForAI      bool
	err               error
	responseBuffer    string
	activeUpdateChan  <-chan orchestrator.MessageUpdate
	activeErrorChan   <-chan error
	activePane        ImplementPaneType
	currentTaskIndex  int
	totalTasks        int
	completedTasks    int
	phaseComplete     bool
	layoutCache       LayoutCache
	// Interrupt state tracking for enhanced keyboard handling
	interruptRequested bool      // Track if user requested interruption
	lastInterruptTime  time.Time // Prevent accidental double-interrupts
	lastEscTime        time.Time // Track last ESC press for double-ESC clear
}

// NewImplementModel creates a new implement model
func NewImplementModel(session *types.WorkflowSession, eventBus *events.EventBus) ImplementModel {
	chatView := components.NewChatView(80, 24)
	implementPreview := components.NewImplementPreview(80, 24)

	return ImplementModel{
		session:          session,
		eventBus:         eventBus,
		chatView:         chatView,
		implementPreview: implementPreview,
		ready:            false,
		activePane:       ImplementChatPane,
	}
}

// Init implements tea.Model
func (m ImplementModel) Init() tea.Cmd {
	return tea.Batch(
		m.chatView.Init(),
		m.implementPreview.Init(),
		initializeImplementAgent(m.session, m.eventBus),
	)
}

// initializeImplementAgent initializes the AI agent for implementation
func initializeImplementAgent(session *types.WorkflowSession, eventBus *events.EventBus) tea.Cmd {
	return func() tea.Msg {
		apiKey := orchestrator.GetAPIKey()
		orch := orchestrator.NewImplementOrchestrator(session, apiKey, eventBus)

		if err := orch.Initialize(); err != nil {
			return ImplementAgentErrorMsg{Err: err}
		}

		return ImplementAgentInitializedMsg{Orchestrator: orch}
	}
}

// Update implements tea.Model
func (m ImplementModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ImplementAgentInitializedMsg:
		m.orchestrator = msg.Orchestrator
		if len(m.session.ImplChatHistory) == 0 {
			return m, m.startImplementation()
		}
		return m, nil

	case ImplementAgentErrorMsg:
		m.chatView.AddError(msg.Err)
		m.waitingForAI = false
		m.chatView.ClearLoadingState()
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		m.err = msg.Err
		return m, nil

	case implementAgentChannelsReady:
		m.activeUpdateChan = msg.updateChan
		m.activeErrorChan = msg.errorChan
		return m, waitForImplementUpdate(m.activeUpdateChan, m.activeErrorChan)

	case implementAgentUpdate:
		switch msg.update.Type {
		case "text":
			m.responseBuffer += msg.update.Content
			m.chatView.AddOrUpdateAssistantMessage(m.responseBuffer, true)

		case "tool_use":
			toolMsg := types.ChatMessage{
				Role:      "tool",
				Content:   msg.update.Content,
				Timestamp: time.Now(),
			}
			m.chatView.AddToolUse(toolMsg)

		case "tool_error":
			m.chatView.AddError(fmt.Errorf("Tool failed: %s", msg.update.Content))

		case "message_done":
			if len(m.responseBuffer) > 0 {
				assistantMsg := types.ChatMessage{
					Role:      "assistant",
					Content:   m.responseBuffer,
					Timestamp: time.Now(),
				}
				m.session.ImplChatHistory = append(m.session.ImplChatHistory, assistantMsg)
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
				m.session.ImplChatHistory = append(m.session.ImplChatHistory, assistantMsg)
				m.responseBuffer = ""
			}
			m.chatView.FinishStreaming()
			m.waitingForAI = false
			m.chatView.ClearLoadingState()
			m.activeUpdateChan = nil
			m.activeErrorChan = nil
			return m, nil
		}

		return m, waitForImplementUpdate(m.activeUpdateChan, m.activeErrorChan)

	case implementAgentStreamComplete:
		if len(m.responseBuffer) > 0 {
			assistantMsg := types.ChatMessage{
				Role:      "assistant",
				Content:   m.responseBuffer,
				Timestamp: time.Now(),
			}
			m.session.ImplChatHistory = append(m.session.ImplChatHistory, assistantMsg)
			m.responseBuffer = ""
		}
		m.chatView.FinishStreaming()
		m.waitingForAI = false
		m.chatView.ClearLoadingState()
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := lipgloss.Height(m.renderHeader())
		footerHeight := lipgloss.Height(m.renderFooter())
		availableHeight := msg.Height - headerHeight - footerHeight
		if availableHeight < 5 {
			availableHeight = 5
		}

		chatWidth := int(float64(msg.Width) * 0.6)
		previewWidth := msg.Width - chatWidth

		m.chatView.SetSize(chatWidth-4, availableHeight-4)
		m.implementPreview.SetSize(previewWidth-4, availableHeight-4)

		if !m.ready {
			m.ready = true
			if len(m.session.ImplChatHistory) == 0 {
				m.chatView.AddWelcomeMessage("Implementation", true)
			}
			m.chatView.ClearInput()
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			// Ctrl+C always quits
			m.quitting = true
			if m.orchestrator != nil {
				_ = m.orchestrator.Stop()
			}
			return m, tea.Quit

		case tea.KeyEsc:
			now := time.Now()
			if m.waitingForAI && m.orchestrator != nil {
				// ESC interrupts agent if waiting
				m.interruptRequested = true
				m.lastInterruptTime = now
				m.waitingForAI = false
				m.chatView.ClearLoadingState()
				m.activeUpdateChan = nil
				m.activeErrorChan = nil

				// Add immediate visual feedback
				m.chatView.AddMessage(types.ChatMessage{
					Role:      "assistant",
					Content:   "Interrupting...",
					Timestamp: now,
				})

				// Re-focus the chat input to ensure it remains responsive
				if m.activePane == ImplementChatPane {
					cmd := m.chatView.Focus()
					cmds = append(cmds, cmd)
				}

				// Call interrupt asynchronously to avoid blocking UI
				go func() {
					_ = m.orchestrator.Interrupt()
				}()
			} else if m.activePane == ImplementChatPane {
				// Double-ESC clears input when not waiting for AI
				if now.Sub(m.lastEscTime) < implementDoubleEscapeTimeout {
					m.chatView.ClearInput()
					m.lastEscTime = time.Time{} // Reset to prevent triple-ESC
				} else {
					m.lastEscTime = now
				}
			}
			return m, tea.Batch(cmds...)

		case tea.KeyTab:
			if m.activePane == ImplementChatPane {
				m.activePane = ImplementPreviewPane
				m.chatView.Blur()
			} else {
				m.activePane = ImplementChatPane
				cmd := m.chatView.Focus()
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case tea.KeyEnter:
			// Handle Enter key for sending messages
			if m.activePane == ImplementChatPane {
				input := m.chatView.GetInput()
				if input != "" {
					// Only send message if there's actual content
					// Multi-line input is handled by the textarea itself when Shift+Enter is pressed
					return m, m.sendMessage(input)
				}
			}
		}
	}

	if m.activePane == ImplementChatPane {
		var chatCmd tea.Cmd
		m.chatView, chatCmd = m.chatView.Update(msg)
		cmds = append(cmds, chatCmd)
	} else {
		var previewCmd tea.Cmd
		m.implementPreview, previewCmd = m.implementPreview.Update(msg)
		cmds = append(cmds, previewCmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m ImplementModel) View() string {
	if m.err != nil && !m.ready {
		return implementErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return implementQuitStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing implementation..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	availableHeight := m.height - headerHeight - footerHeight

	chatWidth := int(float64(m.width) * 0.6)
	previewWidth := m.width - chatWidth

	chatPane := m.renderPane(
		"Chat",
		m.chatView.View(),
		chatWidth,
		availableHeight,
		m.activePane == ImplementChatPane,
	)

	previewPane := m.renderPane(
		"Implementation Progress",
		m.implementPreview.View(),
		previewWidth,
		availableHeight,
		m.activePane == ImplementPreviewPane,
	)

	content := lipgloss.JoinHorizontal(lipgloss.Top, chatPane, previewPane)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// ViewContent returns just the content without header/footer for embedding in workflow
func (m ImplementModel) ViewContent() string {
	if m.err != nil && !m.ready {
		return implementErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return implementQuitStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing implementation..."
	}

	// Update layout cache if needed using optimized approach
	m.updateLayoutCache()

	// Calculate available height using cached heights
	availableHeight := m.height - m.layoutCache.HeaderHeight - m.layoutCache.FooterHeight

	chatWidth := int(float64(m.width) * 0.6)
	previewWidth := m.width - chatWidth

	chatPane := m.renderPane(
		"Chat",
		m.chatView.View(),
		chatWidth,
		availableHeight,
		m.activePane == ImplementChatPane,
	)

	previewPane := m.renderPane(
		"Implementation Progress",
		m.implementPreview.View(),
		previewWidth,
		availableHeight,
		m.activePane == ImplementPreviewPane,
	)

	// Return just the content (panes) without header/footer
	return lipgloss.JoinHorizontal(lipgloss.Top, chatPane, previewPane)
}

func (m *ImplementModel) renderHeader() string {
	title := fmt.Sprintf("Collab Implement: %s", m.session.FriendlyName)

	progressStr := ""
	if m.totalTasks > 0 {
		progressStr = fmt.Sprintf(" [%d/%d tasks]", m.completedTasks, m.totalTasks)
	}

	subtitle := fmt.Sprintf("Feature #%03d - %s - Phase: Implementation%s",
		m.session.FeatureNumber,
		m.session.Slug,
		progressStr,
	)

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		implementTitleStyle.Render(title),
		implementSubtitleStyle.Render(subtitle),
	)

	return implementHeaderBoxStyle.Width(m.width).Render(headerContent)
}

func (m *ImplementModel) renderFooter() string {
	var shortcuts []string

	if m.activePane == ImplementChatPane {
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

	// Render footer without loading state (moved to ChatView)
	footer := implementFooterStyle.Render(strings.Join(shortcuts, " • "))
	return implementFooterBoxStyle.Width(m.width).Render(footer)
}

func (m *ImplementModel) renderPane(title, content string, width, height int, active bool) string {
	borderStyle := implementPaneBorderStyle
	if active {
		borderStyle = implementSelectedPaneBorderStyle
	}

	paneStyle := borderStyle.
		MaxWidth(width).
		MaxHeight(height).
		Width(width - 2).
		Height(height - 2)

	return paneStyle.Render(content)
}

func (m *ImplementModel) startImplementation() tea.Cmd {
	// Load tasks file
	tasksContent := ""
	if content, err := os.ReadFile(m.session.TasksFile); err == nil {
		tasksContent = string(content)
	}

	initialPrompt := fmt.Sprintf("Please implement the feature: %s\n\nTasks:\n%s",
		m.session.FriendlyName, tasksContent)

	msg := types.ChatMessage{
		Role:      "user",
		Content:   initialPrompt,
		Timestamp: time.Now(),
	}
	m.session.ImplChatHistory = append(m.session.ImplChatHistory, msg)
	m.chatView.AddMessage(msg)
	m.waitingForAI = true
	m.chatView.SetLoadingState("implement")

	if m.orchestrator == nil {
		return func() tea.Msg {
			return ImplementAgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	m.responseBuffer = ""
	return startImplementAgentStreaming(m.orchestrator, initialPrompt)
}

func (m *ImplementModel) sendMessage(input string) tea.Cmd {
	msg := types.ChatMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	}
	m.session.ImplChatHistory = append(m.session.ImplChatHistory, msg)
	m.chatView.AddMessage(msg)
	m.chatView.ClearInput()
	m.waitingForAI = true
	m.chatView.SetLoadingState("implement")

	if m.orchestrator == nil {
		return func() tea.Msg {
			return ImplementAgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	m.responseBuffer = ""
	return startImplementAgentStreaming(m.orchestrator, input)
}

// IsPhaseComplete returns whether implementation is complete
func (m *ImplementModel) IsPhaseComplete() bool {
	return m.phaseComplete
}

// Streaming functions

func startImplementAgentStreaming(orch *orchestrator.ImplementOrchestrator, input string) tea.Cmd {
	return func() tea.Msg {
		updateChan, errorChan := orch.SendMessage(input)
		return implementAgentChannelsReady{
			updateChan: updateChan,
			errorChan:  errorChan,
		}
	}
}

func waitForImplementUpdate(updateChan <-chan orchestrator.MessageUpdate, errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case update, ok := <-updateChan:
			if !ok {
				return implementAgentStreamComplete{}
			}
			return implementAgentUpdate{update: update}
		case err, ok := <-errorChan:
			if ok && err != nil {
				return ImplementAgentErrorMsg{Err: err}
			}
			return implementAgentStreamComplete{}
		}
	}
}

// Message types

type ImplementAgentInitializedMsg struct {
	Orchestrator *orchestrator.ImplementOrchestrator
}

type ImplementAgentErrorMsg struct {
	Err error
}

type implementAgentChannelsReady struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

type implementAgentUpdate struct {
	update orchestrator.MessageUpdate
}

type implementAgentStreamComplete struct{}

// Styles
// updateLayoutCache updates the cached header and footer heights if needed
func (m *ImplementModel) updateLayoutCache() {
	if m.layoutCache.Dirty || m.width != m.layoutCache.LastWidth || m.height != m.layoutCache.LastHeight {
		m.layoutCache.HeaderHeight = lipgloss.Height(m.renderHeader())
		m.layoutCache.FooterHeight = lipgloss.Height(m.renderFooter())
		m.layoutCache.LastWidth = m.width
		m.layoutCache.LastHeight = m.height
		m.layoutCache.Dirty = false
	}
}

// invalidateLayoutCache marks the layout cache as dirty, requiring recalculation
func (m *ImplementModel) invalidateLayoutCache() {
	m.layoutCache.Dirty = true
}

var (
	implementTitleStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Padding(0, 1)

	implementSubtitleStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Padding(0, 1)

	implementHeaderBoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.Border)

	implementFooterStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Padding(0, 1)

	implementFooterBoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(theme.Border)

	implementPaneBorderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(1)

	implementSelectedPaneBorderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderActive).
		Padding(1)

	implementErrorStyle = lipgloss.NewStyle().
		Foreground(theme.Error).
		Bold(true).
		Padding(1)

	implementQuitStyle = lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true).
		Padding(1)
)
