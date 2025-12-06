package tui

import (
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/orchestrator"
	"github.com/dphaener/shiki-cli/internal/storage"
	"github.com/dphaener/shiki-cli/internal/tui/components"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// BugModel manages the bug fix workflow TUI.
// It wraps PhaseModel with bug-specific configuration and phase transitions.
type BugModel struct {
	*PhaseModel
	session         *types.BugSession
	bugPreview      *components.BugPreview
	progressStepper components.ProgressStepper
	approvalBar     components.ApprovalBar
	approvalView    *components.ApprovalView
	width           int
	height          int
	ready           bool
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

func (w *bugPreviewWrapper) GetPhase() types.PreviewPhase {
	return w.preview.GetPhase()
}

func (w *bugPreviewWrapper) SetPhase(phase types.PreviewPhase) {
	w.preview.SetPhase(phase)
}

func (w *bugPreviewWrapper) UpdatePreview(msg tea.Msg) tea.Cmd {
	updated, cmd := w.preview.Update(msg)
	*w.preview = updated
	return cmd
}

// convertBugStatusToWorkflowStatus converts BugPhaseStatus to WorkflowPhaseStatus
func convertBugStatusToWorkflowStatus(bugStatus types.BugPhaseStatus) types.WorkflowPhaseStatus {
	switch bugStatus {
	case types.BugPhaseStatusPending:
		return types.PhaseStatusPending
	case types.BugPhaseStatusCurrent:
		return types.PhaseStatusCurrent
	case types.BugPhaseStatusComplete:
		return types.PhaseStatusComplete
	default:
		return types.PhaseStatusPending
	}
}

// getBugDisplayPhases returns display phases for bug workflow with proper status
// Maps 5 bug phases to 3 display phases: Plan+ApprovePlan→"Plan", Tasks+ApproveTasks→"Tasks", Implement→"Implement"
func getBugDisplayPhases(currentPhase types.BugPhase) []components.DisplayPhase {
	var bugPhase types.BugPhase
	displayPhases := bugPhase.GetDisplayPhases()
	result := make([]components.DisplayPhase, len(displayPhases))

	for i, phase := range displayPhases {
		bugStatus := bugPhase.GetPhaseStatus(phase, currentPhase)
		result[i] = components.DisplayPhase{
			Name:   bugPhase.GetPhaseName(phase),
			Status: convertBugStatusToWorkflowStatus(bugStatus),
		}
	}

	return result
}

// convertBugPhaseToWorkflowPhase converts a bug phase to equivalent workflow phase for ApprovalBar compatibility
func convertBugPhaseToWorkflowPhase(bugPhase types.BugPhase) types.WorkflowPhase {
	switch bugPhase {
	case types.BugPhasePlan:
		return types.WorkflowPhasePlan
	case types.BugPhaseApprovePlan:
		return types.WorkflowPhaseApprovePlan
	case types.BugPhaseTasks:
		return types.WorkflowPhaseTasks
	case types.BugPhaseApproveTasks:
		return types.WorkflowPhaseApproveTasks
	case types.BugPhaseImplement:
		return types.WorkflowPhaseImplement
	case types.BugPhaseComplete:
		return types.WorkflowPhaseComplete
	default:
		return types.WorkflowPhasePlan
	}
}

// NewBugModel creates a new bug workflow model
func NewBugModel(session *types.BugSession, eventBus *events.EventBus) BugModel {
	bugPreview := components.NewBugPreview(80, 24)
	bugPreview.SetPhase(session.CurrentPhase)

	// Construct task progress file path
	taskProgressFile := filepath.Join(session.BugDir, "task-progress.md")
	bugPreview.SetFiles(session.PlanFile, session.TasksFile, taskProgressFile)
	previewWrapper := &bugPreviewWrapper{preview: &bugPreview}

	// Create progress stepper
	progressStepper := components.NewProgressStepper(getBugDisplayPhases(session.CurrentPhase))

	// Create approval bar (convert bug phase to workflow phase for compatibility)
	approvalBar := components.NewApprovalBar(convertBugPhaseToWorkflowPhase(session.CurrentPhase))

	// Create the orchestrator
	apiKey := orchestrator.GetAPIKey()
	orch := orchestrator.NewBugOrchestrator(session, apiKey, eventBus)

	// Create the model first (we need it for closure references)
	model := &BugModel{
		session:         session,
		bugPreview:      &bugPreview,
		progressStepper: progressStepper,
		approvalBar:     approvalBar,
		width:           80,  // default
		height:          24, // default
		ready:           false,
	}

	// Create the phase model config
	config := PhaseModelConfig{
		PhaseType:          PhaseBug,
		Title:              fmt.Sprintf("Collab Bug Fix: %s", session.Title),
		PreviewTitle:       "Bug Fix Preview",
		OutputTemplateName: model.getOutputTemplateName(),
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

	// Initialize the phase (including approval view if needed)
	// Note: initializeBugPhase() returns a command, but in constructor we don't execute it
	// The command will be executed during the first Init() call
	model.initializeBugPhase()

	return *model
}

// initializeBugPhase creates the appropriate view for the current bug phase
func (m *BugModel) initializeBugPhase() tea.Cmd {
	// Clear approval view when not in approval phase
	m.approvalView = nil

	// Clear chat view state when transitioning phases to ensure fresh context
	if m.PhaseModel != nil {
		m.PhaseModel.chatView.ResetScrollState()
		// Clear messages for fresh phase start (emulate feature workflow behavior)
		m.PhaseModel.chatView.SetMessages([]types.ChatMessage{})
	}

	switch m.session.CurrentPhase {
	case types.BugPhasePlan:
		// Initialize agent for plan phase
		return m.reinitializeAgent()
	case types.BugPhaseApprovePlan:
		// Show approval view for plan
		m.approvalView = components.NewApprovalView(convertBugPhaseToWorkflowPhase(m.session.CurrentPhase), m.session.PlanFile)
		return nil
	case types.BugPhaseTasks:
		// Initialize agent for tasks phase
		return m.reinitializeAgent()
	case types.BugPhaseApproveTasks:
		// Show approval view for tasks
		m.approvalView = components.NewApprovalView(convertBugPhaseToWorkflowPhase(m.session.CurrentPhase), m.session.TasksFile)
		return nil
	case types.BugPhaseImplement:
		// Initialize agent for implement phase
		return m.reinitializeAgent()
	default:
		return nil
	}
}

// reinitializeAgent creates a new agent instance for the current working phase
func (m *BugModel) reinitializeAgent() tea.Cmd {
	if m.PhaseModel == nil {
		return nil
	}

	// Clear chat history when entering tasks phase to ensure fresh context
	if m.session != nil && m.session.CurrentPhase == types.BugPhaseTasks {
		m.session.TasksChatHistory = []types.ChatMessage{}
	}

	// Return a command that will reinitialize the agent
	return func() tea.Msg {
		// Create new orchestrator for this phase
		apiKey := orchestrator.GetAPIKey()
		orch := orchestrator.NewBugOrchestrator(m.session, apiKey, m.PhaseModel.eventBus)

		// Set phase-specific instructions
		instructions := m.getPhaseInstructions()
		if instructions != "" {
			orch.SetPlanInstructions(instructions)
		}

		// Initialize the agent
		if err := orch.Initialize(); err != nil {
			return BugAgentErrorMsg{Err: err}
		}

		return BugAgentInitializedMsg{Orchestrator: orch}
	}
}

// Init implements tea.Model
func (m BugModel) Init() tea.Cmd {
	// Load template content FIRST
	m.loadBugTemplateContent()

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
	case bugPhaseChangedMsg:
		// Update session state
		m.session.CurrentPhase = msg.to
		m.session.UpdatedAt = time.Now()

		// Load new template for the phase
		m.loadBugTemplateContent()

		// Initialize the new phase (including approval view if needed)
		initCmd := m.initializeBugPhase()

		// Save bug session state
		_ = storage.SaveBugSession(m.session)

		// Update UI components
		m.progressStepper.SetPhases(getBugDisplayPhases(m.session.CurrentPhase))
		m.approvalBar.SetPhase(convertBugPhaseToWorkflowPhase(m.session.CurrentPhase))

		// Update approval view size if it was just created
		if m.approvalView != nil && m.width > 0 {
			headerHeight := 4
			m.approvalView.SetSize(m.width, m.height-headerHeight)
		}

		// Check if workflow is complete
		if m.session.CurrentPhase == types.BugPhaseComplete {
			m.PhaseModel.quitting = true
			return m, tea.Quit
		}

		// Return the initialization command if there is one
		return m, initCmd

	case BugAgentInitializedMsg:
		// Replace the orchestrator in PhaseModel
		m.PhaseModel.orchestrator = msg.Orchestrator
		// Set phase-specific instructions
		msg.Orchestrator.SetPlanInstructions(m.getPhaseInstructions())

		// Inject the phase prompt to start the conversation
		prompt := m.getPhasePrompt()
		if prompt != "" {
			// Send the initial prompt using PhaseModel's system
			return m, tea.Sequence(
				func() tea.Msg { return PhaseAgentReadyMsg{} },
				m.PhaseModel.sendMessage(prompt),
			)
		}

		// Signal that agent is ready
		return m, func() tea.Msg { return PhaseAgentReadyMsg{} }

	case BugAgentErrorMsg:
		return m, func() tea.Msg { return PhaseAgentErrorMsg{Err: msg.Err} }

	case events.Event:
		// Handle file update events for task-progress.md
		if msg.Type == types.EventFileUpdated {
			if payload, ok := msg.Payload.(events.FileUpdatedPayload); ok {
				return m.handleTaskProgressFileUpdate(payload)
			}
		}

	case TaskProgressRefreshMsg:
		// Task progress file was updated - refresh the preview
		m.refreshBugPreview()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update progress stepper size
		m.progressStepper.SetWidth(msg.Width)

		// Update approval bar size
		m.approvalBar.SetWidth(msg.Width)

		// Update approval view size if active
		if m.approvalView != nil {
			headerHeight := 4 // approximate header height
			m.approvalView.SetSize(msg.Width, msg.Height-headerHeight)
		}

		if !m.ready {
			m.ready = true
		}

	case tea.KeyMsg:
		// Handle approval phase shortcuts (matching feature workflow)
		if m.isApprovalPhase() {
			switch msg.String() {
			case "y", "Y":
				// Approve and advance
				return m.approveAndContinue()
			case "b", "B":
				// Go back to previous phase
				if m.session.CurrentPhase.CanGoBack(m.session.CurrentPhase) {
					return m.goToPreviousPhase()
				}
			case "e", "E":
				// Continue editing - go back to the working phase
				return m.goBackToWorkingPhase()
			case "up", "k", "K":
				// Scroll up in approval view
				if m.approvalView != nil {
					m.approvalView.ScrollUp(3)
				}
				return m, nil
			case "down", "j", "J":
				// Scroll down in approval view
				if m.approvalView != nil {
					m.approvalView.ScrollDown(3)
				}
				return m, nil
			case "pgup":
				// Page up in approval view
				if m.approvalView != nil {
					m.approvalView.ScrollUp(10)
				}
				return m, nil
			case "pgdown":
				// Page down in approval view
				if m.approvalView != nil {
					m.approvalView.ScrollDown(10)
				}
				return m, nil
			}
		}

		// Global shortcuts (matching feature workflow)
		switch msg.String() {
		case "ctrl+c":
			// Quit application (handled by PhaseModel, but ensure consistency)
			// Let it fall through to PhaseModel delegation
		case "ctrl+d", "ctrl+D":
			// Mark current working phase as done and advance to approval
			if m.isWorkingPhase() {
				return m.advanceToApproval()
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
	if !m.ready {
		return "Initializing bug workflow..."
	}

	var parts []string

	// Header with progress stepper
	header := m.renderHeader()
	parts = append(parts, header)

	// Main content - either approval view or phase model
	if m.approvalView != nil {
		// In approval phase - show full-screen approval view
		headerHeight := lipgloss.Height(header)
		m.approvalView.SetSize(m.width, m.height-headerHeight)
		parts = append(parts, m.approvalView.View())
	} else {
		// In working phase - show phase model content
		// Calculate available height for content
		headerHeight := lipgloss.Height(header)
		footerHeight := lipgloss.Height(m.renderFooter())
		contentHeight := m.height - headerHeight - footerHeight

		// Get view from phase model content
		content := m.renderPhaseContent(contentHeight)
		parts = append(parts, content)

		// Footer with approval bar (only in working phases)
		footer := m.renderFooter()
		parts = append(parts, footer)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// ViewContent returns just the content without header/footer for embedding in workflow
func (m BugModel) ViewContent() string {
	return m.PhaseModel.ViewContent()
}

// renderHeader renders the bug workflow header with progress stepper
func (m *BugModel) renderHeader() string {
	title := fmt.Sprintf("Collab Bug Fix: %s", m.session.Title)

	// Update stepper with current phase
	m.progressStepper.SetPhases(getBugDisplayPhases(m.session.CurrentPhase))

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		bugTitleStyle.Render(title),
		m.progressStepper.View(),
	)

	return bugHeaderStyle.Width(m.width).Render(headerContent)
}

// renderFooter renders the bug workflow footer with approval bar
func (m *BugModel) renderFooter() string {
	m.approvalBar.SetPhase(convertBugPhaseToWorkflowPhase(m.session.CurrentPhase))
	return bugFooterStyle.Width(m.width).Render(m.approvalBar.View())
}

// renderPhaseContent renders the content from the active phase model
func (m *BugModel) renderPhaseContent(contentHeight int) string {
	if m.PhaseModel == nil {
		return "No active phase"
	}

	// Get view from phase model
	content := m.PhaseModel.ViewContent()

	// Style the content area
	return bugContentStyle.
		Width(m.width).
		Height(contentHeight).
		Render(content)
}

// refreshBugPreview updates the bug preview from files
func (m *BugModel) refreshBugPreview() {
	planFile := m.session.PlanFile
	tasksFile := m.session.TasksFile
	bugProgressFile := m.session.BugProgressFile

	m.bugPreview.SetFiles(planFile, tasksFile, bugProgressFile)
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

// getOutputTemplateName returns the template name for the current phase
func (m *BugModel) getOutputTemplateName() string {
	switch m.session.CurrentPhase {
	case types.BugPhasePlan:
		return "plan"
	case types.BugPhaseTasks:
		return "tasks"
	case types.BugPhaseImplement:
		return "task-progress"
	default:
		return ""
	}
}

// loadBugTemplateContent loads the appropriate template for the current phase
func (m *BugModel) loadBugTemplateContent() {
	switch m.session.CurrentPhase {
	case types.BugPhasePlan:
		m.loadBugPlanTemplateContent()
	case types.BugPhaseTasks:
		m.loadBugTasksTemplateContent()
	case types.BugPhaseImplement:
		m.loadBugImplementTemplateContent()
	}
}

// loadBugPlanTemplateContent loads the plan template and sets it to preview
func (m *BugModel) loadBugPlanTemplateContent() {
	// Update the config template name and reload
	m.PhaseModel.config.OutputTemplateName = "plan"
	if templateContent := m.PhaseModel.loadOutputTemplate(); templateContent != "" {
		m.bugPreview.SetContent(templateContent)
	}
}

// loadBugTasksTemplateContent loads the tasks template and sets it to preview
func (m *BugModel) loadBugTasksTemplateContent() {
	// Update the config template name and reload
	m.PhaseModel.config.OutputTemplateName = "tasks"
	if templateContent := m.PhaseModel.loadOutputTemplate(); templateContent != "" {
		m.bugPreview.SetContent(templateContent)
	}
}

// loadBugImplementTemplateContent loads the task-progress template and sets it to preview
func (m *BugModel) loadBugImplementTemplateContent() {
	// Update the config template name and reload
	m.PhaseModel.config.OutputTemplateName = "task-progress"
	if templateContent := m.PhaseModel.loadOutputTemplate(); templateContent != "" {
		m.bugPreview.SetContent(templateContent)
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

// isWorkingPhase returns true if current phase is a working phase (not approval)
func (m *BugModel) isWorkingPhase() bool {
	return !m.isApprovalPhase() && m.session.CurrentPhase != types.BugPhaseComplete
}

// advanceToApproval advances from a working phase to its approval phase
func (m BugModel) advanceToApproval() (BugModel, tea.Cmd) {
	prevPhase := m.session.CurrentPhase
	var targetPhase types.BugPhase

	switch m.session.CurrentPhase {
	case types.BugPhasePlan:
		targetPhase = types.BugPhaseApprovePlan
	case types.BugPhaseTasks:
		targetPhase = types.BugPhaseApproveTasks
	case types.BugPhaseImplement:
		// Implement phase goes directly to complete
		targetPhase = types.BugPhaseComplete
	default:
		// Not in a working phase
		return m, nil
	}

	return m, func() tea.Msg {
		return bugPhaseChangedMsg{
			from: prevPhase,
			to:   targetPhase,
		}
	}
}

// goBackToWorkingPhase goes back to the working phase (from approval)
func (m BugModel) goBackToWorkingPhase() (BugModel, tea.Cmd) {
	prevPhase := m.session.CurrentPhase
	var targetPhase types.BugPhase

	switch m.session.CurrentPhase {
	case types.BugPhaseApprovePlan:
		targetPhase = types.BugPhasePlan
	case types.BugPhaseApproveTasks:
		targetPhase = types.BugPhaseTasks
	default:
		return m, nil // Not in an approval phase
	}

	return m, func() tea.Msg {
		return bugPhaseChangedMsg{
			from: prevPhase,
			to:   targetPhase,
		}
	}
}

// approveAndContinue approves the current phase and moves to the next
func (m BugModel) approveAndContinue() (BugModel, tea.Cmd) {
	prevPhase := m.session.CurrentPhase
	nextPhase := m.session.CurrentPhase.GetNextPhase(m.session.CurrentPhase)

	return m, func() tea.Msg {
		return bugPhaseChangedMsg{
			from:      prevPhase,
			to:        nextPhase,
			saveCheck: true, // Save checkpoint when approving
		}
	}
}

// goToPreviousPhase goes back to the previous phase
func (m BugModel) goToPreviousPhase() (BugModel, tea.Cmd) {
	currentPhase := m.session.CurrentPhase
	targetPhase := m.session.CurrentPhase.GetPreviousPhase(m.session.CurrentPhase)

	return m, func() tea.Msg {
		return bugPhaseChangedMsg{
			from: currentPhase,
			to:   targetPhase,
		}
	}
}

// handleTaskProgressFileUpdate handles file update events for task-progress.md
func (m BugModel) handleTaskProgressFileUpdate(payload events.FileUpdatedPayload) (tea.Model, tea.Cmd) {
	// Check if the updated file is the task-progress.md file for this session
	if m.session.BugProgressFile == "" {
		return m, nil
	}

	// Normalize paths for comparison
	updateFile := filepath.Clean(payload.Path)
	expectedFile := filepath.Clean(m.session.BugProgressFile)

	if updateFile == expectedFile {
		// The task progress file was updated, refresh the preview
		m.refreshBugPreview()

		return m, func() tea.Msg {
			return TaskProgressRefreshMsg{Operation: payload.Operation}
		}
	}

	return m, nil
}

// Message types for phase transitions

// bugPhaseChangedMsg represents a bug workflow phase transition
type bugPhaseChangedMsg struct {
	from      types.BugPhase
	to        types.BugPhase
	saveCheck bool // whether to save a checkpoint for the 'from' phase
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

// Styles for bug workflow - matching WorkflowModel styles for consistency
var (
	bugTitleStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Padding(0, 1)

	bugHeaderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.Border)

	bugFooterStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(theme.Border)

	bugContentStyle = lipgloss.NewStyle()
)
