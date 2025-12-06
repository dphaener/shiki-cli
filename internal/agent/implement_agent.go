package agent

import (
	"fmt"

	"github.com/dphaener/shiki-cli/internal/templates"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// NewImplementAgent creates an agent configured for implementation
func NewImplementAgent(session *types.WorkflowSession) *types.Agent {
	systemPrompt := buildImplementSystemPrompt(session)

	return &types.Agent{
		ID:           "implement-agent",
		Name:         "Implementation Assistant",
		Role:         "Feature implementation collaborator",
		SystemPrompt: systemPrompt,
		Model:        "claude-sonnet-4-20250514",
		WorkspaceDir: "", // Not needed for implement mode
		MemoryFile:   "", // Not needed for implement mode
	}
}

// buildImplementSystemPrompt creates the system prompt for implementation workflow
func buildImplementSystemPrompt(session *types.WorkflowSession) string {
	// Create template processor
	processor := templates.NewTemplateProcessor("templates")

	// Create unified phase context
	context := templates.NewPhaseContext().
		WithCore(session.FriendlyName, session.FeatureNumber, session.Slug, "implement").
		WithPaths(session.SpecFile, session.PlanFile, session.TasksFile, "", "", "", session.FeatureDir, "")

	// Load and process the system prompt template
	prompt, err := processor.LoadSystemPrompt("implement", context)
	if err != nil {
		// Fallback to minimal prompt
		return fmt.Sprintf("Help implement the feature: %s", session.FriendlyName)
	}

	return prompt
}
