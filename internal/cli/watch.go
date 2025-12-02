package cli

import (
	"context"
	"path/filepath"

	"github.com/dphaener/shiki-cli/internal/config"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/logging"
	"github.com/dphaener/shiki-cli/internal/storage"
	"github.com/dphaener/shiki-cli/internal/tui"
	"github.com/spf13/cobra"
)

// NewWatchCommand creates the watch command
func NewWatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <session-id>",
		Short: "Attach TUI to monitor a session",
		Long: `Attach an interactive TUI to monitor a running or completed session.

For running sessions, displays real-time updates via EventBus.
For completed sessions, allows inspection of final state and artifacts.

Keyboard shortcuts:
  ↑/↓, j/k     Navigate turns
  ←/→, h/l     Switch panes
  Tab          Cycle files
  PgUp/PgDn    Scroll file content
  q, Esc       Quit
  Ctrl+C       Pause session (if running)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			// Load configuration
			cfg := config.Default()

			// Find session workspace directory
			workspaceDir := filepath.Join(cfg.WorkspaceDir, sessionID)

			// Load session state
			session, err := storage.LoadSessionState(workspaceDir)
			if err != nil {
				PrintError("Failed to load session: %s", err.Error())
				return ExitWithCode(ExitError)
			}

			// Create event bus for TUI updates
			// For completed sessions, this will be mostly inactive
			// For running sessions, this should connect to the live orchestrator's bus
			bus := events.NewEventBus(100)
			defer bus.Shutdown()

			// Start assistant event logger
			assistantLogger, err := logging.NewAssistantEventLogger()
			if err != nil {
				PrintWarning("Failed to initialize assistant logger: %v", err)
			} else {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				assistantLogger.Start(ctx, bus)
				defer assistantLogger.Close()
			}

			PrintInfo("Launching TUI for session %s...", sessionID)
			PrintInfo("Status: %s", session.Status)

			// Run TUI
			if err := tui.Run(session, bus); err != nil {
				if tui.IsPausedError(err) {
					PrintInfo("Session paused by user")
					return ExitWithCode(ExitUserInterrupt)
				}
				PrintError("TUI error: %s", err.Error())
				return ExitWithCode(ExitError)
			}

			return ExitWithCode(ExitSuccess)
		},
	}

	return cmd
}
