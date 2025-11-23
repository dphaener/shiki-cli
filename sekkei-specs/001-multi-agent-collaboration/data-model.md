---
description: "Data model for multi-agent collaboration system"
---

# Data Model

This document defines the core entities, their attributes, relationships, and state transitions for the multi-agent collaboration CLI.

## Entity Definitions

### Session

Represents a single collaboration run with unique ID, workspace, state, and metadata.

**Attributes**:
- `id` (string): Unique identifier in format `YYYYMMDD-HHMMSS-<random>`, e.g., `20251123-143022-a7f3`
- `task_name` (string): Human-readable name from task template
- `task_template_path` (string): Original path to task template file
- `workspace_dir` (string): Absolute path to session workspace directory
- `status` (enum): Current state - `running`, `paused`, `completed`, `incomplete`, `error`
- `created_at` (timestamp): When session was created
- `started_at` (timestamp): When first turn began
- `completed_at` (timestamp, optional): When session ended
- `current_turn` (int): Current turn number (0-indexed)
- `max_turns` (int): Maximum allowed turns from template
- `agent1` (Agent): First agent configuration
- `agent2` (Agent): Second agent configuration
- `turn_history` ([]Turn): Ordered list of completed turns
- `total_cost` (float64): Cumulative cost in USD across all turns
- `total_tokens` (int): Cumulative token count across all turns
- `deliverable_path` (string, optional): Path to final deliverable.md if submitted

**Relationships**:
- Has exactly 2 Agents (agent1, agent2)
- Has 0 to max_turns Turns
- Has 0 or 1 Deliverable
- Contains Workspace

**State Transitions**:
```
[created] → running → {completed, incomplete, error}
running → paused → running
paused → {completed, incomplete, error}
```

**Persistence**: Serialized to `session_state.json` in workspace directory

---

### Agent

An AI agent process with ID, role, system prompt, workspace, and configuration.

**Attributes**:
- `id` (string): Unique identifier, e.g., `agent_1`, `agent_2`
- `name` (string): Human-readable name from template, e.g., "Architect", "Reviewer"
- `role` (string): Role description from template
- `system_prompt` (string): System prompt provided to agent
- `model` (string): Claude model identifier, e.g., `claude-sonnet-4`
- `workspace_dir` (string): Absolute path to agent's workspace
- `memory_file` (string): Path to agent's private memory.md
- `total_turns` (int): Number of turns this agent has executed
- `total_tokens` (int): Cumulative tokens consumed by this agent
- `total_cost` (float64): Cumulative cost in USD for this agent
- `environment_vars` (map[string]string): Environment variables for agent process

**Relationships**:
- Belongs to exactly 1 Session
- Has 0 to N Turns
- Owns 1 memory file

**Runtime State** (not serialized):
- `client` (*claude.Client): SDK client instance
- `message_chan` (chan claude.Message): Message receive channel
- `error_chan` (chan error): Error receive channel
- `process_pid` (int): OS process ID when running

---

### Turn

A single execution cycle for one agent with start time, end time, metrics, and status.

**Attributes**:
- `number` (int): Turn number (0-indexed)
- `agent_id` (string): Which agent executed this turn
- `started_at` (timestamp): When turn started
- `completed_at` (timestamp): When turn completed
- `duration_ms` (int64): Turn duration in milliseconds
- `status` (enum): `in_progress`, `completed`, `error`, `timeout`
- `tokens_used` (int): Tokens consumed in this turn
- `cost` (float64): Cost in USD for this turn
- `tool_calls` (int): Number of MCP tool invocations
- `messages_sent` (int): Number of messages sent to other agent
- `error_message` (string, optional): Error details if status = error

**Relationships**:
- Belongs to exactly 1 Session
- Executed by exactly 1 Agent
- May have 0 to N ToolInvocations

**State Transitions**:
```
in_progress → {completed, error, timeout}
```

**Persistence**: Stored in Session.turn_history array

---

### Message

Communication between agents stored as Markdown file with YAML frontmatter.

**File Structure**:
```markdown
---
from: agent_1
to: agent_2
timestamp: 2025-11-23T14:30:45Z
turn: 5
---

[Markdown content of message]
```

**Attributes**:
- `id` (string): Message ID, e.g., `msg-<turn>-<agent_id>-<seq>`
- `from` (string): Sending agent ID
- `to` (string): Receiving agent ID
- `timestamp` (timestamp): When message was sent
- `turn` (int): Turn number when sent
- `content` (string): Markdown message body
- `file_path` (string): Path to message file, e.g., `messages/05-agent_1-to-agent_2.md`

