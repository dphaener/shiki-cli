package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/storage"
	"github.com/darinhaener/collab/internal/tui/components"
	"github.com/darinhaener/collab/internal/tui/theme"
	"github.com/darinhaener/collab/pkg/types"
)


// WorkflowModel manages the unified feature workflow TUI
type WorkflowModel struct {
	session  *types.WorkflowSession
	eventBus *events.EventBus

	// Embedded phase models
	specifyModel   *SpecifyModel
	planModel      *PlanModel
	tasksModel     *TasksModel
	implementModel *ImplementModel

	// UI components
	progressStepper components.ProgressStepper
	approvalBar     components.ApprovalBar
	confirmModal    *components.ConfirmModal
	approvalView    *components.ApprovalView

	// Layout state
	width        int
	height       int
	ready        bool
	quitting     bool
	err          error
	showingModal bool
	waitingForAI bool
	layoutCache  LayoutCache

	// Current active model for delegation
	activePhaseModel tea.Model
}

// NewWorkflowModel creates a new workflow model
func NewWorkflowModel(session *types.WorkflowSession, eventBus *events.EventBus) WorkflowModel {
	// Create progress stepper
	stepper := components.NewProgressStepper(types.GetDisplayPhases(), session.CurrentPhase)

	// Create approval bar
	approvalBar := components.NewApprovalBar(session.CurrentPhase)

	model := WorkflowModel{
		session:         session,
		eventBus:        eventBus,
		progressStepper: stepper,
		approvalBar:     approvalBar,
		ready:           false,
	}

	// Initialize the appropriate phase model based on current phase
	model.initializePhaseModel()

	return model
}

// initializePhaseModel creates the model for the current phase
func (m *WorkflowModel) initializePhaseModel() {
	// Clear approval view when not in approval phase
	m.approvalView = nil

	switch m.session.CurrentPhase {
	case types.WorkflowPhaseSpecify:
		// Create specify session from workflow session
		specifySession := m.createSpecifySession()
		specModel := NewSpecifyModel(specifySession, m.eventBus)
		m.specifyModel = &specModel
		m.activePhaseModel = &specModel

	case types.WorkflowPhaseApproveSpec:
		// Show approval view for spec
		m.approvalView = components.NewApprovalView(m.session.CurrentPhase, m.session.SpecFile)
		m.activePhaseModel = nil

	case types.WorkflowPhasePlan:
		// Create plan session from workflow session
		planSession := m.createPlanSession()
		planModel := NewPlanModel(planSession, m.eventBus)
		m.planModel = &planModel
		m.activePhaseModel = &planModel

	case types.WorkflowPhaseApprovePlan:
		// Show approval view for plan
		m.approvalView = components.NewApprovalView(m.session.CurrentPhase, m.session.PlanFile)
		m.activePhaseModel = nil

	case types.WorkflowPhaseTasks:
		// Create tasks model
		tasksModel := NewTasksModel(m.session, m.eventBus)
		m.tasksModel = &tasksModel
		m.activePhaseModel = &tasksModel

	case types.WorkflowPhaseApproveTasks:
		// Show approval view for tasks
		m.approvalView = components.NewApprovalView(m.session.CurrentPhase, m.session.TasksFile)
		m.activePhaseModel = nil

	case types.WorkflowPhaseImplement:
		// Create implement model
		implementModel := NewImplementModel(m.session, m.eventBus)
		m.implementModel = &implementModel
		m.activePhaseModel = &implementModel
	}
}

// createSpecifySession creates a SpecifySession from WorkflowSession
func (m *WorkflowModel) createSpecifySession() *types.SpecifySession {
	var chatHistory []types.ChatMessage
	if m.session.SpecifySession != nil {
		chatHistory = m.session.SpecifySession.ChatHistory
	}

	return &types.SpecifySession{
		ID:                     m.session.ID + "-specify",
		FeatureDesc:            m.session.FeatureDesc,
		FeatureNumber:          m.session.FeatureNumber,
		Slug:                   m.session.Slug,
		FriendlyName:           m.session.FriendlyName,
		Phase:                  types.PhaseDiscovery,
		ChatHistory:            chatHistory,
		CurrentSpec:            m.session.CurrentSpec,
		CreatedAt:              m.session.CreatedAt,
		UpdatedAt:              time.Now(),
		Status:                 types.SessionRunning,
		SkipDiscoveryQuestions: m.session.FeatureDesc != "",
		SpecFile:               m.session.SpecFile,
		SpecDir:                m.session.FeatureDir,
		ChecklistDir:           m.session.ChecklistDir,
	}
}

