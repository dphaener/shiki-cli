---
description: "Complete CLI tool for orchestrating multi-agent AI collaboration"
---

# Feature Specification: Multi-Agent Collaboration CLI

**Feature Branch**: `001-multi-agent-collaboration`
**Created**: 2025-11-23
**Status**: Draft

## Overview

A production-ready command-line tool that enables two AI agents to collaborate on complex tasks through file-based communication, turn-based orchestration, and real-time terminal UI monitoring. The tool migrates a Ruby MVP to Go, providing a single-binary deployment with rich terminal user experience, session management, and comprehensive observability.

**Why This Matters**: Development teams and AI researchers need a reliable way to orchestrate multi-agent collaboration for complex tasks like architecture design, code review, and technical analysis. This tool transforms agent collaboration from a prototype into a production-ready workflow with pause/resume capabilities, real-time monitoring, and human-readable artifacts.

## User Scenarios & Testing

### User Story 1 - Start Agent Collaboration on Task (Priority: P1)

A developer wants to collaborate two AI agents on a design task, monitor their progress in real-time, and receive a final deliverable when both agents agree on the solution.

**Why this priority**: Core functionality - without this, the tool provides no value.

**Independent Test**: Run a simple task template with two agents and verify completion with deliverable.

**Acceptance Scenarios**:

1. **Given** a valid task template file, **When** user runs `collab run task.md --watch`, **Then** the system creates a session, spawns both agents, launches an interactive TUI showing real-time progress, and saves a deliverable when both agents approve
2. **Given** a task is running, **When** user views the TUI, **Then** they see turn history, current file contents, token usage, cost tracking, and collaboration status
3. **Given** agents are collaborating, **When** either agent writes a message or updates shared context, **Then** the TUI updates in real-time showing the new content

---

### User Story 2 - Run Headless Collaboration (Priority: P1)

A developer wants to run agent collaboration in the background without TUI for CI/CD pipelines or long-running tasks.

**Why this priority**: Essential for automation and batch processing scenarios.

**Independent Test**: Run collaboration without --watch flag and verify completion via exit code and output files.

**Acceptance Scenarios**:

1. **Given** a task template, **When** user runs `collab run task.md` (no --watch flag), **Then** the system executes in headless mode, prints turn-by-turn progress to stdout, and exits with code 0 on success
2. **Given** a headless session is running, **When** collaboration completes, **Then** the final deliverable path is printed to stdout
3. **Given** a headless session encounters an error, **When** the error occurs, **Then** the system exits with non-zero code and prints detailed error information

---

### User Story 3 - Pause and Resume Collaboration (Priority: P1)

A user needs to pause a running collaboration (e.g., due to cost limits or time constraints) and resume it later from the exact same state.

**Why this priority**: Critical for long-running tasks and cost management.

**Independent Test**: Start a task, pause mid-execution, resume, and verify continuation from correct turn.

**Acceptance Scenarios**:

1. **Given** a collaboration is running, **When** user presses Ctrl+C or sends SIGTERM, **Then** the system saves current state to session_state.json, gracefully shuts down agents, and preserves all files
2. **Given** a paused session exists, **When** user runs `collab resume <session-id>`, **Then** the system restores state, respawns agents, and continues from the next turn
3. **Given** a paused session, **When** user resumes with --watch, **Then** the TUI displays full turn history from before the pause plus new turns

---

### User Story 4 - Inspect Session Artifacts (Priority: P2)

A user wants to examine messages, shared context, agent memory, and logs from a completed or paused collaboration session.

**Why this priority**: Important for debugging and understanding agent interactions, but not critical for basic operation.

**Independent Test**: Complete a session and use show commands to verify all artifacts are accessible.

**Acceptance Scenarios**:

1. **Given** a completed session, **When** user runs `collab show <session-id>`, **Then** they see a summary with status, duration, turn count, token usage, cost, and artifact locations
2. **Given** a session exists, **When** user runs `collab show <session-id> --messages`, **Then** all messages between agents are displayed in chronological order with metadata
3. **Given** a completed session, **When** user runs `collab show <session-id> --deliverable`, **Then** the final approved deliverable content is displayed

---

### User Story 5 - Manage Sessions (Priority: P2)

A user wants to list all sessions, filter by status, and clean up old completed sessions.

**Why this priority**: Necessary for workspace hygiene but not blocking for core workflows.

**Independent Test**: Create multiple sessions with different statuses and verify list/filter/cleanup operations.

**Acceptance Scenarios**:

1. **Given** multiple sessions exist, **When** user runs `collab list sessions`, **Then** they see a table of all sessions with ID, status, turns, duration, and start time
2. **Given** sessions with various statuses, **When** user runs `collab list sessions --status running`, **Then** only running sessions are displayed
3. **Given** old completed sessions exist, **When** user runs `collab clean --older-than 7d`, **Then** sessions older than 7 days are removed after confirmation

