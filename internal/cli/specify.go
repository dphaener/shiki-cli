package cli

import (
	"fmt"
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
and validate quality before saving to your local spec storage.

Examples:
  collab specify "Add user authentication"
  collab specify "Real-time collaboration dashboard"

The specification will be saved to ~/.local/share/collab-cli/specs/`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			featureDesc := strings.Join(args, " ")

			// Check for list/resume subcommands
			if featureDesc == "list" {
				return listSpecs()
			}

			if strings.HasPrefix(featureDesc, "resume ") {
				slug := strings.TrimPrefix(featureDesc, "resume ")
				return resumeSpec(slug)
			}

			// Create new specify session
			return startNewSpec(featureDesc)
		},
	}

	return cmd
}

// startNewSpec creates a new specification session
func startNewSpec(featureDesc string) error {
	// Get next feature number
	featureNumber, err := storage.GetNextFeatureNumber()
	if err != nil {
		PrintError("Failed to get feature number: %v", err)
		return ExitWithCode(ExitError)
	}

	// Generate friendly name (simple version - first 50 chars)
	friendlyName := featureDesc
	if len(friendlyName) > 50 {
		friendlyName = friendlyName[:50]
	}

	// Create slug
	slug := storage.CreateSlug(friendlyName, featureNumber)

	// Create session
	session := &types.SpecifySession{
		ID:            fmt.Sprintf("spec-%d-%s", time.Now().Unix(), slug),
		FeatureDesc:   featureDesc,
		FeatureNumber: featureNumber,
		Slug:          slug,
		FriendlyName:  friendlyName,
		Phase:         types.PhaseDiscovery,
		ChatHistory:   []types.ChatMessage{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Status:        types.SessionRunning,
	}

	PrintSuccess("Starting specification for: %s", friendlyName)
	PrintSuccess("Feature number: %03d", featureNumber)
	PrintSuccess("Slug: %s", slug)
	PrintInfo("")
	PrintInfo("Launching interactive specification TUI...")

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
