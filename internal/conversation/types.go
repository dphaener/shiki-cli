// Package conversation provides core domain types for chat/turn-based functionality.
// This package serves both collaboration mode (multi-agent) and specify mode (single-agent chat).
package conversation

// PartType identifies the type of content within a message.
type PartType string

const (
	PartTypeText       PartType = "text"
	PartTypeToolCall   PartType = "tool_call"
	PartTypeToolResult PartType = "tool_result"
	PartTypeReasoning  PartType = "reasoning"
)

// Role identifies the message author.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// ToolCallStatus tracks the lifecycle of a tool invocation.
type ToolCallStatus string

const (
	ToolCallPending   ToolCallStatus = "pending"
	ToolCallRunning   ToolCallStatus = "running"
	ToolCallCompleted ToolCallStatus = "completed"
	ToolCallFailed    ToolCallStatus = "failed"
)

// FinishReason indicates why a message stream ended.
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonToolUse   FinishReason = "tool_use"
	FinishReasonMaxTokens FinishReason = "max_tokens"
	FinishReasonError     FinishReason = "error"
)

// SessionMode identifies the type of conversation.
type SessionMode string

const (
	ModeCollaboration SessionMode = "collaboration"
	ModeSpecify       SessionMode = "specify"
)

// SessionStatus represents conversation state.
type SessionStatus string

const (
	SessionStatusCreated   SessionStatus = "created"
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusPaused    SessionStatus = "paused"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusError     SessionStatus = "error"
)

// ParticipantType distinguishes agents from users.
type ParticipantType string

const (
	ParticipantAgent ParticipantType = "agent"
	ParticipantUser  ParticipantType = "user"
)
