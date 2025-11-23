package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// Update handles messages and updates model state (Bubbletea Update function)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case eventMsg:
		return m.handleEvent(msg.event)

	case fileContentMsg:
		m.fileContent = msg.content
		if msg.filename != "" {
			m.currentFile = msg.filename
		}
		return m, waitForEvent(m.eventSub)

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

	case "up", "k":
		// Navigate up in active pane
		if m.selectedPane == "turns" && m.selectedTurn > 0 {
			m.selectedTurn--
			m.scrollOffset = max(0, m.scrollOffset-1)
		} else if m.selectedPane == "file" && m.scrollOffset > 0 {
			m.scrollOffset--
		}

	case "down", "j":
		// Navigate down in active pane
		if m.selectedPane == "turns" && m.selectedTurn < len(m.turnHistory)-1 {
			m.selectedTurn++
			m.scrollOffset = min(m.scrollOffset+1, len(m.turnHistory)-1)
		} else if m.selectedPane == "file" {
			m.scrollOffset++
		}

	case "left", "h":
		// Switch to turns pane
		m.selectedPane = "turns"

	case "right", "l":
		// Switch to file pane
		m.selectedPane = "file"

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

	case types.EventTurnError:
		payload := event.Payload.(events.TurnErrorPayload)
		// Update turn with error
		for i := range m.turnHistory {
			if m.turnHistory[i].Number == payload.Turn.Number {
				m.turnHistory[i] = *payload.Turn
				break
			}
		}

	case types.EventFileUpdated:
		payload := event.Payload.(events.FileUpdatedPayload)
		// Reload file if it's currently displayed
		if m.currentFile == payload.Path {
			return m, loadFileContent(m.workspaceDir, m.currentFile)
		}
		// Refresh file list in case new files were created
		m.fileList = getWorkspaceFiles(m.workspaceDir)

	case types.EventSessionCompleted:
		payload := event.Payload.(events.SessionCompletedPayload)
		m.session.Status = types.SessionCompleted
		m.session.CompletedAt = &time.Time{}
		*m.session.CompletedAt = time.Now()
		m.session.DeliverablePath = payload.DeliverablePath

	case types.EventSessionPaused:
		m.session.Status = types.SessionPaused
		m.paused = true

	case types.EventSessionError:
		payload := event.Payload.(events.SessionErrorPayload)
		m.session.Status = types.SessionError
		m.err = &pauseError{msg: payload.Error}
	}

	// Continue listening for events
	return m, waitForEvent(m.eventSub)
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
