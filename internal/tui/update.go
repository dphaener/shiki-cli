package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dphaener/shiki-cli/internal/broker"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/tui/components"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// Update handles messages and updates model state (Bubbletea Update function)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case eventMsg:
		return m.handleEvent(msg.event)

	case brokerEventMsg:
		return m.handleBrokerEvent(msg.event)

	case fileContentMsg:
		m.fileContent = msg.content
		if msg.filename != "" {
			m.currentFile = msg.filename
		}
		// Only continue listening if session is still active
		if m.session.Status == types.SessionRunning {
			return m, waitForEvent(m.eventSub)
		}
		return m, nil

	case deliverableContentMsg:
		if msg.err != nil {
			// Handle error - fallback to showing error message
			m.deliverableContent = "Error loading deliverable: " + msg.err.Error()
			m.deliverableRenderedContent = m.deliverableContent
		} else {
			m.deliverableContent = msg.content
			// Start with raw content, glamour will render async
			m.deliverableRenderedContent = msg.content
		}
		m.deliverablePath = msg.path
		// Switch to completion view
		m.viewMode = ViewModeCompletion
		// Trigger async glamour rendering
		return m, doRenderDeliverable(m.deliverableContent, m.width)

	case deliverableRenderedMsg:
		// Cache the glamour-rendered content
		m.deliverableRenderedContent = msg.content
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	default:
		return m, nil
	}
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		// Quit TUI
		return m, tea.Quit

	case "ctrl+c":
		// Pause session (if running)
		m.paused = true
		return m, tea.Quit

	case "h":
		// Toggle view mode or navigate agent panes
		if m.viewMode == ViewModeCompletion {
			// In completion view, 'h' switches back to agent history
			m.viewMode = ViewModeAgents
		} else {
			// In agent view, 'h' (or 'left') navigates to previous agent pane
			if m.selectedAgent > 0 {
				m.selectedAgent--
			}
		}

	case "up", "k":
		// Scroll up
		if m.viewMode == ViewModeCompletion {
			// Scroll up in completion view
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		} else {
			// Scroll up in selected agent pane
			if m.selectedAgent >= 0 && m.selectedAgent < len(m.activeAgents) {
				agentID := m.activeAgents[m.selectedAgent]
				if agentState, exists := m.agentOutputs[agentID]; exists {
					// Disable autoscroll when user manually scrolls up
					agentState.AutoScroll = false
					if agentState.ScrollOffset > 0 {
						agentState.ScrollOffset--
					}
				}
			}
		}

	case "down", "j":
		// Scroll down
		if m.viewMode == ViewModeCompletion {
			// Scroll down in completion view
			m.scrollOffset++
		} else {
			// Scroll down in selected agent pane
			if m.selectedAgent >= 0 && m.selectedAgent < len(m.activeAgents) {
				agentID := m.activeAgents[m.selectedAgent]
				if agentState, exists := m.agentOutputs[agentID]; exists {
					agentState.ScrollOffset++
					// Note: AutoScroll will be re-enabled automatically in the
					// render function if we've scrolled to the bottom
				}
			}
		}

	case "left":
		// Switch to previous agent pane (only in agent view)
		if m.viewMode == ViewModeAgents && m.selectedAgent > 0 {
			m.selectedAgent--
		}

	case "right", "l":
		// Switch to next agent pane (only in agent view)
		if m.viewMode == ViewModeAgents && m.selectedAgent < len(m.activeAgents)-1 {
			m.selectedAgent++
		}

	case "tab":
		// Cycle through files
		if len(m.fileList) > 0 {
			m.selectedFile = (m.selectedFile + 1) % len(m.fileList)
			m.currentFile = m.fileList[m.selectedFile]
			m.scrollOffset = 0
			return m, loadFileContent(m.workspaceDir, m.currentFile)
		}

	case "shift+tab":
		// Cycle through files (reverse)
		if len(m.fileList) > 0 {
			m.selectedFile--
			if m.selectedFile < 0 {
				m.selectedFile = len(m.fileList) - 1
			}
			m.currentFile = m.fileList[m.selectedFile]
			m.scrollOffset = 0
			return m, loadFileContent(m.workspaceDir, m.currentFile)
		}

	case "pgup":
		// Page up in file viewer
		if m.selectedPane == "file" {
			m.scrollOffset = max(0, m.scrollOffset-10)
		}

	case "pgdown":
		// Page down in file viewer
		if m.selectedPane == "file" {
			m.scrollOffset += 10
		}

	case "home":
		// Go to top
		if m.selectedPane == "turns" {
			m.selectedTurn = 0
			m.scrollOffset = 0
		} else {
			m.scrollOffset = 0
		}

	case "end":
		// Go to bottom
		if m.selectedPane == "turns" && len(m.turnHistory) > 0 {
			m.selectedTurn = len(m.turnHistory) - 1
			m.scrollOffset = len(m.turnHistory) - 1
		}
	}

	return m, nil
}

