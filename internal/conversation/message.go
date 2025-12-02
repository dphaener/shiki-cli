package conversation

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/dphaener/shiki-cli/internal/diff"
	"github.com/google/uuid"
)

// Part represents a discrete piece of message content.
// Each message can contain multiple parts of different types.
type Part interface {
	Type() PartType
	Timestamp() time.Time
}

// partWrapper is used for JSON marshaling/unmarshaling of Part interface.
type partWrapper struct {
	PartType PartType        `json:"type"`
	Data     json.RawMessage `json:"data"`
}

// TextPart contains plain text or markdown content.
type TextPart struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (p TextPart) Type() PartType       { return PartTypeText }
func (p TextPart) Timestamp() time.Time { return p.CreatedAt }

// NewTextPart creates a new text part with generated ID.
func NewTextPart(content string) TextPart {
	return TextPart{
		ID:        uuid.New().String(),
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// ToolCallPart represents a tool invocation request.
type ToolCallPart struct {
	ID           string                 `json:"id"`
	ToolName     string                 `json:"tool_name"`
	Input        map[string]interface{} `json:"input"`
	Status       ToolCallStatus         `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	FinishedAt   *time.Time             `json:"finished_at,omitempty"`
	// Tool chain fields for rollup behavior
	ChainID      string                 `json:"chain_id,omitempty"`      // ID of tool chain this belongs to
	IsReplaced   bool                   `json:"is_replaced,omitempty"`   // True if replaced by later tool in chain
	Resource     string                 `json:"resource,omitempty"`      // Primary resource this tool operates on
}

func (p ToolCallPart) Type() PartType       { return PartTypeToolCall }
func (p ToolCallPart) Timestamp() time.Time { return p.CreatedAt }

// NewToolCallPart creates a new tool call part.
func NewToolCallPart(id, toolName string, input map[string]interface{}) ToolCallPart {
	return ToolCallPart{
		ID:        id,
		ToolName:  toolName,
		Input:     input,
		Status:    ToolCallPending,
		CreatedAt: time.Now(),
	}
}

// MarkRunning updates the tool call status to running.
func (p *ToolCallPart) MarkRunning() {
	p.Status = ToolCallRunning
	now := time.Now()
	p.StartedAt = &now
}

// MarkCompleted updates the tool call status to completed.
func (p *ToolCallPart) MarkCompleted() {
	p.Status = ToolCallCompleted
	now := time.Now()
	p.FinishedAt = &now
}

// MarkFailed updates the tool call status to failed.
func (p *ToolCallPart) MarkFailed() {
	p.Status = ToolCallFailed
	now := time.Now()
	p.FinishedAt = &now
}

// SetChainInfo updates the tool chain information.
func (p *ToolCallPart) SetChainInfo(chainID, resource string) {
	p.ChainID = chainID
	p.Resource = resource
}

// MarkReplaced marks this tool as replaced by a later tool in the chain.
func (p *ToolCallPart) MarkReplaced() {
	p.IsReplaced = true
}

// UnmarkReplaced removes the replaced status (for undo operations).
func (p *ToolCallPart) UnmarkReplaced() {
	p.IsReplaced = false
}

// IsInChain returns true if this tool is part of a tool chain.
func (p *ToolCallPart) IsInChain() bool {
	return p.ChainID != ""
}

// ToolResultPart contains the result of a tool invocation.
type ToolResultPart struct {
	ID         string         `json:"id"`
	ToolCallID string         `json:"tool_call_id"`
	Content    string         `json:"content"`
	IsError    bool           `json:"is_error"`
	CreatedAt  time.Time      `json:"created_at"`
	Diff       *diff.FileDiff `json:"diff,omitempty"` // Optional diff information for Write tool
}

func (p ToolResultPart) Type() PartType       { return PartTypeToolResult }
func (p ToolResultPart) Timestamp() time.Time { return p.CreatedAt }

// NewToolResultPart creates a new tool result part.
func NewToolResultPart(toolCallID, content string, isError bool) ToolResultPart {
	return ToolResultPart{
		ID:         uuid.New().String(),
		ToolCallID: toolCallID,
		Content:    content,
		IsError:    isError,
		CreatedAt:  time.Now(),
		Diff:       nil,
	}
}

// NewToolResultPartWithDiff creates a new tool result part with diff information.
func NewToolResultPartWithDiff(toolCallID, content string, isError bool, fileDiff *diff.FileDiff) ToolResultPart {
	return ToolResultPart{
		ID:         uuid.New().String(),
		ToolCallID: toolCallID,
		Content:    content,
		IsError:    isError,
		CreatedAt:  time.Now(),
		Diff:       fileDiff,
	}
}

// ReasoningPart contains model reasoning/thinking content.
type ReasoningPart struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (p ReasoningPart) Type() PartType       { return PartTypeReasoning }
func (p ReasoningPart) Timestamp() time.Time { return p.CreatedAt }

// NewReasoningPart creates a new reasoning part.
func NewReasoningPart(content string) ReasoningPart {
	return ReasoningPart{
		ID:        uuid.New().String(),
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// Message represents a single message in a conversation.
type Message struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	ParticipantID string    `json:"participant_id"`
	Role          Role      `json:"role"`
	Parts         []Part    `json:"-"` // Custom marshaling below
	Turn          int       `json:"turn"`
	IsStreaming   bool      `json:"is_streaming"`
	StreamBuffer  string    `json:"-"` // Not persisted
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewMessage creates a new message with generated ID.
func NewMessage(sessionID, participantID string, role Role, turn int) *Message {
	now := time.Now()
	return &Message{
		ID:            uuid.New().String(),
		SessionID:     sessionID,
		ParticipantID: participantID,
		Role:          role,
		Parts:         make([]Part, 0),
		Turn:          turn,
		IsStreaming:   false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// AddPart appends a part to the message.
func (m *Message) AddPart(part Part) {
	m.Parts = append(m.Parts, part)
	m.UpdatedAt = time.Now()
}

// AddText is a convenience method to add a text part.
func (m *Message) AddText(content string) {
	m.AddPart(NewTextPart(content))
}

// AddToolCall is a convenience method to add a tool call part.
func (m *Message) AddToolCall(id, toolName string, input map[string]interface{}) {
	m.AddPart(NewToolCallPart(id, toolName, input))
}

// AddToolResult is a convenience method to add a tool result part.
func (m *Message) AddToolResult(toolCallID, content string, isError bool) {
	m.AddPart(NewToolResultPart(toolCallID, content, isError))
}

// AddToolResultWithDiff is a convenience method to add a tool result part with diff information.
func (m *Message) AddToolResultWithDiff(toolCallID, content string, isError bool, fileDiff *diff.FileDiff) {
	m.AddPart(NewToolResultPartWithDiff(toolCallID, content, isError, fileDiff))
}

// GetText returns all text content concatenated.
func (m *Message) GetText() string {
	var text strings.Builder
	for _, part := range m.Parts {
		if tp, ok := part.(TextPart); ok {
			text.WriteString(tp.Content)
		}
	}
	return text.String()
}

// GetToolCalls returns all tool call parts.
func (m *Message) GetToolCalls() []ToolCallPart {
	var calls []ToolCallPart
	for _, part := range m.Parts {
		if tc, ok := part.(ToolCallPart); ok {
			calls = append(calls, tc)
		}
	}
	return calls
}

// GetToolResults returns all tool result parts.
func (m *Message) GetToolResults() []ToolResultPart {
	var results []ToolResultPart
	for _, part := range m.Parts {
		if tr, ok := part.(ToolResultPart); ok {
			results = append(results, tr)
		}
	}
	return results
}

// FindToolCall finds a tool call by ID.
func (m *Message) FindToolCall(toolCallID string) *ToolCallPart {
	for i := range m.Parts {
		if tc, ok := m.Parts[i].(ToolCallPart); ok && tc.ID == toolCallID {
			return &tc
		}
	}
	return nil
}

// UpdateToolCallStatus updates the status of a tool call by ID.
func (m *Message) UpdateToolCallStatus(toolCallID string, status ToolCallStatus) bool {
	for i := range m.Parts {
		if tc, ok := m.Parts[i].(ToolCallPart); ok && tc.ID == toolCallID {
			tc.Status = status
			now := time.Now()
			switch status {
			case ToolCallRunning:
				tc.StartedAt = &now
			case ToolCallCompleted, ToolCallFailed:
				tc.FinishedAt = &now
			}
			m.Parts[i] = tc
			m.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// UpdateToolCallChain updates the tool chain information for a tool call.
func (m *Message) UpdateToolCallChain(toolCallID, chainID, resource string) bool {
	for i := range m.Parts {
		if tc, ok := m.Parts[i].(ToolCallPart); ok && tc.ID == toolCallID {
			tc.SetChainInfo(chainID, resource)
			m.Parts[i] = tc
			m.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// MarkToolCallReplaced marks a tool call as replaced in its chain.
func (m *Message) MarkToolCallReplaced(toolCallID string) bool {
	for i := range m.Parts {
		if tc, ok := m.Parts[i].(ToolCallPart); ok && tc.ID == toolCallID {
			tc.MarkReplaced()
			m.Parts[i] = tc
			m.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// GetToolCallsByChain returns all tool calls belonging to a specific chain.
func (m *Message) GetToolCallsByChain(chainID string) []ToolCallPart {
	var chainTools []ToolCallPart
	for _, part := range m.Parts {
		if tc, ok := part.(ToolCallPart); ok && tc.ChainID == chainID {
			chainTools = append(chainTools, tc)
		}
	}
	return chainTools
}

// GetActiveToolCalls returns tool calls that are not replaced (visible in UI).
func (m *Message) GetActiveToolCalls() []ToolCallPart {
	var activeCalls []ToolCallPart
	for _, part := range m.Parts {
		if tc, ok := part.(ToolCallPart); ok && !tc.IsReplaced {
			activeCalls = append(activeCalls, tc)
		}
	}
	return activeCalls
}

// StartStreaming marks the message as actively streaming.
func (m *Message) StartStreaming() {
	m.IsStreaming = true
	m.StreamBuffer = ""
	m.UpdatedAt = time.Now()
}

// AppendStreamDelta appends text to the stream buffer.
func (m *Message) AppendStreamDelta(delta string) {
	m.StreamBuffer += delta
	m.UpdatedAt = time.Now()
}

// FinishStreaming finalizes streaming and converts buffer to a text part.
func (m *Message) FinishStreaming() {
	if m.StreamBuffer != "" {
		m.AddText(m.StreamBuffer)
		m.StreamBuffer = ""
	}
	m.IsStreaming = false
	m.UpdatedAt = time.Now()
}

// messageJSON is used for JSON marshaling.
type messageJSON struct {
	ID            string        `json:"id"`
	SessionID     string        `json:"session_id"`
	ParticipantID string        `json:"participant_id"`
	Role          Role          `json:"role"`
	Parts         []partWrapper `json:"parts"`
	Turn          int           `json:"turn"`
	IsStreaming   bool          `json:"is_streaming"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// MarshalJSON implements custom JSON marshaling for Message.
func (m Message) MarshalJSON() ([]byte, error) {
	parts := make([]partWrapper, len(m.Parts))
	for i, p := range m.Parts {
		data, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		parts[i] = partWrapper{
			PartType: p.Type(),
			Data:     data,
		}
	}

	return json.Marshal(messageJSON{
		ID:            m.ID,
		SessionID:     m.SessionID,
		ParticipantID: m.ParticipantID,
		Role:          m.Role,
		Parts:         parts,
		Turn:          m.Turn,
		IsStreaming:   m.IsStreaming,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Message.
func (m *Message) UnmarshalJSON(data []byte) error {
	var mj messageJSON
	if err := json.Unmarshal(data, &mj); err != nil {
		return err
	}

	m.ID = mj.ID
	m.SessionID = mj.SessionID
	m.ParticipantID = mj.ParticipantID
	m.Role = mj.Role
	m.Turn = mj.Turn
	m.IsStreaming = mj.IsStreaming
	m.CreatedAt = mj.CreatedAt
	m.UpdatedAt = mj.UpdatedAt
	m.Parts = make([]Part, len(mj.Parts))

	for i, pw := range mj.Parts {
		var part Part
		switch pw.PartType {
		case PartTypeText:
			var tp TextPart
			if err := json.Unmarshal(pw.Data, &tp); err != nil {
				return err
			}
			part = tp
		case PartTypeToolCall:
			var tc ToolCallPart
			if err := json.Unmarshal(pw.Data, &tc); err != nil {
				return err
			}
			part = tc
		case PartTypeToolResult:
			var tr ToolResultPart
			if err := json.Unmarshal(pw.Data, &tr); err != nil {
				return err
			}
			part = tr
		case PartTypeReasoning:
			var rp ReasoningPart
			if err := json.Unmarshal(pw.Data, &rp); err != nil {
				return err
			}
			part = rp
		}
		m.Parts[i] = part
	}

	return nil
}
