package agent

import (
	"fmt"

	"github.com/dphaener/shiki-cli/internal/prompts"
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
	prompt, err := prompts.LoadPlanPrompt(prompts.PlanPromptData{
		FriendlyName:  session.FriendlyName,
		FeatureNumber: session.FeatureNumber,
		SpecSlug:      session.SpecSlug,
		Phase:         string(session.Phase),
		SpecFile:      session.SpecFile,
		PlanFile:      session.PlanFile,
		ContractsDir:  session.ContractsDir,
		SpecDir:       session.SpecDir,
	})
	if err != nil {
		// Fallback to minimal prompt
		return fmt.Sprintf("Help create an implementation plan for: %s", session.FriendlyName)
	}

	return prompt
}