// handleEvent processes EventBus events
func (m Model) handleEvent(event events.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case types.EventTurnStarted:
		payload := event.Payload.(events.TurnStartedPayload)
		// Add in-progress turn to history
		m.turnHistory = append(m.turnHistory, *payload.Turn)
		m.selectedTurn = len(m.turnHistory) - 1
		m.session.CurrentTurn = payload.Turn.Number

		// Update agent state
		if agentState, exists := m.agentOutputs[payload.AgentID]; exists {
			agentState.Status = types.TurnInProgress
			agentState.CurrentTurn = payload.Turn.Number
		}

	case types.EventTurnCompleted:
		payload := event.Payload.(events.TurnCompletedPayload)
		// Update turn in history
		for i := range m.turnHistory {
			if m.turnHistory[i].Number == payload.Turn.Number {
				m.turnHistory[i] = *payload.Turn
				break
			}
		}
		// Update session totals
		m.session.TotalCost += payload.Turn.Cost
		m.session.TotalTokens += payload.Turn.TokensUsed

		// Update agent state
		if agentState, exists := m.agentOutputs[payload.Turn.AgentID]; exists {
			agentState.Status = types.TurnCompleted
			agentState.TotalCost += payload.Turn.Cost
			agentState.TotalTokens += payload.Turn.TokensUsed
		}

	case types.EventTurnError:
		payload := event.Payload.(events.TurnErrorPayload)
		// Update turn with error
		for i := range m.turnHistory {
			if m.turnHistory[i].Number == payload.Turn.Number {
				m.turnHistory[i] = *payload.Turn
				break
			}
		}

		// Update agent state
		if agentState, exists := m.agentOutputs[payload.Turn.AgentID]; exists {
			agentState.Status = types.TurnError
		}

	case types.EventFileUpdated:
		payload := event.Payload.(events.FileUpdatedPayload)
		// Reload file if it's currently displayed
		if m.currentFile == payload.Path {
			return m, loadFileContent(m.workspaceDir, m.currentFile)
		}
		// Refresh file list in case new files were created
		m.fileList = getWorkspaceFiles(m.workspaceDir)

	case types.EventToolInvoked:
		payload := event.Payload.(events.ToolInvokedPayload)
		// Add tool invocation to activity log (keep last 50)
		activity := ToolActivity{
			ToolName:  payload.ToolName,
			AgentID:   payload.AgentID,
			Timestamp: event.Timestamp,
		}
		m.toolActivity = append(m.toolActivity, activity)
		if len(m.toolActivity) > 50 {
			m.toolActivity = m.toolActivity[len(m.toolActivity)-50:]
		}

		// Add to agent output pane with rich tool information
		if agentState, exists := m.agentOutputs[payload.AgentID]; exists {
			entry := components.OutputEntry{
				Type:      components.OutputTypeToolUse,
				Content:   payload.ToolName,
				Timestamp: event.Timestamp.Format("15:04:05"),
				Turn:      agentState.CurrentTurn,
				Args:      payload.Args, // Pass tool arguments for rich display
			}
			agentState.Outputs = append(agentState.Outputs, entry)
			// AutoScroll will automatically show new content when enabled
		}

	case types.EventAssistantMessage:
		payload := event.Payload.(events.AssistantMessagePayload)
		// Add assistant message to agent output pane
		if agentState, exists := m.agentOutputs[payload.AgentID]; exists {
			entry := components.OutputEntry{
				Type:      components.OutputTypeMessage,
				Content:   payload.Content,
				Timestamp: event.Timestamp.Format("15:04:05"),
			}
			agentState.Outputs = append(agentState.Outputs, entry)
			// AutoScroll will automatically show new content when enabled
		}

	case types.EventSessionCompleted:
		payload := event.Payload.(events.SessionCompletedPayload)
		m.session.Status = types.SessionCompleted
		m.session.CompletedAt = &time.Time{}
		*m.session.CompletedAt = time.Now()
		m.session.DeliverablePath = payload.DeliverablePath

		// Load deliverable content and switch to completion view
		// Don't wait for more events - session is complete
		return m, loadDeliverableContent(payload.DeliverablePath)

	case types.EventSessionPaused:
		m.session.Status = types.SessionPaused
		m.paused = true
		// Don't wait for more events when paused
		return m, nil

	case types.EventSessionResumed:
		// Reset all viewport components to ensure clean state
		// This fixes the scroll bug that occurs after resumption

		// Reset each agent's output state
		for _, agentState := range m.agentOutputs {
			if agentState != nil {
				// Reset auto-scroll and scroll offset
				agentState.ResetScrollState()
			}
		}

		// Reset file view scroll state
		m.scrollOffset = 0

		// Reset selected positions to latest turn
		if len(m.turnHistory) > 0 {
			m.selectedTurn = len(m.turnHistory) - 1
		}

		// Update session status
		m.session.Status = types.SessionRunning
		m.paused = false

		// Force re-render of all viewport content
		return m, tea.Batch(
			loadFileContent(m.workspaceDir, m.currentFile),
			waitForEvent(m.eventSub),
		)

	case types.EventSessionError:
		payload := event.Payload.(events.SessionErrorPayload)
		m.session.Status = types.SessionError
		m.err = &pauseError{msg: payload.Error}
		// Don't wait for more events on error
		return m, nil
	}

	// Continue listening for events (only if session is still active)
	if m.session.Status == types.SessionRunning {
		return m, waitForEvent(m.eventSub)
	}
	return m, nil
}

