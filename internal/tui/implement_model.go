package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/orchestrator"
	"github.com/dphaener/shiki-cli/internal/templates"
	"github.com/dphaener/shiki-cli/internal/tui/components"
	"github.com/dphaener/shiki-cli/pkg/types"
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

func (w *implementPreviewWrapper) GetPhase() types.PreviewPhase {
	return w.preview.GetPhase()
}

func (w *implementPreviewWrapper) SetPhase(phase types.PreviewPhase) {
	w.preview.SetPhase(phase)
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
		PhaseType:          PhaseImplement,
		Title:              fmt.Sprintf("Collab Implement: %s", session.FriendlyName),
		PreviewTitle:       "Implementation Progress",
		OutputTemplateName: "task-progress",
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
		OnToolUseFunc: func(toolName string, args map[string]interface{}) {
			// Refresh preview after tool use - agent may have written files
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
	// Load template content first
	m.loadImplementTemplateContent()

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
		// Replace the orchestrator in PhaseModel - cast to interface
		m.PhaseModel.orchestrator = orchestrator.PhaseOrchestrator(msg.Orchestrator)

		// Start task progress tracking if available
		if msg.Orchestrator.IsTaskProgressEnabled() {
			msg.Orchestrator.StartTaskProgress()
		}

		// Refresh the preview to show the task progress file
		m.refreshImplementPreview()

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

	case events.Event:
		// Handle file update events for task-progress.md
		if msg.Type == types.EventFileUpdated {
			if payload, ok := msg.Payload.(events.FileUpdatedPayload); ok {
				return m.handleFileUpdateEvent(payload)
			}
		}

	case TaskProgressRefreshMsg:
		// Task progress file was updated - show appropriate feedback
		operation := msg.Operation
		var statusMsg string
		switch operation {
		case "created":
			statusMsg = "Task progress tracking started"
		case "modified":
			statusMsg = "Task progress updated"
		case "deleted":
			statusMsg = "Task progress file deleted"
		default:
			statusMsg = "Task progress refreshed"
		}

		// Update UI to show progress has been refreshed
		m.refreshImplementPreview()

		// Could emit a brief status message here if desired
		_ = statusMsg
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
	// Check if feature directory is available
	if m.session.FeatureDir == "" {
		m.implementPreview.SetSummary("Implementation progress will appear here when the agent begins work.")
		return
	}

	progressFilePath := filepath.Join(m.session.FeatureDir, "task-progress.md")

	// Check if progress file exists
	if _, err := os.Stat(progressFilePath); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet - show loading message
			loadingContent := "Loading tasks...\n\nInitializing task progress tracking.\nThe agent will create task-progress.md when implementation begins."
			m.implementPreview.SetSummary(loadingContent)
		} else {
			// Other error accessing file
			errorContent := fmt.Sprintf("Unable to access task progress file:\n%v\n\nPlease check file permissions and try again.", err)
			m.implementPreview.SetSummary(errorContent)
		}
		return
	}

	// File exists, try to load progress with retry mechanism
	maxRetries := 3
	var loadErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		loadErr = m.implementPreview.LoadProgressFromFile(progressFilePath)
		if loadErr == nil {
			// Success - break out of retry loop
			break
		}

		// Check if it's a temporary file access error
		if os.IsNotExist(loadErr) {
			// File was deleted between check and load - unlikely but possible
			m.implementPreview.SetSummary("Task progress file was removed during loading. Refreshing...")
			return
		}

		if strings.Contains(loadErr.Error(), "permission denied") ||
			strings.Contains(loadErr.Error(), "device or resource busy") {
			// Temporary access issue - retry after a short delay
			if attempt < maxRetries-1 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		// For validation/parsing errors, don't retry - these are permanent
		break
	}

	if loadErr != nil {
		// Loading failed - show detailed error message
		var errorMsg string
		if strings.Contains(loadErr.Error(), "validation failed") {
			errorMsg = fmt.Sprintf("Task progress file format error:\n%v\n\nThe file may be corrupted or in an unexpected format.", loadErr)
		} else if strings.Contains(loadErr.Error(), "parse") {
			errorMsg = fmt.Sprintf("Failed to parse task progress:\n%v\n\nPlease check the file format in task-progress.md", loadErr)
		} else if strings.Contains(loadErr.Error(), "permission denied") {
			errorMsg = fmt.Sprintf("Permission error accessing task progress:\n%v\n\nPlease check file permissions for task-progress.md", loadErr)
		} else {
			errorMsg = fmt.Sprintf("Task progress loading error:\n%v\n\nRefreshing the view may resolve this issue.", loadErr)
		}
		m.implementPreview.SetSummary(errorMsg)
	}

	// If we reach here, loading was successful - the preview will show the progress data
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

