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

// buildBugSystemPrompt creates the system prompt for bug fix workflow using a hybrid template approach.
//
// This function implements a hybrid template strategy to ensure bug mode implement phase
// receives the critical task-progress.md management instructions that are mandatory for
// the preview system to function correctly.
//
// Template Loading Strategy:
// - All phases: Load the base "bug" template containing bug-specific context and workflow
// - Implement phase only: Additionally load the "implement" template and merge it
//   to include mandatory task progress tracking instructions
//
// This hybrid approach solves the bug where implement phase in bug mode was missing
// the task-progress.md creation and update instructions that the preview system requires.
//
// The merged prompt contains:
// 1. Bug context (ID, title, description, current phase)
// 2. Bug-specific workflow guidance
// 3. For implement phase: Task progress management instructions (MANDATORY)
//
// Fallback behavior: If template loading fails, a minimal fallback prompt is provided
// to ensure system resilience.
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

	// HYBRID TEMPLATE APPROACH: For implement phase only, merge implement template
	// This is the core fix for bug 013-implement-task-progress-preview-broken
	if session.CurrentPhase == types.BugPhaseImplement {
		implementPrompt, err := processor.LoadSystemPrompt("implement", context)
		if err == nil {
			// Successfully loaded implement template - merge it with bug template
			// The implement template contains critical task-progress.md instructions:
			// - "Create task-progress.md as first action (MANDATORY)"
			// - "Update task-progress.md for every status change (MANDATORY)"
			// - "Use explicit task IDs (T001, T002, etc.) in all status updates (MANDATORY)"
			prompt += "\n\n## Implementation Instructions\n\n" + implementPrompt
		}
		// If implement template fails to load, continue with bug-only template
		// This ensures graceful degradation - bug workflow continues without implement instructions
	}

	return prompt
}