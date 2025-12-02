package agent

import (
	"fmt"

	"github.com/dphaener/shiki-cli/internal/prompts"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// NewTasksAgent creates an agent configured for task generation
func NewTasksAgent(session *types.WorkflowSession) *types.Agent {
	systemPrompt := buildTasksSystemPrompt(session)

	return &types.Agent{
		ID:           "tasks-agent",
		Name:         "Tasks Generation Assistant",
		Role:         "Implementation task breakdown collaborator",
		SystemPrompt: systemPrompt,
		Model:        "claude-sonnet-4-20250514",
		WorkspaceDir: "", // Not needed for tasks mode
		MemoryFile:   "", // Not needed for tasks mode
	}
}

// buildTasksSystemPrompt creates the system prompt for tasks workflow
func buildTasksSystemPrompt(session *types.WorkflowSession) string {
	prompt, err := prompts.LoadTasksPrompt(prompts.TasksPromptData{
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
		return fmt.Sprintf("Help generate implementation tasks for: %s", session.FriendlyName)
	}

	return prompt
}
