---
work_package_id: WP07
title: "Orchestrator"
priority: P1
status: planned
subtasks:
  - T036
  - T037
  - T038
  - T039
  - T040
  - T041
dependencies:
  - WP01
  - WP02
  - WP03
  - WP06
lane: planned
history:
  - timestamp: "2025-11-23"
    action: created
    status: planned
---

# Work Package WP07: Orchestrator

**Objective**: Implement session lifecycle and turn-based execution loop with pause/resume (zero data loss), completion detection, and max_turns enforcement.

**Priority**: P1 (Core orchestration logic, required by WP08 CLI)

**Estimated Effort**: 8-10 hours

## Context

The Orchestrator is the heart of the system, managing the turn-based execution loop (agent1 → agent2 → agent1...), persisting state after each turn, handling pause/resume with exact state restoration, and detecting completion when both agents submit matching deliverables.

**Critical Requirements**:
- Sequential turn execution (no concurrent agents per CON-003)
- State persistence after every turn (FR-015)
- Pause on SIGTERM/SIGINT with graceful save
- Resume from exact turn number with all history
- Completion on matching deliverable submissions
- Max turns enforcement → status=incomplete

## Detailed Guidance

### T036: Implement Session lifecycle

Create `internal/orchestrator/session.go`:
```go
package orchestrator

import (
    "github.com/yourusername/collab/pkg/types"
    "github.com/yourusername/collab/internal/storage"
    "github.com/yourusername/collab/internal/template"
)

func NewSession(tmpl *template.TaskTemplate, workspaceDir string) *types.Session {
    now := time.Now()
    
    return &types.Session{
        ID:               generateSessionID(),
        TaskName:         extractTaskName(tmpl.TaskBody),
        WorkspaceDir:     workspaceDir,
        Status:           types.SessionRunning,
        CreatedAt:        now,
        StartedAt:        &now,
        CurrentTurn:      0,
        MaxTurns:         tmpl.MaxTurns,
        Agent1:           agentFromTemplate(tmpl, "agent_1"),
        Agent2:           agentFromTemplate(tmpl, "agent_2"),
        TurnHistory:      []types.Turn{},
        TotalCost:        0.0,
        TotalTokens:      0,
    }
}

func (s *types.Session) SaveState() error {
    return storage.SaveSessionState(s)
}

func LoadSession(workspaceDir string) (*types.Session, error) {
    return storage.LoadSessionState(workspaceDir)
}
```

### T037-T041: Turn execution and orchestration loop

Create `internal/orchestrator/orchestrator.go`:
```go
package orchestrator

import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

type Orchestrator struct {
    session      *types.Session
    agentManager *agent.Manager
    mcpServer    *mcp.Server
    eventBus     *events.EventBus
}

func (o *Orchestrator) Run(ctx context.Context) error {
    // Set up signal handling for pause
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    go func() {
        <-sigChan
        o.session.Status = types.SessionPaused
        o.session.SaveState()
        cancel()
    }()
    
    // Main turn loop
    for o.session.CurrentTurn < o.session.MaxTurns {
        select {
        case <-ctx.Done():
            return nil // Paused
        default:
        }
        
        // Alternate agents
        currentAgent := o.session.Agent1
        if o.session.CurrentTurn%2 == 1 {
            currentAgent = o.session.Agent2
        }
        
        // Execute turn
        turn, err := o.executeTurn(ctx, &currentAgent)
        if err != nil {
            o.handleTurnError(err)
            return err
        }
        
        // Update session
        o.session.TurnHistory = append(o.session.TurnHistory, *turn)
        o.session.TotalCost += turn.Cost
        o.session.TotalTokens += turn.TokensUsed
        o.session.CurrentTurn++
        
        // Persist state
        if err := o.session.SaveState(); err != nil {
            return fmt.Errorf("save state: %w", err)
        }
        
        // Check completion
        if o.checkCompletion() {
            o.session.Status = types.SessionCompleted
            o.session.SaveState()
            return nil
        }
    }
    
    // Max turns reached without completion
    o.session.Status = types.SessionIncomplete
    o.session.SaveState()
    return nil
}

func (o *Orchestrator) executeTurn(ctx context.Context, agent *types.Agent) (*types.Turn, error) {
    turn := &types.Turn{
        Number:    o.session.CurrentTurn,
        AgentID:   agent.ID,
        StartedAt: time.Now(),
        Status:    types.TurnInProgress,
    }
    
    // Emit TurnStarted event
    o.eventBus.Publish(events.NewTurnStarted(turn, agent.ID, o.session.ID))
    
    // Execute via agent manager
    result, err := o.agentManager.StartTurn(ctx, agent, buildQuery(o.session))
    if err != nil {
        turn.Status = types.TurnError
        turn.ErrorMessage = err.Error()
        return turn, err
    }
    
    // Complete turn
    now := time.Now()
    turn.CompletedAt = &now
    turn.DurationMS = int64(result.Duration.Milliseconds())
    turn.TokensUsed = result.TokensUsed
    turn.Cost = result.Cost
    turn.ToolCalls = result.ToolCalls
    turn.Status = types.TurnCompleted
    
    // Emit TurnCompleted event
    o.eventBus.Publish(events.Event{
        Type:      types.EventTurnCompleted,
        SessionID: o.session.ID,
        Timestamp: time.Now(),
        Payload:   events.TurnCompletedPayload{Turn: turn},
    })
    
    return turn, nil
}

func (o *Orchestrator) checkCompletion() bool {
    // Check if SessionCompleted event was emitted by MCP server
    // (happens when both agents submit matching deliverables)
    return o.session.DeliverablePath != ""
}
```

## Test Strategy

```go
func TestOrchestrator(t *testing.T) {
    // Mock agent manager and MCP server
    bus := events.NewEventBus(100)
    defer bus.Shutdown()
    
    session := &types.Session{
        ID:          "test",
        CurrentTurn: 0,
        MaxTurns:    4,
        Status:      types.SessionRunning,
    }
    
    orch := &Orchestrator{
        session:  session,
        eventBus: bus,
        // ... mock managers
    }
    
    ctx := context.Background()
    err := orch.Run(ctx)
    require.NoError(t, err)
    
    // Verify turn alternation
    assert.Equal(t, 4, len(session.TurnHistory))
    assert.Equal(t, "agent_1", session.TurnHistory[0].AgentID)
    assert.Equal(t, "agent_2", session.TurnHistory[1].AgentID)
}
```

## Definition of Done

- [ ] Session creation and state persistence
- [ ] Turn execution with metrics collection
- [ ] Turn alternation (agent1 → agent2 → agent1...)
- [ ] Pause handling (SIGTERM/SIGINT)
- [ ] Resume from exact turn with state restoration
- [ ] Completion detection
- [ ] Max turns enforcement
- [ ] Tests with >90% coverage

## References

- [spec.md](../spec.md): FR-011 to FR-019, SC-002
- [plan.md](../plan.md): Phase 6 (lines 461-503), Decision 4 (graceful degradation)
