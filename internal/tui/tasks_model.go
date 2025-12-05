package tui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/orchestrator"
	"github.com/dphaener/shiki-cli/internal/tui/components"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// TasksModel manages the tasks generation TUI.
// It wraps PhaseModel with tasks-specific configuration.
type TasksModel struct {
	*PhaseModel
	session       *types.WorkflowSession
	tasksPreview  *components.TasksPreview
	phaseComplete bool
}

// tasksPreviewWrapper wraps TasksPreview to implement PreviewUpdater
type tasksPreviewWrapper struct {
	preview *components.TasksPreview
}

func (w *tasksPreviewWrapper) Init() tea.Cmd {
	return w.preview.Init()
}

func (w *tasksPreviewWrapper) View() string {
	return w.preview.View()
}

func (w *tasksPreviewWrapper) SetSize(width, height int) {
	w.preview.SetSize(width, height)
}

func (w *tasksPreviewWrapper) SetContent(content string) {
	w.preview.SetContent(content)
}

func (w *tasksPreviewWrapper) GetPhase() types.PreviewPhase {
	return w.preview.GetPhase()
}

func (w *tasksPreviewWrapper) SetPhase(phase types.PreviewPhase) {
	w.preview.SetPhase(phase)
}

func (w *tasksPreviewWrapper) UpdatePreview(msg tea.Msg) tea.Cmd {
	updated, cmd := w.preview.Update(msg)
	*w.preview = updated
	return cmd
}

// NewTasksModel creates a new tasks model
func NewTasksModel(session *types.WorkflowSession, eventBus *events.EventBus) TasksModel {
	tasksPreview := components.NewTasksPreview(80, 24)
	previewWrapper := &tasksPreviewWrapper{preview: &tasksPreview}

	// Create the orchestrator
	apiKey := orchestrator.GetAPIKey()
	orch := orchestrator.NewTasksOrchestrator(session, apiKey, eventBus)

	// Create the model first (we need it for closure references)
	model := &TasksModel{
		session:      session,
		tasksPreview: &tasksPreview,
	}

	// Create the phase model config
	config := PhaseModelConfig{
		PhaseType:    PhaseTasks,
		Title:        fmt.Sprintf("Collab Tasks: %s", session.FriendlyName),
		PreviewTitle: "Tasks Preview",
		SubtitleFunc: func() string {
			return fmt.Sprintf("Feature #%03d - %s - Phase: Tasks Generation",
				session.FeatureNumber,
				session.Slug,
			)
		},
		InitialPromptFunc: func() string {
			return fmt.Sprintf("Please generate implementation tasks for the feature: %s. The spec is at %s and the plan is at %s.",
				session.FriendlyName, session.SpecFile, session.PlanFile)
		},
		RefreshPreviewFunc: func() {
			model.refreshTasksPreviewFromFile()
		},
		GetChatHistoryFunc: func() []types.ChatMessage {
			return session.TasksChatHistory
		},
		AppendChatHistoryFunc: func(msg types.ChatMessage) {
			session.TasksChatHistory = append(session.TasksChatHistory, msg)
		},
		LoadingStateLabel: "tasks",
	}

	// Create the PhaseModel
	model.PhaseModel = NewPhaseModel(config, orch, previewWrapper, eventBus)

	return *model
}

// Init implements tea.Model
func (m TasksModel) Init() tea.Cmd {
	return tea.Batch(
		m.PhaseModel.Init(),
		initializeTasksAgent(m.session, m.PhaseModel.eventBus),
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
	// Handle tasks-specific messages first
	switch msg := msg.(type) {
	case TasksAgentInitializedMsg:
		// Replace the orchestrator in PhaseModel
		m.PhaseModel.orchestrator = msg.Orchestrator
		// Signal that agent is ready
		return m, func() tea.Msg { return PhaseAgentReadyMsg{} }

	case TasksAgentErrorMsg:
		return m, func() tea.Msg { return PhaseAgentErrorMsg{Err: msg.Err} }
	}

	// Delegate to PhaseModel
	updatedModel, cmd := m.PhaseModel.Update(msg)
	m.PhaseModel = updatedModel.(*PhaseModel)

	// Check for phase completion after stream complete
	if _, ok := msg.(phaseStreamCompleteMsg); ok {
		if m.tasksFileExists() {
			m.phaseComplete = true
		}
	}

	return m, cmd
}

// View implements tea.Model
func (m TasksModel) View() string {
	return m.PhaseModel.View()
}

// ViewContent returns just the content without header/footer for embedding in workflow
func (m TasksModel) ViewContent() string {
	return m.PhaseModel.ViewContent()
}

// refreshTasksPreviewFromFile reads the tasks file and updates preview
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

// tasksFileExists checks if the tasks file has been created
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

// Message types for agent communication

// TasksAgentInitializedMsg indicates the agent is ready
type TasksAgentInitializedMsg struct {
	Orchestrator *orchestrator.TasksOrchestrator
}

// TasksAgentErrorMsg represents an error from the AI agent
type TasksAgentErrorMsg struct {
	Err error
}

// LayoutCache is defined in layout.go

// Legacy message types for backward compatibility
type tasksAgentChannelsReady struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

type tasksAgentUpdate struct {
	update orchestrator.MessageUpdate
}

type tasksAgentStreamComplete struct{}

// Styles are now handled by PhaseModel's shared styles

// Compatibility: keep updateLayoutCache and invalidateLayoutCache as no-ops
// since they're referenced in workflow_model.go
func (m *TasksModel) updateLayoutCache() {
	// Layout is handled by PhaseModel
}

func (m *TasksModel) invalidateLayoutCache() {
	// Layout is handled by PhaseModel
}

// Additional compatibility methods that may be referenced elsewhere

// GetSession returns the workflow session
func (m *TasksModel) GetSession() *types.WorkflowSession {
	return m.session
}

// SetEventBus sets the event bus
func (m *TasksModel) SetEventBus(eventBus *events.EventBus) {
	m.PhaseModel.eventBus = eventBus
}

// Keyboard interaction constants (kept for any external references)
const (
	tasksDoubleEscapeTimeout  = 2 * time.Second
	tasksMessageClearShortcut = "ctrl+u"
)

// TasksPaneType represents which pane is currently active in tasks mode
// Kept for backward compatibility
type TasksPaneType = PaneType

const (
	TasksChatPane    TasksPaneType = ChatPane
	TasksPreviewPane TasksPaneType = PreviewPane
)
