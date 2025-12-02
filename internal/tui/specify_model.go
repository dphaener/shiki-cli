package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/orchestrator"
	"github.com/dphaener/shiki-cli/internal/tui/components"
	"github.com/dphaener/shiki-cli/internal/tui/theme"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// PaneType represents which pane is currently active
type PaneType int

const (
	ChatPane PaneType = iota
	PreviewPane
)

// Keyboard interaction constants
const (
	// doubleEscapeTimeout is the time window for detecting double ESC key presses
	doubleEscapeTimeout = 2 * time.Second
	// messageClearShortcut defines the keyboard combination for clearing messages
	messageClearShortcut = "ctrl+u"
)

// SpecifyModel manages the specify mode TUI.
// It wraps PhaseModel with specify-specific configuration.
type SpecifyModel struct {
	*PhaseModel
	session     *types.SpecifySession
	specPreview *components.SpecPreview
}

// specPreviewWrapper wraps SpecPreview to implement PreviewUpdater
type specPreviewWrapper struct {
	preview *components.SpecPreview
}

func (w *specPreviewWrapper) Init() tea.Cmd {
	return w.preview.Init()
}

func (w *specPreviewWrapper) View() string {
	return w.preview.View()
}

func (w *specPreviewWrapper) SetSize(width, height int) {
	w.preview.SetSize(width, height)
}

func (w *specPreviewWrapper) SetContent(content string) {
	w.preview.SetContent(content)
}

func (w *specPreviewWrapper) UpdatePreview(msg tea.Msg) tea.Cmd {
	updated, cmd := w.preview.Update(msg)
	*w.preview = updated
	return cmd
}

// NewSpecifyModel creates a new specify model
func NewSpecifyModel(session *types.SpecifySession, eventBus *events.EventBus) SpecifyModel {
	specPreview := components.NewSpecPreview(80, 24)
	previewWrapper := &specPreviewWrapper{preview: &specPreview}

	// Create the orchestrator
	apiKey := orchestrator.GetAPIKey()
	orch := orchestrator.NewSpecifyOrchestrator(session, apiKey, eventBus)

	// Create the model first (we need it for closure references)
	model := &SpecifyModel{
		session:     session,
		specPreview: &specPreview,
	}

	// Create the phase model config
	config := PhaseModelConfig{
		PhaseType:    PhaseSpecify,
		Title:        fmt.Sprintf("Collab Specify: %s", session.FriendlyName),
		PreviewTitle: "Specification Preview",
		SubtitleFunc: func() string {
			return fmt.Sprintf("Feature #%03d - %s - Phase: %s",
				session.FeatureNumber,
				session.Slug,
				session.Phase,
			)
		},
		InitialPromptFunc: func() string {
			if session.FeatureDesc != "" {
				return fmt.Sprintf("I want to create a feature: %s", session.FeatureDesc)
			}
			return "Hello, I'd like to specify a new feature."
		},
		RefreshPreviewFunc: func() {
			model.refreshSpecPreviewFromFile()
		},
		OnToolUseFunc: func(toolName string, args map[string]interface{}) {
			// Refresh preview after tool use - agent may have written spec
			model.refreshSpecPreviewFromFile()
		},
		GetChatHistoryFunc: func() []types.ChatMessage {
			return session.ChatHistory
		},
		AppendChatHistoryFunc: func(msg types.ChatMessage) {
			session.ChatHistory = append(session.ChatHistory, msg)
		},
		LoadingStateLabel: "specify",
	}

	// Create the PhaseModel
	model.PhaseModel = NewPhaseModel(config, orch, previewWrapper, eventBus)

	return *model
}

// Init implements tea.Model
func (m SpecifyModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.PhaseModel.Init(),
		initializeAgent(m.session, m.PhaseModel.eventBus),
	)
}

// initializeAgent initializes the AI agent
func initializeAgent(session *types.SpecifySession, eventBus *events.EventBus) tea.Cmd {
	return func() tea.Msg {
		apiKey := orchestrator.GetAPIKey()
		orch := orchestrator.NewSpecifyOrchestrator(session, apiKey, eventBus)

		if err := orch.Initialize(); err != nil {
			return AgentErrorMsg{Err: err}
		}

		return AgentInitializedMsg{Orchestrator: orch}
	}
}

