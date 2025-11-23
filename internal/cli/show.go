package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/darinhaener/collab/internal/orchestrator"
	"github.com/darinhaener/collab/internal/storage"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/spf13/cobra"
)

// NewShowCommand creates the show command
func NewShowCommand() *cobra.Command {
	var (
		showMessages    bool
		showDeliverable bool
		showLogs        bool
		showContext     bool
		outputFormat    string
	)

	cmd := &cobra.Command{
		Use:   "show <session-id>",
		Short: "Display detailed information about a session",
		Long: `Display detailed information about a collaboration session.

Can optionally show messages, deliverable content, logs, and shared context.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			// Get workspace directory
			workspaceRoot, _ := cmd.Flags().GetString("workspace")
			if workspaceRoot == "" {
				workspaceRoot = getDefaultWorkspaceDir()
			}

			// Load session
			workspaceDir := filepath.Join(workspaceRoot, sessionID)
			session, err := orchestrator.LoadSession(workspaceDir)
			if err != nil {
				PrintError("Failed to load session %s: %v", sessionID, err)
				return ExitWithCode(ExitError)
			}

			// Display session summary
			if outputFormat == "json" {
				return displaySessionJSON(session, workspaceDir, showMessages, showDeliverable, showContext)
			}

			displaySessionSummary(session)

			// Display optional sections
			if showMessages {
				fmt.Println()
				displayMessages(workspaceDir)
			}

			if showContext {
				fmt.Println()
				displaySharedContext(workspaceDir)
			}

			if showDeliverable && session.DeliverablePath != "" {
				fmt.Println()
				displayDeliverable(session.DeliverablePath)
			}

			if showLogs {
				fmt.Println()
				displayLogs(workspaceDir)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showMessages, "messages", "m", false, "Show all messages between agents")
	cmd.Flags().BoolVarP(&showDeliverable, "deliverable", "d", false, "Show final deliverable content")
	cmd.Flags().BoolVarP(&showLogs, "logs", "l", false, "Show orchestrator logs")
	cmd.Flags().BoolVar(&showContext, "context", false, "Show shared_context.md content")
	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text|json)")

	return cmd
}

// displaySessionSummary displays the session summary
func displaySessionSummary(session *types.Session) {
	fmt.Printf("Session: %s\n", session.ID)
	fmt.Printf("Status: %s\n", session.Status)
	fmt.Printf("Task: %s\n", session.TaskName)
	fmt.Println()

	fmt.Printf("Agents:\n")
	fmt.Printf("  Agent 1: %s (%s)\n", session.Agent1.Name, session.Agent1.Role)
	fmt.Printf("    Model: %s\n", session.Agent1.Model)
	fmt.Printf("    Turns: %d, Tokens: %d, Cost: $%.2f\n",
		session.Agent1.TotalTurns, session.Agent1.TotalTokens, session.Agent1.TotalCost)
	fmt.Printf("  Agent 2: %s (%s)\n", session.Agent2.Name, session.Agent2.Role)
	fmt.Printf("    Model: %s\n", session.Agent2.Model)
	fmt.Printf("    Turns: %d, Tokens: %d, Cost: $%.2f\n",
		session.Agent2.TotalTurns, session.Agent2.TotalTokens, session.Agent2.TotalCost)
	fmt.Println()

	fmt.Printf("Progress:\n")
	fmt.Printf("  Turns: %d/%d\n", session.CurrentTurn, session.MaxTurns)
	fmt.Printf("  Duration: %s\n", formatDuration(calculateDuration(session)))
	fmt.Printf("  Total Cost: $%.2f\n", session.TotalCost)
	fmt.Printf("  Total Tokens: %d\n", session.TotalTokens)
	fmt.Println()

	fmt.Printf("Timeline:\n")
	fmt.Printf("  Created: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
	if session.StartedAt != nil {
		fmt.Printf("  Started: %s\n", session.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if session.CompletedAt != nil {
		fmt.Printf("  Completed: %s\n", session.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()

	fmt.Printf("Workspace: %s\n", session.WorkspaceDir)
	if session.DeliverablePath != "" {
		fmt.Printf("Deliverable: %s\n", session.DeliverablePath)
	}
}

// displayMessages displays all messages between agents
func displayMessages(workspaceDir string) {
	fmt.Println("=== Messages ===")

	messagesPath := filepath.Join(workspaceDir, "messages.md")
	content, err := os.ReadFile(messagesPath)
	if err != nil {
		PrintWarning("No messages found")
		return
	}

	fmt.Println(string(content))
}

// displaySharedContext displays shared context
func displaySharedContext(workspaceDir string) {
	fmt.Println("=== Shared Context ===")

	contextPath := filepath.Join(workspaceDir, "shared_context.md")
	content, err := os.ReadFile(contextPath)
	if err != nil {
		PrintWarning("No shared context found")
		return
	}

	fmt.Println(string(content))
}

// displayDeliverable displays the deliverable content
func displayDeliverable(deliverablePath string) {
	fmt.Println("=== Deliverable ===")

	content, err := os.ReadFile(deliverablePath)
	if err != nil {
		PrintWarning("Could not read deliverable: %v", err)
		return
	}

	fmt.Println(string(content))
}

// displayLogs displays orchestrator logs
func displayLogs(workspaceDir string) {
	fmt.Println("=== Logs ===")

	logsPath := filepath.Join(workspaceDir, "orchestrator.log")
	content, err := os.ReadFile(logsPath)
	if err != nil {
		PrintWarning("No logs found")
		return
	}

	fmt.Println(string(content))
}

// displaySessionJSON displays session in JSON format
func displaySessionJSON(session *types.Session, workspaceDir string, includeMessages, includeDeliverable, includeContext bool) error {
	output := map[string]interface{}{
		"session": session,
	}

	if includeMessages {
		messages, err := storage.ReadMessages(workspaceDir, "", 0)
		if err == nil {
			output["messages"] = messages
		}
	}

	if includeContext {
		contextPath := filepath.Join(workspaceDir, "shared_context.md")
		if content, err := os.ReadFile(contextPath); err == nil {
			output["shared_context"] = string(content)
		}
	}

	if includeDeliverable && session.DeliverablePath != "" {
		if content, err := os.ReadFile(session.DeliverablePath); err == nil {
			output["deliverable"] = string(content)
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
