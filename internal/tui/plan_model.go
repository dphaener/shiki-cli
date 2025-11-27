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


// PlanPaneType represents which pane is currently active in plan mode
type PlanPaneType int

const (
	PlanChatPane PlanPaneType = iota
	PlanPreviewPane
)

// Keyboard interaction constants
const (
	// doubleEscapeTimeout is the time window for detecting double ESC key presses
	planDoubleEscapeTimeout = 2 * time.Second
	// messageClearShortcut defines the keyboard combination for clearing messages
	planMessageClearShortcut = "ctrl+u"
)

// PlanModel manages the plan mode TUI
type PlanModel struct {
	session          *types.PlanSession
	orchestrator     *orchestrator.PlanOrchestrator
	eventBus         *events.EventBus
	chatView         components.ChatView
	planPreview      components.PlanPreview
	width            int
	height           int
	ready            bool
	quitting         bool
	waitingForAI     bool
	err              error
	responseBuffer   string // Buffer for accumulating assistant response
	activeUpdateChan <-chan orchestrator.MessageUpdate
	activeErrorChan  <-chan error
	activePane       PlanPaneType // Which pane is currently active
	layoutCache      LayoutCache
	// Interrupt state tracking for enhanced keyboard handling
	interruptRequested bool      // Track if user requested interruption
	lastInterruptTime  time.Time // Prevent accidental double-interrupts
}

// NewPlanModel creates a new plan model
func NewPlanModel(session *types.PlanSession, eventBus *events.EventBus) PlanModel {
	// Initialize with default sizes - will be updated on first WindowSizeMsg
	chatView := components.NewChatView(80, 24)
	planPreview := components.NewPlanPreview(80, 24)

	return PlanModel{
		session:     session,
		eventBus:    eventBus,
		chatView:    chatView,
		planPreview: planPreview,
		ready:       false,
		activePane:  PlanChatPane, // Start with chat pane active
	}
}

// Init implements tea.Model
func (m PlanModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen, // Enter alt screen FIRST to isolate from terminal
		m.chatView.Init(),
		m.planPreview.Init(),
		initializePlanAgent(m.session, m.eventBus),
	)
}

// initializePlanAgent initializes the AI agent for planning
func initializePlanAgent(session *types.PlanSession, eventBus *events.EventBus) tea.Cmd {
	return func() tea.Msg {
		apiKey := orchestrator.GetAPIKey()
		orch := orchestrator.NewPlanOrchestrator(session, apiKey, eventBus)

		if err := orch.Initialize(); err != nil {
			return PlanAgentErrorMsg{Err: err}
		}

		return PlanAgentInitializedMsg{Orchestrator: orch}
	}
}

// clearPlanInputAfterDelay returns a cmd that clears input after a brief delay
func clearPlanInputAfterDelay() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return clearPlanInputMsg{}
	})
}

// clearPlanInputMsg signals to clear the input field
type clearPlanInputMsg struct{}

