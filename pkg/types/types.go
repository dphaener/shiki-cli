package types

import "time"

// SessionStatus represents the current state of a collaboration session
type SessionStatus string

const (
	SessionRunning    SessionStatus = "running"
	SessionPaused     SessionStatus = "paused"
	SessionCompleted  SessionStatus = "completed"
	SessionIncomplete SessionStatus = "incomplete"
	SessionError      SessionStatus = "error"
)

// TurnStatus represents the current state of a turn
type TurnStatus string

const (
	TurnInProgress TurnStatus = "in_progress"
	TurnCompleted  TurnStatus = "completed"
	TurnError      TurnStatus = "error"
	TurnTimeout    TurnStatus = "timeout"
)

// EventType identifies the type of event
type EventType string

const (
	EventSessionCreated         EventType = "session.created"
	EventSessionStarted         EventType = "session.started"
	EventSessionPaused          EventType = "session.paused"
	EventSessionResumed         EventType = "session.resumed"
	EventSessionCompleted       EventType = "session.completed"
	EventSessionIncomplete      EventType = "session.incomplete"
	EventSessionError           EventType = "session.error"
	EventTurnStarted            EventType = "turn.started"
	EventTurnCompleted          EventType = "turn.completed"
	EventTurnError              EventType = "turn.error"
	EventMessageSent            EventType = "message.sent"
	EventFileUpdated            EventType = "file.updated"
	EventToolInvoked            EventType = "tool.invoked"
	EventAssistantMessage       EventType = "assistant.message"
	EventCostThresholdExceeded  EventType = "cost.threshold_exceeded"
)

// Session represents a single collaboration run
type Session struct {
	ID                 string        `json:"id"`
	TaskName           string        `json:"task_name"`
	TaskTemplatePath   string        `json:"task_template_path"`
	WorkspaceDir       string        `json:"workspace_dir"`
	Status             SessionStatus `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
	StartedAt          *time.Time    `json:"started_at,omitempty"`
	CompletedAt        *time.Time    `json:"completed_at,omitempty"`
	CurrentTurn        int           `json:"current_turn"`
	MaxTurns           int           `json:"max_turns"`
	Agent1             Agent         `json:"agent1"`
	Agent2             Agent         `json:"agent2"`
	TurnHistory        []Turn        `json:"turn_history"`
	TotalCost          float64       `json:"total_cost"`
	TotalTokens        int           `json:"total_tokens"`
	DeliverablePath    string        `json:"deliverable_path,omitempty"`
	DeliverableAgent1  string        `json:"deliverable_agent1,omitempty"`
	DeliverableAgent2  string        `json:"deliverable_agent2,omitempty"`
}

// Agent represents an AI agent process
type Agent struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Role            string            `json:"role"`
	SystemPrompt    string            `json:"system_prompt"`
	Model           string            `json:"model"`
	WorkspaceDir    string            `json:"workspace_dir"`
	MemoryFile      string            `json:"memory_file"`
	TotalTurns      int               `json:"total_turns"`
	TotalTokens     int               `json:"total_tokens"`
	TotalCost       float64           `json:"total_cost"`
	EnvironmentVars map[string]string `json:"environment_vars,omitempty"`
}

// Turn represents a single execution cycle for one agent
type Turn struct {
	Number       int        `json:"number"`
	AgentID      string     `json:"agent_id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	DurationMs   int64      `json:"duration_ms"`
	Status       TurnStatus `json:"status"`
	TokensUsed   int        `json:"tokens_used"`
	Cost         float64    `json:"cost"`
	ToolCalls    int        `json:"tool_calls"`
	MessagesSent int        `json:"messages_sent"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// Message represents communication between agents
type Message struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp"`
	Turn      int       `json:"turn"`
	Content   string    `json:"content"`
	FilePath  string    `json:"file_path"`
}

// TaskTemplate represents the YAML frontmatter and markdown body of a task file
type TaskTemplate struct {
	// Agent 1 configuration
	Agent1Name         string `yaml:"agent1_name"`
	Agent1Role         string `yaml:"agent1_role"`
	Agent1SystemPrompt string `yaml:"agent1_system_prompt"`
	Agent1Model        string `yaml:"agent1_model,omitempty"`

	// Agent 2 configuration
	Agent2Name         string `yaml:"agent2_name"`
	Agent2Role         string `yaml:"agent2_role"`
	Agent2SystemPrompt string `yaml:"agent2_system_prompt"`
	Agent2Model        string `yaml:"agent2_model,omitempty"`

	// Session configuration
	MaxTurns             int    `yaml:"max_turns"`
	TurnTimeoutSeconds   int    `yaml:"turn_timeout_seconds,omitempty"`
	TurnDelayMs          int    `yaml:"turn_delay_ms,omitempty"`
	CompletionCriteria   string `yaml:"completion_criteria,omitempty"`

	// Optional workspace structure
	WorkspaceStructure []WorkspaceItem `yaml:"workspace_structure,omitempty"`

	// Cost limits
	CostLimits *CostLimits `yaml:"cost_limits,omitempty"`

	// Task body (markdown content after frontmatter)
	TaskBody string `yaml:"-"`
}

// WorkspaceItem represents a file or directory to create in the workspace
type WorkspaceItem struct {
	Path    string `yaml:"path"`
	Type    string `yaml:"type"` // "file" or "directory"
	Content string `yaml:"content,omitempty"`
}

// CostLimits represents cost constraints for the session
type CostLimits struct {
	PerAgent float64 `yaml:"per_agent,omitempty"`
	Total    float64 `yaml:"total,omitempty"`
}

// Workspace represents the file-based session directory
type Workspace struct {
	RootDir   string    `json:"root_dir"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
}
