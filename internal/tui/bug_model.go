package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/internal/storage"
	"github.com/darinhaener/collab/internal/tui/components"
	"github.com/darinhaener/collab/pkg/types"
)

// BugModel manages the bug fix workflow TUI.
// It wraps PhaseModel with bug-specific configuration and phase transitions.
type BugModel struct {
	*PhaseModel
	session    *types.BugSession
	bugPreview *components.BugPreview
}

// bugPreviewWrapper wraps BugPreview to implement PreviewUpdater
type bugPreviewWrapper struct {
	preview *components.BugPreview
}

func (w *bugPreviewWrapper) Init() tea.Cmd {
	return w.preview.Init()
}

func (w *bugPreviewWrapper) View() string {
	return w.preview.View()
}

func (w *bugPreviewWrapper) SetSize(width, height int) {
	w.preview.SetSize(width, height)
}

func (w *bugPreviewWrapper) SetContent(content string) {
	w.preview.SetContent(content)
}

func (w *bugPreviewWrapper) UpdatePreview(msg tea.Msg) tea.Cmd {
	updated, cmd := w.preview.Update(msg)
	*w.preview = updated
	return cmd
}

// NewBugModel creates a new bug workflow model
func NewBugModel(session *types.BugSession, eventBus *events.EventBus) BugModel {
	bugPreview := components.NewBugPreview(80, 24)
	bugPreview.SetPhase(session.CurrentPhase)
	bugPreview.SetFiles(session.PlanFile, session.TasksFile)
	previewWrapper := &bugPreviewWrapper{preview: &bugPreview}

	// Create the orchestrator
	apiKey := orchestrator.GetAPIKey()
	orch := orchestrator.NewBugOrchestrator(session, apiKey, eventBus)

	// Create the model first (we need it for closure references)
	model := &BugModel{
		session:    session,
		bugPreview: &bugPreview,
	}

	// Create the phase model config
	config := PhaseModelConfig{
		PhaseType:    PhaseBug,
		Title:        fmt.Sprintf("Collab Bug Fix: %s", session.Title),
		PreviewTitle: "Bug Fix Preview",
		SubtitleFunc: func() string {
			return fmt.Sprintf("Bug #%s - Phase: %s",
				session.ID[:8],
				components.FormatBugPhase(session.CurrentPhase),
			)
		},
		InitialPromptFunc: func() string {
			return model.getPhasePrompt()
		},
		RefreshPreviewFunc: func() {
			model.refreshBugPreview()
		},
		OnToolUseFunc: func(toolName string, args map[string]interface{}) {
			// Refresh preview after tool use - agent may have written files
			model.refreshBugPreview()
		},
		GetChatHistoryFunc: func() []types.ChatMessage {
			return model.getCurrentChatHistory()
		},
		AppendChatHistoryFunc: func(msg types.ChatMessage) {
			model.appendToChatHistory(msg)
		},
		LoadingStateLabel: "bug",
	}

	// Create the PhaseModel
	model.PhaseModel = NewPhaseModel(config, orch, previewWrapper, eventBus)

	return *model
}

// Init implements tea.Model
func (m BugModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.PhaseModel.Init(),
		initializeBugAgent(m.session, m.PhaseModel.eventBus),
	)
}

// initializeBugAgent initializes the AI agent for bug fixing
func initializeBugAgent(session *types.BugSession, eventBus *events.EventBus) tea.Cmd {
	return func() tea.Msg {
		apiKey := orchestrator.GetAPIKey()
		orch := orchestrator.NewBugOrchestrator(session, apiKey, eventBus)

		if err := orch.Initialize(); err != nil {
			return BugAgentErrorMsg{Err: err}
		}

		return BugAgentInitializedMsg{Orchestrator: orch}
	}
}