// Update implements tea.Model
func (m PlanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case PlanAgentInitializedMsg:
		m.orchestrator = msg.Orchestrator
		// Handle initialization - start the planning conversation
		if len(m.session.ChatHistory) == 0 {
			return m, m.startPlanConversation()
		}
		return m, nil

	case PlanAgentErrorMsg:
		// Show error inline in chat instead of replacing entire view
		m.chatView.AddError(msg.Err)
		m.waitingForAI = false
		m.chatView.ClearLoadingState()
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		// Also store for potential full-screen error display
		m.err = msg.Err
		return m, nil

	case planAgentChannelsReady:
		// Store channels and start listening for updates
		m.activeUpdateChan = msg.updateChan
		m.activeErrorChan = msg.errorChan
		return m, waitForPlanUpdate(m.activeUpdateChan, m.activeErrorChan)

	case planAgentUpdate:
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

			// Refresh preview from file - agent may have written plan
			m.refreshPlanPreviewFromFile()

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

				// Check if this response contains plan content
				m.updatePlanPreviewFromMessage(m.responseBuffer)

				m.responseBuffer = ""
			}
			// Mark streaming as finished so next text starts a NEW message
			m.chatView.FinishStreaming()
			// Continue listening - don't stop!

		case "complete":
			// Final response - add to session history and update plan
			if len(m.responseBuffer) > 0 {
				assistantMsg := types.ChatMessage{
					Role:      "assistant",
					Content:   m.responseBuffer,
					Timestamp: time.Now(),
				}
				m.session.ChatHistory = append(m.session.ChatHistory, assistantMsg)

				// Update phase indicators based on message content
				m.updatePlanPreviewFromMessage(m.responseBuffer)

				m.responseBuffer = ""
			}
			// Mark streaming as finished
			m.chatView.FinishStreaming()
			// Refresh preview from file to show actual plan content
			m.refreshPlanPreviewFromFile()
			m.waitingForAI = false
			m.chatView.ClearLoadingState()
			m.activeUpdateChan = nil
			m.activeErrorChan = nil
			return m, nil
		}

		// Continue listening for more updates
		return m, waitForPlanUpdate(m.activeUpdateChan, m.activeErrorChan)

	case planAgentStreamComplete:
		// Stream ended - finalize response if we have one
		if len(m.responseBuffer) > 0 {
			assistantMsg := types.ChatMessage{
				Role:      "assistant",
				Content:   m.responseBuffer,
				Timestamp: time.Now(),
			}
			m.session.ChatHistory = append(m.session.ChatHistory, assistantMsg)

			// Update phase indicators based on message content
			m.updatePlanPreviewFromMessage(m.responseBuffer)

			m.responseBuffer = ""
		}
		// Mark streaming as finished
		m.chatView.FinishStreaming()
		// Refresh preview from file to show actual plan content
		m.refreshPlanPreviewFromFile()
		m.waitingForAI = false
		m.chatView.ClearLoadingState()
		m.activeUpdateChan = nil
		m.activeErrorChan = nil
		return m, nil

	case clearPlanInputMsg:
		// Clear input field to remove any late-arriving escape sequences
		m.chatView.ClearInput()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Invalidate layout cache since window size changed
		m.invalidateLayoutCache()

		// Update layout cache with new dimensions
		m.updateLayoutCache()

		// Available height for content using cached heights
		availableHeight := msg.Height - m.layoutCache.HeaderHeight - m.layoutCache.FooterHeight
		if availableHeight < 5 {
			availableHeight = 5 // Minimum
		}

		// Split width: 60% chat, 40% preview
		chatWidth := int(float64(msg.Width) * 0.6)
		previewWidth := msg.Width - chatWidth

		// Account for pane borders (2) + padding (2) = 4 total for both width and height
		m.chatView.SetSize(chatWidth-4, availableHeight-4)
		m.planPreview.SetSize(previewWidth-4, availableHeight-4)

		if !m.ready {
			m.ready = true
			// Only add welcome message if we haven't auto-started
			if len(m.session.ChatHistory) == 0 {
				m.chatView.AddWelcomeMessage(m.session.FriendlyName, true)
			}
			// Clear any escape sequences that leaked into textarea
			m.chatView.ClearInput()
			// Schedule one more clear after a brief delay to catch any late terminal responses
			cmds = append(cmds, clearPlanInputAfterDelay())
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Ctrl+C always quits
			m.quitting = true
			// Cleanup orchestrator
			if m.orchestrator != nil {
				_ = m.orchestrator.Stop()
			}
			return m, tea.Quit

		case "esc":
			// ESC interrupts agent if waiting, otherwise does nothing
			if m.waitingForAI && m.orchestrator != nil {
				now := time.Now()
				// Call interrupt and set flag
				_ = m.orchestrator.Interrupt()
				m.interruptRequested = true
				m.lastInterruptTime = now
				// Clean up waiting state
				m.waitingForAI = false
				m.chatView.ClearLoadingState()
				m.activeUpdateChan = nil
				m.activeErrorChan = nil
				// Add visual feedback
				m.chatView.AddMessage(types.ChatMessage{
					Role:      "assistant",
					Content:   "Interrupted by user",
					Timestamp: now,
				})
			}
			return m, nil

		case "tab":
			// Switch between panes
			if m.activePane == PlanChatPane {
				m.activePane = PlanPreviewPane
				// Blur chat textarea when switching away
				m.chatView.Blur()
			} else {
				m.activePane = PlanChatPane
				// Focus chat textarea when switching to it
				cmd := m.chatView.Focus()
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)

		case planMessageClearShortcut, "ctrl+shift+u":
			// Clear message input when chat pane is active (support both ctrl+u and ctrl+shift+u)
			if m.activePane == PlanChatPane {
				m.chatView.ClearInput()
			}
			return m, nil

		case "enter":
			// Only send message if chat pane is active
			if m.activePane == PlanChatPane {
				input := m.chatView.GetInput()
				if input != "" {
					return m, m.sendMessage(input)
				}
			}
		}
	}

	// Route keyboard events to the active pane
	if m.activePane == PlanChatPane {
		// Update chat view (handles viewport scrolling and textarea input)
		var chatCmd tea.Cmd
		m.chatView, chatCmd = m.chatView.Update(msg)
		cmds = append(cmds, chatCmd)
	} else {
		// Update plan preview (handles viewport scrolling)
		var previewCmd tea.Cmd
		m.planPreview, previewCmd = m.planPreview.Update(msg)
		cmds = append(cmds, previewCmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m PlanModel) View() string {
	// Only show full-screen error if not ready yet (critical init failure)
	// Once ready, errors are shown inline in the chat
	if m.err != nil && !m.ready {
		return planErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return planQuitMessageStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing..."
	}

	// Render components and cache heights
	header := m.renderHeader()
	footer := m.renderFooter()
	m.layoutCache.HeaderHeight = lipgloss.Height(header)
	m.layoutCache.FooterHeight = lipgloss.Height(footer)
	m.layoutCache.LastWidth = m.width
	m.layoutCache.LastHeight = m.height
	m.layoutCache.Dirty = false

	// Calculate available height using cached heights
	availableHeight := m.height - m.layoutCache.HeaderHeight - m.layoutCache.FooterHeight

	// Split panes: 60% chat, 40% preview
	chatWidth := int(float64(m.width) * 0.6)
	previewWidth := m.width - chatWidth

	// Render panes with proper sizing
	chatPane := m.renderPane(
		"Chat",
		m.chatView.View(),
		chatWidth,
		availableHeight,
		m.activePane == PlanChatPane, // Active if ChatPane is selected
	)

	previewPane := m.renderPane(
		"Plan Preview",
		m.planPreview.View(),
		previewWidth,
		availableHeight,
		m.activePane == PlanPreviewPane, // Active if PreviewPane is selected
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

// ViewContent returns just the content without header/footer for embedding in workflow
func (m PlanModel) ViewContent() string {
	if m.err != nil && !m.ready {
		return planErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return planQuitMessageStyle.Render("Goodbye!")
	}

	if !m.ready {
		return "Initializing..."
	}

	// Update layout cache if needed using optimized approach
	m.updateLayoutCache()

	// Calculate available height using cached heights
	availableHeight := m.height - m.layoutCache.HeaderHeight - m.layoutCache.FooterHeight

	// Split panes: 60% chat, 40% preview
	chatWidth := int(float64(m.width) * 0.6)
	previewWidth := m.width - chatWidth

	// Render panes with proper sizing
	chatPane := m.renderPane(
		"Chat",
		m.chatView.View(),
		chatWidth,
		availableHeight,
		m.activePane == PlanChatPane, // Active if ChatPane is selected
	)

	previewPane := m.renderPane(
		"Plan Preview",
		m.planPreview.View(),
		previewWidth,
		availableHeight,
		m.activePane == PlanPreviewPane, // Active if PreviewPane is selected
	)

	// Return just the content (panes) without header/footer
	return lipgloss.JoinHorizontal(lipgloss.Top, chatPane, previewPane)
}

// renderHeader renders the TUI header
func (m *PlanModel) renderHeader() string {
	title := fmt.Sprintf("Collab Plan: %s", m.session.FriendlyName)
	subtitle := fmt.Sprintf("Feature #%03d - %s - Phase: %s",
		m.session.FeatureNumber,
		m.session.SpecSlug,
		m.session.Phase,
	)

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		planTitleStyle.Render(title),
		planSubtitleStyle.Render(subtitle),
	)

	return planHeaderBoxStyle.Width(m.width).Render(headerContent)
}

// renderFooter renders the TUI footer with keyboard shortcuts
func (m *PlanModel) renderFooter() string {
	var shortcuts []string

	if m.activePane == PlanChatPane {
		shortcuts = []string{
			"Enter: Send",
			"Shift+Enter: New Line",
			"Ctrl+U: Clear",
			"Tab: Switch to Preview",
			"↑/↓/PgUp/PgDn: Scroll",
			"Esc: Interrupt",
			"Ctrl+C: Quit",
		}
	} else {
		shortcuts = []string{
			"Tab: Switch to Chat",
			"↑/↓/PgUp/PgDn: Scroll",
			"Ctrl+C: Quit",
		}
	}

	// Render footer without loading state (moved to ChatView)
	footer := planFooterStyle.Render(strings.Join(shortcuts, " • "))
	return planFooterBoxStyle.Width(m.width).Render(footer)
}

// renderPane renders a pane with border and title
func (m *PlanModel) renderPane(title, content string, width, height int, active bool) string {
	borderStyle := planPaneBorderStyle
	if active {
		borderStyle = planSelectedPaneBorderStyle
	}

	// Use MaxWidth and MaxHeight instead of fixed Width/Height for better flexibility
	paneStyle := borderStyle.
		MaxWidth(width).
		MaxHeight(height).
		Width(width - 2).  // Account for borders
		Height(height - 2) // Account for borders

	return paneStyle.Render(content)
}

// startPlanConversation starts the planning conversation
func (m *PlanModel) startPlanConversation() tea.Cmd {
	// Send an initial message to start the planning process
	initialPrompt := fmt.Sprintf("I want to create an implementation plan for the feature: %s. Please help me plan this.", m.session.FriendlyName)

	// Add user message to chat
	msg := types.ChatMessage{
		Role:      "user",
		Content:   initialPrompt,
		Timestamp: time.Now(),
	}
	m.session.ChatHistory = append(m.session.ChatHistory, msg)
	m.chatView.AddMessage(msg)
	m.waitingForAI = true
	m.chatView.SetLoadingState("plan")

	// Send to AI agent
	if m.orchestrator == nil {
		return func() tea.Msg {
			return PlanAgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	// Reset response buffer and start streaming
	m.responseBuffer = ""
	return startPlanAgentStreaming(m.orchestrator, initialPrompt)
}

// sendMessage sends a user message and triggers AI response
func (m *PlanModel) sendMessage(input string) tea.Cmd {
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
	m.chatView.SetLoadingState("plan")

	// Send to AI agent
	if m.orchestrator == nil {
		return func() tea.Msg {
			return PlanAgentErrorMsg{Err: fmt.Errorf("agent not initialized")}
		}
	}

	// Reset response buffer and start streaming
	m.responseBuffer = ""
	return startPlanAgentStreaming(m.orchestrator, input)
}

// startPlanAgentStreaming initiates the agent query and returns channels
func startPlanAgentStreaming(orch *orchestrator.PlanOrchestrator, input string) tea.Cmd {
	return func() tea.Msg {
		updateChan, errorChan := orch.SendMessage(input)
		return planAgentChannelsReady{
			updateChan: updateChan,
			errorChan:  errorChan,
		}
	}
}

// waitForPlanUpdate waits for the next message from the agent
func waitForPlanUpdate(updateChan <-chan orchestrator.MessageUpdate, errorChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case update, ok := <-updateChan:
			if !ok {
				// Channel closed
				return planAgentStreamComplete{}
			}
			return planAgentUpdate{update: update}

		case err, ok := <-errorChan:
			if ok && err != nil {
				return PlanAgentErrorMsg{Err: err}
			}
			return planAgentStreamComplete{}
		}
	}
}

// planAgentChannelsReady signals that streaming channels are ready
type planAgentChannelsReady struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

// planAgentUpdate carries a single update from the agent
type planAgentUpdate struct {
	update orchestrator.MessageUpdate
}

// planAgentStreamComplete signals that streaming is complete
type planAgentStreamComplete struct{}

// refreshPlanPreviewFromFile reads the plan file from disk and updates the preview
func (m *PlanModel) refreshPlanPreviewFromFile() {
	if m.session.PlanFile == "" {
		return
	}

	content, err := os.ReadFile(m.session.PlanFile)
	if err != nil {
		return // File may not exist yet
	}

	// Strip YAML frontmatter if present
	planContent := stripPlanFrontmatter(string(content))
	if planContent == "" {
		return
	}

	m.planPreview.SetContent(planContent)
	m.session.CurrentPlan = planContent

	// Update phase based on content
	if containsPlanSections(planContent) {
		m.session.Phase = types.PlanPhaseDesign
		m.planPreview.SetPhase(types.PlanPhaseDesign)
	}
}

// stripPlanFrontmatter removes YAML frontmatter from markdown content
func stripPlanFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2])
	}
	return content
}

