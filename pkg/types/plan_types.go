package types

import "time"

// PlanPhase represents the current phase of the planning workflow
type PlanPhase string

const (
	PlanPhaseInterrogation PlanPhase = "interrogation"
	PlanPhaseResearch      PlanPhase = "research"
	PlanPhaseDesign        PlanPhase = "design"
	PlanPhaseComplete      PlanPhase = "complete"
)

// PlanSession represents an implementation planning session
type PlanSession struct {
	ID            string        `json:"id"`
	SpecSlug      string        `json:"spec_slug"`
	FeatureNumber int           `json:"feature_number"`
	FriendlyName  string        `json:"friendly_name"`
	Phase         PlanPhase     `json:"phase"`
	ChatHistory   []ChatMessage `json:"chat_history"`
	CurrentPlan   string        `json:"current_plan"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	Status        SessionStatus `json:"status"`
	// Paths
	SpecFile     string `json:"spec_file"`
	SpecDir      string `json:"spec_dir"`
	PlanFile     string `json:"plan_file"`
	ContractsDir string `json:"contracts_dir"`
}

// Plan event types
const (
	EventPlanChatMessage     EventType = "plan.chat_message"
	EventPlanGenerated       EventType = "plan.generated"
	EventPlanPhaseTransition EventType = "plan.phase_transition"
	EventPlanComplete        EventType = "plan.complete"
)