// createPlanSession creates a PlanSession from WorkflowSession
func (m *WorkflowModel) createPlanSession() *types.PlanSession {
	var chatHistory []types.ChatMessage
	if m.session.PlanSession != nil {
		chatHistory = m.session.PlanSession.ChatHistory
	}

	return &types.PlanSession{
		ID:            m.session.ID + "-plan",
		SpecSlug:      m.session.Slug,
		FeatureNumber: m.session.FeatureNumber,
		FriendlyName:  m.session.FriendlyName,
		Phase:         types.PlanPhaseInterrogation,
		ChatHistory:   chatHistory,
		CurrentPlan:   m.session.CurrentPlan,
		CreatedAt:     m.session.CreatedAt,
		UpdatedAt:     time.Now(),
		Status:        types.SessionRunning,
		SpecFile:      m.session.SpecFile,
		SpecDir:       m.session.FeatureDir,
		PlanFile:      m.session.PlanFile,
		ContractsDir:  m.session.ContractsDir,
	}
}

// Init implements tea.Model
func (m WorkflowModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.EnterAltScreen,
	}

	// Initialize the active phase model
	if m.activePhaseModel != nil {
		cmds = append(cmds, m.activePhaseModel.Init())
	}

	return tea.Batch(cmds...)
}

// Update implements tea.Model
func (m WorkflowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case workflowPhaseChangedMsg:
		// Save checkpoint if requested (for approval transitions)
		if msg.saveCheck {
			var chatHistory []types.ChatMessage
			var artifactPath string

			switch msg.from {
			case types.WorkflowPhaseApproveSpec:
				if m.session.SpecifySession != nil {
					chatHistory = m.session.SpecifySession.ChatHistory
				}
				artifactPath = m.session.SpecFile
			case types.WorkflowPhaseApprovePlan:
				if m.session.PlanSession != nil {
					chatHistory = m.session.PlanSession.ChatHistory
				}
				artifactPath = m.session.PlanFile
			case types.WorkflowPhaseApproveTasks:
				chatHistory = m.session.TasksChatHistory
				artifactPath = m.session.TasksFile
			}

			_ = storage.SaveCheckpoint(m.session, msg.from, chatHistory, artifactPath)
		}

		// Update session state
		m.session.CurrentPhase = msg.to
		m.session.UpdatedAt = time.Now()

		// Initialize the new phase model
		m.initializePhaseModel()

		// Save workflow state
		_ = storage.SaveWorkflowSession(m.session)

		// Update UI components
		m.progressStepper.SetCurrentPhase(m.session.CurrentPhase)
		m.approvalBar.SetPhase(m.session.CurrentPhase)

		// Update approval view size if it was just created
		if m.approvalView != nil && m.width > 0 {
			headerHeight := 4
			m.approvalView.SetSize(m.width, m.height-headerHeight)
		}

		// Check if workflow is complete
		if m.session.CurrentPhase == types.WorkflowPhaseComplete {
			m.quitting = true
			return m, tea.Quit
		}

		// Initialize the new phase model and send it a window size message
		var cmds []tea.Cmd
		if m.activePhaseModel != nil {
			// Call Init on the new model
			cmds = append(cmds, m.activePhaseModel.Init())
			// Send it the current window size
			cmds = append(cmds, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			})
		}

		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Invalidate layout cache since window size changed
		m.invalidateLayoutCache()

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
		// Handle modal first if showing
		if m.showingModal && m.confirmModal != nil {
			switch msg.String() {
			case "enter", "y", "Y":
				// Confirm go back
				m.showingModal = false
				m.confirmModal = nil
				return m, m.goBackToPhase()
			case "esc", "n", "N":
				// Cancel
				m.showingModal = false
				m.confirmModal = nil
				return m, nil
			}
			return m, nil
		}

		// Handle approval phase shortcuts
		if types.IsApprovalPhase(m.session.CurrentPhase) {
			switch msg.String() {
			case "y", "Y":
				// Approve and advance
				return m, m.approveAndAdvance()
			case "b", "B":
				// Show go-back confirmation
				if types.CanGoBack(m.session.CurrentPhase) {
					m.showingModal = true
					m.confirmModal = components.NewConfirmModal(
						"Go Back?",
						"This will preserve current artifacts as drafts.\nYou'll return to the previous phase to make changes.",
						[]string{"Cancel", "Go Back"},
					)
				}
				return m, nil
			case "e", "E":
				// Continue editing - go back to the working phase
				return m, m.goBackToWorkingPhase()
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

		// Global shortcuts
		switch msg.String() {
		case "ctrl+c":
			// Only Ctrl+C quits the application
			// In approval phase, don't quit immediately - prompt
			if types.IsApprovalPhase(m.session.CurrentPhase) {
				// Save state before quitting
				_ = storage.SaveWorkflowSession(m.session)
			}
			m.quitting = true
			return m, tea.Quit
		case "ctrl+d", "ctrl+D":
			// Mark current working phase as done and advance to approval
			if m.isWorkingPhase() {
				return m, m.advanceToApproval()
			}
		}
	}

	// Delegate to active phase model
	if m.activePhaseModel != nil {
		var cmd tea.Cmd
		m.activePhaseModel, cmd = m.activePhaseModel.Update(msg)
		cmds = append(cmds, cmd)

		// Sync waiting state with the active model
		m.syncWaitingState()

		// Check for phase completion signals from embedded models
		if m.checkPhaseCompletion() {
			cmds = append(cmds, m.transitionToApproval())
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m WorkflowModel) View() string {
	if m.err != nil && !m.ready {
		return workflowErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.quitting {
		return workflowQuitStyle.Render("Workflow paused. Resume with: collab feature resume " + m.session.Slug)
	}

	if !m.ready {
		return "Initializing workflow..."
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
	} else if m.activePhaseModel != nil {
		// In working phase - show phase model content
		// First render footer to cache its height
		footer := m.renderFooter()
		m.layoutCache.FooterHeight = lipgloss.Height(footer)
		m.layoutCache.HeaderHeight = lipgloss.Height(header)
		m.layoutCache.LastWidth = m.width
		m.layoutCache.LastHeight = m.height
		m.layoutCache.Dirty = false

		content := m.renderPhaseContent()
		parts = append(parts, content)

		// Footer with approval bar (only in working phases)
		parts = append(parts, footer)
	}

	view := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Overlay modal if showing
	if m.showingModal && m.confirmModal != nil {
		view = m.overlayModal(view)
	}

	return view
}

// renderHeader renders the workflow header with progress stepper
func (m *WorkflowModel) renderHeader() string {
	title := fmt.Sprintf("Collab Feature: %s", m.session.FriendlyName)

	// Update stepper with current phase
	m.progressStepper.SetCurrentPhase(m.session.CurrentPhase)

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		workflowTitleStyle.Render(title),
		m.progressStepper.View(),
	)

	return workflowHeaderStyle.Width(m.width).Render(headerContent)
}

// renderFooter renders the workflow footer with approval bar
func (m *WorkflowModel) renderFooter() string {
	m.approvalBar.SetPhase(m.session.CurrentPhase)
	m.approvalBar.SetWaitingForAI(m.waitingForAI)
	return workflowFooterStyle.Width(m.width).Render(m.approvalBar.View())
}

// renderPhaseContent renders the content from the active phase model
func (m *WorkflowModel) renderPhaseContent() string {
	if m.activePhaseModel == nil {
		return "No active phase"
	}

	// Calculate available height for content using cached heights
	// Heights are cached in View() method before calling renderPhaseContent()
	contentHeight := m.height - m.layoutCache.HeaderHeight - m.layoutCache.FooterHeight

	// Get view from active model - use ViewContent if available to avoid duplicate headers/footers
	var content string
	if contentProvider, ok := m.activePhaseModel.(ContentProvider); ok {
		content = contentProvider.ViewContent()
	} else {
		content = m.activePhaseModel.View()
	}

	// Style the content area
	return workflowContentStyle.
		Width(m.width).
		Height(contentHeight).
		Render(content)
}

// overlayModal renders a modal over the current view
func (m *WorkflowModel) overlayModal(background string) string {
	if m.confirmModal == nil {
		return background
	}

	modalView := m.confirmModal.View()

	// Create overlay by placing modal on background (centered)
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modalView,
		lipgloss.WithWhitespaceBackground(lipgloss.AdaptiveColor{Light: "0", Dark: "0"}),
	)
}

// checkPhaseCompletion checks if the current phase has completed
func (m *WorkflowModel) checkPhaseCompletion() bool {
	// This would check if the embedded model signals completion
	// For now, we rely on explicit user action in approval phases
	return false
}

// transitionToApproval moves to the approval gate for current phase
func (m *WorkflowModel) transitionToApproval() tea.Cmd {
	prevPhase := m.session.CurrentPhase
	nextPhase := types.GetNextPhase(prevPhase)

	return func() tea.Msg {
		return workflowPhaseChangedMsg{
			from: prevPhase,
			to:   nextPhase,
		}
	}
}

// approveAndAdvance approves the current phase and advances to next
func (m *WorkflowModel) approveAndAdvance() tea.Cmd {
	prevPhase := m.session.CurrentPhase
	nextPhase := types.GetNextPhase(prevPhase)

	return func() tea.Msg {
		return workflowPhaseChangedMsg{
			from:      prevPhase,
			to:        nextPhase,
			saveCheck: true, // Save checkpoint when approving
		}
	}
}

// goBackToPhase goes back to the previous phase
func (m *WorkflowModel) goBackToPhase() tea.Cmd {
	prevPhase := m.session.CurrentPhase
	targetPhase := types.GetPreviousPhase(prevPhase)

	return func() tea.Msg {
		return workflowPhaseChangedMsg{
			from: prevPhase,
			to:   targetPhase,
		}
	}
}

// goBackToWorkingPhase goes back to the working phase (from approval)
func (m *WorkflowModel) goBackToWorkingPhase() tea.Cmd {
	prevPhase := m.session.CurrentPhase
	var targetPhase types.WorkflowPhase

	switch prevPhase {
	case types.WorkflowPhaseApproveSpec:
		targetPhase = types.WorkflowPhaseSpecify
	case types.WorkflowPhaseApprovePlan:
		targetPhase = types.WorkflowPhasePlan
	case types.WorkflowPhaseApproveTasks:
		targetPhase = types.WorkflowPhaseTasks
	default:
		targetPhase = prevPhase
	}

	return func() tea.Msg {
		return workflowPhaseChangedMsg{
			from: prevPhase,
			to:   targetPhase,
		}
	}
}

// advanceToApproval advances from a working phase to its approval phase
func (m *WorkflowModel) advanceToApproval() tea.Cmd {
	prevPhase := m.session.CurrentPhase
	var targetPhase types.WorkflowPhase

	switch prevPhase {
	case types.WorkflowPhaseSpecify:
		targetPhase = types.WorkflowPhaseApproveSpec
	case types.WorkflowPhasePlan:
		targetPhase = types.WorkflowPhaseApprovePlan
	case types.WorkflowPhaseTasks:
		targetPhase = types.WorkflowPhaseApproveTasks
	case types.WorkflowPhaseImplement:
		targetPhase = types.WorkflowPhaseComplete
	default:
		return nil // Not in a working phase
	}

	return func() tea.Msg {

		return workflowPhaseChangedMsg{
			from: prevPhase,
			to:   targetPhase,
		}
	}
}

// Message types

type workflowPhaseChangedMsg struct {
	from       types.WorkflowPhase
	to         types.WorkflowPhase
	saveCheck  bool // whether to save a checkpoint for the 'from' phase
}

// syncWaitingState synchronizes the workflow's waitingForAI state with the active phase model
func (m *WorkflowModel) syncWaitingState() {
	switch model := m.activePhaseModel.(type) {
	case *SpecifyModel:
		m.waitingForAI = model.waitingForAI
	case *PlanModel:
		m.waitingForAI = model.waitingForAI
	case *TasksModel:
		m.waitingForAI = model.waitingForAI
	case *ImplementModel:
		m.waitingForAI = model.waitingForAI
	}
}

// Styles for workflow mode
var (
	workflowTitleStyle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Padding(0, 1)

	workflowHeaderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.Border)

	workflowFooterStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(theme.Border)

	workflowContentStyle = lipgloss.NewStyle()

	workflowErrorStyle = lipgloss.NewStyle().
		Foreground(theme.Error).
		Bold(true).
		Padding(1)

	workflowQuitStyle = lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true).
		Padding(1)
)

