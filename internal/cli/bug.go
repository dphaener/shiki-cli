package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dphaener/shiki-cli/internal/storage"
	"github.com/dphaener/shiki-cli/internal/tui"
	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewBugCommand creates the bug command
func NewBugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bug [bug-title]",
		Short: "Bug fix workflow with 3-phase approach",
		Long: `Start or resume a bug fix workflow with a streamlined 3-phase approach.

The bug command provides a focused workflow for systematic bug resolution:
  Plan → Approve → Tasks → Approve → Implement → Complete

Each phase has an approval gate where you can approve to continue, go back to
make changes, or quit and resume later.

Examples:
  collab bug "Login button broken"               # Start new bug workflow
  collab bug                                     # Prompts for bug title
  collab bug list                                # List active bugs
  collab bug status                              # Show current bug status
  collab bug resume <bug-id>                     # Resume existing workflow`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle no arguments - prompt for bug description
			if len(args) == 0 {
				return startNewBugWorkflow("")
			}

			input := strings.Join(args, " ")

			// Check for subcommands
			if input == "list" {
				return listBugWorkflows()
			}

			if input == "status" {
				return showBugStatus()
			}

			if strings.HasPrefix(input, "resume ") {
				bugID := strings.TrimPrefix(input, "resume ")
				return resumeBugWorkflow(bugID)
			}

			// Start new workflow with bug description
			return startNewBugWorkflow(input)
		},
	}

	return cmd
}

// startNewBugWorkflow creates and starts a new bug fix workflow
func startNewBugWorkflow(bugTitle string) error {
	var finalBugTitle string

	if bugTitle != "" {
		// WITH ARGUMENT: Validate the provided bug title
		if err := validateBugTitle(bugTitle); err != nil {
			PrintError("Invalid bug title: %v", err)
			return ExitWithCode(ExitError)
		}
		finalBugTitle = strings.TrimSpace(bugTitle)
	} else {
		// NO ARGUMENT: Prompt for bug title
		PrintInfo("Creating a new bug fix workflow...")
		var err error
		finalBugTitle, err = promptForBugTitle()
		if err != nil {
			PrintError("Failed to get bug title: %v", err)
			return ExitWithCode(ExitError)
		}
	}

	// Setup bug workflow session
	result, err := storage.SetupBugSession(storage.BugSetupConfig{
		Title:       finalBugTitle,
		Description: "", // Agent will ask for details in first turn
	})
	if err != nil {
		PrintError("Failed to setup bug workflow: %v", err)
		return ExitWithCode(ExitError)
	}

	PrintSuccess("Starting bug fix workflow: %s", finalBugTitle)
	PrintSuccess("Bug ID: %s", result.BugID)
	PrintInfo("")
	PrintInfo("Launching bug fix workflow TUI...")

	// Create bug session
	session := &types.BugSession{
		ID:           result.BugID,
		Title:        finalBugTitle,
		Description:  "", // Agent will ask for details in first turn
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     result.PlanFile,
		TasksFile:    result.TasksFile,
		BugDir:       result.BugDir,
		Checkpoints:  make(map[types.BugPhase]*types.PhaseCheckpoint),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       types.SessionRunning,
	}

	// Save initial bug session state
	if err := storage.SaveBugSession(session); err != nil {
		PrintError("Failed to save bug session state: %v", err)
		return ExitWithCode(ExitError)
	}

	// Launch TUI in bug mode
	return tui.RunBugMode(session)
}

// resumeBugWorkflow resumes an existing bug workflow
func resumeBugWorkflow(bugID string) error {
	// Load existing bug session
	session, err := storage.LoadBugSession(bugID)
	if err != nil {
		PrintError("Failed to load bug workflow: %v", err)
		return ExitWithCode(ExitError)
	}

	phaseName := session.CurrentPhase.GetPhaseName(session.CurrentPhase)
	PrintSuccess("Resuming bug workflow: %s", session.Title)
	PrintSuccess("Bug ID: %s", session.ID)
	PrintSuccess("Current phase: %s", phaseName)
	PrintInfo("")
	PrintInfo("Launching bug fix workflow TUI...")

	// Update session status
	session.Status = types.SessionRunning
	session.UpdatedAt = time.Now()

	// Launch TUI in bug mode
	return tui.RunBugMode(session)
}

