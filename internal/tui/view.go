package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/tui/components"
)

// View renders the complete TUI (Bubbletea View function)
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing TUI..."
	}

	// Route to appropriate view based on view mode
	switch m.viewMode {
	case ViewModeCompletion:
		return m.renderCompletionView()
	case ViewModeAgents:
		fallthrough
	default:
		return m.renderAgentView()
	}
}

// renderAgentView renders the agent conversation view (original view)
func (m Model) renderAgentView() string {
	// Render header (1 line + padding)
	header := components.RenderHeader(m.session, m.width)

	// Calculate pane dimensions
	// Layout: header (2-3 lines) + split panes (remaining) + status bar (1 line)
	headerHeight := lipgloss.Height(header)
	statusBarHeight := 1
	paneHeight := m.height - headerHeight - statusBarHeight

	// Render agent output panes horizontally
	numAgents := len(m.activeAgents)
	if numAgents == 0 {
		numAgents = 1 // Prevent division by zero
	}
	agentPaneWidth := m.width / numAgents

	var agentPanes []string
	for i, agentID := range m.activeAgents {
		if agentState, exists := m.agentOutputs[agentID]; exists {
			isActive := i == m.selectedAgent
			pane := components.RenderAgentOutput(agentState, agentPaneWidth, paneHeight, isActive)
			agentPanes = append(agentPanes, pane)
		}
	}

	// Join agent panes horizontally
	var panes string
	if len(agentPanes) > 0 {
		panes = lipgloss.JoinHorizontal(lipgloss.Top, agentPanes...)
	} else {
		// Fallback if no agents
		panes = "No active agents"
	}

	// Get most recent tool activity
	recentTool := ""
	if len(m.toolActivity) > 0 {
		last := m.toolActivity[len(m.toolActivity)-1]
		recentTool = fmt.Sprintf("%s: %s", last.AgentID, last.ToolName)
	}

	// Render status bar
	statusBar := components.RenderStatusBar(
		len(m.activeAgents),
		m.selectedAgent,
		m.width,
		recentTool,
	)

	// Join all sections vertically
	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		panes,
		statusBar,
	)

	return view
}

// renderCompletionView renders the completion view showing the deliverable
func (m Model) renderCompletionView() string {
	// Create completion view state
	state := components.CompletionViewState{
		Session:            m.session,
		DeliverableContent: m.deliverableContent,
		DeliverablePath:    m.deliverablePath,
		ScrollOffset:       m.scrollOffset,
	}

	// Render the completion view
	return components.RenderCompletionView(state, m.width, m.height)
}
