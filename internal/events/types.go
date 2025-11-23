package events

import (
	"time"

	"github.com/darinhaener/collab/pkg/types"
)

// Event wraps typed events with common metadata
type Event struct {
	Type      types.EventType
	SessionID string
	Timestamp time.Time
	Payload   interface{}
}

// Event payload types for structured access
type (
	SessionCreatedPayload struct {
		Session *types.Session
	}

	SessionStartedPayload struct {
		Session *types.Session
	}

	SessionPausedPayload struct {
		Session *types.Session
	}

	SessionResumedPayload struct {
		Session *types.Session
	}

	SessionCompletedPayload struct {
		Session         *types.Session
		DeliverablePath string
	}

	SessionIncompletePayload struct {
		Session *types.Session
	}

	SessionErrorPayload struct {
		Session *types.Session
		Error   string
	}

	TurnStartedPayload struct {
		Turn    *types.Turn
		AgentID string
	}

	TurnCompletedPayload struct {
		Turn *types.Turn
	}

	TurnErrorPayload struct {
		Turn  *types.Turn
		Error string
	}

	MessageSentPayload struct {
		Message *types.Message
	}

	FileUpdatedPayload struct {
		Path      string
		Operation string // "created", "modified", "deleted"
	}

	ToolInvokedPayload struct {
		ToolName string
		AgentID  string
		Turn     int
		Args     map[string]interface{}
	}

	CostThresholdExceededPayload struct {
		AgentID     string // empty for session-level
		CurrentCost float64
		Limit       float64
	}
)

// Helper constructors for common events
func NewSessionCreated(session *types.Session) Event {
	return Event{
		Type:      types.EventSessionCreated,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Payload:   SessionCreatedPayload{Session: session},
	}
}

func NewSessionStarted(session *types.Session) Event {
	return Event{
		Type:      types.EventSessionStarted,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Payload:   SessionStartedPayload{Session: session},
	}
}

func NewSessionPaused(session *types.Session) Event {
	return Event{
		Type:      types.EventSessionPaused,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Payload:   SessionPausedPayload{Session: session},
	}
}

func NewSessionResumed(session *types.Session) Event {
	return Event{
		Type:      types.EventSessionResumed,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Payload:   SessionResumedPayload{Session: session},
	}
}

func NewSessionCompleted(session *types.Session, deliverablePath string) Event {
	return Event{
		Type:      types.EventSessionCompleted,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Payload:   SessionCompletedPayload{Session: session, DeliverablePath: deliverablePath},
	}
}

func NewSessionIncomplete(session *types.Session) Event {
	return Event{
		Type:      types.EventSessionIncomplete,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Payload:   SessionIncompletePayload{Session: session},
	}
}

func NewSessionError(session *types.Session, err string) Event {
	return Event{
		Type:      types.EventSessionError,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Payload:   SessionErrorPayload{Session: session, Error: err},
	}
}

func NewTurnStarted(turn *types.Turn, agentID, sessionID string) Event {
	return Event{
		Type:      types.EventTurnStarted,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   TurnStartedPayload{Turn: turn, AgentID: agentID},
	}
}

func NewTurnCompleted(turn *types.Turn, sessionID string) Event {
	return Event{
		Type:      types.EventTurnCompleted,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   TurnCompletedPayload{Turn: turn},
	}
}

func NewTurnError(turn *types.Turn, sessionID, err string) Event {
	return Event{
		Type:      types.EventTurnError,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   TurnErrorPayload{Turn: turn, Error: err},
	}
}

func NewMessageSent(message *types.Message, sessionID string) Event {
	return Event{
		Type:      types.EventMessageSent,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   MessageSentPayload{Message: message},
	}
}

func NewFileUpdated(path, operation, sessionID string) Event {
	return Event{
		Type:      types.EventFileUpdated,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   FileUpdatedPayload{Path: path, Operation: operation},
	}
}

func NewToolInvoked(toolName, agentID, sessionID string, turn int, args map[string]interface{}) Event {
	return Event{
		Type:      types.EventToolInvoked,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   ToolInvokedPayload{ToolName: toolName, AgentID: agentID, Turn: turn, Args: args},
	}
}

func NewCostThresholdExceeded(agentID, sessionID string, currentCost, limit float64) Event {
	return Event{
		Type:      types.EventCostThresholdExceeded,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   CostThresholdExceededPayload{AgentID: agentID, CurrentCost: currentCost, Limit: limit},
	}
}
