package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/internal/tui/components"
	"github.com/darinhaener/collab/pkg/types"
)

// PlanModel manages the plan mode TUI.
// It wraps PhaseModel with plan-specific configuration.
type PlanModel struct {
	*PhaseModel
	session     *types.PlanSession
	planPreview *components.PlanPreview
}

// planPreviewWrapper wraps PlanPreview to implement PreviewUpdater
type planPreviewWrapper struct {
	preview *components.PlanPreview
}

func (w *planPreviewWrapper) Init() tea.Cmd {
	return w.preview.Init()
}

func (w *planPreviewWrapper) View() string {
	return w.preview.View()
}

func (w *planPreviewWrapper) SetSize(width, height int) {
	w.preview.SetSize(width, height)
}

func (w *planPreviewWrapper) SetContent(content string) {
	w.preview.SetContent(content)
}

func (w *planPreviewWrapper) UpdatePreview(msg tea.Msg) tea.Cmd {
	updated, cmd := w.preview.Update(msg)
	*w.preview = updated
	return cmd
}

// NewPlanModel creates a new plan model
func NewPlanModel(session *types.PlanSession, eventBus *events.EventBus) PlanModel {
	planPreview := components.NewPlanPreview(80, 24)
	previewWrapper := &planPreviewWrapper{preview: &planPreview}

	// Create the orchestrator
	apiKey := orchestrator.GetAPIKey()
	orch := orchestrator.NewPlanOrchestrator(session, apiKey, eventBus)

	// Create the model first (we need it for closure references)
	model := &PlanModel{
		session:     session,
		planPreview: &planPreview,
	}

	// Create the phase model config
	config := PhaseModelConfig{
		PhaseType:    PhasePlan,
		Title:        fmt.Sprintf("Collab Plan: %s", session.FriendlyName),
		PreviewTitle: "Plan Preview",
		SubtitleFunc: func() string {
			return fmt.Sprintf("Feature #%03d - %s - Phase: %s",
				session.FeatureNumber,
				session.SpecSlug,
				session.Phase,
			)
		},
		InitialPromptFunc: func() string {
			return fmt.Sprintf("I want to create an implementation plan for the feature: %s. Please help me plan this.",
				session.FriendlyName)
		},
		RefreshPreviewFunc: func() {
			model.refreshPlanPreviewFromFile()
		},
		OnToolUseFunc: func(toolName string, args map[string]interface{}) {
			// Refresh preview after tool use - agent may have written plan
			model.refreshPlanPreviewFromFile()
		},
		GetChatHistoryFunc: func() []types.ChatMessage {
			return session.ChatHistory
		},
		AppendChatHistoryFunc: func(msg types.ChatMessage) {
			session.ChatHistory = append(session.ChatHistory, msg)
		},
		LoadingStateLabel: "plan",
	}

	// Create the PhaseModel
	model.PhaseModel = NewPhaseModel(config, orch, previewWrapper, eventBus)

	return *model
}

// Init implements tea.Model
func (m PlanModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.PhaseModel.Init(),
		initializePlanAgent(m.session, m.PhaseModel.eventBus),
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

// Update implements tea.Model
func (m PlanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle plan-specific messages first
	switch msg := msg.(type) {
	case PlanAgentInitializedMsg:
		// Replace the orchestrator in PhaseModel
		m.PhaseModel.orchestrator = msg.Orchestrator
		// Signal that agent is ready
		return m, func() tea.Msg { return PhaseAgentReadyMsg{} }

	case PlanAgentErrorMsg:
		return m, func() tea.Msg { return PhaseAgentErrorMsg{Err: msg.Err} }
	}

	// Delegate to PhaseModel
	updatedModel, cmd := m.PhaseModel.Update(msg)
	m.PhaseModel = updatedModel.(*PhaseModel)

	return m, cmd
}

// View implements tea.Model
func (m PlanModel) View() string {
	return m.PhaseModel.View()
}

// ViewContent returns just the content without header/footer for embedding in workflow
func (m PlanModel) ViewContent() string {
	return m.PhaseModel.ViewContent()
}

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

// Legacy message types for backward compatibility
type planAgentChannelsReady struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

type planAgentUpdate struct {
	update orchestrator.MessageUpdate
}

type planAgentStreamComplete struct{}

// clearPlanInputMsg signals to clear the input field
type clearPlanInputMsg struct{}

// Compatibility methods

func (m *PlanModel) updateLayoutCache() {
	// Layout is handled by PhaseModel
}

func (m *PlanModel) invalidateLayoutCache() {
	// Layout is handled by PhaseModel
}

// Keyboard interaction constants (kept for any external references)
const (
	planDoubleEscapeTimeout  = 2 * time.Second
	planMessageClearShortcut = "ctrl+u"
)

// PlanPaneType represents which pane is currently active in plan mode
type PlanPaneType = PaneType

const (
	PlanChatPane    PlanPaneType = ChatPane
	PlanPreviewPane PlanPaneType = PreviewPane
)
