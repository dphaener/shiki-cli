package agent

import (
	"fmt"

	"github.com/dphaener/shiki-cli/internal/templates"
	"github.com/dphaener/shiki-cli/pkg/types"
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
	// Create template processor
	processor := templates.NewTemplateProcessor("templates")

	// Create unified phase context
	context := templates.NewPhaseContext().
		WithCore("", 0, "", string(session.CurrentPhase)).
		WithPaths("", session.PlanFile, session.TasksFile, "", "", "", "", session.BugDir).
		WithBugData(session.ID, session.Title, session.Description)

	// Load and process the system prompt template
	prompt, err := processor.LoadSystemPrompt("bug", context)
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