package events

import (
	"time"

	"github.com/darinhaener/collab/pkg/types"
)

// Event wraps typed events with common metadata.
//
// Deprecated: Use broker.Event interface and typed events from internal/broker instead.
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

	AssistantMessagePayload struct {
		AgentID string
		Turn    int
		Content string
	}

	CostThresholdExceededPayload struct {
		AgentID     string // empty for session-level
		CurrentCost float64
		Limit       float64
	}

	// Assistant logging event payloads
	AssistantRequestPayload struct {
		SessionID      string                 `json:"session_id"`
		OrchestratorID string                 `json:"orchestrator"`
		AgentID        string                 `json:"agent_id"`
		ConversationID string                 `json:"conversation_id"`
		Content        string                 `json:"content"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
	}

	AssistantResponsePayload struct {
		SessionID      string                 `json:"session_id"`
		OrchestratorID string                 `json:"orchestrator"`
		AgentID        string                 `json:"agent_id"`
		ConversationID string                 `json:"conversation_id"`
		Content        string                 `json:"content"`
		TokensUsed     int                    `json:"tokens_used,omitempty"`
		TimingMs       int64                  `json:"timing_ms,omitempty"`
		Model          string                 `json:"model,omitempty"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
	}

	AssistantErrorPayload struct {
		SessionID      string                 `json:"session_id"`
		OrchestratorID string                 `json:"orchestrator"`
		AgentID        string                 `json:"agent_id"`
		ErrorType      string                 `json:"error_type"`
		ErrorMessage   string                 `json:"error_message"`
		Context        string                 `json:"context,omitempty"`
		RecoveryAction string                 `json:"recovery_action,omitempty"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
	}

	AssistantMetadataPayload struct {
		SessionID       string                 `json:"session_id"`
		OrchestratorID  string                 `json:"orchestrator"`
		AgentID         string                 `json:"agent_id"`
		TokensUsed      int                    `json:"tokens_used,omitempty"`
		Cost            float64                `json:"cost,omitempty"`
		DurationMs      int64                  `json:"duration_ms,omitempty"`
		Model           string                 `json:"model,omitempty"`
		PerformanceData map[string]interface{} `json:"performance_data,omitempty"`
		Metadata        map[string]interface{} `json:"metadata,omitempty"`
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

func NewAssistantMessage(agentID, sessionID string, turn int, content string) Event {
	return Event{
		Type:      types.EventAssistantMessage,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload:   AssistantMessagePayload{AgentID: agentID, Turn: turn, Content: content},
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

// Helper constructors for assistant logging events
func NewAssistantRequest(sessionID, orchestratorID, agentID, conversationID, content string, metadata map[string]interface{}) Event {
	return Event{
		Type:      types.EventAssistantRequest,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload: AssistantRequestPayload{
			SessionID:      sessionID,
			OrchestratorID: orchestratorID,
			AgentID:        agentID,
			ConversationID: conversationID,
			Content:        content,
			Metadata:       metadata,
		},
	}
}

func NewAssistantResponse(sessionID, orchestratorID, agentID, conversationID, content string, tokensUsed int, timingMs int64, model string, metadata map[string]interface{}) Event {
	return Event{
		Type:      types.EventAssistantResponse,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload: AssistantResponsePayload{
			SessionID:      sessionID,
			OrchestratorID: orchestratorID,
			AgentID:        agentID,
			ConversationID: conversationID,
			Content:        content,
			TokensUsed:     tokensUsed,
			TimingMs:       timingMs,
			Model:          model,
			Metadata:       metadata,
		},
	}
}

func NewAssistantError(sessionID, orchestratorID, agentID, errorType, errorMessage, context, recoveryAction string, metadata map[string]interface{}) Event {
	return Event{
		Type:      types.EventAssistantError,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload: AssistantErrorPayload{
			SessionID:      sessionID,
			OrchestratorID: orchestratorID,
			AgentID:        agentID,
			ErrorType:      errorType,
			ErrorMessage:   errorMessage,
			Context:        context,
			RecoveryAction: recoveryAction,
			Metadata:       metadata,
		},
	}
}

func NewAssistantMetadata(sessionID, orchestratorID, agentID string, tokensUsed int, cost float64, durationMs int64, model string, performanceData, metadata map[string]interface{}) Event {
	return Event{
		Type:      types.EventAssistantMetadata,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Payload: AssistantMetadataPayload{
			SessionID:       sessionID,
			OrchestratorID:  orchestratorID,
			AgentID:         agentID,
			TokensUsed:      tokensUsed,
			Cost:            cost,
			DurationMs:      durationMs,
			Model:           model,
			PerformanceData: performanceData,
			Metadata:        metadata,
		},
	}
}
