package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/darinhaener/collab/internal/storage"
	"github.com/darinhaener/collab/internal/tui"
	"github.com/darinhaener/collab/pkg/types"
	"github.com/spf13/cobra"
)

// NewSpecifyCommand creates the specify command
func NewSpecifyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "specify [description]",
		Short: "Create feature specification through AI collaboration",
		Long: `Create a feature specification through interactive AI collaboration.

The specify command launches an interactive TUI with AI-assisted specification
generation. The AI will ask clarifying questions, generate a structured spec,
and validate quality before saving.

When called with a description, the AI will proceed directly to generating
a specification draft. When called without arguments, the AI will ask what
feature you'd like to specify.

Examples:
  collab specify "Add user authentication"
  collab specify "Real-time collaboration dashboard"
  collab specify                                      # AI asks what to specify

Subcommands:
  collab specify list                                 # List all specs
  collab specify resume <slug>                        # Resume existing spec`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle no arguments - launch TUI and let AI ask
			if len(args) == 0 {
				return startNewSpec("")
			}

			featureDesc := strings.Join(args, " ")

			// Check for list/resume subcommands
			if featureDesc == "list" {
				return listSpecs()
			}

			if strings.HasPrefix(featureDesc, "resume ") {
				slug := strings.TrimPrefix(featureDesc, "resume ")
				return resumeSpec(slug)
			}

			// Create new specify session with description
			return startNewSpec(featureDesc)
		},
	}

	return cmd
}

// startNewSpec creates a new specification session
func startNewSpec(featureDesc string) error {
	var featureNumber int
	var slug string
	var friendlyName string
	var specFile string
	var specDir string
	var checklistDir string
	var err error

	if featureDesc != "" {
		// WITH ARGUMENT: Setup feature directories first, then launch TUI
		friendlyName = featureDesc
		if len(friendlyName) > 50 {
			friendlyName = friendlyName[:50]
		}

		result, err := storage.SetupFeature(storage.FeatureSetupConfig{
			FriendlyName: friendlyName,
			Description:  featureDesc,
		})
		if err != nil {
			PrintError("Failed to setup feature: %v", err)
			return ExitWithCode(ExitError)
		}

		featureNumber = result.FeatureNumber
		slug = result.Slug
		specFile = result.SpecFile
		specDir = result.FeatureDir
		checklistDir = result.ChecklistDir

		PrintSuccess("Starting specification for: %s", friendlyName)
		PrintSuccess("Feature number: %03d", featureNumber)
		PrintSuccess("Slug: %s", slug)
		PrintSuccess("Feature directory: %s", result.FeatureDir)
		PrintInfo("")
		PrintInfo("Launching interactive specification TUI...")
	} else {
		// NO ARGUMENT: Get feature number but don't create directories yet
		// AI will prompt for description, then we can create directories
		featureNumber, err = storage.GetNextFeatureNumber()
		if err != nil {
			PrintError("Failed to get feature number: %v", err)
			return ExitWithCode(ExitError)
		}

		// Use placeholder values - AI will ask for the real description
		friendlyName = "New Feature"
		slug = storage.CreateSlug(friendlyName, featureNumber)

		// Set up paths using storage functions
		specDir = storage.GetSpecDir(slug)
		specFile = filepath.Join(specDir, "spec.md")
		checklistDir = filepath.Join(specDir, "checklists")

		PrintInfo("Launching interactive specification TUI...")
		PrintInfo("The AI will ask what feature you'd like to specify.")
		PrintInfo("")
	}

	// Create session
	session := &types.SpecifySession{
		ID:                     fmt.Sprintf("spec-%d-%s", time.Now().Unix(), slug),
		FeatureDesc:            featureDesc,
		FeatureNumber:          featureNumber,
		Slug:                   slug,
		FriendlyName:           friendlyName,
		Phase:                  types.PhaseDiscovery,
		ChatHistory:            []types.ChatMessage{},
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		Status:                 types.SessionRunning,
		SkipDiscoveryQuestions: featureDesc != "",
		SpecFile:               specFile,
		SpecDir:                specDir,
		ChecklistDir:           checklistDir,
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
		FeatureDesc:   spec.FriendlyName,
		FeatureNumber: spec.Number,
		Slug:          spec.Slug,
		FriendlyName:  spec.FriendlyName,
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

	PrintSuccess("Resuming specification: %s", spec.FriendlyName)
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
		PrintInfo("  %03d - %s", spec.Number, spec.FriendlyName)
		PrintInfo("      Slug: %s", spec.Slug)
		PrintInfo("      Status: %s", status)
		PrintInfo("      Created: %s", spec.CreatedAt.Format("2006-01-02"))
		PrintInfo("")
	}

	PrintInfo("To resume a spec: collab specify resume <slug>")
	return nil
}
