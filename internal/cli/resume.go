package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/spf13/cobra"
)

// NewResumeCommand creates the resume command
func NewResumeCommand() *cobra.Command {
	var watchMode bool

	cmd := &cobra.Command{
		Use:   "resume <session-id>",
		Short: "Resume a paused collaboration session",
		Long: `Resume a paused collaboration session from where it left off.

The session must be in 'paused' status. Completed, incomplete, or errored
sessions cannot be resumed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			// Get workspace directory from flag or default
			workspaceRoot, _ := cmd.Flags().GetString("workspace")
			if workspaceRoot == "" {
				workspaceRoot = getDefaultWorkspaceDir()
			}

			// Load session
			workspaceDir := filepath.Join(workspaceRoot, sessionID)
			session, err := orchestrator.LoadSession(workspaceDir)
			if err != nil {
				PrintError("Failed to load session %s: %v", sessionID, err)
				PrintInfo("Check that the session exists with: collab list")
				return ExitWithCode(ExitValidation)
			}

			// Validate session can be resumed
			if session.Status != types.SessionPaused {
				PrintError("Session %s cannot be resumed (status: %s)", sessionID, session.Status)
				PrintInfo("Only paused sessions can be resumed")
				return ExitWithCode(ExitValidation)
			}

			PrintSuccess("Resuming session: %s", sessionID)
			PrintInfo("Status: paused at turn %d/%d", session.CurrentTurn, session.MaxTurns)
			PrintInfo("Previous cost: $%.2f", session.TotalCost)
			PrintInfo("Continuing from turn %d...", session.CurrentTurn+1)

			// Get API key (optional - claude CLI will use saved credentials if not set)
			apiKey := os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				PrintInfo("Using Claude Code saved credentials (from /login)")
				PrintInfo("If authentication fails, either run /login in Claude Code or set ANTHROPIC_API_KEY")
			}

			// Resume session
			if watchMode {
				PrintInfo("Watch mode not yet implemented - running headless")
				return resumeHeadless(session, apiKey)
			}

			return resumeHeadless(session, apiKey)
		},
	}

	cmd.Flags().BoolVar(&watchMode, "watch", false, "Launch interactive TUI to monitor progress")

	return cmd
}

// resumeHeadless resumes the session without TUI
func resumeHeadless(session *types.Session, apiKey string) error {
	// Create event bus
	eventBus := events.NewEventBus(100)
	defer eventBus.Shutdown()

	// Subscribe to events for progress display
	subscriber := eventBus.Subscribe("cli-resume")
	go displayEvents(subscriber.Events(), session)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(session, eventBus, apiKey)

	// Initialize orchestrator
	ctx := context.Background()
	if err := orch.Initialize(ctx); err != nil {
		PrintError("Failed to initialize orchestrator: %v", err)
		return ExitWithCode(ExitError)
	}
	defer orch.Shutdown()

	fmt.Println()

	// Resume orchestrator
	startTime := time.Now()
	if err := orch.Resume(ctx); err != nil {
		// Check if it was a user interrupt
		if ctx.Err() == context.Canceled {
			PrintInfo("Session paused by user")
			return ExitWithCode(ExitUserInterrupt)
		}

		PrintError("Session error: %v", err)
		return ExitWithCode(ExitError)
	}

	// Display final summary
	duration := time.Since(startTime)
	return displayFinalSummary(session, duration)
}
