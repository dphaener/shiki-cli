# Architecture Overview

## System Architecture

Collab is a production-ready Go CLI tool that orchestrates collaboration between two AI agents through file-based communication, turn-based execution, and real-time terminal monitoring.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      User Interface                         │
│  ┌──────────────┐                    ┌─────────────────┐    │
│  │ CLI Commands │                    │  Terminal UI    │    │
│  │   (Cobra)    │                    │  (Bubbletea)    │    │
│  └──────┬───────┘                    └────────┬────────┘    │
│         │                                     │             │
│         │        ┌────────────────────────────┘             │
│         ▼        ▼                                          │
├─────────────────────────────────────────────────────────────┤
│                    Orchestrator Layer                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Session Lifecycle │ Turn Execution │ Completion       │   │
│  │ Pause/Resume      │ Metrics        │ Detection        │   │
│  └────────┬──────────────────┬──────────────────┬────────┘   │
│           │                  │                  │            │
├───────────┼──────────────────┼──────────────────┼────────────┤
│           │                  │                  │            │
│  ┌────────▼────────┐  ┌──────▼────────┐  ┌─────▼─────────┐  │
│  │ Agent Manager   │  │  MCP Server   │  │  EventBus     │  │
│  │ (SDK wrapper)   │  │  (STDIO)      │  │  (channels)   │  │
│  └────────┬────────┘  └───────┬───────┘  └───────┬───────┘  │
│           │                   │                  │          │
├───────────┼───────────────────┼──────────────────┼──────────┤
│           │                   │                  │          │
│  ┌────────▼───────────────────▼───────┐  ┌───────▼───────┐  │
│  │       Storage Layer                │  │   Logging     │  │
│  │  Atomic Writes │ File Watching     │  │   (JSON)      │  │
│  │  Session State │ Workspace Mgmt    │  │               │  │
│  └────────────────────────────────────┘  └───────────────┘  │
└─────────────────────────────────────────────────────────────┘
              │                        │
              ▼                        ▼
       ┌──────────────┐        ┌──────────────┐
       │   Agent 1    │        │   Agent 2    │
       │ Claude CLI   │        │ Claude CLI   │
       └──────────────┘        └──────────────┘
