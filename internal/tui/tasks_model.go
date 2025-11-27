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

// TasksPaneType represents which pane is currently active in tasks mode
type TasksPaneType int

const (
	TasksChatPane TasksPaneType = iota
	TasksPreviewPane
)

// Keyboard interaction constants
const (
	// doubleEscapeTimeout is the time window for detecting double ESC key presses
	tasksDoubleEscapeTimeout = 2 * time.Second
	// messageClearShortcut defines the keyboard combination for clearing messages
	tasksMessageClearShortcut = "ctrl+u"
)

// TasksModel manages the tasks generation TUI
type TasksModel struct {
	session          *types.WorkflowSession
	orchestrator     *orchestrator.TasksOrchestrator
	eventBus         *events.EventBus
	chatView         components.ChatView
	tasksPreview     components.TasksPreview
	width            int
	height           int
	ready            bool
	quitting         bool
	waitingForAI     bool
	err              error
	responseBuffer   string
	activeUpdateChan <-chan orchestrator.MessageUpdate
	activeErrorChan  <-chan error
	activePane       TasksPaneType
	phaseComplete    bool
	layoutCache      LayoutCache
	// Interrupt state tracking for enhanced keyboard handling
	interruptRequested bool      // Track if user requested interruption
	lastInterruptTime  time.Time // Prevent accidental double-interrupts
	lastEscTime        time.Time // Track last ESC press for double-ESC clear
}

// NewTasksModel creates a new tasks model
func NewTasksModel(session *types.WorkflowSession, eventBus *events.EventBus) TasksModel {
	chatView := components.NewChatView(80, 24)
	tasksPreview := components.NewTasksPreview(80, 24)

	return TasksModel{
		session:      session,
		eventBus:     eventBus,
		chatView:     chatView,
		tasksPreview: tasksPreview,
		ready:        false,
		activePane:   TasksChatPane,
	}
}

// Init implements tea.Model
func (m TasksModel) Init() tea.Cmd {
	return tea.Batch(
		m.chatView.Init(),
		m.tasksPreview.Init(),
		initializeTasksAgent(m.session, m.eventBus),
	)
}

// initializeTasksAgent initializes the AI agent for task generation
func initializeTasksAgent(session *types.WorkflowSession, eventBus *events.EventBus) tea.Cmd {
	return func() tea.Msg {
		apiKey := orchestrator.GetAPIKey()
		orch := orchestrator.NewTasksOrchestrator(session, apiKey, eventBus)

		if err := orch.Initialize(); err != nil {
			return TasksAgentErrorMsg{Err: err}
		}

		return TasksAgentInitializedMsg{Orchestrator: orch}
	}
}

