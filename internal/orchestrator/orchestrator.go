package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dphaener/shiki-cli/internal/agent"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// Orchestrator manages the collaboration session lifecycle
type Orchestrator struct {
	session      *types.Session
	agentManager *agent.Manager
	eventBus     *events.EventBus
	apiKey       string
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(
	session *types.Session,
	eventBus *events.EventBus,
	apiKey string,
) *Orchestrator {
	return &Orchestrator{
		session:  session,
		eventBus: eventBus,
		apiKey:   apiKey,
	}
}

// Initialize sets up the orchestrator components (agent manager)
// Note: MCP server removed - agents use file-based communication instead
func (o *Orchestrator) Initialize(ctx context.Context) error {
	// Create agent manager
	o.agentManager = agent.NewManager(o.eventBus, o.session.ID, o.apiKey)

	// Spawn agent1 (no MCP tools - uses built-in tools only)
	if _, err := o.agentManager.SpawnAgent(ctx, &o.session.Agent1); err != nil {
		return fmt.Errorf("spawn agent1: %w", err)
	}

	// Spawn agent2 (no MCP tools - uses built-in tools only)
	if _, err := o.agentManager.SpawnAgent(ctx, &o.session.Agent2); err != nil {
		return fmt.Errorf("spawn agent2: %w", err)
	}

	// Emit SessionStarted event
	o.eventBus.Publish(events.NewSessionStarted(o.session))

	return nil
}

// Run executes the main orchestration loop
func (o *Orchestrator) Run(ctx context.Context) error {
	// Set up signal handling for pause
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Handle signals in goroutine
	go func() {
		select {
		case <-sigChan:
			// Graceful pause
			if err := o.Pause(); err != nil {
				fmt.Fprintf(os.Stderr, "Error pausing session: %v\n", err)
			}
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	// Main turn loop
	for {
		// Check if we should continue
		shouldContinue, deliverablePath, err := o.ShouldContinue()
		if err != nil {
			return fmt.Errorf("check continuation: %w", err)
		}

		if !shouldContinue {
			// Session completed or max turns reached
			if deliverablePath != "" {
				// Completed successfully
				if err := CompleteSession(o.session, deliverablePath); err != nil {
					return fmt.Errorf("complete session: %w", err)
				}
				o.eventBus.Publish(events.NewSessionCompleted(o.session, deliverablePath))
			} else {
				// Max turns reached without completion
				if err := MarkIncomplete(o.session); err != nil {
					return fmt.Errorf("mark incomplete: %w", err)
				}
				o.eventBus.Publish(events.NewSessionIncomplete(o.session))
			}
			break
		}

		// Check for cancellation (pause request)
		select {
		case <-ctx.Done():
			return nil // Paused
		default:
		}

		// Validate turn execution is possible
		if err := o.ValidateTurnExecution(); err != nil {
			return fmt.Errorf("validate turn: %w", err)
		}

		// Get current agent (alternates each turn)
		currentAgent := o.GetCurrentAgent()

		// Execute turn
		turn, err := o.ExecuteTurn(ctx, currentAgent)
		if err != nil {
			// Handle turn error
			if turnErr := o.handleTurnError(err, turn); turnErr != nil {
				return fmt.Errorf("handle turn error: %w", turnErr)
			}
			return fmt.Errorf("turn execution: %w", err)
		}

		// Update session state
		if err := o.UpdateSessionFromTurn(turn); err != nil {
			return fmt.Errorf("update session: %w", err)
		}
	}

	return nil
}

// Resume restores a paused session and continues execution
func (o *Orchestrator) Resume(ctx context.Context) error {
	// Validate session can be resumed
	if err := ResumeSession(o.session); err != nil {
		return fmt.Errorf("validate resume: %w", err)
	}

	// Save resumed status
	if err := SaveSession(o.session); err != nil {
		return fmt.Errorf("save resumed session: %w", err)
	}

	// Emit SessionResumed event
	o.eventBus.Publish(events.NewSessionResumed(o.session))

	// Re-initialize if needed
	if o.agentManager == nil {
		if err := o.Initialize(ctx); err != nil {
			return fmt.Errorf("re-initialize: %w", err)
		}
	}

	// Continue execution
	return o.Run(ctx)
}

// Pause gracefully pauses the session and saves state
func (o *Orchestrator) Pause() error {
	if err := PauseSession(o.session); err != nil {
		return fmt.Errorf("pause session: %w", err)
	}

	// Emit SessionPaused event
	o.eventBus.Publish(events.NewSessionPaused(o.session))

	return nil
}

// Shutdown gracefully shuts down all orchestrator components
func (o *Orchestrator) Shutdown() error {
	var errs []error

	// Stop agents
	if o.agentManager != nil {
		if err := o.agentManager.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("shutdown agent manager: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// handleTurnError processes a turn error and updates session state
func (o *Orchestrator) handleTurnError(err error, turn *types.Turn) error {
	// Record the error turn in history if it exists
	if turn != nil {
		o.session.TurnHistory = append(o.session.TurnHistory, *turn)
		o.session.CurrentTurn++
	}

	// Mark session as error
	if markErr := MarkError(o.session, err.Error()); markErr != nil {
		return fmt.Errorf("mark error: %w", markErr)
	}

	// Emit SessionError event
	o.eventBus.Publish(events.NewSessionError(o.session, err.Error()))

	return nil
}

// GetSession returns the current session
func (o *Orchestrator) GetSession() *types.Session {
	return o.session
}

// GetProgress returns current progress statistics
func (o *Orchestrator) GetProgress() map[string]interface{} {
	return map[string]interface{}{
		"session_id":   o.session.ID,
		"current_turn": o.session.CurrentTurn,
		"max_turns":    o.session.MaxTurns,
		"status":       o.session.Status,
		"total_cost":   o.session.TotalCost,
		"total_tokens": o.session.TotalTokens,
		"duration":     o.getDuration(),
		"completion":   o.GetCompletionSummary(),
	}
}

// getDuration calculates session duration
func (o *Orchestrator) getDuration() time.Duration {
	if o.session.StartedAt == nil {
		return 0
	}

	endTime := time.Now()
	if o.session.CompletedAt != nil {
		endTime = *o.session.CompletedAt
	}

	return endTime.Sub(*o.session.StartedAt)
}
