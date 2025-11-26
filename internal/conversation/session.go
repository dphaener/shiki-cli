package conversation

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a conversation session.
// This is the unified model for both collaboration and specify modes.
type Session struct {
	ID     string        `json:"id"`
	Mode   SessionMode   `json:"mode"`
	Status SessionStatus `json:"status"`
	Title  string        `json:"title"`

	// Participants in this session
	Participants []Participant `json:"participants"`

	// Turn management
	CurrentTurn       int    `json:"current_turn"`
	MaxTurns          int    `json:"max_turns"`
	ActiveParticipant string `json:"active_participant"` // ID of currently active participant

	// Aggregate metrics
	TotalTokens int     `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Mode-specific metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Collaboration mode specific
	WorkDir        string `json:"work_dir,omitempty"`
	TaskFile       string `json:"task_file,omitempty"`
	DeliverablePath string `json:"deliverable_path,omitempty"`

	// Specify mode specific
	FeatureID   string `json:"feature_id,omitempty"`
	FeatureName string `json:"feature_name,omitempty"`
}

// NewSession creates a new session with the given mode.
func NewSession(mode SessionMode, title string) *Session {
	now := time.Now()
	return &Session{
		ID:           uuid.New().String(),
		Mode:         mode,
		Status:       SessionStatusCreated,
		Title:        title,
		Participants: make([]Participant, 0),
		CurrentTurn:  0,
		MaxTurns:     100, // Default, can be overridden
		CreatedAt:    now,
		UpdatedAt:    now,
		Metadata:     make(map[string]interface{}),
	}
}

// NewCollaborationSession creates a session for multi-agent collaboration.
func NewCollaborationSession(title, workDir, taskFile string, maxTurns int) *Session {
	s := NewSession(ModeCollaboration, title)
	s.WorkDir = workDir
	s.TaskFile = taskFile
	s.MaxTurns = maxTurns
	return s
}

// NewSpecifySession creates a session for single-agent specify mode.
func NewSpecifySession(featureID, featureName string) *Session {
	s := NewSession(ModeSpecify, featureName)
	s.FeatureID = featureID
	s.FeatureName = featureName
	s.MaxTurns = 0 // Unlimited for specify mode
	return s
}

// AddParticipant adds a participant to the session.
func (s *Session) AddParticipant(p *Participant) {
	s.Participants = append(s.Participants, *p)
	if s.ActiveParticipant == "" {
		s.ActiveParticipant = p.ID
	}
	s.UpdatedAt = time.Now()
}

// GetParticipant returns a participant by ID.
func (s *Session) GetParticipant(id string) *Participant {
	for i := range s.Participants {
		if s.Participants[i].ID == id {
			return &s.Participants[i]
		}
	}
	return nil
}

// GetParticipantByName returns a participant by name.
func (s *Session) GetParticipantByName(name string) *Participant {
	for i := range s.Participants {
		if s.Participants[i].Name == name {
			return &s.Participants[i]
		}
	}
	return nil
}

// GetActiveParticipant returns the currently active participant.
func (s *Session) GetActiveParticipant() *Participant {
	return s.GetParticipant(s.ActiveParticipant)
}

// SetActiveParticipant sets the active participant by ID.
func (s *Session) SetActiveParticipant(id string) {
	s.ActiveParticipant = id
	s.UpdatedAt = time.Now()
}

// NextTurn increments the turn counter and optionally switches participants.
func (s *Session) NextTurn(nextParticipantID string) {
	s.CurrentTurn++
	if nextParticipantID != "" {
		s.ActiveParticipant = nextParticipantID
	}
	s.UpdatedAt = time.Now()
}

// Start marks the session as running.
func (s *Session) Start() {
	s.Status = SessionStatusRunning
	now := time.Now()
	s.StartedAt = &now
	s.UpdatedAt = now
}

// Pause marks the session as paused.
func (s *Session) Pause() {
	s.Status = SessionStatusPaused
	s.UpdatedAt = time.Now()
}

// Resume marks the session as running again.
func (s *Session) Resume() {
	s.Status = SessionStatusRunning
	s.UpdatedAt = time.Now()
}

// Complete marks the session as completed.
func (s *Session) Complete(deliverablePath string) {
	s.Status = SessionStatusCompleted
	s.DeliverablePath = deliverablePath
	now := time.Now()
	s.CompletedAt = &now
	s.UpdatedAt = now
}

// Error marks the session as errored.
func (s *Session) Error() {
	s.Status = SessionStatusError
	s.UpdatedAt = time.Now()
}

// IsRunning returns true if the session is currently running.
func (s *Session) IsRunning() bool {
	return s.Status == SessionStatusRunning
}

// IsCompleted returns true if the session is completed.
func (s *Session) IsCompleted() bool {
	return s.Status == SessionStatusCompleted
}

// CanContinue returns true if the session can accept more turns.
func (s *Session) CanContinue() bool {
	if s.Status != SessionStatusRunning {
		return false
	}
	if s.MaxTurns > 0 && s.CurrentTurn >= s.MaxTurns {
		return false
	}
	return true
}

// UpdateMetrics aggregates metrics from all participants.
func (s *Session) UpdateMetrics() {
	s.TotalTokens = 0
	s.TotalCost = 0
	for _, p := range s.Participants {
		s.TotalTokens += p.TotalTokens
		s.TotalCost += p.TotalCost
	}
	s.UpdatedAt = time.Now()
}

// SetMetadata sets a metadata value.
func (s *Session) SetMetadata(key string, value interface{}) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
}

// GetMetadata retrieves a metadata value.
func (s *Session) GetMetadata(key string) (interface{}, bool) {
	if s.Metadata == nil {
		return nil, false
	}
	v, ok := s.Metadata[key]
	return v, ok
}