// Update implements tea.Model
func (m BugModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle bug-specific messages first
	switch msg := msg.(type) {
	case BugAgentInitializedMsg:
		// Replace the orchestrator in PhaseModel
		m.PhaseModel.orchestrator = msg.Orchestrator
		// Set phase-specific instructions
		msg.Orchestrator.SetPlanInstructions(m.getPhaseInstructions())
		// Signal that agent is ready
		return m, func() tea.Msg { return PhaseAgentReadyMsg{} }

	case BugAgentErrorMsg:
		return m, func() tea.Msg { return PhaseAgentErrorMsg{Err: msg.Err} }

	case tea.KeyMsg:
		// Handle bug-specific key bindings for phase transitions
		switch msg.String() {
		case "ctrl+a":
			// Approve current phase and move to next
			if m.isApprovalPhase() {
				return m.approveAndContinue()
			}
		case "ctrl+b":
			// Go back to previous phase
			if m.session.CurrentPhase.CanGoBack(m.session.CurrentPhase) {
				return m.goToPreviousPhase()
			}
		}
	}

	// Delegate to PhaseModel
	updatedModel, cmd := m.PhaseModel.Update(msg)
	m.PhaseModel = updatedModel.(*PhaseModel)

	return m, cmd
}

// View implements tea.Model
func (m BugModel) View() string {
	return m.PhaseModel.View()
}

// ViewContent returns just the content without header/footer for embedding in workflow
func (m BugModel) ViewContent() string {
	return m.PhaseModel.ViewContent()
}

// refreshBugPreview updates the bug preview from files
func (m *BugModel) refreshBugPreview() {
	m.bugPreview.SetPhase(m.session.CurrentPhase)
	m.bugPreview.RefreshFromFiles()
}

// getCurrentChatHistory returns the chat history for the current phase
func (m *BugModel) getCurrentChatHistory() []types.ChatMessage {
	switch m.session.CurrentPhase {
	case types.BugPhasePlan, types.BugPhaseApprovePlan:
		return m.session.PlanChatHistory
	case types.BugPhaseTasks, types.BugPhaseApproveTasks:
		return m.session.TasksChatHistory
	case types.BugPhaseImplement:
		return m.session.ImplementChatHistory
	default:
		return m.session.PlanChatHistory
	}
}

// appendToChatHistory appends a message to the current phase's chat history
func (m *BugModel) appendToChatHistory(msg types.ChatMessage) {
	switch m.session.CurrentPhase {
	case types.BugPhasePlan, types.BugPhaseApprovePlan:
		m.session.PlanChatHistory = append(m.session.PlanChatHistory, msg)
	case types.BugPhaseTasks, types.BugPhaseApproveTasks:
		m.session.TasksChatHistory = append(m.session.TasksChatHistory, msg)
	case types.BugPhaseImplement:
		m.session.ImplementChatHistory = append(m.session.ImplementChatHistory, msg)
	default:
		m.session.PlanChatHistory = append(m.session.PlanChatHistory, msg)
	}
}

// getPhasePrompt returns the initial prompt for the current phase
func (m *BugModel) getPhasePrompt() string {
	switch m.session.CurrentPhase {
	case types.BugPhasePlan:
		// Ask the agent to ask questions - don't give it context to work with
		return fmt.Sprintf("I need help fixing a bug called '%s'. Please ask me questions to understand the problem before investigating.", m.session.Title)

	case types.BugPhaseApprovePlan:
		return "Please review the bug fix plan. Type 'approve' to continue or provide feedback for changes."

	case types.BugPhaseTasks:
		return "Please break down the bug fix plan into specific implementation tasks."

	case types.BugPhaseApproveTasks:
		return "Please review the task breakdown. Type 'approve' to continue or provide feedback for changes."

	case types.BugPhaseImplement:
		return "Please implement the bug fix following the task breakdown."

	default:
		return fmt.Sprintf("Starting bug fix for: %s", m.session.Title)
	}
}

