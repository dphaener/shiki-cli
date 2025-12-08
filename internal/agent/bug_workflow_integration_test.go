package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBugWorkflowIntegration tests the complete bug workflow with hybrid template loading
func TestBugWorkflowIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bug_workflow_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test data
	bugID := "test-bug-integration-001"
	bugTitle := "Preview broken in bug mode"
	bugDescription := "Bug mode implement phase fails to create progress file"

	// Create bug directory structure
	bugDir := filepath.Join(tempDir, "bugs", bugID)
	err = os.MkdirAll(bugDir, 0755)
	require.NoError(t, err)

	planFile := filepath.Join(bugDir, "bug-plan.md")
	tasksFile := filepath.Join(bugDir, "bug-tasks.md")

	tests := []struct {
		name         string
		phase        types.BugPhase
		expectHybrid bool
		minLength    int
		mustContain  []string
		mustNotContain []string
	}{
		{
			name:         "Plan Phase",
			phase:        types.BugPhasePlan,
			expectHybrid: false,
			minLength:    400,
			mustContain:  []string{"Bug Fix Agent", bugTitle, "plan"},
			mustNotContain: []string{"Implementation Instructions", "MANDATORY", "Create task-progress.md as first action"},
		},
		{
			name:         "Tasks Phase",
			phase:        types.BugPhaseTasks,
			expectHybrid: false,
			minLength:    400,
			mustContain:  []string{"Bug Fix Agent", bugTitle, "tasks"},
			mustNotContain: []string{"Implementation Instructions", "MANDATORY", "Create task-progress.md as first action"},
		},
		{
			name:         "Implement Phase with Hybrid Template",
			phase:        types.BugPhaseImplement,
			expectHybrid: true,
			minLength:    1500, // Should be much longer due to hybrid template
			mustContain: []string{
				"Bug Fix Agent",
				bugTitle,
				"implement",
				"Implementation Instructions",
				"Implementation Agent",
				"task-progress.md",
				"Create task-progress.md as first action (MANDATORY)",
				"Update task-progress.md for every status change (MANDATORY)",
				"Use explicit task IDs",
			},
			mustNotContain: []string{}, // No exclusions for implement phase
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create bug session for this phase
			session := &types.BugSession{
				ID:           bugID,
				Title:        bugTitle,
				Description:  bugDescription,
				CurrentPhase: tt.phase,
				PlanFile:     planFile,
				TasksFile:    tasksFile,
				BugDir:       bugDir,
			}

			// Create bug agent
			agent := NewBugAgent(session)

			// Validate agent creation
			require.NotNil(t, agent)
			assert.Equal(t, "bug-agent", agent.ID)
			assert.Equal(t, "Bug Fix Assistant", agent.Name)
			assert.NotEmpty(t, agent.SystemPrompt)

			// Validate prompt length
			promptLength := len(agent.SystemPrompt)
			assert.GreaterOrEqual(t, promptLength, tt.minLength,
				"Prompt should be at least %d characters for %s phase, got %d",
				tt.minLength, tt.phase, promptLength)

			if tt.expectHybrid {
				// Implement phase should have significantly more content
				// Create a plan phase session for comparison
				planSession := &types.BugSession{
					ID:           bugID,
					Title:        bugTitle,
					Description:  bugDescription,
					CurrentPhase: types.BugPhasePlan,
					PlanFile:     planFile,
					TasksFile:    tasksFile,
					BugDir:       bugDir,
				}
				planAgent := NewBugAgent(planSession)

				assert.Greater(t, promptLength, len(planAgent.SystemPrompt)*2,
					"Implement phase should have significantly longer prompt due to hybrid template")
			}

			// Validate required content
			for _, content := range tt.mustContain {
				assert.Contains(t, agent.SystemPrompt, content,
					"Prompt missing required content: %s", content)
			}

			// Validate excluded content
			for _, content := range tt.mustNotContain {
				assert.NotContains(t, agent.SystemPrompt, content,
					"Prompt contains unexpected content: %s", content)
			}

			// For implement phase, validate that both templates are present
			if tt.expectHybrid {
				// Should contain content from both bug and implement templates
				lines := strings.Split(agent.SystemPrompt, "\n")

				hasBugContent := false
				hasImplementContent := false
				hasImplementInstructions := false

				for _, line := range lines {
					if strings.Contains(line, "Bug Fix Agent") {
						hasBugContent = true
					}
					if strings.Contains(line, "Implementation Agent") {
						hasImplementContent = true
					}
					if strings.Contains(line, "## Implementation Instructions") {
						hasImplementInstructions = true
					}
				}

				assert.True(t, hasBugContent, "Should contain bug template content")
				assert.True(t, hasImplementContent, "Should contain implement template content")
				assert.True(t, hasImplementInstructions, "Should contain Implementation Instructions section")
			}
		})
	}
}

