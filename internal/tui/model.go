package tui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/tui/components"
	"github.com/darinhaener/collab/pkg/types"
)

// ViewMode represents the current view state of the TUI
type ViewMode string

const (
	ViewModeAgents     ViewMode = "agents"
	ViewModeCompletion ViewMode = "completion"
)

// ToolActivity represents a tool invocation for display
type ToolActivity struct {
	ToolName  string
	AgentID   string
	Timestamp time.Time
}

// Model represents the TUI state (Bubbletea Model)
type Model struct {
	session      *types.Session
	turnHistory  []types.Turn
	currentFile  string
	fileContent  string
	fileList     []string
	eventSub     *events.Subscriber
	workspaceDir string
	toolActivity []ToolActivity // Recent tool invocations

	// Agent output tracking
	agentOutputs  map[string]*components.AgentOutputState // Agent ID -> output state
	activeAgents  []string                                // List of agent IDs in display order
	selectedAgent int                                     // Index of selected agent pane

	// View state
	viewMode                   ViewMode
	deliverableContent         string // raw content
	deliverableRenderedContent string // pre-rendered with glamour (cached)
	deliverablePath            string

	// UI state
	selectedTurn int
	selectedPane string // "turns" or "file"
	selectedFile int
	scrollOffset int

	// Terminal dimensions
	width  int
	height int

	// Runtime state
	paused bool
	err    error
}

// NewModel creates a new TUI model from a session
func NewModel(session *types.Session, bus *events.EventBus) Model {
	// Subscribe to relevant events for real-time updates
	sub := bus.Subscribe("tui",
		types.EventTurnStarted,
		types.EventTurnCompleted,
		types.EventTurnError,
		types.EventFileUpdated,
		types.EventToolInvoked,
		types.EventAssistantMessage,
		types.EventSessionCompleted,
		types.EventSessionPaused,
		types.EventSessionError,
	)

	// Get list of files in workspace
	fileList := getWorkspaceFiles(session.WorkspaceDir)

	// Start with shared_context.md if it exists, otherwise first file
	currentFile := "shared_context.md"
	if !contains(fileList, currentFile) && len(fileList) > 0 {
		currentFile = fileList[0]
	}

	// Initialize agent output states
	agentOutputs := make(map[string]*components.AgentOutputState)
	activeAgents := []string{session.Agent1.ID, session.Agent2.ID}

	// Create initial state for each agent
	agentOutputs[session.Agent1.ID] = &components.AgentOutputState{
		AgentID:      session.Agent1.ID,
		Status:       types.TurnCompleted,
		CurrentTurn:  0,
		TotalCost:    session.Agent1.TotalCost,
		TotalTokens:  session.Agent1.TotalTokens,
		Outputs:      make([]components.OutputEntry, 0),
		ScrollOffset: 0,
		AutoScroll:   true, // Start with autoscroll enabled
	}

	agentOutputs[session.Agent2.ID] = &components.AgentOutputState{
		AgentID:      session.Agent2.ID,
		Status:       types.TurnCompleted,
		CurrentTurn:  0,
		TotalCost:    session.Agent2.TotalCost,
		TotalTokens:  session.Agent2.TotalTokens,
		Outputs:      make([]components.OutputEntry, 0),
		ScrollOffset: 0,
		AutoScroll:   true, // Start with autoscroll enabled
	}

	return Model{
		session:       session,
		turnHistory:   session.TurnHistory,
		currentFile:   currentFile,
		fileList:      fileList,
		eventSub:      sub,
		workspaceDir:  session.WorkspaceDir,
		toolActivity:  make([]ToolActivity, 0),
		agentOutputs:  agentOutputs,
		activeAgents:  activeAgents,
		selectedAgent: 0,
		viewMode:      ViewModeAgents,
		selectedPane:  "turns",
		selectedTurn:  len(session.TurnHistory) - 1,
		selectedFile:  indexOf(fileList, currentFile),
	}
}

// Init initializes the model (Bubbletea lifecycle)
func (m Model) Init() tea.Cmd {
	// Start listening for events and load initial file content
	return tea.Batch(
		waitForEvent(m.eventSub),
		loadFileContent(m.workspaceDir, m.currentFile),
	)
}

// waitForEvent waits for an event from the EventBus
func waitForEvent(sub *events.Subscriber) tea.Cmd {
	return func() tea.Msg {
		event := <-sub.Events()
		return eventMsg{event}
	}
}

// loadFileContent loads file content from workspace
func loadFileContent(workspaceDir, filename string) tea.Cmd {
	return func() tea.Msg {
		if filename == "" {
			return fileContentMsg{content: "No file selected"}
		}

		path := filepath.Join(workspaceDir, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return fileContentMsg{
				content: "Error reading file: " + err.Error(),
			}
		}

		return fileContentMsg{
			content:  string(content),
			filename: filename,
		}
	}
}

// loadDeliverableContent loads deliverable content from the specified path
// and pre-renders markdown to avoid blocking the View function
func loadDeliverableContent(path string) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			return deliverableContentMsg{
				content: "",
				path:    "",
				err:     nil,
			}
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return deliverableContentMsg{
				content: "",
				path:    path,
				err:     err,
			}
		}

		// Pre-render markdown here (outside the View loop) to avoid blocking
		// We use a reasonable default width; it will be plain text but readable
		rendered := string(content)
		// Note: glamour rendering removed - it blocks the TUI
		// The content will be displayed as plain text which is fine for deliverables

		return deliverableContentMsg{
			content: rendered,
			path:    path,
			err:     nil,
		}
	}
}

// getWorkspaceFiles returns list of files in workspace directory
func getWorkspaceFiles(workspaceDir string) []string {
	var files []string

	// Walk workspace directory
	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path from workspace
		relPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return nil
		}

		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return []string{}
	}

	return files
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return 0
}

// Messages for Bubbletea message passing
type eventMsg struct {
	event events.Event
}

type fileContentMsg struct {
	content  string
	filename string
}

type deliverableContentMsg struct {
	content string
	path    string
	err     error
}

// deliverableRenderedMsg is sent after glamour rendering completes
type deliverableRenderedMsg struct {
	content string
}

// doRenderDeliverable renders markdown content with glamour in a goroutine
// This prevents blocking the TUI event loop
func doRenderDeliverable(content string, width int) tea.Cmd {
	return func() tea.Msg {
		if width <= 0 {
			width = 80 // default width
		}
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return deliverableRenderedMsg{content: content}
		}
		rendered, err := renderer.Render(content)
		if err != nil {
			return deliverableRenderedMsg{content: content}
		}
		return deliverableRenderedMsg{content: rendered}
	}
}