**Relationships**:
- Belongs to exactly 1 Session
- Sent by exactly 1 Agent
- Received by exactly 1 Agent
- Created during exactly 1 Turn

**Storage**: Individual markdown files in `<workspace>/messages/` directory

---

### Workspace

File-based session directory containing all collaboration artifacts.

**Directory Structure**:
```
<workspace_dir>/
├── session_state.json          # Session metadata and state
├── task.md                      # Copy of original task template
├── shared_context.md            # Shared collaborative knowledge
├── deliverable.md               # Final approved deliverable (if submitted)
├── orchestrator.log             # Structured JSON logs
├── messages/                    # Agent-to-agent messages
│   ├── 00-agent_1-to-agent_2.md
│   ├── 01-agent_2-to-agent_1.md
│   └── ...
├── memory/                      # Per-agent private memory
│   ├── agent_1_memory.md
│   └── agent_2_memory.md
└── [custom files/dirs]          # From template workspace_structure
```

**Attributes**:
- `root_dir` (string): Absolute path to workspace root
- `session_id` (string): Associated session ID
- `created_at` (timestamp): When workspace was created

**Relationships**:
- Belongs to exactly 1 Session
- Contains 0 to N Messages
- Contains 0 or 1 Deliverable
- Contains exactly 2 memory files (one per agent)

**Operations**:
- Initialize: Create directory structure from template workspace_structure
- Watch: Monitor for file changes using fsnotify
- Cleanup: Delete workspace when session is cleaned

---

### TaskTemplate

YAML + Markdown file defining agent configuration and task description.

**File Structure**:
```yaml
---
agent1_name: "Architect"
agent1_role: "System designer"
agent1_system_prompt: "You are an expert system architect..."
agent1_model: "claude-sonnet-4"

agent2_name: "Reviewer"
agent2_role: "Design critic"
agent2_system_prompt: "You are a meticulous reviewer..."
agent2_model: "claude-sonnet-4"

max_turns: 10
turn_timeout_seconds: 300
turn_delay_ms: 0

workspace_structure:
  - path: "designs/"
    type: "directory"
  - path: "designs/draft.md"
    type: "file"
    content: "# Draft Design\n\n"

completion_criteria: "Both agents invoke submit_deliverable with identical content"

cost_limits:
  per_agent: 5.00
  total: 10.00
---

# Task Description

[Markdown task description provided to both agents]
```

**Attributes**:
- `agent1_name` (string, required)
- `agent1_role` (string, required)
- `agent1_system_prompt` (string, required)
- `agent1_model` (string, optional, default: "claude-sonnet-4")
- `agent2_name` (string, required)
- `agent2_role` (string, required)
- `agent2_system_prompt` (string, required)
- `agent2_model` (string, optional, default: "claude-sonnet-4")
- `max_turns` (int, required)
- `turn_timeout_seconds` (int, optional, default: 300)
- `turn_delay_ms` (int, optional, default: 0)
- `workspace_structure` ([]WorkspaceItem, optional)
- `completion_criteria` (string, optional)
- `cost_limits` (CostLimits, optional)
- `task_body` (string): Markdown content after frontmatter

**Validation Rules**:
- All required fields must be present
- max_turns must be > 0
- agent names must be non-empty
- YAML frontmatter must be valid

**Relationships**:
- Used by 0 to N Sessions

---

### Event

State change notification distributed via EventBus to subscribers.

**Base Attributes** (all events):
- `type` (enum): Event type identifier
- `session_id` (string): Associated session ID
- `timestamp` (timestamp): When event occurred

**Event Types**:

#### SessionCreated
- `session` (Session): Newly created session

#### SessionStarted
- `session` (Session): Session beginning execution

#### SessionPaused
- `session` (Session): Session paused by user

#### SessionResumed
- `session` (Session): Session resumed from pause

#### SessionCompleted
- `session` (Session): Session completed successfully
- `deliverable_path` (string): Path to deliverable

#### SessionIncomplete
- `session` (Session): Session ended without completion (max turns reached)

#### SessionError
- `session` (Session): Session encountered critical error
- `error` (string): Error message

#### TurnStarted
- `turn` (Turn): Turn beginning execution
- `agent_id` (string): Which agent is executing

#### TurnCompleted
- `turn` (Turn): Turn completed successfully

