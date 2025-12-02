package agent

import (
	"fmt"

	"github.com/dphaener/shiki-cli/internal/prompts"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// NewSpecifyAgent creates an agent configured for feature specification
func NewSpecifyAgent(session *types.SpecifySession) *types.Agent {
	systemPrompt := buildSpecifySystemPrompt(session)

	return &types.Agent{
		ID:           "specify-agent",
		Name:         "Specification Assistant",
		Role:         "Feature specification collaborator",
		SystemPrompt: systemPrompt,
		Model:        "claude-sonnet-4-20250514",
		WorkspaceDir: "", // Not needed for specify mode
		MemoryFile:   "", // Not needed for specify mode
	}
}

// buildSpecifySystemPrompt creates the system prompt for specification workflow
func buildSpecifySystemPrompt(session *types.SpecifySession) string {
	prompt, err := prompts.LoadSpecifyPrompt(prompts.SpecifyPromptData{
		FriendlyName:   session.FriendlyName,
		FeatureNumber:  session.FeatureNumber,
		Slug:           session.Slug,
		Phase:          string(session.Phase),
		HasDescription: session.FeatureDesc != "",
		FeatureDesc:    session.FeatureDesc,
		SpecFile:       session.SpecFile,
		SpecDir:        session.SpecDir,
		ChecklistDir:   session.ChecklistDir,
	})
	if err != nil {
		// Fallback to minimal prompt
		return fmt.Sprintf("Help create a specification for: %s", session.FriendlyName)
	}

	return prompt
}