// updatePlanPreviewFromMessage updates phase indicators based on message content
// Note: Actual plan content is read from file via refreshPlanPreviewFromFile()
func (m *PlanModel) updatePlanPreviewFromMessage(content string) {
	// Only update phase indicators, not content (content comes from file)
	if m.session.Phase == types.PlanPhaseInterrogation && containsPlanSections(content) {
		m.session.Phase = types.PlanPhaseDesign
		m.planPreview.SetPhase(types.PlanPhaseDesign)
	}
}

// containsPlanSections checks if content has multiple plan sections
func containsPlanSections(content string) bool {
	content = strings.ToLower(content)
	sectionCount := 0
	sections := []string{"summary", "technical context", "implementation phases", "key decisions", "testing strategy"}
	for _, section := range sections {
		if strings.Contains(content, section) {
			sectionCount++
		}
	}
	return sectionCount >= 3
}

// Message types for agent communication

// PlanAgentInitializedMsg indicates the agent is ready
type PlanAgentInitializedMsg struct {
	Orchestrator *orchestrator.PlanOrchestrator
}

// PlanAgentErrorMsg represents an error from the AI agent
type PlanAgentErrorMsg struct {
	Err error
}

// Styles for plan mode using Sekkei Design System theme
var (
	planTitleStyle = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1)

	planSubtitleStyle = lipgloss.NewStyle().
				Foreground(theme.TextMuted).
				Padding(0, 1)

	planHeaderBoxStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(theme.Border)

	planFooterStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1)

	planFooterBoxStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderTop(true).
				BorderForeground(theme.Border)

	planPaneBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(theme.Border).
				Padding(1)

	planSelectedPaneBorderStyle = lipgloss.NewStyle().
					BorderStyle(lipgloss.RoundedBorder()).
					BorderForeground(theme.BorderActive).
					Padding(1)

	planErrorStyle = lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true).
			Padding(1)

	planQuitMessageStyle = lipgloss.NewStyle().
				Foreground(theme.Success).
				Bold(true).
				Padding(1)
)

// updateLayoutCache updates the cached header and footer heights if needed
func (m *PlanModel) updateLayoutCache() {
	if m.layoutCache.Dirty || m.width != m.layoutCache.LastWidth || m.height != m.layoutCache.LastHeight {
		m.layoutCache.HeaderHeight = lipgloss.Height(m.renderHeader())
		m.layoutCache.FooterHeight = lipgloss.Height(m.renderFooter())
		m.layoutCache.LastWidth = m.width
		m.layoutCache.LastHeight = m.height
		m.layoutCache.Dirty = false
	}
}

// invalidateLayoutCache marks the layout cache as dirty, requiring recalculation
func (m *PlanModel) invalidateLayoutCache() {
	m.layoutCache.Dirty = true
}
