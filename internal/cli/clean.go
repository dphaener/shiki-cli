package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewCleanCommand creates the clean command
func NewCleanCommand() *cobra.Command {
	var (
		olderThan string
		status    string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove old or completed sessions",
		Long: `Remove collaboration sessions from the workspace directory.

Can filter by age or status. By default, prompts for confirmation before deleting.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get workspace directory
			workspaceRoot, _ := cmd.Flags().GetString("workspace")
			if workspaceRoot == "" {
				workspaceRoot = getDefaultWorkspaceDir()
			}

			// Load all sessions
			sessions, err := loadAllSessions(workspaceRoot)
			if err != nil {
				PrintError("Failed to load sessions: %v", err)
				return ExitWithCode(ExitError)
			}

			// Filter sessions to delete
			toDelete := filterSessionsForDeletion(sessions, status, olderThan)

			if len(toDelete) == 0 {
				fmt.Println("No sessions match the deletion criteria")
				return nil
			}

			// Display sessions to be deleted
			fmt.Printf("Sessions to be deleted (%d):\n", len(toDelete))
			for _, s := range toDelete {
				fmt.Printf("  - %s (%s, %s)\n", s.ID, s.Status, s.StartedAt.Format("2006-01-02"))
			}
			fmt.Println()

			// Confirm deletion unless --force
			if !force {
				fmt.Print("Are you sure you want to delete these sessions? (yes/no): ")
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response != "yes" && response != "y" {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			// Delete sessions
			deleted := 0
			for _, s := range toDelete {
				if err := os.RemoveAll(s.WorkspaceDir); err != nil {
					PrintWarning("Failed to delete session %s: %v", s.ID, err)
					continue
				}
				deleted++
			}

			PrintSuccess("Deleted %d session(s)", deleted)

			return nil
		},
	}

	cmd.Flags().StringP("workspace", "w", "", "Workspace directory (default: ~/.local/share/collab-cli/sessions)")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "Delete sessions older than duration (e.g., '7d', '24h')")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Delete sessions with specific status")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// filterSessionsForDeletion filters sessions based on deletion criteria
func filterSessionsForDeletion(sessions []*types.Session, status, olderThan string) []*types.Session {
	var filtered []*types.Session

	// Parse olderThan duration
	var cutoffTime time.Time
	if olderThan != "" {
		duration, err := parseDuration(olderThan)
		if err == nil {
			cutoffTime = time.Now().Add(-duration)
		}
	}

	for _, s := range sessions {
		// Filter by status
		if status != "" && !strings.EqualFold(string(s.Status), status) {
			continue
		}

		// Filter by age
		if !cutoffTime.IsZero() && s.StartedAt != nil {
			if s.StartedAt.After(cutoffTime) {
				continue
			}
		}

		filtered = append(filtered, s)
	}

	return filtered
}

// parseDuration parses duration strings like "7d", "24h", "30m"
func parseDuration(s string) (time.Duration, error) {
	// Handle days
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Standard time.Duration parsing
	return time.ParseDuration(s)
}
