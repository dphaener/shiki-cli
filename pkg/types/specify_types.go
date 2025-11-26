package types

import "time"

// SpecifyPhase represents the current phase of the specification workflow
type SpecifyPhase string

const (
	PhaseDiscovery   SpecifyPhase = "discovery"
	PhaseGeneration  SpecifyPhase = "generation"
	PhaseValidation  SpecifyPhase = "validation"
	PhaseComplete    SpecifyPhase = "complete"
)

// SpecifySession represents a feature specification session
type SpecifySession struct {
	ID               string          `json:"id"`
	FeatureDesc      string          `json:"feature_desc"`
	FeatureNumber    int             `json:"feature_number"`
	Slug             string          `json:"slug"`
	FriendlyName     string          `json:"friendly_name"`
	Phase            SpecifyPhase    `json:"phase"`
	ChatHistory      []ChatMessage   `json:"chat_history"`
	CurrentSpec      string          `json:"current_spec"`
	ValidationIssues []string        `json:"validation_issues"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	Status           SessionStatus   `json:"status"`
}

// ChatMessage represents a single chat message in the specification workflow
type ChatMessage struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// SpecifyEventType identifies specify-specific events
type SpecifyEventType string

const (
	EventSpecChatMessage      EventType = "spec.chat_message"
	EventSpecGenerated        EventType = "spec.generated"
	EventSpecValidationResult EventType = "spec.validation_result"
	EventSpecPhaseTransition  EventType = "spec.phase_transition"
	EventSpecComplete         EventType = "spec.complete"
)
