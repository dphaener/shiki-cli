package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/spf13/cobra"
)

// NewListCommand creates the list command
func NewListCommand() *cobra.Command {
	var (
		statusFilter string
		outputFormat string
		limit        int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all collaboration sessions",
		Long: `List all collaboration sessions in the workspace directory.

Sessions can be filtered by status and output in different formats (table, json, csv).`,
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

			// Filter by status if specified
			if statusFilter != "" && statusFilter != "all" {
				sessions = filterSessionsByStatus(sessions, statusFilter)
			}

			// Limit number of sessions
			if limit > 0 && len(sessions) > limit {
				sessions = sessions[:limit]
			}

			// Display sessions in specified format
			switch outputFormat {
			case "table":
				displaySessionsTable(sessions)
			case "json":
				displaySessionsJSON(sessions)
			case "csv":
				displaySessionsCSV(sessions)
			default:
				PrintError("Invalid output format: %s (must be table, json, or csv)", outputFormat)
				return ExitWithCode(ExitError)
			}

			return nil
		},
	}

	cmd.Flags().StringP("workspace", "w", "", "Workspace directory (default: ~/.local/share/collab-cli/sessions)")
	cmd.Flags().StringVarP(&statusFilter, "status", "s", "all", "Filter by status (running|paused|completed|incomplete|error|all)")
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format (table|json|csv)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum sessions to display")

	return cmd
}

// loadAllSessions loads all sessions from the workspace directory
func loadAllSessions(workspaceRoot string) ([]*types.Session, error) {
	// Check if workspace directory exists
	if _, err := os.Stat(workspaceRoot); os.IsNotExist(err) {
		return []*types.Session{}, nil
	}

	// Read all session directories
	entries, err := os.ReadDir(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("read workspace directory: %w", err)
	}

	var sessions []*types.Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to load session
		sessionDir := filepath.Join(workspaceRoot, entry.Name())
		session, err := orchestrator.LoadSession(sessionDir)
		if err != nil {
			// Skip invalid sessions
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// filterSessionsByStatus filters sessions by status
func filterSessionsByStatus(sessions []*types.Session, status string) []*types.Session {
	var filtered []*types.Session
	for _, s := range sessions {
		if strings.EqualFold(string(s.Status), status) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// displaySessionsTable displays sessions in table format
func displaySessionsTable(sessions []*types.Session) {
	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		return
	}

	// Print header
	fmt.Printf("%-21s %-12s %-8s %-10s %-8s %s\n",
		"SESSION ID", "STATUS", "TURNS", "DURATION", "COST", "STARTED")
	fmt.Println(strings.Repeat("-", 85))

	// Print sessions
	for _, s := range sessions {
		duration := calculateDuration(s)
		fmt.Printf("%-21s %-12s %2d/%-5d %-10s $%-7.2f %s\n",
			s.ID,
			s.Status,
			s.CurrentTurn,
			s.MaxTurns,
			formatDuration(duration),
			s.TotalCost,
			s.StartedAt.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\nTotal: %d session(s)\n", len(sessions))
}

// displaySessionsJSON displays sessions in JSON format
func displaySessionsJSON(sessions []*types.Session) {
	type Turns struct {
		Current int `json:"current"`
		Max     int `json:"max"`
	}

	type Cost struct {
		Total  float64 `json:"total"`
		Agent1 float64 `json:"agent1"`
		Agent2 float64 `json:"agent2"`
	}

	type SessionSummary struct {
		ID          string  `json:"id"`
		Status      string  `json:"status"`
		TaskName    string  `json:"task_name"`
		Turns       Turns   `json:"turns"`
		Duration    float64 `json:"duration_seconds"`
		Cost        Cost    `json:"cost"`
		StartedAt   string  `json:"started_at"`
		CompletedAt *string `json:"completed_at,omitempty"`
	}

	type Output struct {
		Sessions []SessionSummary `json:"sessions"`
		Total    int              `json:"total"`
	}

	summaries := make([]SessionSummary, 0, len(sessions))
	for _, s := range sessions {
		duration := calculateDuration(s)

		summary := SessionSummary{
			ID:       s.ID,
			Status:   string(s.Status),
			TaskName: s.TaskName,
			Turns: Turns{
				Current: s.CurrentTurn,
				Max:     s.MaxTurns,
			},
			Duration: duration.Seconds(),
			Cost: Cost{
				Total:  s.TotalCost,
				Agent1: s.Agent1.TotalCost,
				Agent2: s.Agent2.TotalCost,
			},
			StartedAt: s.StartedAt.Format(time.RFC3339),
		}

		if s.CompletedAt != nil {
			completedStr := s.CompletedAt.Format(time.RFC3339)
			summary.CompletedAt = &completedStr
		}

		summaries = append(summaries, summary)
	}

	output := Output{
		Sessions: summaries,
		Total:    len(summaries),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}

// displaySessionsCSV displays sessions in CSV format
func displaySessionsCSV(sessions []*types.Session) {
	// Print header
	fmt.Println("session_id,status,task_name,current_turn,max_turns,duration_seconds,total_cost,started_at,completed_at")

	// Print sessions
	for _, s := range sessions {
		duration := calculateDuration(s)

		completedAt := ""
		if s.CompletedAt != nil {
			completedAt = s.CompletedAt.Format(time.RFC3339)
		}

		fmt.Printf("%s,%s,%s,%d,%d,%.2f,%.2f,%s,%s\n",
			s.ID,
			s.Status,
			s.TaskName,
			s.CurrentTurn,
			s.MaxTurns,
			duration.Seconds(),
			s.TotalCost,
			s.StartedAt.Format(time.RFC3339),
			completedAt)
	}
}

// calculateDuration calculates the session duration
func calculateDuration(s *types.Session) time.Duration {
	if s.StartedAt == nil {
		return 0
	}

	endTime := time.Now()
	if s.CompletedAt != nil {
		endTime = *s.CompletedAt
	}

	return endTime.Sub(*s.StartedAt)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	seconds := d.Seconds()
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	minutes := int(seconds / 60)
	secs := int(seconds) % 60
	return fmt.Sprintf("%dm%ds", minutes, secs)
}