// placeholderView returns a placeholder for phases not yet implemented
func placeholderView(phaseName string, width, height int) string {
	content := fmt.Sprintf("[ %s Phase - Coming Soon ]", phaseName)
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(theme.TextMuted).
		Render(content)
}

// Helper to check if we're in a working phase vs approval phase
func (m *WorkflowModel) isWorkingPhase() bool {
	return !types.IsApprovalPhase(m.session.CurrentPhase) &&
		m.session.CurrentPhase != types.WorkflowPhaseComplete
}

// Helper to get keyboard shortcuts for current phase
func (m *WorkflowModel) getKeyboardShortcuts() []string {
	if types.IsApprovalPhase(m.session.CurrentPhase) {
		shortcuts := []string{
			"Y: Approve",
			"E: Edit",
		}
		if types.CanGoBack(m.session.CurrentPhase) {
			shortcuts = append(shortcuts, "B: Go Back")
		}
		shortcuts = append(shortcuts, "Esc: Quit")
		return shortcuts
	}

	// Working phase shortcuts
	return []string{
		"Enter: Send",
		"Tab: Switch Pane",
		"Esc: Quit",
	}
}

// updateLayoutCache updates the cached header and footer heights if needed
func (m *WorkflowModel) updateLayoutCache() {
	if m.layoutCache.Dirty || m.width != m.layoutCache.LastWidth || m.height != m.layoutCache.LastHeight {
		m.layoutCache.HeaderHeight = lipgloss.Height(m.renderHeader())
		m.layoutCache.FooterHeight = lipgloss.Height(m.renderFooter())
		m.layoutCache.LastWidth = m.width
		m.layoutCache.LastHeight = m.height
		m.layoutCache.Dirty = false
	}
}

// invalidateLayoutCache marks the layout cache as dirty, requiring recalculation
func (m *WorkflowModel) invalidateLayoutCache() {
	m.layoutCache.Dirty = true
}
