package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/internal/tasks"
	"github.com/darinhaener/collab/internal/tui/components"
	"github.com/darinhaener/collab/pkg/types"
)

// ImplementModel manages the implementation TUI.
// It wraps PhaseModel with implementation-specific configuration.
type ImplementModel struct {
	*PhaseModel
	session          *types.WorkflowSession
	implementPreview *components.ImplementPreview
	currentTaskIndex int
	totalTasks       int
	completedTasks   int
	phaseComplete    bool
}

// implementPreviewWrapper wraps ImplementPreview to implement PreviewUpdater
type implementPreviewWrapper struct {
	preview *components.ImplementPreview
}

func (w *implementPreviewWrapper) Init() tea.Cmd {
	return w.preview.Init()
}

func (w *implementPreviewWrapper) View() string {
	return w.preview.View()
}

func (w *implementPreviewWrapper) SetSize(width, height int) {
	w.preview.SetSize(width, height)
}

func (w *implementPreviewWrapper) SetContent(content string) {
	w.preview.SetContent(content)
}

func (w *implementPreviewWrapper) UpdatePreview(msg tea.Msg) tea.Cmd {
	updated, cmd := w.preview.Update(msg)
	*w.preview = updated
	return cmd
}

// NewImplementModel creates a new implement model
func NewImplementModel(session *types.WorkflowSession, eventBus *events.EventBus) ImplementModel {
	implementPreview := components.NewImplementPreview(80, 24)
	previewWrapper := &implementPreviewWrapper{preview: &implementPreview}

	// Create the orchestrator
	apiKey := orchestrator.GetAPIKey()
	orch := orchestrator.NewImplementOrchestrator(session, apiKey, eventBus)

	// Create the model first (we need it for closure references)
	model := &ImplementModel{
		session:          session,
		implementPreview: &implementPreview,
	}

	// Create the phase model config
	config := PhaseModelConfig{
		PhaseType:    PhaseImplement,
		Title:        fmt.Sprintf("Collab Implement: %s", session.FriendlyName),
		PreviewTitle: "Implementation Progress",
		SubtitleFunc: func() string {
			progressStr := ""
			if model.totalTasks > 0 {
				progressStr = fmt.Sprintf(" [%d/%d tasks]", model.completedTasks, model.totalTasks)
			}
			return fmt.Sprintf("Feature #%03d - %s - Phase: Implementation%s",
				session.FeatureNumber,
				session.Slug,
				progressStr,
			)
		},
		InitialPromptFunc: func() string {
			// Load tasks file
			tasksContent := ""
			if content, err := os.ReadFile(session.TasksFile); err == nil {
				tasksContent = string(content)
			}
			return fmt.Sprintf("Please implement the feature: %s\n\nTasks:\n%s",
				session.FriendlyName, tasksContent)
		},
		RefreshPreviewFunc: func() {
			model.refreshImplementPreview()
		},
		GetChatHistoryFunc: func() []types.ChatMessage {
			return session.ImplChatHistory
		},
		AppendChatHistoryFunc: func(msg types.ChatMessage) {
			session.ImplChatHistory = append(session.ImplChatHistory, msg)
		},
		LoadingStateLabel: "implement",
	}

	// Create the PhaseModel
	model.PhaseModel = NewPhaseModel(config, orch, previewWrapper, eventBus)

	return *model
}

