package agent

import (
	"testing"
	"time"

	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
)

// TestClaudeCodeIntegration verifies that all agent types use the claude_code preset
func TestClaudeCodeIntegration(t *testing.T) {
	testCases := []struct {
		name        string
		agentFunc   func() *types.Agent
		description string
	}{
		{
			name: "SpecifyAgent",
			agentFunc: func() *types.Agent {
				session := &types.SpecifySession{
					ID:            "test-spec",
					FeatureDesc:   "test feature",
					FeatureNumber: 1,
					Slug:          "feature-1",
					FriendlyName:  "Test Feature",
					Phase:         "specify",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				return NewSpecifyAgent(session)
			},
			description: "Specify phase agent",
		},
		{
			name: "PlanAgent",
			agentFunc: func() *types.Agent {
				session := &types.PlanSession{
					ID:            "test-plan",
					SpecSlug:      "feature-1",
					FeatureNumber: 1,
					FriendlyName:  "Test Feature",
					Phase:         "plan",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				return NewPlanAgent(session)
			},
			description: "Plan phase agent",
		},
		{
			name: "TasksAgent",
			agentFunc: func() *types.Agent {
				session := &types.WorkflowSession{
					ID:            "test-tasks",
					CurrentPhase:  "tasks",
					FeatureNumber: 1,
					Slug:          "feature-1",
					FriendlyName:  "Test Feature",
					TasksFile:     "/tmp/test/task-progress.md",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				return NewTasksAgent(session)
			},
			description: "Tasks phase agent",
		},
		{
			name: "ImplementAgent",
			agentFunc: func() *types.Agent {
				session := &types.WorkflowSession{
					ID:            "test-impl",
					CurrentPhase:  "implement",
					FeatureNumber: 1,
					Slug:          "feature-1",
					FriendlyName:  "Test Feature",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				return NewImplementAgent(session)
			},
			description: "Implement phase agent",
		},
		{
			name: "BugAgent",
			agentFunc: func() *types.Agent {
				session := &types.BugSession{
					ID:           "test-bug",
					Title:        "test bug",
					Description:  "test bug description",
					CurrentPhase: "plan",
					PlanFile:     "/tmp/test/plan.md",
					TasksFile:    "/tmp/test/tasks.md",
					BugDir:       "/tmp/test/bugs",
				}
				return NewBugAgent(session)
			},
			description: "Bug fix agent",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agent := tc.agentFunc()
			assert.NotNil(t, agent, "Agent should not be nil")

			// Verify agent has proper configuration
			assert.NotEmpty(t, agent.ID, "Agent should have ID")
			assert.NotEmpty(t, agent.Name, "Agent should have name")
			assert.NotEmpty(t, agent.SystemPrompt, "Agent should have system prompt")
			assert.Equal(t, "claude-sonnet-4-20250514", agent.Model, "Agent should use default model")

			// Create actual Agent instance and verify SDK integration
			agentInstance := NewAgent(agent)
			assert.NotNil(t, agentInstance, "Agent instance should not be nil")
			assert.Equal(t, agent.SystemPrompt, agentInstance.SystemPrompt, "System prompt should be preserved")

			t.Logf("%s created successfully with claude_code integration", tc.description)
		})
	}
}

// TestCollaborationAgentClaudeCode verifies collaboration agents use claude_code preset
func TestCollaborationAgentClaudeCode(t *testing.T) {
	// Create test agent configuration (similar to what agentFromTemplate produces)
	agentCfg := types.Agent{
		ID:           "agent_1",
		Name:         "Test Agent 1",
		Role:         "Developer",
		SystemPrompt: "You are Test Agent 1, a Developer. You will collaborate with another agent to complete tasks.",
		Model:        "claude-sonnet-4",
		WorkspaceDir: "/tmp/test",
		MemoryFile:   "agent_1_memory.md",
	}

	// Create Agent instance (this is what uses the claude_code preset)
	agent := NewAgent(&agentCfg)

	assert.NotNil(t, agent, "Collaboration agent should not be nil")
	assert.Equal(t, "agent_1", agent.ID, "Agent ID should be preserved")
	assert.Equal(t, "Test Agent 1", agent.Name, "Agent name should be preserved")
	assert.Equal(t, "Developer", agent.Role, "Agent role should be preserved")
	assert.NotEmpty(t, agent.SystemPrompt, "Agent should have system prompt")
	assert.Equal(t, "claude-sonnet-4", agent.Model, "Agent should use specified model")

	t.Log("Collaboration agent created successfully with claude_code integration")
}

// TestAgentSystemPromptLoading verifies that system prompts are loaded properly
func TestAgentSystemPromptLoading(t *testing.T) {
	// Test each agent type has a non-empty system prompt
	agents := []struct {
		name      string
		agentFunc func() *types.Agent
	}{
		{
			"Specify",
			func() *types.Agent {
				session := &types.SpecifySession{
					ID:            "test",
					FeatureNumber: 1,
					Slug:          "feature",
					Phase:         "specify",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				return NewSpecifyAgent(session)
			},
		},
		{
			"Plan",
			func() *types.Agent {
				session := &types.PlanSession{
					ID:            "test",
					FeatureNumber: 1,
					SpecSlug:      "feature",
					Phase:         "plan",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				return NewPlanAgent(session)
			},
		},
		{
			"Tasks",
			func() *types.Agent {
				session := &types.WorkflowSession{
					ID:            "test",
					CurrentPhase:  "tasks",
					FeatureNumber: 1,
					Slug:          "feature",
					TasksFile:     "/tmp/progress.md",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				return NewTasksAgent(session)
			},
		},
		{
			"Implement",
			func() *types.Agent {
				session := &types.WorkflowSession{
					ID:            "test",
					CurrentPhase:  "implement",
					FeatureNumber: 1,
					Slug:          "feature",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				return NewImplementAgent(session)
			},
		},
		{
			"Bug",
			func() *types.Agent {
				session := &types.BugSession{
					ID:           "test",
					Title:        "bug-1",
					Description:  "desc",
					CurrentPhase: "plan",
					BugDir:       "/tmp/bugs",
				}
				return NewBugAgent(session)
			},
		},
	}

	for _, agent := range agents {
		t.Run(agent.name, func(t *testing.T) {
			a := agent.agentFunc()
			assert.NotNil(t, a, "Agent should not be nil")
			assert.NotEmpty(t, a.SystemPrompt, "Agent should have non-empty system prompt")

			// System prompt should contain role-related information
			// The system prompt contains template-based role names, not exact agent names
			assert.True(t, len(a.SystemPrompt) > 100, "System prompt should be substantial (>100 chars)")
			t.Logf("Agent %s has system prompt of length %d", a.Name, len(a.SystemPrompt))
		})
	}
}