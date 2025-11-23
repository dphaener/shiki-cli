package cli

import (
	"github.com/spf13/cobra"
)

// NewWatchCommand creates the watch command
func NewWatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <session-id>",
		Short: "Attach TUI to monitor a session",
		Long: `Attach an interactive TUI to monitor a running or completed session.

For running sessions, displays real-time updates.
For completed sessions, allows replay and inspection.

Note: TUI functionality will be implemented in WP09.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			PrintError("Watch mode not yet implemented (WP09)")
			PrintInfo("Session ID: %s", sessionID)
			PrintInfo("For now, use: collab show %s", sessionID)

			return ExitWithCode(ExitError)
		},
	}

	return cmd
}
