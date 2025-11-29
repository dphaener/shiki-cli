package types

import "time"

// BugPhase represents the current phase of the bug fix workflow
type BugPhase string

const (
	BugPhasePlan         BugPhase = "plan"          // Combined discovery + planning
	BugPhaseApprovePlan  BugPhase = "approve_plan"  // Plan approval gate
	BugPhaseTasks        BugPhase = "tasks"         // Task breakdown
	BugPhaseApproveTasks BugPhase = "approve_tasks" // Tasks approval gate
	BugPhaseImplement    BugPhase = "implement"     // Execution
	BugPhaseComplete     BugPhase = "complete"      // Bug fix complete
)

// BugPhaseStatus represents the visual status of a bug workflow phase
type BugPhaseStatus string

const (
	BugPhaseStatusPending  BugPhaseStatus = "pending"
	BugPhaseStatusCurrent  BugPhaseStatus = "current"
	BugPhaseStatusComplete BugPhaseStatus = "complete"
)

// BugSession represents a complete bug fix workflow
type BugSession struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	CurrentPhase BugPhase  `json:"current_phase"`

	// File paths
	PlanFile     string `json:"plan_file"`
	TasksFile    string `json:"tasks_file"`
	BugDir       string `json:"bug_dir"`

	// Phase checkpoints for go-back support
	Checkpoints map[BugPhase]*PhaseCheckpoint `json:"checkpoints"`

	// Chat histories for each phase
	PlanChatHistory       []ChatMessage `json:"plan_chat_history,omitempty"`
	TasksChatHistory      []ChatMessage `json:"tasks_chat_history,omitempty"`
	ImplementChatHistory  []ChatMessage `json:"implement_chat_history,omitempty"`

	// Current artifact content
	CurrentPlan  string `json:"current_plan,omitempty"`
	CurrentTasks string `json:"current_tasks,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Status
	Status SessionStatus `json:"status"`
}

// BugTransition represents a state transition in the bug workflow
type BugTransition struct {
	From      BugPhase
	To        BugPhase
	Timestamp time.Time
}

// GetDisplayPhases returns the main phases for progress stepper display
// (excludes approval sub-phases) - 3 phases for bug workflow vs 4 for feature workflow
func (b BugPhase) GetDisplayPhases() []BugPhase {
	return []BugPhase{
		BugPhasePlan,
		BugPhaseTasks,
		BugPhaseImplement,
	}
}

// GetPhaseStatus returns the visual status of a phase relative to current phase
func (b BugPhase) GetPhaseStatus(phase, currentPhase BugPhase) BugPhaseStatus {
	displayPhases := b.GetDisplayPhases()

	// Find indices
	phaseIdx := -1
	currentIdx := -1
	for i, p := range displayPhases {
		if p == phase {
			phaseIdx = i
		}
		// Map approval phases to their parent phase
		switch currentPhase {
		case BugPhaseApprovePlan:
			if p == BugPhasePlan {
				currentIdx = i
			}
		case BugPhaseApproveTasks:
			if p == BugPhaseTasks {
				currentIdx = i
			}
		default:
			if p == currentPhase {
				currentIdx = i
			}
		}
	}

	if phaseIdx < 0 || currentIdx < 0 {
		return BugPhaseStatusPending
	}

	if phaseIdx < currentIdx {
		return BugPhaseStatusComplete
	} else if phaseIdx == currentIdx {
		return BugPhaseStatusCurrent
	}
	return BugPhaseStatusPending
}

// GetPhaseName returns a human-readable name for the phase
func (b BugPhase) GetPhaseName(phase BugPhase) string {
	names := map[BugPhase]string{
		BugPhasePlan:         "Plan",
		BugPhaseApprovePlan:  "Approve Plan",
		BugPhaseTasks:        "Tasks",
		BugPhaseApproveTasks: "Approve Tasks",
		BugPhaseImplement:    "Implement",
		BugPhaseComplete:     "Complete",
	}
	if name, ok := names[phase]; ok {
		return name
	}
	return string(phase)
}

// CanGoBack returns true if the current phase supports going back
func (b BugPhase) CanGoBack(phase BugPhase) bool {
	// Can't go back from the first phase or its approval
	return phase != BugPhasePlan
}

// GetPreviousPhase returns the phase to go back to
func (b BugPhase) GetPreviousPhase(phase BugPhase) BugPhase {
	transitions := map[BugPhase]BugPhase{
		BugPhaseApprovePlan:  BugPhasePlan,
		BugPhaseTasks:        BugPhaseApprovePlan,
		BugPhaseApproveTasks: BugPhaseTasks,
		BugPhaseImplement:    BugPhaseApproveTasks,
	}
	if prev, ok := transitions[phase]; ok {
		return prev
	}
	return phase
}

// GetNextPhase returns the next phase after approval
func (b BugPhase) GetNextPhase(phase BugPhase) BugPhase {
	transitions := map[BugPhase]BugPhase{
		BugPhasePlan:         BugPhaseApprovePlan,
		BugPhaseApprovePlan:  BugPhaseTasks,
		BugPhaseTasks:        BugPhaseApproveTasks,
		BugPhaseApproveTasks: BugPhaseImplement,
		BugPhaseImplement:    BugPhaseComplete,
	}
	if next, ok := transitions[phase]; ok {
		return next
	}
	return phase
}

// IsApprovalPhase returns true if the phase is an approval gate
func (b BugPhase) IsApprovalPhase(phase BugPhase) bool {
	return phase == BugPhaseApprovePlan ||
		phase == BugPhaseApproveTasks
}

// Bug workflow event types
const (
	EventBugWorkflowPhaseTransition EventType = "bug_workflow.phase_transition"
	EventBugWorkflowApproved        EventType = "bug_workflow.approved"
	EventBugWorkflowGoBack          EventType = "bug_workflow.go_back"
	EventBugWorkflowComplete        EventType = "bug_workflow.complete"
)