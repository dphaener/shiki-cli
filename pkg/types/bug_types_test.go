package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBugPhaseGetDisplayPhases(t *testing.T) {
	var bugPhase BugPhase
	phases := bugPhase.GetDisplayPhases()

	expected := []BugPhase{BugPhasePlan, BugPhaseTasks, BugPhaseImplement}
	assert.Equal(t, expected, phases)
}

func TestBugPhaseGetPhaseStatus(t *testing.T) {
	var bugPhase BugPhase

	tests := []struct {
		name         string
		phase        BugPhase
		currentPhase BugPhase
		expected     BugPhaseStatus
	}{
		{
			name:         "plan phase when current is plan",
			phase:        BugPhasePlan,
			currentPhase: BugPhasePlan,
			expected:     BugPhaseStatusCurrent,
		},
		{
			name:         "plan phase when current is approve plan",
			phase:        BugPhasePlan,
			currentPhase: BugPhaseApprovePlan,
			expected:     BugPhaseStatusCurrent,
		},
		{
			name:         "plan phase when current is tasks",
			phase:        BugPhasePlan,
			currentPhase: BugPhaseTasks,
			expected:     BugPhaseStatusComplete,
		},
		{
			name:         "tasks phase when current is plan",
			phase:        BugPhaseTasks,
			currentPhase: BugPhasePlan,
			expected:     BugPhaseStatusPending,
		},
		{
			name:         "tasks phase when current is approve tasks",
			phase:        BugPhaseTasks,
			currentPhase: BugPhaseApproveTasks,
			expected:     BugPhaseStatusCurrent,
		},
		{
			name:         "implement phase when current is tasks",
			phase:        BugPhaseImplement,
			currentPhase: BugPhaseTasks,
			expected:     BugPhaseStatusPending,
		},
		{
			name:         "implement phase when current is implement",
			phase:        BugPhaseImplement,
			currentPhase: BugPhaseImplement,
			expected:     BugPhaseStatusCurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bugPhase.GetPhaseStatus(tt.phase, tt.currentPhase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBugPhaseGetPhaseName(t *testing.T) {
	var bugPhase BugPhase

	tests := []struct {
		phase    BugPhase
		expected string
	}{
		{BugPhasePlan, "Plan"},
		{BugPhaseApprovePlan, "Approve Plan"},
		{BugPhaseTasks, "Tasks"},
		{BugPhaseApproveTasks, "Approve Tasks"},
		{BugPhaseImplement, "Implement"},
		{BugPhaseComplete, "Complete"},
		{BugPhase("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			result := bugPhase.GetPhaseName(tt.phase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBugPhaseCanGoBack(t *testing.T) {
	var bugPhase BugPhase

	tests := []struct {
		phase    BugPhase
		expected bool
	}{
		{BugPhasePlan, false},        // Can't go back from first phase
		{BugPhaseApprovePlan, true},
		{BugPhaseTasks, true},
		{BugPhaseApproveTasks, true},
		{BugPhaseImplement, true},
		{BugPhaseComplete, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			result := bugPhase.CanGoBack(tt.phase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBugPhaseGetPreviousPhase(t *testing.T) {
	var bugPhase BugPhase

	tests := []struct {
		phase    BugPhase
		expected BugPhase
	}{
		{BugPhasePlan, BugPhasePlan},            // No previous phase
		{BugPhaseApprovePlan, BugPhasePlan},
		{BugPhaseTasks, BugPhaseApprovePlan},
		{BugPhaseApproveTasks, BugPhaseTasks},
		{BugPhaseImplement, BugPhaseApproveTasks},
		{BugPhaseComplete, BugPhaseComplete},    // No previous phase defined
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			result := bugPhase.GetPreviousPhase(tt.phase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBugPhaseGetNextPhase(t *testing.T) {
	var bugPhase BugPhase

	tests := []struct {
		phase    BugPhase
		expected BugPhase
	}{
		{BugPhasePlan, BugPhaseApprovePlan},
		{BugPhaseApprovePlan, BugPhaseTasks},
		{BugPhaseTasks, BugPhaseApproveTasks},
		{BugPhaseApproveTasks, BugPhaseImplement},
		{BugPhaseImplement, BugPhaseComplete},
		{BugPhaseComplete, BugPhaseComplete},    // No next phase
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			result := bugPhase.GetNextPhase(tt.phase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBugPhaseIsApprovalPhase(t *testing.T) {
	var bugPhase BugPhase

	tests := []struct {
		phase    BugPhase
		expected bool
	}{
		{BugPhasePlan, false},
		{BugPhaseApprovePlan, true},
		{BugPhaseTasks, false},
		{BugPhaseApproveTasks, true},
		{BugPhaseImplement, false},
		{BugPhaseComplete, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			result := bugPhase.IsApprovalPhase(tt.phase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBugPhaseWorkflow(t *testing.T) {
	// Test the complete workflow progression
	var bugPhase BugPhase

	// Start at plan phase
	currentPhase := BugPhasePlan

	// Progress through all phases
	phases := []BugPhase{
		BugPhasePlan,
		BugPhaseApprovePlan,
		BugPhaseTasks,
		BugPhaseApproveTasks,
		BugPhaseImplement,
		BugPhaseComplete,
	}

	for i, expectedPhase := range phases {
		assert.Equal(t, expectedPhase, currentPhase, "Phase %d should be %s", i, expectedPhase)

		if currentPhase == BugPhaseComplete {
			break
		}

		currentPhase = bugPhase.GetNextPhase(currentPhase)
	}
}

func TestBugPhaseBackwardNavigation(t *testing.T) {
	// Test going back through phases
	var bugPhase BugPhase

	// Start from implement phase and go back
	currentPhase := BugPhaseImplement

	// Should be able to go back to: approve_tasks, tasks, approve_plan, plan
	expectedPrevious := []BugPhase{
		BugPhaseApproveTasks,
		BugPhaseTasks,
		BugPhaseApprovePlan,
		BugPhasePlan,
	}

	for _, expectedPhase := range expectedPrevious {
		assert.True(t, bugPhase.CanGoBack(currentPhase), "Should be able to go back from %s", currentPhase)
		currentPhase = bugPhase.GetPreviousPhase(currentPhase)
		assert.Equal(t, expectedPhase, currentPhase, "Previous phase should be %s", expectedPhase)
	}

	// Can't go back from plan phase
	assert.False(t, bugPhase.CanGoBack(currentPhase), "Should not be able to go back from plan phase")
}