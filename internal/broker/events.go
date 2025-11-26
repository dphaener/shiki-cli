// Package broker provides a typed pub/sub event system for the chat infrastructure.
// This replaces the untyped internal/events package with strongly-typed events.
package broker

import (
	"time"

	"github.com/darinhaener/collab/internal/conversation"
)

// EventType identifies event categories.
type EventType string

const (
	// Session events
	EventSessionCreated EventType = "session.created"
	EventSessionUpdated EventType = "session.updated"
	EventSessionStarted EventType = "session.started"
	EventSessionPaused  EventType = "session.paused"
	EventSessionCompleted EventType = "session.completed"
	EventSessionError   EventType = "session.error"

	// Message events
	EventMessageCreated EventType = "message.created"
	EventMessageUpdated EventType = "message.updated"

	// Part events
	EventPartAdded EventType = "part.added"

	// Streaming events
	EventStreamStarted  EventType = "stream.started"
	EventStreamDelta    EventType = "stream.delta"
	EventStreamFinished EventType = "stream.finished"

	// Tool events
	EventToolStarted   EventType = "tool.started"
	EventToolCompleted EventType = "tool.completed"

	// Turn events
	EventTurnStarted   EventType = "turn.started"
	EventTurnCompleted EventType = "turn.completed"
	EventTurnError     EventType = "turn.error"
)

// Event is the base interface for all events.
type Event interface {
	EventType() EventType
	SessionID() string
	Timestamp() time.Time
}

// BaseEvent provides common event fields.
type BaseEvent struct {
	Type      EventType `json:"type"`
	Session   string    `json:"session_id"`
	CreatedAt time.Time `json:"timestamp"`
}

func (e BaseEvent) EventType() EventType { return e.Type }
func (e BaseEvent) SessionID() string    { return e.Session }
func (e BaseEvent) Timestamp() time.Time { return e.CreatedAt }

func newBaseEvent(eventType EventType, sessionID string) BaseEvent {
	return BaseEvent{
		Type:      eventType,
		Session:   sessionID,
		CreatedAt: time.Now(),
	}
}

// SessionCreatedEvent is emitted when a session is created.
type SessionCreatedEvent struct {
	BaseEvent
	Session *conversation.Session `json:"session"`
}

func NewSessionCreatedEvent(session *conversation.Session) SessionCreatedEvent {
	return SessionCreatedEvent{
		BaseEvent: newBaseEvent(EventSessionCreated, session.ID),
		Session:   session,
	}
}

// SessionUpdatedEvent is emitted when session state changes.
type SessionUpdatedEvent struct {
	BaseEvent
	Session *conversation.Session `json:"session"`
	Changes []string              `json:"changes"` // Field names that changed
}

func NewSessionUpdatedEvent(session *conversation.Session, changes ...string) SessionUpdatedEvent {
	return SessionUpdatedEvent{
		BaseEvent: newBaseEvent(EventSessionUpdated, session.ID),
		Session:   session,
		Changes:   changes,
	}
}

// SessionStartedEvent is emitted when a session starts running.
type SessionStartedEvent struct {
	BaseEvent
	Session *conversation.Session `json:"session"`
}

func NewSessionStartedEvent(session *conversation.Session) SessionStartedEvent {
	return SessionStartedEvent{
		BaseEvent: newBaseEvent(EventSessionStarted, session.ID),
		Session:   session,
	}
}

// SessionPausedEvent is emitted when a session is paused.
type SessionPausedEvent struct {
	BaseEvent
	Session *conversation.Session `json:"session"`
}

func NewSessionPausedEvent(session *conversation.Session) SessionPausedEvent {
	return SessionPausedEvent{
		BaseEvent: newBaseEvent(EventSessionPaused, session.ID),
		Session:   session,
	}
}

// SessionCompletedEvent is emitted when a session completes.
type SessionCompletedEvent struct {
	BaseEvent
	Session         *conversation.Session `json:"session"`
	DeliverablePath string                `json:"deliverable_path"`
}

