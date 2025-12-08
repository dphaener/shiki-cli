package agent

import (
	"strings"
	"testing"

	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBugAgent(t *testing.T) {
	session := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "This is a test bug",
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     "/tmp/plan.md",
		TasksFile:    "/tmp/tasks.md",
		BugDir:       "/tmp/bug",
	}

	agent := NewBugAgent(session)

	assert.Equal(t, "bug-agent", agent.ID)
	assert.Equal(t, "Bug Fix Assistant", agent.Name)
	assert.Equal(t, "Bug analysis and resolution collaborator", agent.Role)
	assert.Equal(t, "claude-sonnet-4-20250514", agent.Model)
	assert.NotEmpty(t, agent.SystemPrompt)
}

func TestBuildBugSystemPrompt_PlanPhase(t *testing.T) {
	session := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "This is a test bug",
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     "/tmp/plan.md",
		TasksFile:    "/tmp/tasks.md",
		BugDir:       "/tmp/bug",
	}

	prompt := buildBugSystemPrompt(session)

	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Bug Fix Agent")
	assert.Contains(t, prompt, "Test Bug")
	assert.Contains(t, prompt, "plan")
	// Should not contain implement instructions in plan phase
	assert.NotContains(t, prompt, "Implementation Instructions")
	assert.NotContains(t, prompt, "task-progress.md")
}

func TestBuildBugSystemPrompt_TasksPhase(t *testing.T) {
	session := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "This is a test bug",
		CurrentPhase: types.BugPhaseTasks,
		PlanFile:     "/tmp/plan.md",
		TasksFile:    "/tmp/tasks.md",
		BugDir:       "/tmp/bug",
	}

	prompt := buildBugSystemPrompt(session)

	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Bug Fix Agent")
	assert.Contains(t, prompt, "Test Bug")
	assert.Contains(t, prompt, "tasks")
	// Should not contain implement instructions in tasks phase
	assert.NotContains(t, prompt, "Implementation Instructions")
	assert.NotContains(t, prompt, "task-progress.md")
}

func TestBuildBugSystemPrompt_ImplementPhase(t *testing.T) {
	session := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "This is a test bug",
		CurrentPhase: types.BugPhaseImplement,
		PlanFile:     "/tmp/plan.md",
		TasksFile:    "/tmp/tasks.md",
		BugDir:       "/tmp/bug",
	}

	prompt := buildBugSystemPrompt(session)

	require.NotEmpty(t, prompt)

	// Should contain bug-specific content
	assert.Contains(t, prompt, "Bug Fix Agent")
	assert.Contains(t, prompt, "Test Bug")
	assert.Contains(t, prompt, "implement")

	// Should contain hybrid template content with implement instructions
	assert.Contains(t, prompt, "Implementation Instructions")

	// Should contain the critical task progress instructions from implement template
	assert.Contains(t, prompt, "task-progress.md")
	assert.Contains(t, prompt, "MANDATORY")

	// Verify the prompt contains both bug context and implement instructions
	lines := strings.Split(prompt, "\n")
	hasImplementInstructions := false

	for _, line := range lines {
		if strings.Contains(line, "## Implementation Instructions") {
			hasImplementInstructions = true
			break
		}
	}

	assert.True(t, hasImplementInstructions, "Prompt should contain Implementation Instructions section")
}

func TestBuildBugSystemPrompt_ImplementPhase_ContainsTaskProgressInstructions(t *testing.T) {
	session := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "This is a test bug",
		CurrentPhase: types.BugPhaseImplement,
		PlanFile:     "/tmp/plan.md",
		TasksFile:    "/tmp/tasks.md",
		BugDir:       "/tmp/bug",
	}

	prompt := buildBugSystemPrompt(session)

	// Verify that the hybrid prompt contains the mandatory task progress instructions
	mandatoryInstructions := []string{
		"Create task-progress.md as first action (MANDATORY)",
		"Update task-progress.md for every status change (MANDATORY)",
		"Use explicit task IDs",
		"MANDATORY",
	}

	for _, instruction := range mandatoryInstructions {
		assert.Contains(t, prompt, instruction, "Missing critical instruction: %s", instruction)
	}
}

func TestBuildBugSystemPrompt_HybridTemplateLength(t *testing.T) {
	// Test that implement phase produces significantly longer prompt due to hybrid template
	planSession := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "This is a test bug",
		CurrentPhase: types.BugPhasePlan,
		PlanFile:     "/tmp/plan.md",
		TasksFile:    "/tmp/tasks.md",
		BugDir:       "/tmp/bug",
	}

	implementSession := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "This is a test bug",
		CurrentPhase: types.BugPhaseImplement,
		PlanFile:     "/tmp/plan.md",
		TasksFile:    "/tmp/tasks.md",
		BugDir:       "/tmp/bug",
	}

	planPrompt := buildBugSystemPrompt(planSession)
	implementPrompt := buildBugSystemPrompt(implementSession)

	// Implement phase should have significantly more content due to hybrid template
	assert.Greater(t, len(implementPrompt), len(planPrompt)*2,
		"Implement phase prompt should be significantly longer due to hybrid template loading")

	// Verify implement prompt contains both templates
	assert.Contains(t, implementPrompt, "Bug Fix Agent") // From bug template
	assert.Contains(t, implementPrompt, "Implementation Agent") // From implement template
}

func TestBuildBugSystemPrompt_FallbackBehavior(t *testing.T) {
	// Test with minimal session to trigger fallback
	session := &types.BugSession{
		ID:           "test-bug-001",
		Title:        "Test Bug",
		Description:  "This is a test bug",
		CurrentPhase: types.BugPhasePlan,
	}

	prompt := buildBugSystemPrompt(session)

	// Should still generate a valid prompt even if templates fail to load
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Test Bug")
}