#### TurnError
- `turn` (Turn): Turn encountered error
- `error` (string): Error message

#### MessageSent
- `message` (Message): Message sent between agents

#### FileUpdated
- `path` (string): Path to updated file
- `operation` (enum): `created`, `modified`, `deleted`

#### ToolInvoked
- `tool_name` (string): Name of MCP tool called
- `agent_id` (string): Which agent invoked tool
- `turn` (int): Current turn number
- `args` (map[string]interface{}): Tool arguments

#### CostThresholdExceeded
- `agent_id` (string, optional): If per-agent limit exceeded
- `current_cost` (float64): Current cost
- `limit` (float64): Configured limit

**Distribution**: Published to EventBus, delivered to subscribers via Go channels

---

### Tool (MCP)

MCP tool implementation providing file operations to agents.

**Tool Definitions**:

#### send_message
Sends a markdown message to the other agent.

**Arguments**:
- `content` (string, required): Markdown message content

**Returns**:
- `message_path` (string): Path to created message file

**Validation**:
- content must be non-empty
- content must be valid UTF-8

---

#### read_messages
Reads all messages from the other agent.

**Arguments**:
- `since_turn` (int, optional): Only return messages from turns >= since_turn

**Returns**:
- `messages` ([]Message): Array of message objects with metadata

**Validation**:
- since_turn must be >= 0 if provided

---

#### write_shared_context
Overwrites shared_context.md with new content.

**Arguments**:
- `content` (string, required): New content for shared context

**Returns**:
- `success` (bool): true

**Validation**:
- content must be valid UTF-8
- atomic write (temp file + rename)

---

#### read_shared_context
Reads current shared_context.md content.

**Arguments**: None

**Returns**:
- `content` (string): Current shared context markdown

---

#### update_memory
Overwrites agent's private memory.md with new content.

**Arguments**:
- `content` (string, required): New memory content

**Returns**:
- `success` (bool): true

**Validation**:
- content must be valid UTF-8
- atomic write
- agent can only update their own memory

---

#### read_memory
Reads agent's private memory.md content.

**Arguments**: None

**Returns**:
- `content` (string): Current memory markdown

**Validation**:
- agent can only read their own memory

---

#### submit_deliverable
Submits final deliverable for review. Session completes when both agents submit identical content.

**Arguments**:
- `content` (string, required): Final deliverable markdown
- `approved` (bool, required): Whether agent approves this deliverable

**Returns**:
- `status` (string): `pending` (waiting for other agent) or `completed` (both approved)