// Update implements tea.Model
func (m TasksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TasksAgentInitializedMsg:
		m.orchestrator = msg.Orchestrator
		if len(m.session.TasksChatHistory) == 0 {
			return m, m.startTasksGeneration()
		}
		return m, nil

	case TasksAgentErrorMsg:
		m.chatView.AddError(msg.Err)
		m.waitingForAI = false
		m.chatView.ClearLoadingState()
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		m.err = msg.Err
		return m, nil

	case tasksAgentChannelsReady:
		m.activeUpdateChan = msg.updateChan
		m.activeErrorChan = msg.errorChan
		return m, waitForTasksUpdate(m.activeUpdateChan, m.activeErrorChan)

	case tasksAgentUpdate:
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
			m.refreshTasksPreviewFromFile()

		case "tool_error":
			m.chatView.AddError(fmt.Errorf("Tool failed: %s", msg.update.Content))

		case "message_done":
			if len(m.responseBuffer) > 0 {
				assistantMsg := types.ChatMessage{
					Role:      "assistant",
					Content:   m.responseBuffer,
					Timestamp: time.Now(),
				}
				m.session.TasksChatHistory = append(m.session.TasksChatHistory, assistantMsg)
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
				m.session.TasksChatHistory = append(m.session.TasksChatHistory, assistantMsg)
				m.responseBuffer = ""
			}
			m.chatView.FinishStreaming()
			m.refreshTasksPreviewFromFile()
			m.waitingForAI = false
			m.chatView.ClearLoadingState()
			m.activeUpdateChan = nil
			m.activeErrorChan = nil

			// Check if tasks file was created
			if m.tasksFileExists() {
				m.phaseComplete = true
			}
			return m, nil
		}

		return m, waitForTasksUpdate(m.activeUpdateChan, m.activeErrorChan)

	case tasksAgentStreamComplete:
		if len(m.responseBuffer) > 0 {
			assistantMsg := types.ChatMessage{
				Role:      "assistant",
				Content:   m.responseBuffer,
				Timestamp: time.Now(),
			}
			m.session.TasksChatHistory = append(m.session.TasksChatHistory, assistantMsg)
			m.responseBuffer = ""
		}
		m.chatView.FinishStreaming()
		m.refreshTasksPreviewFromFile()
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
		m.tasksPreview.SetSize(previewWidth-4, availableHeight-4)

		if !m.ready {
			m.ready = true
			if len(m.session.TasksChatHistory) == 0 {
				m.chatView.AddWelcomeMessage("Tasks Generation", true)
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
				if m.activePane == TasksChatPane {
					cmd := m.chatView.Focus()
					cmds = append(cmds, cmd)
				}

				// Call interrupt asynchronously to avoid blocking UI
				go func() {
					_ = m.orchestrator.Interrupt()
				}()
			} else if m.activePane == TasksChatPane {
				// Double-ESC clears input when not waiting for AI
				if now.Sub(m.lastEscTime) < tasksDoubleEscapeTimeout {
					m.chatView.ClearInput()
					m.lastEscTime = time.Time{} // Reset to prevent triple-ESC
				} else {
					m.lastEscTime = now
				}
			}
			return m, tea.Batch(cmds...)

		case tea.KeyTab:
			if m.activePane == TasksChatPane {
				m.activePane = TasksPreviewPane
				m.chatView.Blur()
			} else {
				m.activePane = TasksChatPane
				cmd := m.chatView.Focus()
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case tea.KeyEnter:
			// Handle Enter key for sending messages
			if m.activePane == TasksChatPane {
				input := m.chatView.GetInput()
				if input != "" {
					// Only send message if there's actual content
					// Multi-line input is handled by the textarea itself when Shift+Enter is pressed
					return m, m.sendMessage(input)
				}
			}
		}
	}

	if m.activePane == TasksChatPane {
		var chatCmd tea.Cmd
		m.chatView, chatCmd = m.chatView.Update(msg)
		cmds = append(cmds, chatCmd)
	} else {
		var previewCmd tea.Cmd
		m.tasksPreview, previewCmd = m.tasksPreview.Update(msg)
		cmds = append(cmds, previewCmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m TasksModel) View() string {
	if m.err != nil && !m.ready {
		return tasksErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return tasksQuitStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing tasks generation..."
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
		m.activePane == TasksChatPane,
	)

	previewPane := m.renderPane(
		"Tasks Preview",
		m.tasksPreview.View(),
		previewWidth,
		availableHeight,
		m.activePane == TasksPreviewPane,
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
func (m TasksModel) ViewContent() string {
	if m.err != nil && !m.ready {
		return tasksErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return tasksQuitStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing tasks generation..."
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
		m.activePane == TasksChatPane,
	)

	previewPane := m.renderPane(
		"Tasks Preview",
		m.tasksPreview.View(),
		previewWidth,
		availableHeight,
		m.activePane == TasksPreviewPane,
	)

	// Return just the content (panes) without header/footer
	return lipgloss.JoinHorizontal(lipgloss.Top, chatPane, previewPane)
}

func (m *TasksModel) renderHeader() string {
	title := fmt.Sprintf("Collab Tasks: %s", m.session.FriendlyName)
	subtitle := fmt.Sprintf("Feature #%03d - %s - Phase: Tasks Generation",
		m.session.FeatureNumber,
		m.session.Slug,
	)

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		tasksTitleStyle.Render(title),
		tasksSubtitleStyle.Render(subtitle),
	)

	return tasksHeaderBoxStyle.Width(m.width).Render(headerContent)
}

func (m *TasksModel) renderFooter() string {
	var shortcuts []string

	if m.activePane == TasksChatPane {
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
	footer := tasksFooterStyle.Render(strings.Join(shortcuts, " • "))
	return tasksFooterBoxStyle.Width(m.width).Render(footer)
}

func (m *TasksModel) renderPane(title, content string, width, height int, active bool) string {
	borderStyle := tasksPaneBorderStyle
	if active {
		borderStyle = tasksSelectedPaneBorderStyle
	}

	paneStyle := borderStyle.
		MaxWidth(width).
		MaxHeight(height).
		Width(width - 2).
		Height(height - 2)

	return paneStyle.Render(content)
}

func (m *TasksModel) startTasksGeneration() tea.Cmd {
	initialPrompt := fmt.Sprintf("Please generate implementation tasks for the feature: %s. The spec is at %s and the plan is at %s.",
		m.session.FriendlyName, m.session.SpecFile, m.session.PlanFile)

	msg := types.ChatMessage{
		Role:      "user",
		Content:   initialPrompt,
		Timestamp: time.Now(),
	}
	m.session.TasksChatHistory = append(m.session.TasksChatHistory, msg)
	m.chatView.AddMessage(msg)
	m.waitingForAI = true
	m.chatView.SetLoadingState("tasks")

	if m.orchestrator == nil {
		return func() tea.Msg {
			return TasksAgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	m.responseBuffer = ""
	return startTasksAgentStreaming(m.orchestrator, initialPrompt)
}

func (m *TasksModel) sendMessage(input string) tea.Cmd {
	msg := types.ChatMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	}
	m.session.TasksChatHistory = append(m.session.TasksChatHistory, msg)
	m.chatView.AddMessage(msg)
	m.chatView.ClearInput()
	m.waitingForAI = true
	m.chatView.SetLoadingState("tasks")

	if m.orchestrator == nil {
		return func() tea.Msg {
			return TasksAgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	m.responseBuffer = ""
	return startTasksAgentStreaming(m.orchestrator, input)
}

func (m *TasksModel) refreshTasksPreviewFromFile() {
	if m.session.TasksFile == "" {
		return
	}

	content, err := os.ReadFile(m.session.TasksFile)
	if err != nil {
		return
	}

	m.tasksPreview.SetContent(string(content))
	m.session.CurrentTasks = string(content)
}

func (m *TasksModel) tasksFileExists() bool {
	if m.session.TasksFile == "" {
		return false
	}
	_, err := os.Stat(m.session.TasksFile)
	return err == nil
}

// IsPhaseComplete returns whether tasks generation is complete
func (m *TasksModel) IsPhaseComplete() bool {
	return m.phaseComplete
}

// Streaming functions

func startTasksAgentStreaming(orch *orchestrator.TasksOrchestrator, input string) tea.Cmd {
	return func() tea.Msg {
		updateChan, errorChan := orch.SendMessage(input)
		return tasksAgentChannelsReady{
			updateChan: updateChan,
			errorChan:  errorChan,
		}
	}
}

func waitForTasksUpdate(updateChan <-chan orchestrator.MessageUpdate, errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case update, ok := <-updateChan:
			if !ok {
				return tasksAgentStreamComplete{}
			}
			return tasksAgentUpdate{update: update}
		case err, ok := <-errorChan:
			if ok && err != nil {
				return TasksAgentErrorMsg{Err: err}
			}
			return tasksAgentStreamComplete{}
		}
	}
}

// Message types

type TasksAgentInitializedMsg struct {
	Orchestrator *orchestrator.TasksOrchestrator
}

type TasksAgentErrorMsg struct {
	Err error
}

type tasksAgentChannelsReady struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

type tasksAgentUpdate struct {
	update orchestrator.MessageUpdate
}

type tasksAgentStreamComplete struct{}

// Styles
// updateLayoutCache updates the cached header and footer heights if needed
func (m *TasksModel) updateLayoutCache() {
	if m.layoutCache.Dirty || m.width != m.layoutCache.LastWidth || m.height != m.layoutCache.LastHeight {
		m.layoutCache.HeaderHeight = lipgloss.Height(m.renderHeader())
		m.layoutCache.FooterHeight = lipgloss.Height(m.renderFooter())
		m.layoutCache.LastWidth = m.width
		m.layoutCache.LastHeight = m.height
		m.layoutCache.Dirty = false
	}
}

// invalidateLayoutCache marks the layout cache as dirty, requiring recalculation
func (m *TasksModel) invalidateLayoutCache() {
	m.layoutCache.Dirty = true
}

var (
	tasksTitleStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Padding(0, 1)

	tasksSubtitleStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Padding(0, 1)

	tasksHeaderBoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.Border)

	tasksFooterStyle = lipgloss.NewStyle().
		Foreground(theme.TextMuted).
		Padding(0, 1)

	tasksFooterBoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(theme.Border)

	tasksPaneBorderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(1)

	tasksSelectedPaneBorderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderActive).
		Padding(1)

	tasksErrorStyle = lipgloss.NewStyle().
		Foreground(theme.Error).
		Bold(true).
		Padding(1)

	tasksQuitStyle = lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true).
		Padding(1)
)
