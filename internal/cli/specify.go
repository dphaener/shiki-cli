package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dphaener/shiki-cli/internal/storage"
	"github.com/dphaener/shiki-cli/internal/tui"
	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewSpecifyCommand creates the specify command
func NewSpecifyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "specify [feature-name]",
		Short: "Create feature specification through AI collaboration",
		Long: `Create a feature specification through interactive AI collaboration.

The specify command launches an interactive TUI with AI-assisted specification
generation. The AI will ask clarifying questions, generate a structured spec,
and validate quality before saving.

When called with a feature name, the AI will proceed directly to generating
a specification draft for that named feature. When called without arguments,
you will be prompted to provide a feature name.

Examples:
  collab specify "user-authentication"
  collab specify "Real-time Collaboration Dashboard"
  collab specify                                      # Prompts for feature name

Subcommands:
  collab specify list                                 # List all specs
  collab specify resume <slug>                        # Resume existing spec`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle no arguments - prompt for feature name
			if len(args) == 0 {
				return startNewSpec("")
			}

			featureName := strings.Join(args, " ")

			// Check for list/resume subcommands
			if featureName == "list" {
				return listSpecs()
			}

			if strings.HasPrefix(featureName, "resume ") {
				slug := strings.TrimPrefix(featureName, "resume ")
				return resumeSpec(slug)
			}

			// Create new specify session with feature name
			return startNewSpec(featureName)
		},
	}

	return cmd
}

// startNewSpec creates a new specification session
func startNewSpec(featureName string) error {
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
		PrintInfo("Creating a new feature specification...")
		finalFeatureName, err = storage.PromptForFeatureName()
		if err != nil {
			PrintError("Failed to get feature name: %v", err)
			return ExitWithCode(ExitError)
		}
	}

	// Setup feature directories with validated name
	result, err := storage.SetupFeature(storage.FeatureSetupConfig{
		FeatureName: finalFeatureName,
	})
	if err != nil {
		PrintError("Failed to setup feature: %v", err)
		return ExitWithCode(ExitError)
	}

	PrintSuccess("Starting specification for: %s", finalFeatureName)
	PrintSuccess("Feature number: %03d", result.FeatureNumber)
	PrintSuccess("Slug: %s", result.Slug)
	PrintSuccess("Feature directory: %s", result.FeatureDir)
	PrintInfo("")
	PrintInfo("Launching interactive specification TUI...")

	// Create session
	session := &types.SpecifySession{
		ID:                     fmt.Sprintf("spec-%d-%s", time.Now().Unix(), result.Slug),
		FeatureDesc:            finalFeatureName, // Now contains the feature name instead of description
		FeatureNumber:          result.FeatureNumber,
		Slug:                   result.Slug,
		FriendlyName:           finalFeatureName,
		Phase:                  types.PhaseDiscovery,
		ChatHistory:            []types.ChatMessage{},
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		Status:                 types.SessionRunning,
		SkipDiscoveryQuestions: false, // Always go through discovery since we only have a name, not a description
		SpecFile:               result.SpecFile,
		SpecDir:                result.FeatureDir,
		ChecklistDir:           result.ChecklistDir,
	}

	// Launch TUI in specify mode
	return tui.RunSpecifyMode(session)
}

// resumeSpec resumes an existing specification session
func resumeSpec(slug string) error {
	// Load existing spec
	spec, err := storage.LoadSpec(slug)
	if err != nil {
		PrintError("Failed to load spec: %v", err)
		return ExitWithCode(ExitError)
	}

	// Get paths using storage functions
	specDir := storage.GetSpecDir(slug)
	specFile := filepath.Join(specDir, "spec.md")
	checklistDir := filepath.Join(specDir, "checklists")

	// Create session from spec
	session := &types.SpecifySession{
		ID:            fmt.Sprintf("spec-%d-%s", time.Now().Unix(), slug),
		FeatureDesc:   spec.FeatureName,
		FeatureNumber: spec.Number,
		Slug:          spec.Slug,
		FriendlyName:  spec.FeatureName,
		Phase:         types.PhaseDiscovery, // TODO: Restore phase from metadata
		ChatHistory:   []types.ChatMessage{},
		CurrentSpec:   spec.Content,
		CreatedAt:     spec.CreatedAt,
		UpdatedAt:     time.Now(),
		Status:        types.SessionRunning,
		SpecFile:      specFile,
		SpecDir:       specDir,
		ChecklistDir:  checklistDir,
	}

	PrintSuccess("Resuming specification: %s", spec.FeatureName)
	PrintSuccess("Feature: %s", spec.Slug)
	PrintInfo("")
	PrintInfo("Launching interactive specification TUI...")

	// Launch TUI in specify mode
	return tui.RunSpecifyMode(session)
}

// listSpecs lists all existing specifications
func listSpecs() error {
	specs, err := storage.ListSpecs()
	if err != nil {
		PrintError("Failed to list specs: %v", err)
		return ExitWithCode(ExitError)
	}

	if len(specs) == 0 {
		PrintInfo("No specifications found.")
		PrintInfo("Create one with: collab specify \"your feature description\"")
		return nil
	}

	PrintInfo("Feature Specifications:")
	PrintInfo("")
	for _, spec := range specs {
		status := spec.Status
		if status == "" {
			status = "draft"
		}
		PrintInfo("  %03d - %s", spec.Number, spec.FeatureName)
		PrintInfo("      Slug: %s", spec.Slug)
		PrintInfo("      Status: %s", status)
		PrintInfo("      Created: %s", spec.CreatedAt.Format("2006-01-02"))
		PrintInfo("")
	}

	PrintInfo("To resume a spec: collab specify resume <slug>")
	return nil
}