// Init implements tea.Model
func (m ImplementModel) Init() tea.Cmd {
	return tea.Batch(
		m.PhaseModel.Init(),
		initializeImplementAgent(m.session, m.PhaseModel.eventBus),
		func() tea.Msg {
			// Initialize task progress file
			if err := m.initializeTaskProgress(); err != nil {
				// Log error but don't fail initialization
				return TaskProgressInitErrorMsg{Err: err}
			}
			return TaskProgressInitializedMsg{}
		},
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
	// Handle implement-specific messages first
	switch msg := msg.(type) {
	case ImplementAgentInitializedMsg:
		// Replace the orchestrator in PhaseModel
		m.PhaseModel.orchestrator = msg.Orchestrator
		// Signal that agent is ready
		return m, func() tea.Msg { return PhaseAgentReadyMsg{} }

	case ImplementAgentErrorMsg:
		return m, func() tea.Msg { return PhaseAgentErrorMsg{Err: msg.Err} }

	case TaskProgressInitializedMsg:
		// Task progress tracking is ready - refresh the preview
		m.refreshImplementPreview()
		return m, nil

	case TaskProgressInitErrorMsg:
		// Task progress initialization failed - show error in preview
		errorContent := fmt.Sprintf("Task progress initialization failed: %v", msg.Err)
		m.implementPreview.SetSummary(errorContent)
		return m, nil
	}

	// Delegate to PhaseModel
	updatedModel, cmd := m.PhaseModel.Update(msg)
	m.PhaseModel = updatedModel.(*PhaseModel)

	// Check for phase completion after stream complete
	if _, ok := msg.(phaseStreamCompleteMsg); ok {
		// Could check for implementation completion markers here
		// For now, phase completes when all tasks are done
	}

	return m, cmd
}

// View implements tea.Model
func (m ImplementModel) View() string {
	return m.PhaseModel.View()
}

// ViewContent returns just the content without header/footer for embedding in workflow
func (m ImplementModel) ViewContent() string {
	return m.PhaseModel.ViewContent()
}

// refreshImplementPreview updates the implementation preview
func (m *ImplementModel) refreshImplementPreview() {
	// Load task progress from progress file if it exists
	if m.session.FeatureDir != "" {
		progressFilePath := filepath.Join(m.session.FeatureDir, "task-progress.md")

		// Try to load progress from file
		if err := m.implementPreview.LoadProgressFromFile(progressFilePath); err != nil {
			// If loading fails, show error or fallback content
			fallbackContent := fmt.Sprintf("Task progress tracking not available.\nReason: %v\n\nThe agent will create task-progress.md when implementation begins.", err)
			m.implementPreview.SetSummary(fallbackContent)
		}
	} else {
		// No feature directory available
		m.implementPreview.SetSummary("Implementation progress will appear here when the agent begins work.")
	}
}

// IsPhaseComplete returns whether implementation is complete
func (m *ImplementModel) IsPhaseComplete() bool {
	return m.phaseComplete
}

// SetTaskProgress updates the task progress indicators
func (m *ImplementModel) SetTaskProgress(current, total, completed int) {
	m.currentTaskIndex = current
	m.totalTasks = total
	m.completedTasks = completed
}

// initializeTaskProgress creates initial task-progress.md file if it doesn't exist
func (m *ImplementModel) initializeTaskProgress() error {
	if m.session.FeatureDir == "" {
		return fmt.Errorf("no feature directory available")
	}

	progressFilePath := filepath.Join(m.session.FeatureDir, "task-progress.md")
	tasksFilePath := m.session.TasksFile

	// Check if progress file already exists
	if _, err := os.Stat(progressFilePath); err == nil {
		// File already exists, just refresh the preview
		return m.implementPreview.RefreshFromProgressFile()
	}

	// Create progress file manager and initialize progress file
	progressManager := tasks.NewProgressFileManager(progressFilePath)
	if err := progressManager.CreateInitialProgressFile(tasksFilePath); err != nil {
		return fmt.Errorf("failed to create initial progress file: %w", err)
	}

	// Load the newly created progress file into the preview
	return m.implementPreview.LoadProgressFromFile(progressFilePath)
}

// Message types for agent communication

// ImplementAgentInitializedMsg indicates the agent is ready
type ImplementAgentInitializedMsg struct {
	Orchestrator *orchestrator.ImplementOrchestrator
}

// ImplementAgentErrorMsg represents an error from the AI agent
type ImplementAgentErrorMsg struct {
	Err error
}

// TaskProgressInitializedMsg indicates task progress tracking is ready
type TaskProgressInitializedMsg struct{}

// TaskProgressInitErrorMsg represents an error during task progress initialization
type TaskProgressInitErrorMsg struct {
	Err error
}

// Legacy message types for backward compatibility
type implementAgentChannelsReady struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

type implementAgentUpdate struct {
	update orchestrator.MessageUpdate
}

type implementAgentStreamComplete struct{}

// Compatibility methods
func (m *ImplementModel) updateLayoutCache() {
	// Layout is handled by PhaseModel
}

func (m *ImplementModel) invalidateLayoutCache() {
	// Layout is handled by PhaseModel
}

// Keyboard interaction constants (kept for any external references)
const (
	implementDoubleEscapeTimeout  = 2 * time.Second
	implementMessageClearShortcut = "ctrl+u"
)

// ImplementPaneType represents which pane is currently active in implement mode
type ImplementPaneType = PaneType

const (
	ImplementChatPane    ImplementPaneType = ChatPane
	ImplementPreviewPane ImplementPaneType = PreviewPane
)