func NewSessionCompletedEvent(session *conversation.Session, deliverablePath string) SessionCompletedEvent {
	return SessionCompletedEvent{
		BaseEvent:       newBaseEvent(EventSessionCompleted, session.ID),
		Session:         session,
		DeliverablePath: deliverablePath,
	}
}

// SessionErrorEvent is emitted when a session encounters an error.
type SessionErrorEvent struct {
	BaseEvent
	Session *conversation.Session `json:"session"`
	Error   string                `json:"error"`
}

func NewSessionErrorEvent(session *conversation.Session, err error) SessionErrorEvent {
	return SessionErrorEvent{
		BaseEvent: newBaseEvent(EventSessionError, session.ID),
		Session:   session,
		Error:     err.Error(),
	}
}

// MessageCreatedEvent is emitted when a new message is created.
type MessageCreatedEvent struct {
	BaseEvent
	Message *conversation.Message `json:"message"`
}

func NewMessageCreatedEvent(msg *conversation.Message) MessageCreatedEvent {
	return MessageCreatedEvent{
		BaseEvent: newBaseEvent(EventMessageCreated, msg.SessionID),
		Message:   msg,
	}
}

// MessageUpdatedEvent is emitted when a message is modified.
type MessageUpdatedEvent struct {
	BaseEvent
	Message *conversation.Message `json:"message"`
}

func NewMessageUpdatedEvent(msg *conversation.Message) MessageUpdatedEvent {
	return MessageUpdatedEvent{
		BaseEvent: newBaseEvent(EventMessageUpdated, msg.SessionID),
		Message:   msg,
	}
}

// PartAddedEvent is emitted when a part is added to a message.
type PartAddedEvent struct {
	BaseEvent
	MessageID     string            `json:"message_id"`
	ParticipantID string            `json:"participant_id"`
	Part          conversation.Part `json:"part"`
}

func NewPartAddedEvent(sessionID, messageID, participantID string, part conversation.Part) PartAddedEvent {
	return PartAddedEvent{
		BaseEvent:     newBaseEvent(EventPartAdded, sessionID),
		MessageID:     messageID,
		ParticipantID: participantID,
		Part:          part,
	}
}

// StreamStartedEvent is emitted when streaming begins.
type StreamStartedEvent struct {
	BaseEvent
	MessageID     string `json:"message_id"`
	ParticipantID string `json:"participant_id"`
	Turn          int    `json:"turn"`
}

func NewStreamStartedEvent(sessionID, messageID, participantID string, turn int) StreamStartedEvent {
	return StreamStartedEvent{
		BaseEvent:     newBaseEvent(EventStreamStarted, sessionID),
		MessageID:     messageID,
		ParticipantID: participantID,
		Turn:          turn,
	}
}

// StreamDeltaEvent is emitted during streaming.
type StreamDeltaEvent struct {
	BaseEvent
	MessageID     string `json:"message_id"`
	ParticipantID string `json:"participant_id"`
	Delta         string `json:"delta"`
	Index         int    `json:"index"`
}

func NewStreamDeltaEvent(sessionID, messageID, participantID, delta string, index int) StreamDeltaEvent {
	return StreamDeltaEvent{
		BaseEvent:     newBaseEvent(EventStreamDelta, sessionID),
		MessageID:     messageID,
		ParticipantID: participantID,
		Delta:         delta,
		Index:         index,
	}
}

// StreamFinishedEvent is emitted when streaming completes.
type StreamFinishedEvent struct {
	BaseEvent
	MessageID     string                     `json:"message_id"`
	ParticipantID string                     `json:"participant_id"`
	FinishReason  conversation.FinishReason  `json:"finish_reason"`
	InputTokens   int                        `json:"input_tokens"`
	OutputTokens  int                        `json:"output_tokens"`
}

