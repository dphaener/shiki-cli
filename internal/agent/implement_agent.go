package agent

import (
	"fmt"

	"github.com/dphaener/shiki-cli/internal/prompts"
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
	prompt, err := prompts.LoadImplementPrompt(prompts.ImplementPromptData{
		FriendlyName:  session.FriendlyName,
		FeatureNumber: session.FeatureNumber,
		Slug:          session.Slug,
		SpecFile:      session.SpecFile,
		PlanFile:      session.PlanFile,
		TasksFile:     session.TasksFile,
		FeatureDir:    session.FeatureDir,
	})
	if err != nil {
		// Fallback to minimal prompt
		return fmt.Sprintf("Help implement the feature: %s", session.FriendlyName)
	}

	return prompt
}