---

### User Story 6 - Validate Task Templates (Priority: P2)

A user wants to validate their task template before running to catch configuration errors early.

**Why this priority**: Improves user experience by catching errors before expensive agent runs.

**Independent Test**: Create valid and invalid templates, run validation, verify appropriate success/error messages.

**Acceptance Scenarios**:

1. **Given** a valid task template, **When** user runs `collab validate task.md`, **Then** validation passes with checkmarks for all criteria
2. **Given** a template with missing required fields, **When** user runs `collab validate task.md`, **Then** validation fails with specific error messages for each missing field
3. **Given** a template with invalid YAML, **When** user runs `collab validate task.md`, **Then** validation fails with line number and syntax error details

---

### User Story 7 - Create Task Templates (Priority: P3)

A user wants to quickly create a new task template from an interactive wizard or copy an existing template.

**Why this priority**: Nice to have for convenience, but users can manually create templates.

**Independent Test**: Use init command and verify generated template is valid and usable.

**Acceptance Scenarios**:

1. **Given** user wants a new template, **When** they run `collab init template my-task`, **Then** an interactive wizard prompts for agent names, roles, and generates a valid template file
2. **Given** a template exists, **When** user edits and saves it, **Then** the template passes validation
3. **Given** user runs init, **When** they complete the wizard, **Then** the template location is printed and ready to use

---

### User Story 8 - Watch Running Session (Priority: P3)

A user wants to attach a TUI to an already-running headless collaboration session to monitor its progress.

**Why this priority**: Useful but not essential - users can start with --watch or inspect artifacts.

**Independent Test**: Start headless session, attach TUI, verify real-time updates appear.

**Acceptance Scenarios**:

1. **Given** a session is running headless, **When** user runs `collab watch <session-id>`, **Then** a TUI launches showing all past turns and updates in real-time for new turns
2. **Given** a TUI is attached, **When** user presses 'q', **Then** the TUI exits but the session continues running
3. **Given** a completed session, **When** user runs `collab watch <session-id> --replay`, **Then** the TUI displays the full turn history in replay mode

---

### Edge Cases

- **What happens when an agent process crashes mid-turn?** The orchestrator detects the crash, emits an ErrorOccurred event, saves session state, and exits with an error. The session can be resumed with retry logic.
- **How does the system handle agent API key missing or invalid?** Validation occurs during agent spawn; the system fails fast with a clear error message directing the user to set ANTHROPIC_API_KEY.
- **What happens when both agents never approve a deliverable and max turns is reached?** The session ends with status "incomplete", saves all artifacts, and reports the maximum turn limit reached.
- **What happens when the workspace directory is not writable?** The system fails during initialization with a clear error message about permissions.
- **How does the system handle concurrent runs in the same workspace?** Each session creates a unique timestamped directory to prevent conflicts.
- **What happens when a turn exceeds the timeout limit?** The turn is aborted, an ErrorOccurred event is emitted, and the session can be resumed with extended timeout.
- **How does the system handle malformed task templates?** The template parser validates YAML frontmatter and required fields; if invalid, the run command fails before spawning agents.
- **What happens when file watcher detects file changes outside agent actions?** FileUpdated events are emitted for all changes; the TUI updates regardless of source.
- **What happens when a user tries to resume a completed session?** The system detects the "completed" status and rejects the resume request with an error message.
- **What happens when disk space is exhausted during a session?** File write operations fail with disk space errors; the session enters error state and saves what it can.

## Requirements

### Functional Requirements

#### CLI Commands

- **FR-001**: System MUST provide `collab run [task-file]` command to start new collaboration sessions with agent orchestration
- **FR-002**: System MUST provide `--watch` flag to launch interactive TUI during collaboration
- **FR-003**: System MUST provide `collab resume [session-id]` command to continue paused sessions from saved state
- **FR-004**: System MUST provide `collab list sessions` command to display all sessions with filtering options
- **FR-005**: System MUST provide `collab show [session-id]` command to inspect session artifacts (messages, context, memory, deliverable, logs)
- **FR-006**: System MUST provide `collab validate [task-file]` command to validate template syntax and required fields
- **FR-007**: System MUST provide `collab init template [name]` command to create new task templates
- **FR-008**: System MUST provide `collab clean` command to remove old sessions with safety prompts
- **FR-009**: System MUST provide `collab watch [session-id]` command to attach TUI to running sessions
- **FR-010**: System MUST support global flags: --verbose, --no-color, --config, --workspace

#### Orchestration

