package agent

import (
	"fmt"

	"github.com/dphaener/shiki-cli/internal/templates"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// NewPlanAgent creates an agent configured for implementation planning
func NewPlanAgent(session *types.PlanSession) *types.Agent {
	systemPrompt := buildPlanSystemPrompt(session)

	return &types.Agent{
		ID:           "plan-agent",
		Name:         "Planning Assistant",
		Role:         "Implementation planning collaborator",
		SystemPrompt: systemPrompt,
		Model:        "claude-sonnet-4-20250514",
		WorkspaceDir: "", // Not needed for plan mode
		MemoryFile:   "", // Not needed for plan mode
	}
}

// buildPlanSystemPrompt creates the system prompt for planning workflow
func buildPlanSystemPrompt(session *types.PlanSession) string {
	// Create template processor
	processor := templates.NewTemplateProcessor("templates")

	// Create unified phase context
	context := templates.NewPhaseContext().
		WithCore(session.FriendlyName, session.FeatureNumber, session.SpecSlug, string(session.Phase)).
		WithPaths(session.SpecFile, session.PlanFile, "", session.SpecDir, session.ContractsDir, "", "", "")

	// Load and process the system prompt template
	prompt, err := processor.LoadSystemPrompt("plan", context)
	if err != nil {
		// Fallback to minimal prompt
		return fmt.Sprintf("Help create an implementation plan for: %s", session.FriendlyName)
	}

	return prompt
}