// initializeTaskProgress initializes task progress tracking
// This now delegates to the orchestrator which handles file creation
func (m *ImplementModel) initializeTaskProgress() error {
	if m.session.FeatureDir == "" {
		return fmt.Errorf("no feature directory available")
	}

	progressFilePath := filepath.Join(m.session.FeatureDir, "task-progress.md")

	// Check if progress file already exists
	if _, err := os.Stat(progressFilePath); err == nil {
		// File already exists, just refresh the preview
		return m.implementPreview.RefreshFromProgressFile()
	}

	// The orchestrator will handle file creation during Initialize()
	// Here we just need to prepare the preview to load it once it's created
	return nil
}

// handleFileUpdateEvent handles file update events for task progress monitoring
func (m *ImplementModel) handleFileUpdateEvent(payload events.FileUpdatedPayload) (tea.Model, tea.Cmd) {
	// Check if the updated file is the task-progress.md file for this session
	if m.session.FeatureDir == "" {
		return m, nil
	}

	expectedProgressFile := filepath.Join(m.session.FeatureDir, "task-progress.md")

	// Normalize paths for comparison
	updateFile := filepath.Clean(payload.Path)
	expectedFile := filepath.Clean(expectedProgressFile)

	if updateFile == expectedFile {
		// The task progress file was updated, refresh the preview
		go func() {
			// Add retry mechanism for file access with exponential backoff
			maxRetries := 5
			baseDelay := 50 * time.Millisecond

			for attempt := 0; attempt < maxRetries; attempt++ {
				// Calculate delay with exponential backoff
				delay := time.Duration(attempt) * baseDelay
				if delay > 0 {
					time.Sleep(delay)
				}

				err := m.implementPreview.RefreshFromProgressFile()
				if err == nil {
					// Success - break out of retry loop
					return
				}

				// Check if it's a temporary file access error
				if os.IsNotExist(err) {
					// File might have been temporarily deleted/moved during write
					continue
				}

				if strings.Contains(err.Error(), "permission denied") ||
					strings.Contains(err.Error(), "device or resource busy") {
					// Temporary access issue - retry
					continue
				}

				// For other errors, log and set error message on final attempt
				if attempt == maxRetries-1 {
					errorMsg := fmt.Sprintf("Failed to refresh task progress after %d attempts: %v", maxRetries, err)
					m.implementPreview.SetSummary(errorMsg)
				}
			}
		}()

		// Show a brief loading indicator
		return m, func() tea.Msg {
			return TaskProgressRefreshMsg{Operation: payload.Operation}
		}
	}

	return m, nil
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

// TaskProgressRefreshMsg indicates the task progress file was updated
type TaskProgressRefreshMsg struct {
	Operation string // "created", "modified", "deleted"
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

// loadImplementTemplateContent loads the task-progress output template with proper session context
func (m *ImplementModel) loadImplementTemplateContent() {
	// Create context with session data for template variables
	context := templates.NewPhaseContext().
		WithCore(m.session.FriendlyName, m.session.FeatureNumber, m.session.Slug, string(m.session.CurrentPhase))

	// Load template content with context
	templateContent := m.PhaseModel.LoadOutputTemplateWithContext(context)
	if templateContent != "" {
		// Set template content in preview
		m.implementPreview.SetContent(templateContent)
	}
}
