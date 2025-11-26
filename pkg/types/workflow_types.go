package types

import "time"

// WorkflowPhase represents the current phase of the unified feature workflow
type WorkflowPhase string

const (
	WorkflowPhaseSpecify      WorkflowPhase = "specify"
	WorkflowPhaseApproveSpec  WorkflowPhase = "approve_spec"
	WorkflowPhasePlan         WorkflowPhase = "plan"
	WorkflowPhaseApprovePlan  WorkflowPhase = "approve_plan"
	WorkflowPhaseTasks        WorkflowPhase = "tasks"
	WorkflowPhaseApproveTasks WorkflowPhase = "approve_tasks"
	WorkflowPhaseImplement    WorkflowPhase = "implement"
	WorkflowPhaseComplete     WorkflowPhase = "complete"
)

// WorkflowPhaseStatus represents the visual status of a workflow phase
type WorkflowPhaseStatus string

const (
	PhaseStatusPending  WorkflowPhaseStatus = "pending"
	PhaseStatusCurrent  WorkflowPhaseStatus = "current"
	PhaseStatusComplete WorkflowPhaseStatus = "complete"
)

// PhaseCheckpoint stores the state at an approval point
type PhaseCheckpoint struct {
	Phase       WorkflowPhase `json:"phase"`
	CompletedAt time.Time     `json:"completed_at"`
	ChatHistory []ChatMessage `json:"chat_history"`
	ArtifactRef string        `json:"artifact_ref"` // Path to the artifact (spec.md, plan.md, etc.)
}

