---
work_package_id: WP09
title: "Terminal UI (TUI)"
priority: P2
status: planned
subtasks:
  - T052
  - T053
  - T054
  - T055
  - T056
  - T057
  - T058
  - T059
dependencies:
  - WP03
  - WP07
lane: planned
history:
  - timestamp: "2025-11-23"
    action: created
    status: planned
---

# Work Package WP09: Terminal UI (TUI)

**Objective**: Implement Bubbletea-based real-time monitoring interface with split-pane layout, keyboard navigation, and <100ms event updates.

**Priority**: P2 (User experience enhancement)

**Estimated Effort**: 6-8 hours

## Context

The TUI provides real-time visualization of agent collaboration using the Bubbletea framework (Elm architecture: Model/View/Update). Updates appear within 100ms of events via EventBus subscription.

**Layout** (contracts/cli-interface.md:474-498):
```
┌─────────────────────────────────────────────────────────┐
│ Session: ID │ Task │ Duration │ Cost                    │
├──────────────────────────┬──────────────────────────────┤
│                          │                              │
│  Turn History            │  File Viewer                 │
│  (turn list)             │  (file content)              │
│                          │                              │
├──────────────────────────┴──────────────────────────────┤
│ Status │ Files │ Keys                                   │
└─────────────────────────────────────────────────────────┘
```

**Keyboard Shortcuts**: ↑/↓ (turns), Tab (files), PgUp/PgDn (scroll), q (quit), Ctrl+C (pause)

## Detailed Guidance

### T052-T059: TUI Implementation

Create `internal/tui/model.go`:
```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/yourusername/collab/pkg/types"
    "github.com/yourusername/collab/internal/events"
)

type Model struct {
    session       *types.Session
    turnHistory   []types.Turn
    currentFile   string
    fileContent   string
    eventSub      *events.Subscriber
    
    // UI state
    selectedTurn  int
    selectedPane  string // "turns" or "file"
    
    width         int
    height        int
}

func NewModel(session *types.Session, bus *events.EventBus) Model {
    sub := bus.Subscribe("tui",
        types.EventTurnStarted,
        types.EventTurnCompleted,
        types.EventFileUpdated,
        types.EventSessionCompleted,
    )
    
    return Model{
        session:      session,
        turnHistory:  session.TurnHistory,
        currentFile:  "shared_context.md",
        eventSub:     sub,
        selectedPane: "turns",
    }
}

func (m Model) Init() tea.Cmd {
    return waitForEvent(m.eventSub)
}
```

Create `internal/tui/update.go`:
```go
package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q":
            return m, tea.Quit
        case "ctrl+c":
            // Pause session
            return m, pauseSession
        case "up":
            if m.selectedPane == "turns" && m.selectedTurn > 0 {
                m.selectedTurn--
            }
        case "down":
            if m.selectedPane == "turns" && m.selectedTurn < len(m.turnHistory)-1 {
                m.selectedTurn++
            }
        case "tab":
            m.cycleFile()
        }
        
    case events.Event:
        return m.handleEvent(msg)
        
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    
    return m, waitForEvent(m.eventSub)
}

func (m *Model) handleEvent(event events.Event) (Model, tea.Cmd) {
    switch event.Type {
    case types.EventTurnCompleted:
        payload := event.Payload.(events.TurnCompletedPayload)
        m.turnHistory = append(m.turnHistory, *payload.Turn)
        
    case types.EventFileUpdated:
        payload := event.Payload.(events.FileUpdatedPayload)
        if m.isCurrentFile(payload.Path) {
            m.reloadFile()
        }
    }
    
    return *m, nil
}
```

Create `internal/tui/view.go`:
```go
package tui

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
    if m.width == 0 {
        return "Initializing..."
    }
    
    // Header
    header := renderHeader(m.session, m.width)
    
    // Split panes
    leftWidth := m.width / 2
    rightWidth := m.width - leftWidth
    paneHeight := m.height - 3 // Header + status bar
    
    leftPane := renderTurnList(m.turnHistory, m.selectedTurn, leftWidth, paneHeight)
    rightPane := renderFileViewer(m.fileContent, m.currentFile, rightWidth, paneHeight)
    
    // Status bar
    statusBar := renderStatusBar(m, m.width)
    
    return lipgloss.JoinVertical(lipgloss.Left,
        header,
        lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane),
        statusBar,
    )
}
```

Implement components (T055-T058) in `internal/tui/components/`:
- `turn_list.go`: Scrollable turn history
- `file_viewer.go`: File content with scrolling
- `status_bar.go`: Bottom status with keybindings
- `header.go`: Session info header

Create `internal/tui/styles.go` (T059):
```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
    HeaderStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("15")).
        Background(lipgloss.Color("62"))
        
    TurnStyle = lipgloss.NewStyle().
        Padding(0, 1)
        
    SelectedTurnStyle = TurnStyle.Copy().
        Background(lipgloss.Color("240"))
)
```

## Test Strategy

Snapshot tests for view rendering:
```go
func TestTUIView(t *testing.T) {
    model := NewModel(mockSession(), mockEventBus())
    model.width = 80
    model.height = 24
    
    view := model.View()
    
    // Verify layout elements present
    assert.Contains(t, view, "Session:")
    assert.Contains(t, view, "Turn History")
    assert.Contains(t, view, "File Viewer")
}
```

## Definition of Done

- [ ] All 8 subtasks completed
- [ ] TUI renders in 80x24 terminal
- [ ] Updates appear <100ms after events
- [ ] Keyboard shortcuts work
- [ ] File viewer displays markdown correctly
- [ ] TUI exits cleanly on 'q'
- [ ] Ctrl+C pauses session

## References

- [spec.md](../spec.md): FR-042 to FR-051, SC-004
- [plan.md](../plan.md): Phase 8 (lines 557-597)
- [contracts/cli-interface.md](../contracts/cli-interface.md): TUI specification
