---
work_package_id: WP06
title: "Agent Manager"
priority: P1
status: planned
subtasks:
  - T031
  - T032
  - T033
  - T034
  - T035
dependencies:
  - WP01
  - WP03
  - WP05
lane: planned
history:
  - timestamp: "2025-11-23"
    action: created
    status: planned
---

# Work Package WP06: Agent Manager

**Objective**: Integrate claude-agent-sdk-go for agent lifecycle management, health monitoring, and metrics collection with <1s crash detection and no orphaned processes.

**Priority**: P1 (Required by WP07 orchestrator)

**Estimated Effort**: 5-7 hours

## Context

The Agent Manager wraps the claude-agent-sdk-go, spawning agent processes, monitoring their health, collecting metrics (tokens, cost), and ensuring graceful shutdown. Each agent is a separate OS process running the Claude CLI with MCP tool access.

**Key Requirements**:
- SDK integration for agent subprocess management
- Health checks every 5s, detect crashes within 1s (SC-005)
- Metrics collection: tokens, cost within 1% accuracy (SC-010)
- Turn timeout enforcement with context.WithTimeout
- Clean shutdown: SIGTERM → 5s grace → SIGKILL

## Detailed Guidance

### T031: Implement Agent struct

Create `internal/agent/agent.go`:
```go
package agent

import (
    "context"
    "github.com/yourusername/collab/pkg/types"
    "github.com/connerohnesorge/claude-agent-sdk-go/client"
)

type Agent struct {
    ID           string
    Name         string
    Role         string
    SystemPrompt string
    Model        string
    WorkspaceDir string
    
    client       *client.Client
    processPID   int
    isHealthy    bool
}

func (a *Agent) Start(mcpEndpoint string) error {
    // Initialize SDK client
    // Configure environment variables
    // Start Claude CLI subprocess
    return nil
}

func (a *Agent) Stop() error {
    // Send SIGTERM
    // Wait 5s
    // Send SIGKILL if needed
    return nil
}

func (a *Agent) IsHealthy() bool {
    return a.isHealthy
}
```

### T032-T034: Implement manager components

Create `internal/agent/manager.go`:
```go
package agent

import (
    "context"
    "fmt"
    "time"
)

type Manager struct {
    agents   map[string]*Agent
    eventBus *events.EventBus
}

func NewManager(eventBus *events.EventBus) *Manager {
    return &Manager{
        agents:   make(map[string]*Agent),
        eventBus: eventBus,
    }
}

func (m *Manager) SpawnAgent(cfg *types.Agent, mcpEndpoint string) (*Agent, error) {
    agent := &Agent{
        ID:           cfg.ID,
        Name:         cfg.Name,
        Role:         cfg.Role,
        SystemPrompt: cfg.SystemPrompt,
        Model:        cfg.Model,
        WorkspaceDir: cfg.WorkspaceDir,
    }
    
    if err := agent.Start(mcpEndpoint); err != nil {
        return nil, fmt.Errorf("start agent: %w", err)
    }
    
    m.agents[agent.ID] = agent
    
    // Start health check goroutine
    go m.monitorHealth(agent)
    
    return agent, nil
}

func (m *Manager) StartTurn(ctx context.Context, agent *Agent, query string) (*TurnResult, error) {
    // Create timeout context
    turnCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()
    
    start := time.Now()
    
    // Send query to agent via SDK
    response, err := agent.client.Send(turnCtx, query)
    if err != nil {
        return nil, fmt.Errorf("agent turn failed: %w", err)
    }
    
    // Collect metrics
    duration := time.Since(start)
    tokens := response.TokensUsed
    cost := calculateCost(tokens, agent.Model)
    
    return &TurnResult{
        Duration:   duration,
        TokensUsed: tokens,
        Cost:       cost,
        Response:   response.Content,
    }, nil
}

func (m *Manager) monitorHealth(agent *Agent) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        // Check if process is still running
        isAlive := checkProcessAlive(agent.processPID)
        
        if !isAlive && agent.isHealthy {
            // Process crashed
            agent.isHealthy = false
            m.eventBus.Publish(events.Event{
                Type: types.EventSessionError,
                Payload: events.SessionErrorPayload{
                    Error: fmt.Sprintf("Agent %s crashed", agent.ID),
                },
            })
        }
        
        agent.isHealthy = isAlive
    }
}

func (m *Manager) Shutdown() error {
    for _, agent := range m.agents {
        agent.Stop()
    }
    return nil
}
```

### T035: Implement metrics collection

Create `internal/agent/metrics.go`:
```go
package agent

const (
    // Model pricing (per 1M tokens)
    SonnetInputPrice  = 3.00  // $3/1M input tokens
    SonnetOutputPrice = 15.00 // $15/1M output tokens
)

func calculateCost(tokens int, model string) float64 {
    // Simplified: assume 50/50 input/output split
    tokensMillions := float64(tokens) / 1_000_000.0
    
    switch model {
    case "claude-sonnet-4":
        avgPrice := (SonnetInputPrice + SonnetOutputPrice) / 2.0
        return tokensMillions * avgPrice
    default:
        return 0.0
    }
}

type TurnResult struct {
    Duration   time.Duration
    TokensUsed int
    Cost       float64
    Response   string
    ToolCalls  int
}
```

## Test Strategy

Use mocks for SDK client (no actual agent spawning in unit tests):
```go
func TestSpawnAgent(t *testing.T) {
    bus := events.NewEventBus(10)
    defer bus.Shutdown()
    
    mgr := NewManager(bus)
    
    cfg := &types.Agent{
        ID:           "agent_1",
        Name:         "Test Agent",
        SystemPrompt: "You are a test agent",
        Model:        "claude-sonnet-4",
    }
    
    // Mock SDK integration
    agent, err := mgr.SpawnAgent(cfg, "stdio://mcp-server")
    require.NoError(t, err)
    assert.Equal(t, "agent_1", agent.ID)
}
```

## Definition of Done

- [ ] All 5 subtasks completed
- [ ] Agent struct wraps SDK client
- [ ] Health checks detect crashes within 1s
- [ ] Metrics collection accurate within 1%
- [ ] Turn timeout enforced
- [ ] Clean shutdown tested (no orphaned processes)
- [ ] Tests use mocks for SDK

## References

- [spec.md](../spec.md): FR-036 to FR-041, SC-005, SC-010
- [plan.md](../plan.md): Phase 5 (lines 425-459), Decision on SDK integration
