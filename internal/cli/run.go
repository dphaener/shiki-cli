package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/internal/storage"
	"github.com/darinhaener/collab/internal/template"
	"github.com/darinhaener/collab/internal/tui"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/spf13/cobra"
)

// NewRunCommand creates the run command
func NewRunCommand() *cobra.Command {
	var (
		watchMode bool
		sessionID string
	)

	cmd := &cobra.Command{
		Use:   "run <task-file>",
		Short: "Start a new collaboration session",
		Long: `Start a new collaboration session from a task template file.

The task template defines the agents, their roles, and the task to complete.
The session runs until both agents submit matching deliverables or max turns is reached.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskFile := args[0]

			// Get workspace directory from flag or default
			workspaceRoot, _ := cmd.Flags().GetString("workspace")
			if workspaceRoot == "" {
				workspaceRoot = getDefaultWorkspaceDir()
			}

			// Parse and validate template
			tmpl, err := template.ParseTemplate(taskFile)
			if err != nil {
				PrintError("Failed to parse template: %v", err)
				return ExitWithCode(ExitValidation)
			}

			if err := template.ValidateTemplate(tmpl); err != nil {
				PrintError("Template validation failed: %v", err)
				return ExitWithCode(ExitValidation)
			}

			// Create session
			session, workspaceDir, err := createSession(tmpl, taskFile, workspaceRoot, sessionID)
			if err != nil {
				PrintError("Failed to create session: %v", err)
				return ExitWithCode(ExitError)
			}

			PrintSuccess("Session created: %s", session.ID)
			PrintSuccess("Workspace: %s", workspaceDir)

			// Get API key (optional - claude CLI will use saved credentials if not set)
			apiKey := os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				PrintInfo("Using Claude Code saved credentials (from /login)")
				PrintInfo("If authentication fails, either run /login in Claude Code or set ANTHROPIC_API_KEY")
			}

			// Run session
			if watchMode {
			return runWithTUI(session, apiKey)
			}

			return runHeadless(session, apiKey)
		},
	}

	cmd.Flags().StringP("workspace", "w", "", "Workspace directory (default: ~/.local/share/collab-cli/sessions)")
	cmd.Flags().BoolVar(&watchMode, "watch", false, "Launch interactive TUI to monitor progress")
	cmd.Flags().StringVar(&sessionID, "session-id", "", "Override automatic session ID")

	return cmd
}

// createSession creates a new session and workspace
func createSession(tmpl *template.TaskTemplate, templatePath, workspaceRoot, sessionID string) (*types.Session, string, error) {
	// Create session
	session, err := orchestrator.NewSession(tmpl, templatePath, "")
	if err != nil {
		return nil, "", fmt.Errorf("create session: %w", err)
	}

	// Override session ID if provided
	if sessionID != "" {
		session.ID = sessionID
	}

	// Create workspace directory
	workspaceDir := filepath.Join(workspaceRoot, session.ID)
	if err := os.MkdirAll(workspaceDir, 0700); err != nil {
		return nil, "", fmt.Errorf("create workspace directory: %w", err)
	}

	// Create workspace structure from template (this creates the actual workspace in a nested dir)
	actualWorkspaceDir, err := storage.CreateWorkspace(session.ID, workspaceDir, tmpl.WorkspaceStructure)
	if err != nil {
		return nil, "", fmt.Errorf("create workspace structure: %w", err)
	}

	// Update session with the actual workspace directory
	session.WorkspaceDir = actualWorkspaceDir
	session.Agent1.WorkspaceDir = actualWorkspaceDir
	session.Agent2.WorkspaceDir = actualWorkspaceDir

	// Save initial session state
	if err := orchestrator.SaveSession(session); err != nil {
		return nil, "", fmt.Errorf("save session: %w", err)
	}

	return session, workspaceDir, nil
}

// runHeadless runs the session without TUI
func runHeadless(session *types.Session, apiKey string) error {
	// Create event bus
	eventBus := events.NewEventBus(100)
	defer eventBus.Shutdown()

	// Subscribe to events for progress display
	subscriber := eventBus.Subscribe("cli-display")
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

	PrintSuccess("Starting agents...")

	// Run orchestrator
	startTime := time.Now()
	if err := orch.Run(ctx); err != nil {
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

// runWithTUI runs the session with an interactive TUI
func runWithTUI(session *types.Session, apiKey string) error {
	// Create event bus
	eventBus := events.NewEventBus(100)
	defer eventBus.Shutdown()

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(session, eventBus, apiKey)

	// Initialize orchestrator
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := orch.Initialize(ctx); err != nil {
		PrintError("Failed to initialize orchestrator: %v", err)
		return ExitWithCode(ExitError)
	}
	defer orch.Shutdown()

	// Run orchestrator in background goroutine
	orchDone := make(chan error, 1)
	startTime := time.Now()
	go func() {
		orchDone <- orch.Run(ctx)
	}()

	// Run TUI in main thread (blocks until user exits)
	tuiErr := tui.Run(session, eventBus)

	// If TUI exited with pause, cancel orchestrator
	if tui.IsPausedError(tuiErr) {
		cancel() // Signal orchestrator to pause
		<-orchDone // Wait for orchestrator to finish
		PrintInfo("Session paused by user")
		return ExitWithCode(ExitUserInterrupt)
	}

	// If TUI exited with error (not pause), still cancel orchestrator
	if tuiErr != nil {
		cancel()
		<-orchDone
		PrintError("TUI error: %v", tuiErr)
		return ExitWithCode(ExitError)
	}

	// TUI exited normally (session completed or errored), wait for orchestrator
	orchErr := <-orchDone
	if orchErr != nil {
		// Check if it was a user interrupt
		if ctx.Err() == context.Canceled {
			PrintInfo("Session paused by user")
			return ExitWithCode(ExitUserInterrupt)
		}

		PrintError("Session error: %v", orchErr)
		return ExitWithCode(ExitError)
	}

	// Display final summary
	duration := time.Since(startTime)
	return displayFinalSummary(session, duration)
}

// displayEvents displays real-time events during session execution
func displayEvents(eventChan <-chan events.Event, session *types.Session) {
	for event := range eventChan {
		switch event.Type {
		case types.EventTurnStarted:
			if payload, ok := event.Payload.(events.TurnStartedPayload); ok {
				agentName := payload.AgentID
				// Get friendly agent name from session
				if session.Agent1.ID == payload.AgentID {
					agentName = session.Agent1.Name
				} else if session.Agent2.ID == payload.AgentID {
					agentName = session.Agent2.Name
				}
				PrintInfo("Turn %d: %s is thinking...", payload.Turn.Number, agentName)
			}
		case types.EventTurnCompleted:
			displayTurnSummary(event, session)
		case types.EventSessionCompleted:
			// Will be displayed in final summary
		}
	}
}

// displayTurnSummary displays a completed turn's summary
func displayTurnSummary(event events.Event, session *types.Session) {
	payload, ok := event.Payload.(events.TurnCompletedPayload)
	if !ok {
		return
	}

	turn := payload.Turn

	// Get agent name
	var agentName string
	if turn.AgentID == session.Agent1.ID {
		agentName = session.Agent1.Name
	} else {
		agentName = session.Agent2.Name
	}

	// Format duration
	durationSec := float64(turn.DurationMs) / 1000.0

	// Display turn summary
	fmt.Printf("\nTurn %d [%s]: %.1fs, %d tokens, $%.2f, %d tools\n",
		turn.Number, agentName, durationSec, turn.TokensUsed, turn.Cost, turn.ToolCalls)

	// Display messages sent
	if turn.MessagesSent > 0 {
		PrintInfo("→ Sent %d message(s)", turn.MessagesSent)
	}
}

// displayFinalSummary displays the final session summary
func displayFinalSummary(session *types.Session, duration time.Duration) error {
	fmt.Println()

	switch session.Status {
	case types.SessionCompleted:
		PrintSuccess("Session completed in %.1fs", duration.Seconds())
		PrintSuccess("Total cost: $%.2f (%s: $%.2f, %s: $%.2f)",
			session.TotalCost,
			session.Agent1.Name, session.Agent1.TotalCost,
			session.Agent2.Name, session.Agent2.TotalCost)
		PrintSuccess("Deliverable: %s", session.DeliverablePath)
		return ExitWithCode(ExitSuccess)

	case types.SessionIncomplete:
		PrintWarning("Session incomplete (max turns reached)")
		PrintInfo("Total cost: $%.2f", session.TotalCost)
		PrintInfo("Turns completed: %d/%d", session.CurrentTurn, session.MaxTurns)
		return ExitWithCode(ExitIncomplete)

	case types.SessionPaused:
		PrintInfo("Session paused")
		PrintInfo("Total cost so far: $%.2f", session.TotalCost)
		PrintInfo("Resume with: collab resume %s", session.ID)
		return ExitWithCode(ExitUserInterrupt)

	default:
		PrintError("Session ended with status: %s", session.Status)
		return ExitWithCode(ExitError)
	}
}

// getDefaultWorkspaceDir returns the default workspace directory
func getDefaultWorkspaceDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".collab", "sessions")
	}
	return filepath.Join(homeDir, ".local", "share", "collab-cli", "sessions")
}

// setupSignalHandler sets up signal handling for graceful shutdown
func setupSignalHandler(ctx context.Context, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		cancel()
	}()
}