func NewStreamFinishedEvent(sessionID, messageID, participantID string, reason conversation.FinishReason, inputTokens, outputTokens int) StreamFinishedEvent {
	return StreamFinishedEvent{
		BaseEvent:     newBaseEvent(EventStreamFinished, sessionID),
		MessageID:     messageID,
		ParticipantID: participantID,
		FinishReason:  reason,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
	}
}

// ToolStartedEvent is emitted when a tool begins execution.
type ToolStartedEvent struct {
	BaseEvent
	MessageID     string                 `json:"message_id"`
	ParticipantID string                 `json:"participant_id"`
	ToolCallID    string                 `json:"tool_call_id"`
	ToolName      string                 `json:"tool_name"`
	Input         map[string]interface{} `json:"input"`
	Turn          int                    `json:"turn"`
}

func NewToolStartedEvent(sessionID, messageID, participantID, toolCallID, toolName string, input map[string]interface{}, turn int) ToolStartedEvent {
	return ToolStartedEvent{
		BaseEvent:     newBaseEvent(EventToolStarted, sessionID),
		MessageID:     messageID,
		ParticipantID: participantID,
		ToolCallID:    toolCallID,
		ToolName:      toolName,
		Input:         input,
		Turn:          turn,
	}
}

// ToolCompletedEvent is emitted when a tool finishes.
type ToolCompletedEvent struct {
	BaseEvent
	MessageID     string `json:"message_id"`
	ParticipantID string `json:"participant_id"`
	ToolCallID    string `json:"tool_call_id"`
	Output        string `json:"output"`
	IsError       bool   `json:"is_error"`
	DurationMs    int64  `json:"duration_ms"`
}

func NewToolCompletedEvent(sessionID, messageID, participantID, toolCallID, output string, isError bool, durationMs int64) ToolCompletedEvent {
	return ToolCompletedEvent{
		BaseEvent:     newBaseEvent(EventToolCompleted, sessionID),
		MessageID:     messageID,
		ParticipantID: participantID,
		ToolCallID:    toolCallID,
		Output:        output,
		IsError:       isError,
		DurationMs:    durationMs,
	}
}

// TurnStartedEvent is emitted when a turn begins.
type TurnStartedEvent struct {
	BaseEvent
	ParticipantID string `json:"participant_id"`
	Turn          int    `json:"turn"`
}

func NewTurnStartedEvent(sessionID, participantID string, turn int) TurnStartedEvent {
	return TurnStartedEvent{
		BaseEvent:     newBaseEvent(EventTurnStarted, sessionID),
		ParticipantID: participantID,
		Turn:          turn,
	}
}

// TurnCompletedEvent is emitted when a turn completes.
type TurnCompletedEvent struct {
	BaseEvent
	ParticipantID string  `json:"participant_id"`
	Turn          int     `json:"turn"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	Cost          float64 `json:"cost"`
	ToolCalls     int     `json:"tool_calls"`
	DurationMs    int64   `json:"duration_ms"`
}

func NewTurnCompletedEvent(sessionID, participantID string, turn, inputTokens, outputTokens int, cost float64, toolCalls int, durationMs int64) TurnCompletedEvent {
	return TurnCompletedEvent{
		BaseEvent:     newBaseEvent(EventTurnCompleted, sessionID),
		ParticipantID: participantID,
		Turn:          turn,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		Cost:          cost,
		ToolCalls:     toolCalls,
		DurationMs:    durationMs,
	}
}

// TurnErrorEvent is emitted when a turn encounters an error.
type TurnErrorEvent struct {
	BaseEvent
	ParticipantID string `json:"participant_id"`
	Turn          int    `json:"turn"`
	Error         string `json:"error"`
}

func NewTurnErrorEvent(sessionID, participantID string, turn int, err error) TurnErrorEvent {
	return TurnErrorEvent{
		BaseEvent:     newBaseEvent(EventTurnError, sessionID),
		ParticipantID: participantID,
		Turn:          turn,
		Error:         err.Error(),
	}
}
