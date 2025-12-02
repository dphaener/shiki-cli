package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/dphaener/shiki-cli/internal/storage"
	"github.com/dphaener/shiki-cli/internal/tui"
	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewFeatureCommand creates the feature command
func NewFeatureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature [feature-name]",
		Short: "Unified feature development workflow",
		Long: `Start or resume a unified feature development workflow.

The feature command chains together the entire feature development lifecycle:
  Specify → Approve → Plan → Approve → Tasks → Approve → Implement → Complete

Each phase has an approval gate where you can approve to continue, go back to
make changes, or quit and resume later.

Examples:
  collab feature "user-authentication"       # Start new feature workflow
  collab feature                             # Prompts for feature name
  collab feature resume 001-user-auth        # Resume existing workflow
  collab feature list                        # List active workflows`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle no arguments - prompt for feature name
			if len(args) == 0 {
				return startNewWorkflow("")
			}

			input := strings.Join(args, " ")

			// Check for subcommands
			if input == "list" {
				return listWorkflows()
			}

			if strings.HasPrefix(input, "resume ") {
				slug := strings.TrimPrefix(input, "resume ")
				return resumeWorkflow(slug)
			}

			// Start new workflow with feature name
			return startNewWorkflow(input)
		},
	}

	return cmd
}

// startNewWorkflow creates and starts a new unified workflow
func startNewWorkflow(featureName string) error {
	var finalFeatureName string
	var err error

	if featureName != "" {
		// WITH ARGUMENT: Validate the provided feature name
		if err := storage.ValidateFeatureName(featureName); err != nil {
			PrintError("Invalid feature name: %v", err)
			return ExitWithCode(ExitError)
		}
		finalFeatureName = strings.TrimSpace(featureName)
	} else {
		// NO ARGUMENT: Prompt for feature name
		PrintInfo("Creating a new feature workflow...")
		finalFeatureName, err = storage.PromptForFeatureName()
		if err != nil {
			PrintError("Failed to get feature name: %v", err)
			return ExitWithCode(ExitError)
		}
	}

	// Setup workflow directories with validated name
	result, err := storage.SetupWorkflowSession(storage.FeatureSetupConfig{
		FeatureName: finalFeatureName,
	})
	if err != nil {
		PrintError("Failed to setup workflow: %v", err)
		return ExitWithCode(ExitError)
	}

	PrintSuccess("Starting unified workflow for: %s", finalFeatureName)
	PrintSuccess("Feature number: %03d", result.FeatureNumber)
	PrintSuccess("Slug: %s", result.Slug)
	PrintInfo("")
	PrintInfo("Launching unified feature workflow TUI...")

	// Create workflow session
	session := &types.WorkflowSession{
		ID:            fmt.Sprintf("workflow-%d-%s", time.Now().Unix(), result.Slug),
		CurrentPhase:  types.WorkflowPhaseSpecify,
		FeatureNumber: result.FeatureNumber,
		Slug:          result.Slug,
		FriendlyName:  finalFeatureName,
		FeatureDesc:   finalFeatureName, // Now contains the feature name instead of description
		SpecFile:      result.SpecFile,
		PlanFile:      result.PlanFile,
		TasksFile:     result.TasksFile,
		FeatureDir:    result.FeatureDir,
		ChecklistDir:  result.ChecklistDir,
		ContractsDir:  result.ContractsDir,
		Checkpoints:   make(map[types.WorkflowPhase]*types.PhaseCheckpoint),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Status:        types.SessionRunning,
	}

	// Save initial workflow state
	if err := storage.SaveWorkflowSession(session); err != nil {
		PrintError("Failed to save workflow state: %v", err)
		return ExitWithCode(ExitError)
	}

	// Launch TUI in workflow mode
	return tui.RunWorkflowMode(session)
}

// resumeWorkflow resumes an existing workflow
func resumeWorkflow(slug string) error {
	// Load existing workflow session
	session, err := storage.LoadWorkflowSession(slug)
	if err != nil {
		PrintError("Failed to load workflow: %v", err)
		return ExitWithCode(ExitError)
	}

	phaseName := types.GetPhaseName(session.CurrentPhase)
	PrintSuccess("Resuming workflow: %s", session.FriendlyName)
	PrintSuccess("Feature: %s", session.Slug)
	PrintSuccess("Current phase: %s", phaseName)
	PrintInfo("")
	PrintInfo("Launching unified feature workflow TUI...")

	// Update session status
	session.Status = types.SessionRunning
	session.UpdatedAt = time.Now()

	// Launch TUI in workflow mode
	return tui.RunWorkflowMode(session)
}

// listWorkflows lists all active workflows
func listWorkflows() error {
	workflows, err := storage.ListActiveWorkflows()
	if err != nil {
		PrintError("Failed to list workflows: %v", err)
		return ExitWithCode(ExitError)
	}

	if len(workflows) == 0 {
		PrintInfo("No active workflows found.")
		PrintInfo("Start a new workflow with: collab feature \"your feature description\"")
		return nil
	}

	PrintInfo("Active Feature Workflows:")
	PrintInfo("")
	for _, w := range workflows {
		phaseName := types.GetPhaseName(w.CurrentPhase)
		PrintInfo("  %03d - %s", w.FeatureNumber, w.FriendlyName)
		PrintInfo("      Slug: %s", w.Slug)
		PrintInfo("      Phase: %s", phaseName)
		PrintInfo("      Updated: %s", w.UpdatedAt.Format("2006-01-02 15:04"))
		PrintInfo("")
	}

	PrintInfo("To resume a workflow: collab feature resume <slug>")
	return nil
}