**Validation**:
- content must be non-empty
- approved must be true (agents can't submit unapproved deliverables)

**Completion Logic**:
- If both agents have submitted and content matches exactly → session status = completed
- If contents don't match → continue collaboration

---

### SessionState (Persistence)

Serializable snapshot of session for pause/resume functionality.

**JSON Structure**:
```json
{
  "version": "1.0",
  "session": {
    "id": "20251123-143022-a7f3",
    "task_name": "Architecture Review",
    "status": "paused",
    "created_at": "2025-11-23T14:30:22Z",
    "started_at": "2025-11-23T14:30:25Z",
    "current_turn": 7,
    "max_turns": 10
  },
  "agents": [
    {
      "id": "agent_1",
      "name": "Architect",
      "total_turns": 4,
      "total_tokens": 15420,
      "total_cost": 0.42
    },
    {
      "id": "agent_2",
      "name": "Reviewer",
      "total_turns": 3,
      "total_tokens": 12305,
      "total_cost": 0.35
    }
  ],
  "turn_history": [
    {
      "number": 0,
      "agent_id": "agent_1",
      "started_at": "2025-11-23T14:30:25Z",
      "completed_at": "2025-11-23T14:31:03Z",
      "duration_ms": 38000,
      "status": "completed",
      "tokens_used": 2345,
      "cost": 0.06,
      "tool_calls": 2,
      "messages_sent": 1
    }
  ],
  "deliverable_submitted": false,
  "total_cost": 0.77,
  "total_tokens": 27725
}
```

**Not Serialized** (reconstructed on resume):
- EventBus channels
- File watchers
- Agent SDK client instances
- Goroutines
- MCP server process

---

## Relationships Diagram

```
Session (1) ─── (2) Agent
   │
   ├─── (0..max_turns) Turn
   │
   ├─── (1) Workspace
   │       │
   │       ├─── (0..N) Message
   │       ├─── (2) Memory files
   │       └─── (0..1) Deliverable
   │
   └─── (1) TaskTemplate

Turn (N) ─── (1) Agent
Turn (1) ─── (0..N) ToolInvocation

Event (N) ─── (1) Session
```

---

## State Machine: Session Lifecycle

```
┌─────────┐
│ Created │ (session directory initialized, state.json written)
└────┬────┘
     │ run command
     ▼
┌─────────┐
│ Running │◄──┐ (agents executing turns)
└────┬────┘   │
     │         │ resume command
     ├─────────┼─► Paused (Ctrl+C, SIGTERM)
     │         │
     ├─────────┼─► Completed (both agents submit matching deliverable)
     │         │
     ├─────────┼─► Incomplete (max turns reached, no deliverable)
     │         │
     └─────────┴─► Error (agent crash, API failure, disk full)
```

---

## State Machine: Turn Lifecycle

```
┌────────────┐
│ Not Started│
└──────┬─────┘
       │ orchestrator starts turn
       ▼
┌────────────┐
│ In Progress│ (agent executing, SDK client active)
└──────┬─────┘
       │
       ├──► Completed (agent finishes within timeout)
       ├──► Error (agent process crash, API error)
       └──► Timeout (turn exceeds turn_timeout_seconds)
```

---

## File Persistence Strategy

| Entity | Storage Format | Location | Update Frequency |
|--------|---------------|----------|------------------|
| Session | JSON | `session_state.json` | After each turn, on pause |
| Agent | Part of Session JSON | `session_state.json` | After each turn |
| Turn | Part of Session JSON | `session_state.json` | After turn completes |
| Message | Markdown with frontmatter | `messages/<turn>-<from>-to-<to>.md` | On send_message call |
| Workspace | Directory structure | `<sessions_dir>/<session_id>/` | On session create |
| TaskTemplate | YAML + Markdown | User-provided path | Read-only |
| shared_context | Markdown | `shared_context.md` | On write_shared_context call |
| memory | Markdown | `memory/agent_<id>_memory.md` | On update_memory call |
| deliverable | Markdown with metadata | `deliverable.md` | On submit_deliverable (both approve) |
| logs | JSONL (JSON Lines) | `orchestrator.log` | Real-time append |

---

## Atomic Write Pattern

All file writes use atomic rename to prevent corruption:

```go
// Write to temp file
tmpPath := filepath.Join(dir, ".tmp-" + filename)
os.WriteFile(tmpPath, data, 0600)

// Atomic rename
os.Rename(tmpPath, filepath.Join(dir, filename))
```

This ensures that file reads never see partially-written content, critical for pause/resume reliability.

---

## Concurrency Considerations

- **Session state**: Only orchestrator goroutine writes session_state.json
- **Messages**: Agents alternate turns (no concurrent writes to messages/)
- **shared_context**: Agents alternate turns (sequential writes, no locking needed)
- **memory**: Each agent writes only to their own file (no conflicts)
- **EventBus**: Non-blocking publish with select/default to prevent slow subscribers from blocking
- **File watcher**: Single goroutine reads fsnotify events, publishes to EventBus

---

## Validation Rules Summary

### Session
- ID must match format `YYYYMMDD-HHMMSS-<random>`
- max_turns must be > 0
- current_turn must be >= 0 and <= max_turns
- Must have exactly 2 agents

### Agent
- ID must be `agent_1` or `agent_2`
- name, role, system_prompt must be non-empty
- model must be valid Claude model identifier

### Turn
- number must be >= 0 and < session.max_turns
- duration_ms must be >= 0
- tokens_used must be >= 0
- cost must be >= 0.0

### Message
- from and to must be valid agent IDs
- content must be non-empty
- timestamp must be valid RFC3339

### TaskTemplate
- All required fields present (see TaskTemplate attributes)
- YAML frontmatter must parse successfully
- max_turns > 0

---

## Index & Query Patterns

**List all sessions**:
- Scan `<sessions_dir>/**/session_state.json`
- Parse JSON, extract status, created_at, id
- Sort by created_at descending

**Get session by ID**:
- Load `<sessions_dir>/<session_id>/session_state.json`

**List messages for session**:
- Glob `<workspace_dir>/messages/*.md`
- Parse YAML frontmatter for metadata
- Sort by turn, timestamp

**Find running sessions**:
- Filter session list where status = "running"

**Calculate total cost**:
- Sum agent1.total_cost + agent2.total_cost from session_state.json
