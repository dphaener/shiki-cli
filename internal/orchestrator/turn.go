package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// ExecuteTurn runs a single turn for the specified agent
func (o *Orchestrator) ExecuteTurn(ctx context.Context, agentCfg *types.Agent) (*types.Turn, error) {
	turnNumber := o.session.CurrentTurn

	// Create turn record
	turn := &types.Turn{
		Number:    turnNumber,
		AgentID:   agentCfg.ID,
		StartedAt: time.Now(),
		Status:    types.TurnInProgress,
	}

	// Emit TurnStarted event
	o.eventBus.Publish(events.NewTurnStarted(turn, agentCfg.ID, o.session.ID))

	// Build query for agent (includes task context)
	query := o.buildQuery(agentCfg)

	// Execute turn via agent manager with timeout
	result, err := o.agentManager.StartTurn(ctx, agentCfg.ID, query, turnNumber)
	if err != nil {
		// Handle turn error
		turn.Status = types.TurnError
		turn.ErrorMessage = err.Error()
		now := time.Now()
		turn.CompletedAt = &now
		turn.DurationMS = time.Since(turn.StartedAt).Milliseconds()

		// Emit TurnError event
		o.eventBus.Publish(events.NewTurnError(turn, o.session.ID, err.Error()))

		return turn, fmt.Errorf("turn execution failed: %w", err)
	}

	// Complete turn successfully
	now := time.Now()
	turn.CompletedAt = &now
	turn.DurationMS = result.Duration.Milliseconds()
	turn.TokensUsed = result.TokensUsed
	turn.Cost = result.Cost
	turn.ToolCalls = result.ToolCalls
	turn.Status = types.TurnCompleted

	// Update agent-level metrics
	agentCfg.TotalTurns++
	agentCfg.TotalTokens += result.TokensUsed
	agentCfg.TotalCost += result.Cost

	// Emit TurnCompleted event
	o.eventBus.Publish(events.NewTurnCompleted(turn, o.session.ID))

	return turn, nil
}

// buildQuery constructs the query/prompt for an agent's turn
func (o *Orchestrator) buildQuery(agentCfg *types.Agent) string {
	// The query should include:
	// 1. The agent's role and current context
	// 2. Instructions to use MCP tools for collaboration
	// 3. Reference to shared context and messages

	query := fmt.Sprintf(`You are %s, a %s.

Your task is to collaborate with your partner agent to complete the following task:
%s

You have access to the following MCP tools for collaboration (use the full tool names exactly as shown):
- mcp__collaboration__send_message: Send a message to your partner
- mcp__collaboration__read_messages: Read messages from your partner
- mcp__collaboration__write_shared_context: Write to shared context (both agents can see)
- mcp__collaboration__read_shared_context: Read shared context
- mcp__collaboration__update_memory: Update your private memory
- mcp__collaboration__read_memory: Read your private memory
- mcp__collaboration__submit_deliverable: Submit the final deliverable when ready

This is turn %d of %d. Please:
1. Read any new messages from your partner using mcp__collaboration__read_messages
2. Review the shared context using mcp__collaboration__read_shared_context
3. Perform your role's responsibilities
4. Communicate with your partner using mcp__collaboration__send_message
5. Submit a deliverable if you have completed the task (both agents must submit matching deliverables)

Begin your turn.`,
		agentCfg.Name,
		agentCfg.Role,
		o.session.TaskName,
		o.session.CurrentTurn+1,
		o.session.MaxTurns,
	)

	return query
}

// GetCurrentAgent returns the agent that should execute the current turn
func (o *Orchestrator) GetCurrentAgent() *types.Agent {
	// Alternate between agent1 and agent2
	if o.session.CurrentTurn%2 == 0 {
		return &o.session.Agent1
	}
	return &o.session.Agent2
}

// UpdateSessionFromTurn updates session state after a turn completes
func (o *Orchestrator) UpdateSessionFromTurn(turn *types.Turn) error {
	// Append turn to history
	o.session.TurnHistory = append(o.session.TurnHistory, *turn)

	// Update session-level metrics
	o.session.TotalTokens += turn.TokensUsed
	o.session.TotalCost += turn.Cost

	// Increment turn counter
	o.session.CurrentTurn++

	// Persist state after every turn
	if err := SaveSession(o.session); err != nil {
		return fmt.Errorf("save session after turn: %w", err)
	}

	return nil
}

// ValidateTurnExecution checks if a turn can be executed
func (o *Orchestrator) ValidateTurnExecution() error {
	// Check if session is in valid state
	if o.session.Status != types.SessionRunning {
		return fmt.Errorf("session is not running (status: %s)", o.session.Status)
	}

	// Check if max turns reached
	if o.session.CurrentTurn >= o.session.MaxTurns {
		return fmt.Errorf("max turns (%d) reached", o.session.MaxTurns)
	}

	// Check if agents are healthy
	agent1, err := o.agentManager.GetAgent(o.session.Agent1.ID)
	if err != nil {
		return fmt.Errorf("agent1 not found: %w", err)
	}
	if !agent1.IsHealthy() {
		return fmt.Errorf("agent1 is not healthy")
	}

	agent2, err := o.agentManager.GetAgent(o.session.Agent2.ID)
	if err != nil {
		return fmt.Errorf("agent2 not found: %w", err)
	}
	if !agent2.IsHealthy() {
		return fmt.Errorf("agent2 is not healthy")
	}

	return nil
}