```

## Core Components

### 1. CLI Layer (`cmd/shiki/`, `internal/cli/`)

**Responsibility**: User interaction, command parsing, and output formatting.

**Technology**: [spf13/cobra](https://github.com/spf13/cobra) for command structure and flag parsing.

**Commands**:
- `run` - Start new collaboration session
- `resume` - Continue paused session
- `list` - Show all sessions
- `show` - Display session details
- `watch` - Attach TUI to session
- `validate` - Check template validity
- `init` - Create new task template
- `clean` - Remove old sessions
- `version` - Display version info

**Key Files**:
- `cmd/shiki/main.go` - Entry point, root command setup
- `internal/cli/run.go` - Session creation and orchestrator startup
- `internal/cli/resume.go` - Session restoration logic
- `internal/cli/watch.go` - TUI attachment

### 2. Orchestrator Layer (`internal/orchestrator/`)

**Responsibility**: Session lifecycle management, turn-based execution loop, pause/resume.

**Core Concepts**:
- **Session**: Encapsulates entire collaboration (agents, workspace, state)
- **Turn**: Single agent execution unit with timeout and metrics
- **Completion**: Deliverable comparison and session termination

**Execution Flow**:
1. Load/create session from template
2. Initialize workspace with scaffolded files
3. Start MCP server and spawn agents
4. Execute turns alternating between agents (agent1 → agent2 → agent1...)
5. Save state after each turn
6. Check for completion (matching deliverables)
7. Handle pause (SIGTERM/SIGINT) or completion

**Key Files**:
- `orchestrator/session.go` - Session struct, NewSession, SaveState/LoadState
- `orchestrator/turn.go` - ExecuteTurn with timeout and metrics
- `orchestrator/orchestrator.go` - Main Run loop with pause/resume
- `orchestrator/completion.go` - Deliverable comparison

### 3. Agent Manager (`internal/agent/`)

**Responsibility**: Agent lifecycle, health monitoring, metrics collection.

**Technology**: [claude-agent-sdk-go](https://github.com/connerohnesorge/claude-agent-sdk-go) wrapping Claude CLI subprocess.

**Features**:
- Spawn agent processes with MCP connection
- Execute turns with timeout enforcement
- Health checks every 5 seconds
- Token counting and cost calculation
- Graceful shutdown (SIGTERM → 5s → SIGKILL)

**Key Files**:
- `agent/agent.go` - Agent struct wrapping SDK client
- `agent/manager.go` - SpawnAgent, StartTurn, health checks
- `agent/metrics.go` - Token tracking, cost calculation

### 4. MCP Server (`internal/mcp/`)

**Responsibility**: Provide collaboration tools to agents via Model Context Protocol.

**Technology**: STDIO transport with single shared server for both agents.

**Tools Provided** (7 total):
1. `send_message` - Agent sends message to other agent
2. `read_messages` - Agent reads all messages
3. `write_shared_context` - Update shared context
4. `read_shared_context` - Read shared context
5. `update_memory` - Update agent-specific memory
6. `read_memory` - Read agent-specific memory
7. `submit_deliverable` - Submit final deliverable (triggers completion check)

**Implementation**:
- All tool calls emit `ToolInvoked` events for observability
- File operations delegate to storage layer
- submit_deliverable compares both agents' deliverables byte-for-byte
- Error responses include helpful messages per contract specs

**Key Files**:
- `mcp/server.go` - StartMCPServer with STDIO transport
- `mcp/tools.go` - Tool registration with schemas
- `mcp/handlers.go` - Tool handler implementations

### 5. Storage Layer (`internal/storage/`)

**Responsibility**: File-based persistence with atomic writes and file watching.

**Guarantees**:
- **Atomic writes**: Temp file + rename prevents corruption
- **File permissions**: Directories 0700, files 0600
- **Zero data loss**: All writes are atomic, all pauses save state

**Components**:
- `atomic.go` - Atomic write pattern (write temp → fsync → rename)
- `workspace.go` - Workspace creation, cleanup, scaffolding
- `session_state.go` - Session state JSON serialization
- `messages.go` - Message file operations with YAML frontmatter
- `context.go` - Shared context operations
- `memory.go` - Per-agent memory with isolation
- `deliverable.go` - Deliverable with metadata
- `watcher.go` - fsnotify integration (<10ms latency)

**Workspace Structure**:
```
~/.local/share/shiki-cli/sessions/<session-id>/
├── session_state.json       # Persistent state
├── orchestrator.log          # Structured event log (JSON)
├── shared_context.md         # Shared workspace context
├── messages/
│   ├── 001-agent1.md
│   ├── 002-agent2.md
│   └── ...
├── memory/
│   ├── agent1.md
│   └── agent2.md
├── deliverables/
│   ├── agent1.md
│   └── agent2.md
└── <custom files from template>
```

### 6. EventBus (`internal/events/`)

**Responsibility**: Channel-based pub/sub for real-time updates.

**Design**: Non-blocking publish with select/default prevents slow subscribers from blocking.

**Event Types** (14 total):
- `SessionCreated`, `SessionStarted`, `SessionPaused`, `SessionResumed`, `SessionCompleted`
- `TurnStarted`, `TurnCompleted`, `TurnFailed`
- `MessageSent`, `FileUpdated`
- `ToolInvoked`
- `AgentStarted`, `AgentStopped`, `AgentHealthy`, `AgentUnhealthy`

**Performance**:
- Buffer size: 100 events per subscriber
- Target throughput: 10,000 events/sec
- Graceful shutdown: Drain all events within 1s

**Key Files**:
- `events/types.go` - Event type definitions
- `events/bus.go` - EventBus with Subscribe, Publish, Shutdown
- `events/subscriber.go` - Subscription management

### 7. Terminal UI (`internal/tui/`)

**Responsibility**: Real-time visualization of agent collaboration.

**Technology**: [Bubbletea](https://github.com/charmbracelet/bubbletea) (Elm architecture) + [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling.

**Layout** (80x24 target):
```
┌─────────────────────────────────────────────────────────┐
│ Session: abc123 │ Task │ 00:05:32 │ $0.42              │  Header
├──────────────────────────┬──────────────────────────────┤
│                          │                              │
│  Turn History            │  File Viewer                 │  Split
│  #1 agent1  5s  $0.12    │  shared_context.md           │  Panes
│  #2 agent2  3s  $0.08    │                              │
│  #3 agent1  4s  $0.10    │  Current file content...     │
│                          │                              │
├──────────────────────────┴──────────────────────────────┤
│ ⏵ Running │ 5 files │ q=quit Tab=files ↑↓=nav        │  Status
└─────────────────────────────────────────────────────────┘
```

**Features**:
- <100ms event updates via EventBus subscription
- Keyboard navigation (↑↓ turns, Tab files, PgUp/PgDn scroll)
- File viewer with markdown support
- Live cost tracking
- Ctrl+C to pause session

**Key Files**:
- `tui/model.go` - Bubbletea Model with session state
- `tui/update.go` - Update function handling events and keyboard
- `tui/view.go` - View rendering with Lipgloss
- `tui/components/` - Reusable components (header, turn_list, file_viewer, status_bar, bug_preview)
- `tui/styles.go` - Lipgloss style definitions

#### Bug Workflow TUI

**Responsibility**: Interactive UI for systematic bug fixing with AI assistance.

**Architecture**: Built on the same Bubbletea foundation as the collaboration TUI but optimized for single-agent bug fixing workflow.

**Key Components**:

- `BugModel` (`tui/bug_model.go`) - Main model for bug fixing workflow
- `BugPreview` (`tui/components/bug_preview.go`) - Preview component for bug artifacts

**Bug Workflow Phases**:
1. **Plan Phase**: AI analyzes the bug and creates a fix plan (`bug-plan.md`)
2. **Tasks Phase**: AI breaks down the plan into actionable tasks (`bug-tasks.md`)
3. **Implement Phase**: AI executes tasks and tracks progress (`task-progress.md`)

**Preview Pane Features**:
- Real-time file watching for task progress updates
- Phase-specific content display (plan → tasks → implementation progress)
- Graceful error handling for missing or inaccessible files
- YAML frontmatter stripping for clean markdown display

**File Watching**:
- Monitors `task-progress.md` during implement phase
- Exponential backoff retry mechanism for file access
- Handles file permission and access errors gracefully

**Troubleshooting Preview Pane Issues**:

*Problem: Preview pane shows "Permission denied" error*
- **Cause**: Insufficient file permissions for bug directory or files
- **Solution**: Check and fix file permissions: `chmod 644 /path/to/bug-files/*.md`

*Problem: Preview pane is empty during implement phase*
- **Cause**: `task-progress.md` file has not been created yet by the AI agent
- **Solution**: This is normal behavior; file will appear when implementation begins

*Problem: Preview pane not updating in real-time*
- **Cause**: File watching system may have encountered an error
- **Solution**: Try switching to a different phase and back, or restart the bug workflow

*Problem: Preview shows "file is empty" message*
- **Cause**: Markdown file exists but has no content
- **Solution**: Wait for AI agent to write content, or check if file was corrupted

### 8. Configuration (`internal/config/`)

**Responsibility**: XDG-compliant configuration with priority hierarchy.

**Priority**: CLI flags > env vars > config file > defaults

**Configuration File**: `~/.config/shiki-cli/config.json`

**Fields**:
- `workspace_dir` - Session workspace directory
- `log_level` - Logging verbosity (debug, info, warn, error)
- `default_timeout` - Turn timeout (seconds)
- `anthropic_api_key` - Anthropic API key (fallback)

**Key Files**:
- `config/config.go` - Config struct and defaults
- `config/loader.go` - Load from file/env/flags
- `config/xdg.go` - XDG directory resolution

### 9. Template System (`internal/template/`)

**Responsibility**: Parse and validate task templates with YAML frontmatter.

**Template Format**:
```markdown
---
agent1_name: "Architect"
agent1_role: "system designer"
agent1_system_prompt: "You design system architectures..."
agent2_name: "Reviewer"
agent2_role: "code reviewer"
agent2_system_prompt: "You review designs..."
max_turns: 10
workspace_structure:
  - path: "design.md"
    content: "# System Design\n\n"
  - path: "notes/"
    type: "directory"
---
# Task Description

Design a scalable API for...
```

**Validation**:
- Required fields: agent names, roles, system prompts, max_turns
- max_turns > 0
- Valid YAML syntax with helpful error messages

**Key Files**:
- `template/template.go` - TaskTemplate struct
- `template/parser.go` - YAML frontmatter parsing
- `template/validator.go` - Template validation
- `template/scaffold.go` - Workspace scaffolding

### 10. Logging (`internal/logging/`)

**Responsibility**: Structured JSON logging with EventBus integration.

**Format**: JSON Lines (JSONL)

**Levels**: DEBUG, INFO, WARN, ERROR

**Event Subscriber**: All events logged to `orchestrator.log`

**Key Files**:
- `logging/logger.go` - Structured JSON logger
- `logging/events.go` - EventBus subscriber for logging

## Data Flow

### Session Creation Flow

```
User: shiki run task.md --watch
  │
  ├─> CLI parses command, loads task.md
  │
  ├─> Template parser extracts frontmatter + markdown
  │
  ├─> Validator checks required fields
  │
  ├─> Orchestrator creates session
  │     │
  │     ├─> Generate session ID
  │     ├─> Create workspace directory
  │     ├─> Scaffold files from template.workspace_structure
  │     ├─> Save initial session_state.json
  │     └─> Emit SessionCreated event
  │
  ├─> MCP server starts with STDIO transport
  │
  ├─> Agent manager spawns agent1 and agent2
  │     └─> Connect agents to MCP server
  │
  ├─> EventBus starts (if --watch)
  │
  ├─> TUI subscribes to events (if --watch)
  │
  └─> Orchestrator.Run() begins turn loop
```

### Turn Execution Flow

```
Orchestrator.Run()
  │
  ├─> for turn := 1; turn <= max_turns; turn++
  │     │
  │     ├─> Determine active agent (alternating)
  │     │
  │     ├─> Emit TurnStarted event
  │     │
  │     ├─> AgentManager.StartTurn(agent, query, timeout)
  │     │     │
  │     │     ├─> SDK sends query to agent
  │     │     │
  │     │     ├─> Agent executes, invokes MCP tools
  │     │     │     │
  │     │     │     ├─> Tool: send_message
  │     │     │     │   ├─> Storage.WriteMessage()
  │     │     │     │   ├─> File watcher emits FileUpdated
  │     │     │     │   └─> Emit ToolInvoked event
  │     │     │     │
  │     │     │     ├─> Tool: write_shared_context
  │     │     │     │   ├─> Storage.WriteSharedContext()
  │     │     │     │   ├─> File watcher emits FileUpdated
  │     │     │     │   └─> Emit ToolInvoked event
  │     │     │     │
  │     │     │     └─> Tool: submit_deliverable
  │     │     │         ├─> Storage.WriteDeliverable()
  │     │     │         └─> Check if both agents submitted
  │     │     │             └─> If match: Emit SessionCompleted
  │     │     │
  │     │     ├─> Collect metrics (tokens, cost, duration)
  │     │     │
  │     │     └─> Return turn result
  │     │
  │     ├─> Emit TurnCompleted event
  │     │
  │     ├─> SaveSessionState() (atomic write)
  │     │
  │     └─> Check for completion or pause signal
  │
  └─> Session ends: cleanup agents, MCP, EventBus
```

### Event Flow

```
Component emits event
  │
  ├─> EventBus.Publish(event)
  │     │
  │     └─> for each subscriber:
  │           select {
  │             case ch <- event:  // send if buffer available
  │             default:           // drop if buffer full
  │           }
  │
  ├─> TUI subscriber receives event
  │     └─> Update model state
  │         └─> Trigger view re-render (<100ms)
  │
  └─> Logger subscriber receives event
        └─> Write JSON line to orchestrator.log
```

### Pause/Resume Flow

**Pause** (user sends SIGTERM or presses Ctrl+C):
```
Signal received
  │
  ├─> Orchestrator catches signal
  │
  ├─> Set paused flag
  │
  ├─> Current turn completes
  │
  ├─> SaveSessionState() with status=paused
  │
  ├─> Emit SessionPaused event
  │
  ├─> Shutdown agents (SIGTERM → 5s → SIGKILL)
  │
  ├─> Shutdown MCP server
  │
  ├─> Shutdown EventBus (drain events)
  │
  └─> Exit with code 130 (user interrupt)
```

**Resume**:
```
User: shiki resume <session-id>
  │
  ├─> CLI loads session_state.json
  │
  ├─> Validate status == "paused"
  │
  ├─> Orchestrator restores session
  │     │
  │     ├─> Load turn history
  │     ├─> Load metrics
  │     ├─> Restore agents
  │     └─> Restore MCP server
  │
  ├─> Emit SessionResumed event
  │
  └─> Continue turn loop from next turn
```

## Key Design Decisions

### Decision 1: Single Shared MCP Server

**Context**: Need atomic file operations accessible to both agents.

**Options**:
- One MCP server per agent
- Single shared MCP server

**Choice**: Single shared MCP server.

**Rationale**:
- Simplifies atomic writes (no coordination needed)
- Single source of truth for file state
- STDIO transport supports multiple clients

**Risk Mitigation**: Test multi-client early; pivot to per-agent servers if STDIO doesn't support it.

### Decision 2: Non-Blocking EventBus

**Context**: Need real-time TUI updates without blocking orchestrator.

**Options**:
- Blocking publish (wait for all subscribers)
- Non-blocking publish with select/default

**Choice**: Non-blocking publish.

**Rationale**:
- Slow TUI rendering doesn't block turn execution
- Dropped events logged (observable)
- Idiomatic Go pattern

**Trade-off**: TUI may miss events if rendering too slow (mitigated by buffered channels).

### Decision 3: Turn-Based Execution (No Concurrency)

**Context**: Agents could execute concurrently or sequentially.

**Choice**: Sequential turn-based execution.

**Rationale**:
- Simplifies file locking (no concurrent writes)
- Easier debugging (deterministic order)
- Matches collaboration model (agents respond to each other)

**Trade-off**: Slower than parallel execution (acceptable for collaboration use case).

### Decision 4: Template-Driven Scaffolding

**Context**: Workspaces need initial file structure.

**Choice**: Templates define `workspace_structure` field with initial files/directories.

**Rationale**:
- Flexible (supports any directory structure)
- Declarative (no code in templates)
- Pre-populates workspace with starting content

**Example**:
```yaml
workspace_structure:
  - path: "design.md"
    content: "# Design\n\n"
  - path: "notes/"
    type: "directory"
```

### Decision 5: External SDK Integration

**Context**: Need to manage Claude CLI agent processes.

**Options**:
- Implement subprocess management from scratch
- Use existing SDK (claude-agent-sdk-go)

**Choice**: Use claude-agent-sdk-go.

**Rationale**:
- Battle-tested subprocess management
- Handles STDIO communication
- Reduces custom code

**Risk Mitigation**: Vendor dependencies (`go mod vendor`), prepare to fork if SDK unstable.

## Performance Targets

| Metric | Target | Component |
|--------|--------|-----------|
| Session startup | <10s | Orchestrator + Agent Manager |
| TUI update latency | <100ms | EventBus + TUI |
| File write detection | <10ms | File Watcher (fsnotify) |
| EventBus throughput | >10,000 events/s | EventBus |
| Binary size | <50MB | Build flags |
| Turn timeout | 60s default | Configurable |
| Graceful shutdown | <5s | All components |

## Security Considerations

1. **API Key Handling**: Never log API keys, support env vars and config file
2. **File Permissions**: Session directories 0700, files 0600 (user-only)
3. **Process Isolation**: Agent processes run as user, no elevated privileges
4. **Input Validation**: Validate all template fields before execution
5. **Path Traversal**: Prevent `../` in template workspace_structure paths

## Testing Strategy

- **Unit Tests**: All packages have unit tests (>80% coverage target)
- **Integration Tests**: Cross-component tests (CLI commands, MCP tools, TUI)
- **Snapshot Tests**: TUI view rendering
- **Benchmarks**: EventBus throughput, file watcher latency
- **Manual Testing**: Real Claude agents, various templates, pause/resume

## Future Enhancements

- Multi-agent support (>2 agents)
- Streaming turn execution (partial results)
- Web UI (alternative to TUI)
- Remote agent execution (cloud agents)
- Template marketplace (share templates)
- Session replay (visualize completed sessions)
- Cost budgets (halt when cost limit reached)
