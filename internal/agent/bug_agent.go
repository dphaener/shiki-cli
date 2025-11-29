package agent

import (
	"fmt"

	"github.com/darinhaener/collab/internal/prompts"
	"github.com/darinhaener/collab/pkg/types"
)

// NewBugAgent creates an agent configured for bug fix workflow
func NewBugAgent(session *types.BugSession) *types.Agent {
	systemPrompt := buildBugSystemPrompt(session)

	return &types.Agent{
		ID:           "bug-agent",
		Name:         "Bug Fix Assistant",
		Role:         "Bug analysis and resolution collaborator",
		SystemPrompt: systemPrompt,
		Model:        "claude-sonnet-4-20250514",
		WorkspaceDir: "", // Not needed for bug mode
		MemoryFile:   "", // Not needed for bug mode
	}
}

// buildBugSystemPrompt creates the system prompt for bug fix workflow
func buildBugSystemPrompt(session *types.BugSession) string {
	prompt, err := prompts.LoadBugPrompt(prompts.BugPromptData{
		Title:       session.Title,
		Description: session.Description,
		Phase:       string(session.CurrentPhase),
		BugID:       session.ID,
		BugDir:      session.BugDir,
		PlanFile:    session.PlanFile,
		TasksFile:   session.TasksFile,
	})
	if err != nil {
		// Fallback to minimal prompt
		return fmt.Sprintf(`# Bug Fix Assistant

You are helping to fix a bug in a systematic way.

**Bug**: %s
**Description**: %s
**Current Phase**: %s

Work through the bug fix process step by step, focusing on understanding the problem, creating a plan, and implementing a solution.`, session.Title, session.Description, session.CurrentPhase)
	}

	return prompt
}