// Update implements tea.Model
func (m SpecifyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle specify-specific messages first
	switch msg := msg.(type) {
	case AgentInitializedMsg:
		// Replace the orchestrator in PhaseModel
		m.PhaseModel.orchestrator = msg.Orchestrator
		// Signal that agent is ready
		return m, func() tea.Msg { return PhaseAgentReadyMsg{} }

	case AgentErrorMsg:
		return m, func() tea.Msg { return PhaseAgentErrorMsg{Err: msg.Err} }
	}

	// Delegate to PhaseModel
	updatedModel, cmd := m.PhaseModel.Update(msg)
	m.PhaseModel = updatedModel.(*PhaseModel)

	return m, cmd
}

// View implements tea.Model
func (m SpecifyModel) View() string {
	return m.PhaseModel.View()
}

// ViewContent returns just the content without header/footer for embedding in workflow
func (m SpecifyModel) ViewContent() string {
	return m.PhaseModel.ViewContent()
}

// refreshSpecPreviewFromFile reads the spec file from disk and updates the preview
func (m *SpecifyModel) refreshSpecPreviewFromFile() {
	if m.session.SpecFile == "" {
		return
	}

	content, err := os.ReadFile(m.session.SpecFile)
	if err != nil {
		return // File may not exist yet
	}

	// Strip YAML frontmatter if present
	specContent := stripFrontmatter(string(content))
	if specContent == "" {
		return
	}

	m.specPreview.SetContent(specContent)
	m.session.CurrentSpec = specContent

	// Update phase based on content
	if containsSpecSections(specContent) {
		m.session.Phase = types.PhaseGeneration
		m.specPreview.SetPhase(types.PhaseGeneration)
	}
}

// stripFrontmatter removes YAML frontmatter from markdown content
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2])
	}
	return content
}

// containsSpecSections checks if content has multiple spec sections
func containsSpecSections(content string) bool {
	content = strings.ToLower(content)
	sectionCount := 0
	sections := []string{"overview", "requirements", "success criteria", "user scenarios", "assumptions"}
	for _, section := range sections {
		if strings.Contains(content, section) {
			sectionCount++
		}
	}
	return sectionCount >= 3
}

// Message types for agent communication

// AgentInitializedMsg indicates the agent is ready
type AgentInitializedMsg struct {
	Orchestrator *orchestrator.SpecifyOrchestrator
}

// AgentResponseMsg represents a complete AI agent response
type AgentResponseMsg struct {
	Content string
}

// AgentStreamMsg represents a streaming chunk from the AI agent
type AgentStreamMsg struct {
	Chunk string
}

// AgentToolUseMsg represents a tool invocation by the agent
type AgentToolUseMsg struct {
	ToolName string
}

// AgentErrorMsg represents an error from the AI agent
type AgentErrorMsg struct {
	Err error
}

// Legacy message types for backward compatibility
type agentChannelsReady struct {
	updateChan <-chan orchestrator.MessageUpdate
	errorChan  <-chan error
}

type agentUpdate struct {
	update orchestrator.MessageUpdate
}

type agentStreamComplete struct{}

// clearInputMsg signals to clear the input field
type clearInputMsg struct{}

// Compatibility methods
func (m *SpecifyModel) updateLayoutCache() {
	// Layout is handled by PhaseModel
}

func (m *SpecifyModel) invalidateLayoutCache() {
	// Layout is handled by PhaseModel
}

// Styles for specify mode using Sekkei Design System theme
// These are kept for backward compatibility with any external references
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1)

	headerBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(theme.Border)

	footerStyle = lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1)

	footerBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(theme.Border)

	specifyPaneBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(theme.Border).
				Padding(1)

	specifySelectedPaneBorderStyle = lipgloss.NewStyle().
					BorderStyle(lipgloss.RoundedBorder()).
					BorderForeground(theme.BorderActive).
					Padding(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true).
			Padding(1)

	quitMessageStyle = lipgloss.NewStyle().
				Foreground(theme.Success).
				Bold(true).
				Padding(1)
)