// handleBrokerEvent processes typed broker events for tool chain management
func (m Model) handleBrokerEvent(event broker.Event) (tea.Model, tea.Cmd) {
	switch event := event.(type) {
	case broker.ToolStartedEvent:
		// Process tool started event with tool chain manager
		if m.toolChainManager != nil {
			chain := m.toolChainManager.ProcessToolStarted(event)
			if chain != nil {
				// Update conversation message parts if chain has replaced tools
				m.updateMessagePartChainInfo(event.MessageID, event.ToolCallID, chain)
			}
		}

	case broker.ToolCompletedEvent:
		// Process tool completed event with tool chain manager
		if m.toolChainManager != nil {
			chain := m.toolChainManager.ProcessToolCompleted(event)
			if chain != nil {
				// Update tool call status in conversation message
				m.updateMessagePartStatus(event.MessageID, event.ToolCallID, event.IsError)
			}
		}

	case broker.TurnCompletedEvent:
		// Clean up tool chains at turn boundaries
		if m.toolChainManager != nil {
			m.toolChainManager.ProcessTurnCompleted(event.Turn)
		}

	default:
		// Ignore other broker events for now
	}

	// Continue listening for broker events if subscription is active
	var cmd tea.Cmd
	if m.brokerSub != nil && m.session.Status == types.SessionRunning {
		cmd = waitForBrokerEvent(m.brokerSub)
	}

	return m, cmd
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// pauseError is a sentinel error for paused sessions
type pauseError struct {
	msg string
}

func (e *pauseError) Error() string {
	return e.msg
}

// updateMessagePartChainInfo updates tool chain information for a tool call in the conversation
func (m *Model) updateMessagePartChainInfo(messageID, toolCallID string, chain *components.ToolChain) {
	// Find the participant's message and update the tool call's chain info
	for _, agentState := range m.agentOutputs {
		// Note: This is a simplified approach. In a full implementation,
		// we'd need to properly integrate with the conversation.Message system
		// For now, we ensure that tool chain state is maintained in the ToolChainManager
		_ = agentState // unused for now
	}

	// The tool chain manager maintains the authoritative state
	// UI components will query it during rendering
}

// updateMessagePartStatus updates the status of a tool call in the conversation
func (m *Model) updateMessagePartStatus(messageID, toolCallID string, isError bool) {
	// Similar to above - the ToolChainManager maintains the state
	// and the rendering system will query it for display

	// In a more complete implementation, we would also update any
	// conversation.Message instances stored in the model
}