- **FR-011**: System MUST execute agent turns sequentially (agent_1 → agent_2 → agent_1...)
- **FR-012**: System MUST emit events for all state changes (TurnStarted, TurnCompleted, MessageSent, etc.)
- **FR-013**: System MUST track turn count, duration, token usage, and cost per agent
- **FR-014**: System MUST detect completion when both agents invoke submit_deliverable tool
- **FR-015**: System MUST save session state to session_state.json on pause/shutdown
- **FR-016**: System MUST restore full session state from session_state.json on resume
- **FR-017**: System MUST enforce max_turns limit from task template
- **FR-018**: System MUST enforce turn timeout with configurable duration
- **FR-019**: System MUST handle agent process crashes with recoverable error reporting

#### Storage & Workspace

- **FR-020**: System MUST create unique session directories with timestamp-based IDs
- **FR-021**: System MUST use XDG directory specifications for config, data, and cache
- **FR-022**: System MUST store all messages as Markdown files with YAML frontmatter
- **FR-023**: System MUST maintain shared_context.md for collaborative knowledge
- **FR-024**: System MUST maintain per-agent memory.md for private state
- **FR-025**: System MUST save final deliverable.md with approval metadata
- **FR-026**: System MUST copy original task template to session directory
- **FR-027**: System MUST write structured logs to orchestrator.log in JSON format
- **FR-028**: System MUST perform atomic file writes (temp file + rename)
- **FR-029**: System MUST watch workspace files for changes with fsnotify

#### MCP Tool Server

- **FR-030**: System MUST start MCP tool server with STDIO transport on session start
- **FR-031**: System MUST register collaboration tools: send_message, read_messages, write_shared_context, read_shared_context, update_memory, read_memory, submit_deliverable
- **FR-032**: System MUST expose MCP server configuration to agent processes via environment variables
- **FR-033**: System MUST delegate tool calls to storage layer for file operations
- **FR-034**: System MUST emit ToolInvoked events for all tool calls
- **FR-035**: System MUST gracefully shutdown MCP server on session end

#### Agent Management

- **FR-036**: System MUST spawn agent processes using claude-agent-sdk-go
- **FR-037**: System MUST configure agents with role, workspace directory, and environment variables
- **FR-038**: System MUST pass ANTHROPIC_API_KEY to agent processes securely
- **FR-039**: System MUST monitor agent process health during turns
- **FR-040**: System MUST terminate agent processes gracefully on shutdown
- **FR-041**: System MUST collect agent metrics (tokens, cost, tool count) after each turn

#### TUI (Terminal User Interface)

- **FR-042**: System MUST render split-pane layout: turn history (left) and file view (right)
- **FR-043**: System MUST display session header with ID, task name, duration, and total cost
- **FR-044**: System MUST show turn history with turn number, agent, duration, tool count, and token usage
- **FR-045**: System MUST display current file content in right pane with syntax highlighting
- **FR-046**: System MUST support file navigation (messages, context, memory, deliverable)
- **FR-047**: System MUST show status bar with current agent activity and file counts
- **FR-048**: System MUST update in real-time when FileUpdated events occur
- **FR-049**: System MUST support keyboard navigation: arrow keys, Tab, PgUp/PgDown, Home/End
- **FR-050**: System MUST provide help overlay with key bindings
- **FR-051**: System MUST support graceful exit (q or Ctrl+C) without stopping session

#### Configuration

- **FR-052**: System MUST load configuration from ~/.config/collab-cli/config.json
- **FR-053**: System MUST support environment variable overrides (COLLAB_CLI_*)
- **FR-054**: System MUST prioritize: CLI flags > env vars > config file > defaults
- **FR-055**: System MUST validate configuration and fail fast with clear errors

#### Task Templates

- **FR-056**: System MUST parse YAML frontmatter with agent configuration
- **FR-057**: System MUST validate required fields: agent1_name, agent1_role, agent1_system_prompt, agent2_name, agent2_role, agent2_system_prompt, max_turns
- **FR-058**: System MUST support optional fields: initial messages, turn_delay, timeout, workspace_structure, cost_limits, completion_criteria
- **FR-059**: System MUST parse markdown task body and make available to agents

### Key Entities

- **Session**: Represents a single collaboration run with unique ID, workspace, state, and metadata
- **Agent**: An AI agent process with ID, role, system prompt, workspace, and configuration
- **Turn**: A single execution cycle for one agent with start time, end time, metrics, and status
- **Message**: Communication between agents stored as Markdown file with YAML frontmatter
- **Workspace**: File-based session directory containing all collaboration artifacts
- **Task Template**: YAML + Markdown file defining agent configuration and task description
- **Event**: State change notification distributed via EventBus to subscribers
- **Tool**: MCP tool implementation providing file operations to agents
- **SessionState**: Serializable snapshot of session for pause/resume functionality

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users can start a collaboration and see real-time progress in under 10 seconds from command invocation
- **SC-002**: System supports pausing and resuming sessions with zero data loss and exact state restoration
- **SC-003**: All collaboration artifacts (messages, context, memory, deliverable) are human-readable Markdown files
- **SC-004**: TUI updates reflect file changes within 100ms of file system modification
- **SC-005**: System handles agent process crashes gracefully, saving state and allowing resume with max 3 retry attempts
- **SC-006**: Command-line operations complete with appropriate exit codes (0 for success, non-zero for errors)
- **SC-007**: Users can inspect any past session's artifacts without re-running the collaboration
- **SC-008**: System deploys as a single binary with no runtime dependencies beyond the OS
- **SC-009**: Template validation catches 100% of required field violations before agent spawning
- **SC-010**: Cost tracking reports accurate per-agent and total token usage and cost within 1% margin

