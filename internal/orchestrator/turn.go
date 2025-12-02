package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
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
		turn.DurationMs = time.Since(turn.StartedAt).Milliseconds()

		// Emit TurnError event
		o.eventBus.Publish(events.NewTurnError(turn, o.session.ID, err.Error()))

		return turn, fmt.Errorf("turn execution failed: %w", err)
	}

	// Complete turn successfully
	now := time.Now()
	turn.CompletedAt = &now
	turn.DurationMs = result.Duration.Milliseconds()
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
// It injects all collaboration context directly into the prompt
func (o *Orchestrator) buildQuery(agentCfg *types.Agent) string {
	// Read all context for injection
	task, _ := o.ReadTask()
	messages, _ := o.ReadPartnerMessages(agentCfg.ID)
	sharedContext, _ := o.ReadSharedContext()
	memory, _ := o.ReadAgentMemory(agentCfg.ID)
	partnerName := o.GetPartnerName(agentCfg.ID)

	turnNumber := o.session.CurrentTurn + 1

	query := fmt.Sprintf(`You are %s, a %s.

This is turn %d of %d. You are collaborating with %s.

=== TASK ===
%s

=== MESSAGES FROM YOUR PARTNER ===
%s

=== SHARED CONTEXT ===
%s

=== YOUR PRIVATE MEMORY ===
%s

=== INSTRUCTIONS ===
You have built-in tools (Read, Write, Edit, Glob, Grep, Bash) to work with files in your workspace.

**File-Based Communication Protocol:**

To send a message to your partner:
  Use Write tool to create: messages/%s_turn_%02d.md
  (Your partner will see it on their next turn)

To update shared context (both agents can see):
  Use Write tool to update: shared_context.md

To update your private memory:
  Use Write tool to update: memory/%s_memory.md

To submit your final deliverable:
  Use Write tool to create: %s_deliverable.md
  (Session completes when BOTH agents submit matching deliverables)

**On your turn:**
1. Review the task, messages, shared context, and your memory (shown above)
2. Perform your role's responsibilities
3. Send a message to your partner if needed (write to messages/ directory)
4. Update shared context if you have findings to share
5. Update your memory to track your progress
6. When the task is complete, submit your deliverable

Begin your turn.`,
		agentCfg.Name,
		agentCfg.Role,
		turnNumber,
		o.session.MaxTurns,
		partnerName,
		task,
		messages,
		sharedContext,
		memory,
		agentCfg.ID,
		turnNumber,
		agentCfg.ID,
		agentCfg.ID,
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
