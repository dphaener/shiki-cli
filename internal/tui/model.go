package tui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
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

	return Model{
		session:      session,
		turnHistory:  session.TurnHistory,
		currentFile:  currentFile,
		fileList:     fileList,
		eventSub:     sub,
		workspaceDir: session.WorkspaceDir,
		toolActivity: make([]ToolActivity, 0),
		selectedPane: "turns",
		selectedTurn: len(session.TurnHistory) - 1,
		selectedFile: indexOf(fileList, currentFile),
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