## Assumptions

- **AS-001**: Users have Go 1.21+ available for building from source (or will use pre-built binaries)
- **AS-002**: Users have valid Anthropic API key with access to Claude models
- **AS-003**: File system supports atomic rename operations (POSIX compliance)
- **AS-004**: Terminal supports ANSI colors and Unicode characters for TUI rendering
- **AS-005**: Users understand basic CLI patterns and can read Markdown files
- **AS-006**: Agent SDK (claude-agent-sdk-go) is available and properly configured
- **AS-007**: Network connectivity exists for API calls to Anthropic during agent execution
- **AS-008**: Workspace directory has sufficient disk space for session artifacts (typically < 10MB per session)
- **AS-009**: Users operate on systems supporting XDG directory specifications or equivalents (Linux, macOS, Windows)

## Dependencies

- **DEP-001**: Go 1.21+ standard library for core functionality
- **DEP-002**: github.com/spf13/cobra for CLI command structure and flag parsing
- **DEP-003**: Charmbracelet Bubbletea for TUI framework (tea.Model pattern)
- **DEP-004**: Charmbracelet Lipgloss for TUI styling and layout
- **DEP-005**: Charmbracelet Bubbles for reusable TUI components
- **DEP-006**: fsnotify for file system watching
- **DEP-007**: claude-agent-sdk-go for agent process management and MCP integration
- **DEP-008**: YAML parser (gopkg.in/yaml.v3 or equivalent) for template frontmatter

## Out of Scope

- **OOS-001**: Support for more than two agents in a single session
- **OOS-002**: Web-based UI or HTTP API for remote monitoring
- **OOS-003**: Database storage (all data is file-based)
- **OOS-004**: Agent-to-agent direct communication (all communication is file-based)
- **OOS-005**: Built-in agent models or prompts (users provide via templates)
- **OOS-006**: Distributed execution across multiple machines
- **OOS-007**: Real-time collaboration between multiple human users
- **OOS-008**: Integration with version control beyond local workspace
- **OOS-009**: Custom plugin system or extensions beyond MCP tools
- **OOS-010**: Replay with step-through debugging (only replay mode display)
- **OOS-011**: Network-based MCP transport (only STDIO supported initially)
- **OOS-012**: Windows support in initial release (focus on Linux/macOS)

## Constraints

- **CON-001**: Must maintain backward compatibility with Ruby MVP task template format
- **CON-002**: Must use file-based storage (no database) for artifact persistence
- **CON-003**: Must execute agent turns sequentially (no parallel agent execution)
- **CON-004**: Must fit in single terminal window without scrolling for TUI layout
- **CON-005**: Binary size should remain under 50MB for single-binary distribution
- **CON-006**: TUI refresh rate limited to 10 FPS maximum to avoid CPU overhead
- **CON-007**: Session state must be serializable to JSON for pause/resume
- **CON-008**: Must not require sudo or elevated permissions for normal operation
- **CON-009**: Configuration files must be human-editable (JSON/YAML only)

## Security & Privacy

- **SEC-001**: API keys must be passed via environment variables only (never stored in session files)
- **SEC-002**: Session directories must use restrictive permissions (0700) to prevent unauthorized access
- **SEC-003**: Log files must not contain API keys or sensitive credentials
- **SEC-004**: Agent process environment must be isolated with minimal required variables
- **SEC-005**: File paths must be validated to prevent directory traversal attacks
- **SEC-006**: Template parsing must be safe against YAML injection attacks
- **SEC-007**: MCP tool calls must validate arguments before file system operations
- **SEC-008**: Session cleanup must securely delete files containing potentially sensitive collaboration data

## Open Questions

- Should we support custom MCP tool registration beyond the built-in collaboration tools?
- Should session replay mode support variable playback speed (e.g., 2x, 0.5x)?
- Should we provide shell completion for session IDs in bash/zsh?
- Should cost limits be enforced automatically (abort session) or just warn the user?
- Should deliverable format support other formats beyond Markdown (PDF export)?
- Should we support session export/import for sharing collaborations between users?