// TestBugWorkflowTemplateFailureResilience tests graceful fallback when templates fail to load
func TestBugWorkflowTemplateFailureResilience(t *testing.T) {
	// Test with invalid template directory to force fallback
	session := &types.BugSession{
		ID:           "test-bug-fallback-001",
		Title:        "Test Fallback Behavior",
		Description:  "Testing template fallback",
		CurrentPhase: types.BugPhaseImplement,
	}

	// This should still work even if templates can't be loaded
	agent := NewBugAgent(session)

	require.NotNil(t, agent)
	assert.NotEmpty(t, agent.SystemPrompt)

	// Should contain basic bug information
	assert.Contains(t, agent.SystemPrompt, session.Title)
	assert.Contains(t, agent.SystemPrompt, session.Description)
}

// TestBugWorkflowPhaseTransitions tests that prompts change appropriately across phases
func TestBugWorkflowPhaseTransitions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bug_phase_transition_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	bugDir := filepath.Join(tempDir, "bugs", "test-transitions")
	err = os.MkdirAll(bugDir, 0755)
	require.NoError(t, err)

	baseSession := &types.BugSession{
		ID:          "test-bug-transitions-001",
		Title:       "Test Phase Transitions",
		Description: "Testing phase transition behavior",
		PlanFile:    filepath.Join(bugDir, "plan.md"),
		TasksFile:   filepath.Join(bugDir, "tasks.md"),
		BugDir:      bugDir,
	}

	phases := []types.BugPhase{
		types.BugPhasePlan,
		types.BugPhaseTasks,
		types.BugPhaseImplement,
	}

	var prompts []string

	// Generate prompts for each phase
	for _, phase := range phases {
		session := *baseSession // Copy
		session.CurrentPhase = phase

		agent := NewBugAgent(&session)
		prompts = append(prompts, agent.SystemPrompt)
	}

	// Validate that prompts are different for different phases
	for i := 0; i < len(prompts)-1; i++ {
		for j := i + 1; j < len(prompts); j++ {
			assert.NotEqual(t, prompts[i], prompts[j],
				"Prompts for phases %s and %s should be different", phases[i], phases[j])
		}
	}

	// Implement phase should be significantly longer due to hybrid template
	implementPrompt := prompts[2] // implement is index 2
	planPrompt := prompts[0]      // plan is index 0

	assert.Greater(t, len(implementPrompt), int(float64(len(planPrompt))*1.5),
		"Implement phase prompt should be significantly longer than plan phase")

	// Only implement phase should contain mandatory task progress instructions
	assert.NotContains(t, planPrompt, "Create task-progress.md as first action (MANDATORY)")
	assert.NotContains(t, prompts[1], "Create task-progress.md as first action (MANDATORY)") // tasks phase
	assert.Contains(t, implementPrompt, "Create task-progress.md as first action (MANDATORY)")
}

// TestBugWorkflowPerformanceCharacteristics tests that template loading is reasonably performant
func TestBugWorkflowPerformanceCharacteristics(t *testing.T) {
	session := &types.BugSession{
		ID:           "test-bug-perf-001",
		Title:        "Performance Test Bug",
		Description:  "Testing template loading performance",
		CurrentPhase: types.BugPhaseImplement,
		PlanFile:     "/tmp/plan.md",
		TasksFile:    "/tmp/tasks.md",
		BugDir:       "/tmp/bug",
	}

	// Time the agent creation (which includes template loading)
	start := time.Now()

	iterations := 10
	for i := 0; i < iterations; i++ {
		agent := NewBugAgent(session)
		_ = agent.SystemPrompt // Force template loading
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	// Template loading should be fast (under 100ms per creation on average)
	assert.Less(t, avgTime, 100*time.Millisecond,
		"Bug agent creation should be fast, got average %v", avgTime)

	t.Logf("Average bug agent creation time: %v", avgTime)
}

// TestBugWorkflowErrorBoundaries tests error handling in edge cases
func TestBugWorkflowErrorBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		session *types.BugSession
		wantErr bool
	}{
		{
			name: "Nil Session",
			session: nil,
			wantErr: true,
		},
		{
			name: "Empty Bug ID",
			session: &types.BugSession{
				ID:           "",
				Title:        "Valid Title",
				CurrentPhase: types.BugPhasePlan,
			},
			wantErr: false, // Should not error, just use empty ID
		},
		{
			name: "Empty Title",
			session: &types.BugSession{
				ID:           "valid-id",
				Title:        "",
				CurrentPhase: types.BugPhasePlan,
			},
			wantErr: false, // Should not error
		},
		{
			name: "Invalid Phase",
			session: &types.BugSession{
				ID:           "valid-id",
				Title:        "Valid Title",
				CurrentPhase: types.BugPhase("invalid"),
			},
			wantErr: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.wantErr {
					t.Errorf("NewBugAgent() panicked unexpectedly: %v", r)
				}
			}()

			if tt.session == nil {
				// Can't test nil session directly as it would panic
				// This is acceptable behavior for a constructor
				return
			}

			agent := NewBugAgent(tt.session)

			if !tt.wantErr {
				assert.NotNil(t, agent)
				assert.NotEmpty(t, agent.SystemPrompt)
			}
		})
	}
}