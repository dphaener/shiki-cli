// Package chatservice provides service interfaces for managing chat sessions and messages.
package chatservice

import (
	"context"

	"github.com/dphaener/shiki-cli/internal/conversation"
)

// CreateSessionOpts contains options for creating a session.
type CreateSessionOpts struct {
	Title        string
	MaxTurns     int
	WorkDir      string
	TaskFile     string
	FeatureID    string
	FeatureName  string
	Participants []*conversation.Participant
	Metadata     map[string]interface{}
}

// ListSessionOpts contains options for listing sessions.
type ListSessionOpts struct {
	Mode   *conversation.SessionMode
	Status *conversation.SessionStatus
	Limit  int
	Offset int
}

// ListMessagesOpts contains options for listing messages.
type ListMessagesOpts struct {
	ParticipantID string
	SinceTurn     int
	Limit         int
	Order         string // "asc" or "desc"
}

// SessionService manages conversation sessions.
type SessionService interface {
	// Create creates a new session.
	Create(ctx context.Context, mode conversation.SessionMode, opts CreateSessionOpts) (*conversation.Session, error)

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (*conversation.Session, error)

	// Update saves session state.
	Update(ctx context.Context, session *conversation.Session) error

	// Delete removes a session.
	Delete(ctx context.Context, id string) error

	// List retrieves sessions matching the filter.
	List(ctx context.Context, opts ListSessionOpts) ([]*conversation.Session, error)

	// Lifecycle methods

	// Start marks the session as running.
	Start(ctx context.Context, id string) error

	// Pause marks the session as paused.
	Pause(ctx context.Context, id string) error

	// Resume marks the session as running again.
	Resume(ctx context.Context, id string) error

	// Complete marks the session as completed.
	Complete(ctx context.Context, id string, deliverablePath string) error

	// Error marks the session as errored.
	Error(ctx context.Context, id string, err error) error

	// Turn management

	// NextTurn advances to the next turn.
	NextTurn(ctx context.Context, id string, nextParticipantID string) error

	// GetActive returns the currently active session (if any).
	GetActive(ctx context.Context) (*conversation.Session, error)
}

// MessageService manages messages within sessions.
type MessageService interface {
	// CRUD operations

	// Create creates a new message.
	Create(ctx context.Context, sessionID string, msg *conversation.Message) error

	// Get retrieves a message by ID.
	Get(ctx context.Context, id string) (*conversation.Message, error)

	// Update saves message state.
	Update(ctx context.Context, msg *conversation.Message) error

	// Delete removes a message.
	Delete(ctx context.Context, id string) error

	// List retrieves messages for a session.
	List(ctx context.Context, sessionID string, opts ListMessagesOpts) ([]*conversation.Message, error)

	// Streaming support

	// StartStream creates a new streaming message.
	StartStream(ctx context.Context, sessionID, participantID string, turn int) (*conversation.Message, error)

	// AppendDelta appends text to a streaming message.
	AppendDelta(ctx context.Context, messageID string, delta string) error

	// AddPart adds a part to a message.
	AddPart(ctx context.Context, messageID string, part conversation.Part) error

	// FinishStream finalizes a streaming message.
	FinishStream(ctx context.Context, messageID string, reason conversation.FinishReason, inputTokens, outputTokens int) error

	// Tool support

	// UpdateToolCallStatus updates the status of a tool call within a message.
	UpdateToolCallStatus(ctx context.Context, messageID, toolCallID string, status conversation.ToolCallStatus) error

	// Queries

	// GetByParticipant retrieves messages for a specific participant.
	GetByParticipant(ctx context.Context, sessionID, participantID string) ([]*conversation.Message, error)

	// GetLatest retrieves the most recent message for a session.
	GetLatest(ctx context.Context, sessionID string) (*conversation.Message, error)
}

// Store provides persistence for chat entities.
type Store interface {
	// Session operations
	SaveSession(ctx context.Context, session *conversation.Session) error
	GetSession(ctx context.Context, id string) (*conversation.Session, error)
	ListSessions(ctx context.Context, opts ListSessionOpts) ([]*conversation.Session, error)
	DeleteSession(ctx context.Context, id string) error

	// Message operations
	SaveMessage(ctx context.Context, msg *conversation.Message) error
	GetMessage(ctx context.Context, id string) (*conversation.Message, error)
	ListMessages(ctx context.Context, sessionID string, opts ListMessagesOpts) ([]*conversation.Message, error)
	DeleteMessage(ctx context.Context, id string) error
	DeleteSessionMessages(ctx context.Context, sessionID string) error
}
