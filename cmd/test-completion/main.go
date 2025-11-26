// Quick test for completion view TUI
// Run with: go run ./cmd/test-completion
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/darinhaener/collab/pkg/types"
)

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// Minimal model that just shows the completion view
type completionModel struct {
	session            *types.Session
	deliverableContent string // raw content
	renderedContent    string // pre-rendered with glamour (cached)
	deliverablePath    string
	scrollOffset       int
	width              int
	height             int
	needsRender        bool // flag to render once after getting window size
}

func (m completionModel) Init() tea.Cmd {
	return nil
}

// renderMsg is sent after glamour rendering completes
type renderMsg struct {
	content string
}

// doRender renders content with glamour in a goroutine
func doRender(content string, width int) tea.Cmd {
	return func() tea.Msg {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return renderMsg{content: content}
		}
		rendered, err := renderer.Render(content)
		if err != nil {
			return renderMsg{content: content}
		}
		return renderMsg{content: rendered}
	}
}

func (m completionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "down", "j":
			m.scrollOffset++
		case "pgup":
			m.scrollOffset -= 10
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		case "pgdown":
			m.scrollOffset += 10
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Trigger glamour render once we have window size
		if m.needsRender && m.width > 0 {
			m.needsRender = false
			return m, doRender(m.deliverableContent, m.width)
		}
	case renderMsg:
		// Cache the rendered content
		m.renderedContent = msg.content
	}
	return m, nil
}

func (m completionModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Use pre-rendered content if available, otherwise show loading or raw
	content := m.renderedContent
	status := "rendered"
	if content == "" {
		content = m.deliverableContent
		status = "loading..."
	}

	lines := strings.Split(content, "\n")

	// Header
	header := fmt.Sprintf("Status: %s | Scroll: %d | Size: %dx%d | Press q to quit\n\n",
		status, m.scrollOffset, m.width, m.height)

	// Apply scroll offset
	start := m.scrollOffset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + m.height - 4
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]
	return header + joinLines(visible)
}

func main() {
	// Create a temp workspace
	tmpDir, err := os.MkdirTemp("", "collab-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock deliverable
	deliverablePath := filepath.Join(tmpDir, "deliverable.md")
	deliverableContent := `# Test Deliverable

This is a test deliverable to verify the completion view works.

## Summary
- Agent 1 proposed a solution
- Agent 2 approved it

## Details
Lorem ipsum dolor sit amet, consectetur adipiscing elit.
Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

### Section 1
More content here to test scrolling.

### Section 2
Even more content to ensure we can scroll.

### Section 3
And yet more content...

### Section 4
Keep scrolling...

### Section 5
Almost there...

### Section 6
The end!
`
	if err := os.WriteFile(deliverablePath, []byte(deliverableContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing deliverable: %v\n", err)
		os.Exit(1)
	}

	// Create mock session
	now := time.Now()
	completedAt := now.Add(2 * time.Minute)
	session := &types.Session{
		ID:           "test-session",
		TaskName:     "Test Completion View",
		WorkspaceDir: tmpDir,
		Status:       types.SessionCompleted,
		CreatedAt:    now,
		StartedAt:    &now,
		CompletedAt:  &completedAt,
		CurrentTurn:  2,
		MaxTurns:     10,
		Agent1: types.Agent{
			ID:          "agent_1",
			Name:        "Proposer",
			Role:        "Proposes solutions",
			TotalCost:   0.015,
			TotalTokens: 1500,
		},
		Agent2: types.Agent{
			ID:          "agent_2",
			Name:        "Approver",
			Role:        "Reviews and approves",
			TotalCost:   0.012,
			TotalTokens: 1200,
		},
		TurnHistory: []types.Turn{
			{Number: 0, AgentID: "agent_1", Status: types.TurnCompleted},
			{Number: 1, AgentID: "agent_2", Status: types.TurnCompleted},
		},
		DeliverablePath: deliverablePath,
	}

	// Create minimal model directly in completion mode
	model := completionModel{
		session:            session,
		deliverableContent: deliverableContent,
		renderedContent:    "", // will be populated after glamour render
		deliverablePath:    deliverablePath,
		scrollOffset:       0,
		needsRender:        true, // trigger glamour render on first window size
	}

	// Create and run the program
	p := tea.NewProgram(model, tea.WithAltScreen())

	fmt.Println("Starting completion view test (minimal)...")
	fmt.Println("Press 'q' to quit, arrow keys to scroll")
	fmt.Println()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Test completed successfully!")
}