// listBugWorkflows lists all active bug workflows
func listBugWorkflows() error {
	bugs, err := storage.ListActiveBugs()
	if err != nil {
		PrintError("Failed to list bug workflows: %v", err)
		return ExitWithCode(ExitError)
	}

	if len(bugs) == 0 {
		PrintInfo("No active bug workflows found.")
		PrintInfo("Start a new bug workflow with: collab bug \"bug description\"")
		return nil
	}

	PrintInfo("Active Bug Workflows:")
	PrintInfo("")
	for _, bug := range bugs {
		phaseName := bug.CurrentPhase.GetPhaseName(bug.CurrentPhase)
		PrintInfo("  Bug ID: %s", bug.ID)
		PrintInfo("  Title: %s", bug.Title)
		PrintInfo("  Phase: %s", phaseName)
		PrintInfo("  Updated: %s", bug.UpdatedAt.Format("2006-01-02 15:04"))
		PrintInfo("")
	}

	PrintInfo("To resume a bug workflow: collab bug resume <bug-id>")
	return nil
}

// showBugStatus shows all active bug workflow statuses
func showBugStatus() error {
	activeBugs, err := storage.ListActiveBugs()
	if err != nil {
		PrintError("Failed to list active bugs: %v", err)
		return ExitWithCode(ExitError)
	}

	if len(activeBugs) == 0 {
		PrintInfo("No active bug workflows.")
		PrintInfo("Start a new bug workflow with: collab bug \"bug description\"")
		return nil
	}

	PrintInfo("Active Bug Workflow Status:")
	PrintInfo("")

	for i, bug := range activeBugs {
		if i > 0 {
			PrintInfo("") // Add spacing between bugs
		}

		phaseName := bug.CurrentPhase.GetPhaseName(bug.CurrentPhase)
		PrintInfo("  Bug ID: %s", bug.ID)
		PrintInfo("  Title: %s", bug.Title)
		PrintInfo("  Description: %s", bug.Description)
		PrintInfo("  Current Phase: %s", phaseName)
		PrintInfo("  Created: %s", bug.CreatedAt.Format("2006-01-02 15:04"))
		PrintInfo("  Updated: %s", bug.UpdatedAt.Format("2006-01-02 15:04"))

		// Show phase progress
		displayPhases := types.BugPhase("").GetDisplayPhases()
		PrintInfo("  Phase Progress:")
		for _, phase := range displayPhases {
			status := bug.CurrentPhase.GetPhaseStatus(phase, bug.CurrentPhase)
			statusSymbol := "○" // pending
			if status == types.BugPhaseStatusComplete {
				statusSymbol = "●" // complete
			} else if status == types.BugPhaseStatusCurrent {
				statusSymbol = "◐" // current
			}

			phaseDisplayName := phase.GetPhaseName(phase)
			PrintInfo("    %s %s", statusSymbol, phaseDisplayName)
		}

		PrintInfo("  To resume: collab bug resume %s", bug.ID)
	}

	return nil
}

// validateBugTitle validates the bug title
func validateBugTitle(title string) error {
	if len(strings.TrimSpace(title)) == 0 {
		return fmt.Errorf("bug title cannot be empty")
	}

	if len(title) < 3 {
		return fmt.Errorf("bug title must be at least 3 characters")
	}

	if len(title) > 100 {
		return fmt.Errorf("bug title must be less than 100 characters")
	}

	return nil
}

// promptForBugTitle prompts the user for a bug title
func promptForBugTitle() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)

	PrintInfo("Please provide a short title for the bug:")
	PrintInfo("(e.g., 'Login button not working', 'API returns 500 error')")
	PrintInfo("")

	for {
		fmt.Print("Bug title: ")

		if !scanner.Scan() {
			// Handle EOF or interrupted input
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("input error: %w", err)
			}
			return "", fmt.Errorf("input interrupted")
		}

		title := scanner.Text()
		if err := validateBugTitle(title); err != nil {
			PrintError("%v", err)
			PrintInfo("Please try again:")
			continue
		}

		return title, nil
	}
}