// getPhaseInstructions returns detailed instructions for the current phase
func (m *BugModel) getPhaseInstructions() string {
	switch m.session.CurrentPhase {
	case types.BugPhasePlan:
		return fmt.Sprintf(`You are helping to create a comprehensive plan for fixing this bug:

**Bug**: %s
**Description**: %s

Please analyze the problem, investigate the codebase if needed, and create a detailed plan that includes:
1. Root cause analysis
2. Solution approach
3. Implementation steps
4. Testing strategy
5. Risk assessment

Save your plan to: %s`, m.session.Title, m.session.Description, m.session.PlanFile)

	case types.BugPhaseTasks:
		return fmt.Sprintf(`You are helping to break down the bug fix plan into actionable tasks.

Please read the plan from %s and create a detailed task breakdown that includes:
1. Numbered, actionable implementation tasks
2. Dependencies and prerequisites
3. Specific implementation guidance
4. Acceptance criteria for each task
5. Testing and validation steps

Save your task breakdown to: %s`, m.session.PlanFile, m.session.TasksFile)

	case types.BugPhaseImplement:
		return fmt.Sprintf(`You are helping to implement the bug fix based on the planned tasks.

Please read the tasks from %s and help execute them systematically:
1. Implement each task step by step
2. Run tests and validate changes
3. Ensure the bug is completely resolved
4. Document any important decisions or changes`, m.session.TasksFile)

	default:
		return ""
	}
}

// isApprovalPhase returns true if current phase is an approval phase
func (m *BugModel) isApprovalPhase() bool {
	return m.session.CurrentPhase == types.BugPhaseApprovePlan ||
		m.session.CurrentPhase == types.BugPhaseApproveTasks
}

// approveAndContinue approves the current phase and moves to the next
func (m BugModel) approveAndContinue() (BugModel, tea.Cmd) {
	nextPhase := m.session.CurrentPhase.GetNextPhase(m.session.CurrentPhase)
	m.session.CurrentPhase = nextPhase
	m.session.UpdatedAt = time.Now()

	// Save session state
	storage.SaveBugSession(m.session)

	// Update preview for new phase
	m.bugPreview.SetPhase(nextPhase)
	m.bugPreview.RefreshFromFiles()

	// Update orchestrator with new phase instructions
	if orch, ok := m.PhaseModel.orchestrator.(*orchestrator.BugOrchestrator); ok {
		orch.UpdateSession(m.session)
		orch.SetPlanInstructions(m.getPhaseInstructions())
	}

	if nextPhase == types.BugPhaseComplete {
		m.PhaseModel.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// goToPreviousPhase goes back to the previous phase
func (m BugModel) goToPreviousPhase() (BugModel, tea.Cmd) {
	prevPhase := m.session.CurrentPhase.GetPreviousPhase(m.session.CurrentPhase)
	m.session.CurrentPhase = prevPhase
	m.session.UpdatedAt = time.Now()

	// Save session state
	storage.SaveBugSession(m.session)

	// Update preview for previous phase
	m.bugPreview.SetPhase(prevPhase)
	m.bugPreview.RefreshFromFiles()

	// Update orchestrator with previous phase instructions
	if orch, ok := m.PhaseModel.orchestrator.(*orchestrator.BugOrchestrator); ok {
		orch.UpdateSession(m.session)
		orch.SetPlanInstructions(m.getPhaseInstructions())
	}

	return m, nil
}

// Message types for agent communication

// BugAgentInitializedMsg indicates the agent is ready
type BugAgentInitializedMsg struct {
	Orchestrator *orchestrator.BugOrchestrator
}

// BugAgentErrorMsg represents an error from the AI agent
type BugAgentErrorMsg struct {
	Err error
}

// Legacy message types for backward compatibility
type bugAgentInitializedMsg struct {
	orchestrator *orchestrator.BugOrchestrator
}

type bugAgentErrorMsg struct {
	err error
}

type startBugPhaseMsg struct {
	phase types.BugPhase
}

type bugStreamUpdateMsg struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
	update     orchestrator.MessageUpdate
}

type bugStreamCompleteMsg struct{}

// Keyboard interaction constants (kept for any external references)
const (
	bugDoubleEscapeTimeout  = 2 * time.Second
	bugMessageClearShortcut = "ctrl+u"
)

// BugPaneType represents which pane is currently active in bug mode
type BugPaneType = PaneType

const (
	BugChatPane    BugPaneType = ChatPane
	BugPreviewPane BugPaneType = PreviewPane
)
