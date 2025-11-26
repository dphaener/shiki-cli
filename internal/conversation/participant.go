package conversation

import (
	"time"

	"github.com/google/uuid"
)

// Participant represents an agent or user in the conversation.
type Participant struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Role         string          `json:"role"`          // e.g., "architect", "developer", "user"
	Type         ParticipantType `json:"type"`          // agent or user
	SystemPrompt string          `json:"system_prompt,omitempty"`
	Model        string          `json:"model,omitempty"`

	// Metrics
	TotalTokens  int     `json:"total_tokens"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
	ToolCalls    int     `json:"tool_calls"`

	CreatedAt time.Time `json:"created_at"`
}

// NewAgentParticipant creates a new agent participant.
func NewAgentParticipant(name, role, systemPrompt, model string) *Participant {
	return &Participant{
		ID:           uuid.New().String(),
		Name:         name,
		Role:         role,
		Type:         ParticipantAgent,
		SystemPrompt: systemPrompt,
		Model:        model,
		CreatedAt:    time.Now(),
	}
}

// NewUserParticipant creates a new user participant.
func NewUserParticipant(name string) *Participant {
	return &Participant{
		ID:        uuid.New().String(),
		Name:      name,
		Role:      "user",
		Type:      ParticipantUser,
		CreatedAt: time.Now(),
	}
}

// AddTokenUsage records token usage for the participant.
func (p *Participant) AddTokenUsage(input, output int, cost float64) {
	p.InputTokens += input
	p.OutputTokens += output
	p.TotalTokens += input + output
	p.TotalCost += cost
}

// IncrementToolCalls increments the tool call counter.
func (p *Participant) IncrementToolCalls() {
	p.ToolCalls++
}

// IsAgent returns true if this is an agent participant.
func (p *Participant) IsAgent() bool {
	return p.Type == ParticipantAgent
}

// IsUser returns true if this is a user participant.
func (p *Participant) IsUser() bool {
	return p.Type == ParticipantUser
}