// WorkflowSession represents a complete feature development workflow
type WorkflowSession struct {
	ID            string        `json:"id"`
	CurrentPhase  WorkflowPhase `json:"current_phase"`
	FeatureNumber int           `json:"feature_number"`
	Slug          string        `json:"slug"`
	FriendlyName  string        `json:"friendly_name"`
	FeatureDesc   string        `json:"feature_desc"`

	// File paths
	SpecFile     string `json:"spec_file"`
	PlanFile     string `json:"plan_file"`
	TasksFile    string `json:"tasks_file"`
	FeatureDir   string `json:"feature_dir"`
	ChecklistDir string `json:"checklist_dir"`
	ContractsDir string `json:"contracts_dir"`

	// Phase checkpoints for go-back support
	Checkpoints map[WorkflowPhase]*PhaseCheckpoint `json:"checkpoints"`

	// Embedded sessions for each phase
	SpecifySession   *SpecifySession `json:"specify_session,omitempty"`
	PlanSession      *PlanSession    `json:"plan_session,omitempty"`
	TasksChatHistory []ChatMessage   `json:"tasks_chat_history,omitempty"`
	ImplChatHistory  []ChatMessage   `json:"impl_chat_history,omitempty"`

	// Current artifact content
	CurrentSpec  string `json:"current_spec,omitempty"`
	CurrentPlan  string `json:"current_plan,omitempty"`
	CurrentTasks string `json:"current_tasks,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Status
	Status SessionStatus `json:"status"`
}

// WorkflowTransition represents a state transition in the workflow
type WorkflowTransition struct {
	From      WorkflowPhase
	To        WorkflowPhase
	Timestamp time.Time
}

// GetDisplayPhases returns the main phases for progress stepper display
// (excludes approval sub-phases)
func GetDisplayPhases() []WorkflowPhase {
	return []WorkflowPhase{
		WorkflowPhaseSpecify,
		WorkflowPhasePlan,
		WorkflowPhaseTasks,
		WorkflowPhaseImplement,
	}
}

// GetPhaseStatus returns the visual status of a phase relative to current phase
func GetPhaseStatus(phase, currentPhase WorkflowPhase) WorkflowPhaseStatus {
	displayPhases := GetDisplayPhases()

	// Find indices
	phaseIdx := -1
	currentIdx := -1
	for i, p := range displayPhases {
		if p == phase {
			phaseIdx = i
		}
		// Map approval phases to their parent phase
		switch currentPhase {
		case WorkflowPhaseApproveSpec:
			if p == WorkflowPhaseSpecify {
				currentIdx = i
			}
		case WorkflowPhaseApprovePlan:
			if p == WorkflowPhasePlan {
				currentIdx = i
			}
		case WorkflowPhaseApproveTasks:
			if p == WorkflowPhaseTasks {
				currentIdx = i
			}
		default:
			if p == currentPhase {
				currentIdx = i
			}
		}
	}

	if phaseIdx < 0 || currentIdx < 0 {
		return PhaseStatusPending
	}

	if phaseIdx < currentIdx {
		return PhaseStatusComplete
	} else if phaseIdx == currentIdx {
		return PhaseStatusCurrent
	}
	return PhaseStatusPending
}

// GetPhaseName returns a human-readable name for the phase
func GetPhaseName(phase WorkflowPhase) string {
	names := map[WorkflowPhase]string{
		WorkflowPhaseSpecify:      "Specify",
		WorkflowPhaseApproveSpec:  "Approve Spec",
		WorkflowPhasePlan:         "Plan",
		WorkflowPhaseApprovePlan:  "Approve Plan",
		WorkflowPhaseTasks:        "Tasks",
		WorkflowPhaseApproveTasks: "Approve Tasks",
		WorkflowPhaseImplement:    "Implement",
		WorkflowPhaseComplete:     "Complete",
	}
	if name, ok := names[phase]; ok {
		return name
	}
	return string(phase)
}

// CanGoBack returns true if the current phase supports going back
func CanGoBack(phase WorkflowPhase) bool {
	// Can't go back from the first phase or its approval
	return phase != WorkflowPhaseSpecify
}

// GetPreviousPhase returns the phase to go back to
func GetPreviousPhase(phase WorkflowPhase) WorkflowPhase {
	transitions := map[WorkflowPhase]WorkflowPhase{
		WorkflowPhaseApproveSpec:  WorkflowPhaseSpecify,
		WorkflowPhasePlan:         WorkflowPhaseApproveSpec,
		WorkflowPhaseApprovePlan:  WorkflowPhasePlan,
		WorkflowPhaseTasks:        WorkflowPhaseApprovePlan,
		WorkflowPhaseApproveTasks: WorkflowPhaseTasks,
		WorkflowPhaseImplement:    WorkflowPhaseApproveTasks,
	}
	if prev, ok := transitions[phase]; ok {
		return prev
	}
	return phase
}

// GetNextPhase returns the next phase after approval
func GetNextPhase(phase WorkflowPhase) WorkflowPhase {
	transitions := map[WorkflowPhase]WorkflowPhase{
		WorkflowPhaseSpecify:      WorkflowPhaseApproveSpec,
		WorkflowPhaseApproveSpec:  WorkflowPhasePlan,
		WorkflowPhasePlan:         WorkflowPhaseApprovePlan,
		WorkflowPhaseApprovePlan:  WorkflowPhaseTasks,
		WorkflowPhaseTasks:        WorkflowPhaseApproveTasks,
		WorkflowPhaseApproveTasks: WorkflowPhaseImplement,
		WorkflowPhaseImplement:    WorkflowPhaseComplete,
	}
	if next, ok := transitions[phase]; ok {
		return next
	}
	return phase
}

// IsApprovalPhase returns true if the phase is an approval gate
func IsApprovalPhase(phase WorkflowPhase) bool {
	return phase == WorkflowPhaseApproveSpec ||
		phase == WorkflowPhaseApprovePlan ||
		phase == WorkflowPhaseApproveTasks
}

// Workflow event types
const (
	EventWorkflowPhaseTransition EventType = "workflow.phase_transition"
	EventWorkflowApproved        EventType = "workflow.approved"
	EventWorkflowGoBack          EventType = "workflow.go_back"
	EventWorkflowComplete        EventType = "workflow.complete"
)
