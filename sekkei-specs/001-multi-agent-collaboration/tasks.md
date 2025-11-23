---
description: "Work package task list for multi-agent collaboration CLI"
---
*Path: [sekkei-specs/001-multi-agent-collaboration/tasks.md]*

# Work Packages: Multi-Agent Collaboration CLI

**Feature**: Production-ready Go CLI tool for orchestrating two-agent AI collaboration via file-based communication, turn-based execution, and real-time TUI monitoring.

**Inputs**: Design documents from `sekkei-specs/001-multi-agent-collaboration/`
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/ (MCP tools, CLI interface)

**Tests**: Testing is NOT explicitly required per spec. Unit and integration tests will be implemented for critical paths, but comprehensive test coverage is optional.

**Organization**: 69 fine-grained subtasks rolled into 10 work packages. Each work package aligns with an implementation phase and is independently deliverable with clear acceptance criteria.

**Prompt Files**: Each work package references a matching prompt file in `sekkei-specs/001-multi-agent-collaboration/tasks/planned/`

## Path Conventions
- All file paths use absolute paths from `/Users/darinhaener/code/collab/.worktrees/001-multi-agent-collaboration/`
- Work package prompts: `sekkei-specs/001-multi-agent-collaboration/tasks/planned/WPxx-slug.md`
- Core code: `cmd/collab/` (entry point), `internal/` (packages), `pkg/types/` (shared types)

---

## Work Package WP01: Project Setup & Core Types (Priority: P0)

**Goal**: Initialize Go project with dependencies, directory structure, core type definitions, and configuration system.

**Independent Test**: Build produces binary, core types compile, configuration loads from file/env/defaults with correct priority.

**Prompt**: `tasks/planned/WP01-project-setup.md`

### Included Subtasks
- [ ] T001: Initialize Go module and add all dependencies (cobra, bubbletea, lipgloss, bubbles, fsnotify, yaml.v3, claude-agent-sdk-go)
- [ ] T002: Create project directory structure (cmd/, internal/, pkg/, examples/, docs/, scripts/)
- [ ] T003: Define core types in pkg/types/ (Session, Agent, Turn, Message, Event, errors)
- [ ] T004: Implement configuration system (XDG directory resolution, config file loading, env var overrides, priority handling)
- [ ] T005: Create Makefile with targets (build, test, lint, install, clean)
- [ ] T006: Set up golangci-lint configuration

### Implementation Notes
1. Use `go mod init github.com/<user>/collab` (user can customize)
2. All core types must support JSON serialization for session_state.json
3. Configuration priority: CLI flags > env vars > config file > defaults (FR-054)
4. XDG directories: config in ~/.config/collab-cli/, data in ~/.local/share/collab-cli/

### Parallel Opportunities
- T001 (dependencies) and T002 (directory structure) can run in parallel
- T003 (types) and T004 (config) can run in parallel after T001/T002

### Dependencies
- None (foundation work package)

### Risks & Mitigations
- **Risk**: claude-agent-sdk-go may not be stable or compatible
- **Mitigation**: Vendor dependencies with `go mod vendor`, prepare to fork SDK if needed

---

## Work Package WP02: Storage & File Operations (Priority: P0)

**Goal**: Implement file-based persistence layer with atomic writes, workspace management, and file watching.

**Independent Test**: Write files atomically, scaffold workspace from template structure, detect file changes within 10ms, session state round-trips through JSON.

**Prompt**: `tasks/planned/WP02-storage-layer.md`

### Included Subtasks
- [ ] T007: Implement storage/atomic.go (atomic write pattern: temp file + rename)
- [ ] T008: Implement storage/workspace.go (CreateWorkspace, CleanupWorkspace, scaffolding from template structure)
- [ ] T009: Implement storage/session_state.go (SaveSessionState, LoadSessionState with JSON marshal/unmarshal)
- [ ] T010: Implement storage/messages.go (WriteMessage, ReadMessages with YAML frontmatter parsing)
- [ ] T011: Implement storage/context.go (WriteSharedContext, ReadSharedContext)
- [ ] T012: Implement storage/memory.go (WriteMemory, ReadMemory, per-agent isolation)
- [ ] T013: Implement storage/deliverable.go (WriteDeliverable with metadata, ReadDeliverable)
- [ ] T014: Implement storage/watcher.go (fsnotify integration, emit FileUpdated events)

### Implementation Notes
1. All write operations MUST use atomic write pattern (temp + rename) per FR-028
2. File permissions: session directories 0700, files 0600 (SEC-002)
3. File watcher target: <10ms from file write to event emission
4. Workspace structure supports nested directories and initial file content from templates

### Parallel Opportunities
- T007-T013 (all storage operations) can be developed in parallel since they're independent
- T014 (watcher) depends on T007 (atomic writes) but can parallel with others

### Dependencies
- WP01 complete (needs core types for Session, Message, etc.)

### Risks & Mitigations
- **Risk**: fsnotify may not meet <10ms latency on all platforms
- **Mitigation**: Benchmark early on Linux/macOS; implement polling fallback if needed

---

## Work Package WP03: EventBus & Logging (Priority: P0)

**Goal**: Implement channel-based event distribution system with structured logging subscriber.

**Independent Test**: Publish 10,000 events in <1s, slow subscribers don't block publisher, graceful shutdown drains all events, all events logged to JSON.

**Prompt**: `tasks/planned/WP03-eventbus-logging.md`

### Included Subtasks
- [X] T015: Define all event types in events/types.go (SessionCreated, TurnStarted, MessageSent, FileUpdated, etc. - 14 types total) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP03-eventbus-logging.md]
- [X] T016: Implement events/bus.go (NewEventBus, Subscribe, Publish with non-blocking select/default, Shutdown) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP03-eventbus-logging.md]
- [X] T017: Implement events/subscriber.go (subscription management, channel cleanup) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP03-eventbus-logging.md]
- [X] T018: Implement logging/logger.go (structured JSON logger, configurable levels) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP03-eventbus-logging.md]
- [X] T019: Implement logging/events.go (EventBus subscriber that logs all events to orchestrator.log) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP03-eventbus-logging.md]

### Implementation Notes
1. Non-blocking publish: `select { case ch <- event: default: }` prevents slow subscribers from blocking (Decision 2)
2. Event channel buffer size: 100 events (balance between memory and blocking)
3. Graceful shutdown: close publisher channel, drain all subscriber channels within 1s
4. Log format: JSON lines (JSONL) for structured parsing

### Parallel Opportunities
- T015 (types) can run first
- T016, T017 (EventBus implementation) can run together
- T018, T019 (logging) can run in parallel with T016/T017

### Dependencies
- WP01 complete (needs core types for event payloads)

### Risks & Mitigations
- **Risk**: Slow subscribers could cause dropped events with non-blocking publish
- **Mitigation**: Document expected behavior, provide buffered channels, log dropped event counts

---

## Work Package WP04: Template System (Priority: P0)

**Goal**: Parse and validate task templates with YAML frontmatter, markdown body, and workspace structure scaffolding.

**Independent Test**: Parse all example templates successfully, validator catches 100% of required field violations, scaffold complex workspace structures in <10ms.

**Prompt**: `tasks/planned/WP04-template-system.md`

### Included Subtasks
- [X] T020: Define TaskTemplate struct in template/template.go (matches data model spec) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP04-template-system.md]
- [X] T021: Implement template/parser.go (extract YAML frontmatter, parse markdown body, handle malformed YAML with line numbers) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP04-template-system.md]
- [X] T022: Implement template/validator.go (check required fields, validate types/ranges, return detailed errors) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP04-template-system.md]
- [X] T023: Implement template/scaffold.go (create directories and initial files from workspace_structure field) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP04-template-system.md]
- [X] T024: Create example templates (simple-agreement.md, code-review.md, architecture-design.md) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP04-template-system.md]

### Implementation Notes
1. YAML frontmatter delimiter: `---` at start and end (standard markdown convention)
2. Required fields: agent1_name, agent1_role, agent1_system_prompt, agent2_name, agent2_role, agent2_system_prompt, max_turns (FR-057)
3. Validation rules: max_turns > 0, agent names non-empty, YAML valid
4. Scaffold supports nested structures and file content templates

### Parallel Opportunities
- T020 (struct) must complete first
- T021 (parser), T022 (validator), T023 (scaffold) can run in parallel
- T024 (examples) can run last but in parallel with testing

### Dependencies
- WP01 complete (needs core types)
- WP02 complete (scaffold.go uses workspace creation)

### Risks & Mitigations
- **Risk**: YAML parser may not provide helpful error messages for syntax errors
- **Mitigation**: Wrap yaml.v3 errors with line number extraction and user-friendly messages

---

## Work Package WP05: MCP Tool Server (Priority: P1)

**Goal**: Implement MCP server with STDIO transport and 7 collaboration tools (send_message, read_messages, write_shared_context, read_shared_context, update_memory, read_memory, submit_deliverable).

**Independent Test**: Start MCP server, connect client via STDIO, invoke each tool successfully, verify file operations and event emissions, graceful shutdown in <100ms.

**Prompt**: `tasks/planned/WP05-mcp-server.md`

### Included Subtasks
- [X] T025: Implement mcp/server.go (StartMCPServer with STDIO transport, graceful shutdown on context cancellation) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP05-mcp-server.md]
- [X] T026: Implement mcp/tools.go (tool registration with MCP framework, schema definitions per contracts/mcp-tools.json) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP05-mcp-server.md]
- [X] T027: Implement mcp/handlers.go part 1 (handleSendMessage, handleReadMessages with storage delegation and event emission) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP05-mcp-server.md]
- [X] T028: Implement mcp/handlers.go part 2 (handleWriteSharedContext, handleReadSharedContext) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP05-mcp-server.md]
- [X] T029: Implement mcp/handlers.go part 3 (handleUpdateMemory, handleReadMemory with per-agent isolation) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP05-mcp-server.md]
- [X] T030: Implement mcp/handlers.go part 4 (handleSubmitDeliverable with content comparison logic, SessionCompleted event) ✅ [sekkei-specs/001-multi-agent-collaboration/tasks/done/WP05-mcp-server.md]

### Implementation Notes
1. Single shared MCP server for both agents (Decision 1)
2. STDIO transport configured via environment variables exposed to agent processes
3. All tool calls emit ToolInvoked events for observability (FR-034)
4. submit_deliverable compares content byte-for-byte, emits SessionCompleted when both match
5. Error responses include helpful messages per contract specifications

### Parallel Opportunities
- T025 (server setup) must complete first
- T026 (tool registration) depends on T025
- T027-T030 (individual tool handlers) can run in parallel after T026

### Dependencies
- WP01 (types), WP02 (storage), WP03 (EventBus) complete

### Risks & Mitigations
- **Risk**: MCP STDIO transport may not support multiple clients as expected
- **Mitigation**: Test multi-client early; pivot to per-agent servers if needed (Decision 1 mitigation)

---

## Work Package WP06: Agent Manager (Priority: P1)

**Goal**: Integrate claude-agent-sdk-go for agent lifecycle management, health monitoring, and metrics collection.

**Independent Test**: Spawn agents with mocked SDK client, execute turns with metrics collection, detect health check crashes within 1s, graceful shutdown with no orphaned processes.

**Prompt**: `tasks/planned/WP06-agent-manager.md`

### Included Subtasks
- [ ] T031: Implement agent/agent.go (Agent struct wrapping SDK client, Start/Stop/IsHealthy methods)
- [ ] T032: Implement agent/manager.go part 1 (SpawnAgent with SDK client creation, MCP connection setup)
- [ ] T033: Implement agent/manager.go part 2 (StartTurn to send query and await completion, timeout enforcement)
- [ ] T034: Implement agent/manager.go part 3 (health check goroutine to detect process crashes, emit events)
- [ ] T035: Implement agent/metrics.go (token counting, cost calculation per model pricing, per-agent and per-turn tracking)

### Implementation Notes
1. Agent SDK environment: ANTHROPIC_API_KEY (required), MCP server endpoint, workspace directory
2. Health checks run every 5 seconds, detect process crashes within 1s (SC-005)
3. Metrics accuracy target: within 1% of SDK-reported values (SC-010)
4. Turn timeout uses context.WithTimeout for clean cancellation (FR-018)
5. Agent processes terminated with SIGTERM, 5s grace period before SIGKILL

### Parallel Opportunities
- T031 (agent struct) must complete first
- T032-T034 (manager components) can run in parallel
- T035 (metrics) can run in parallel with T032-T034

### Dependencies
- WP01 (types), WP03 (EventBus), WP05 (MCP server) complete

### Risks & Mitigations
- **Risk**: Agent processes may become orphaned on crashes
- **Mitigation**: Track PIDs, implement cleanup on shutdown, test with `ps aux | grep claude`

---

## Work Package WP07: Orchestrator (Priority: P1)

**Goal**: Implement session lifecycle and turn-based execution loop with pause/resume, completion detection, and state persistence.

**Independent Test**: Execute session with alternating turns, persist state after each turn, detect completion with matching deliverables, pause/resume with exact state restoration, enforce max_turns limit.

**Prompt**: `tasks/planned/WP07-orchestrator.md`

### Included Subtasks
- [ ] T036: Implement orchestrator/session.go (Session struct, NewSession from template, SaveState/LoadState for pause/resume)
- [ ] T037: Implement orchestrator/turn.go (ExecuteTurn logic, timeout enforcement, metrics collection, event emission)
- [ ] T038: Implement orchestrator/orchestrator.go part 1 (Run method with turn loop, agent alternation logic)
- [ ] T039: Implement orchestrator/orchestrator.go part 2 (pause handling with SIGTERM/SIGINT, save state and graceful exit)
- [ ] T040: Implement orchestrator/orchestrator.go part 3 (resume logic to restore state and continue from next turn)
- [ ] T041: Implement orchestrator/completion.go (deliverable comparison, SessionCompleted vs continue if mismatch)

### Implementation Notes
1. Turn alternation: agent1 → agent2 → agent1... (sequential, no concurrent execution per CON-003)
2. State persists to session_state.json after each turn and on pause (FR-015)
3. Resume restores exact session state: turn number, metrics, history (SC-002)
4. Completion requires both agents submit_deliverable with byte-for-byte matching content
5. Max turns enforcement stops session with status=incomplete if reached without deliverable

### Parallel Opportunities
- T036 (session) must complete first
- T037 (turn execution) can run in parallel with T036
- T038-T040 (orchestrator loop components) can run in parallel
- T041 (completion) can run in parallel with T038-T040

### Dependencies
- WP01 (types), WP02 (storage), WP03 (EventBus), WP06 (agent manager) complete

### Risks & Mitigations
- **Risk**: Session state serialization may fail for complex states (1000+ turns)
- **Mitigation**: Test with large sessions early, implement pagination or compression if needed

---

## Work Package WP08: CLI Commands (Priority: P2)

**Goal**: Implement all Cobra commands with flag parsing, output formatting, and proper exit codes per CLI interface contract.

**Independent Test**: Each command executes with various flag combinations, produces correct output formats (table/json/csv), returns appropriate exit codes, help text generated automatically.

**Prompt**: `tasks/planned/WP08-cli-commands.md`

### Included Subtasks
- [ ] T042: Implement cmd/collab/main.go (Cobra root command setup, global flags: --verbose, --no-color, --config, --workspace)
- [ ] T043: Implement cli/run.go (parse template, validate, create session, start orchestrator headless or with TUI)
- [ ] T044: Implement cli/resume.go (load session state, validate status=paused, resume orchestrator)
- [ ] T045: Implement cli/list.go (scan workspace for session_state.json, filter by status, format as table/json/csv)
- [ ] T046: Implement cli/show.go (load session, display summary, optional --messages, --deliverable, --logs, --context flags)
- [ ] T047: Implement cli/validate.go (parse template, run validator, display checkmarks or error messages)
- [ ] T048: Implement cli/init.go (interactive wizard for template creation, example template copying)
- [ ] T049: Implement cli/clean.go (filter sessions by age/status, confirm before deletion unless --force)
- [ ] T050: Implement cli/version.go (display version, Go version, build date, commit hash using ldflags)
- [ ] T051: Implement cli/watch.go (attach TUI to running session or replay completed session) [depends on WP09]

### Implementation Notes
1. Exit codes per contract: 0 (success), 1 (incomplete), 2 (error), 3 (validation), 130 (user interrupt)
2. Output formats: table (default), json, csv support for list command
3. Global flags override config file and env vars (priority hierarchy per FR-054)
4. Cobra auto-generates help text and shell completions
5. All commands validate inputs before executing (fail fast with clear errors)

### Parallel Opportunities
- T042 (main) must complete first
- T043-T050 (individual commands) can run in parallel
- T051 (watch) depends on WP09 TUI completion

### Dependencies
- WP01 (types/config), WP04 (templates), WP07 (orchestrator) complete
- T051 depends on WP09 (TUI)

### Risks & Mitigations
- **Risk**: Output formatting may not handle edge cases (empty sessions, long IDs)
- **Mitigation**: Test with various session states, implement truncation and pagination

---

## Work Package WP09: Terminal UI (TUI) (Priority: P2)

**Goal**: Implement Bubbletea-based real-time monitoring interface with split-pane layout, keyboard navigation, and sub-100ms event updates.

**Independent Test**: TUI renders correctly in 80x24 terminal, updates appear within 100ms of events, keyboard shortcuts work, file viewer displays all file types, TUI exits cleanly on 'q'.

**Prompt**: `tasks/planned/WP09-tui.md`

### Included Subtasks
- [ ] T052: Implement tui/model.go (Bubbletea Model struct with session state, turn history, current file, event subscription)
- [ ] T053: Implement tui/update.go (Update function to handle keyboard input and EventBus messages)
- [ ] T054: Implement tui/view.go (View function rendering header, split panes, status bar with Lipgloss)
- [ ] T055: Implement tui/components/turn_list.go (turn history list with turn number, agent, duration, tokens, cost)
- [ ] T056: Implement tui/components/file_viewer.go (file content viewer with scrolling, markdown syntax highlighting)
- [ ] T057: Implement tui/components/status_bar.go (bottom status bar with current activity, file counts, keybindings)
- [ ] T058: Implement tui/components/header.go (top session header with ID, task name, duration, total cost)
- [ ] T059: Implement tui/styles.go (Lipgloss style definitions for colors, borders, padding)

### Implementation Notes
1. Layout: header (1 line) + split panes (turn list | file viewer) + status bar (1 line)
2. Target: fits in 80x24 terminal without overflow (CON-004)
3. Update latency: <100ms from FileUpdated event to display (SC-004)
4. Keyboard shortcuts: ↑/↓ (navigate turns), ←/→ (switch panes), Tab (cycle files), PgUp/PgDn (scroll), ? (help), q (quit), Ctrl+C (pause)
5. Real-time updates via EventBus subscription (TurnStarted, TurnCompleted, FileUpdated, etc.)

### Parallel Opportunities
- T052 (model) must complete first
- T053 (update) and T054 (view) can run in parallel with T052
- T055-T058 (components) can run in parallel
- T059 (styles) can run in parallel with components

### Dependencies
- WP03 (EventBus) complete for event subscription
- WP07 (orchestrator) for session state and turn history

### Risks & Mitigations
- **Risk**: Bubbletea TUI may degrade with large turn history (100+ turns)
- **Mitigation**: Implement virtual scrolling (render only visible viewport), cap display at 100 most recent turns

---

## Work Package WP10: Documentation & Polish (Priority: P3)

**Goal**: Finalize documentation, optimize binary size, add shell completion, polish error messages, and prepare for v0.1.0 release.

**Independent Test**: Documentation is accurate and complete, binary <50MB, shell completions work in bash/zsh, error messages are helpful, version command shows correct build info.

**Prompt**: `tasks/planned/WP10-documentation-polish.md`

### Included Subtasks
- [ ] T060: Write docs/architecture.md (system architecture overview with diagrams)
- [ ] T061: Write docs/task-templates.md (template format guide with examples)
- [ ] T062: Write docs/mcp-tools.md (MCP tool reference documentation)
- [ ] T063: Update README.md (installation, quick start, examples, links to docs)
- [ ] T064: Optimize binary size (use -ldflags="-s -w" to strip debug info, profile with go tool nm, target <50MB)
- [ ] T065: Add shell completion generation (bash/zsh completions with Cobra, include in install script)
- [ ] T066: Create scripts/install.sh (copy binary to $GOPATH/bin or /usr/local/bin, create config directory)
- [ ] T067: Polish error messages (review all error paths, add suggestions for common failures like missing API key)
- [ ] T068: Add version embedding (use -ldflags "-X main.version=..." to embed version/commit/date)
- [ ] T069: Create CHANGELOG.md with release notes for v0.1.0

### Implementation Notes
1. Documentation should cover all 59 functional requirements and 8 user scenarios
2. Binary size target: <50MB after stripping debug symbols (CON-005)
3. Shell completions for session IDs, commands, and flags
4. Error messages should be actionable (e.g., "ANTHROPIC_API_KEY not set. Run: export ANTHROPIC_API_KEY=...")
5. Version info format: "collab version X.Y.Z\nGo version: ...\nBuild date: ...\nCommit: ..."

### Parallel Opportunities
- T060-T063 (documentation) can run in parallel
- T064-T069 (polish tasks) can run in parallel
- All subtasks are independent and can be parallelized

### Dependencies
- All previous work packages complete (this is final polish)

### Risks & Mitigations
- **Risk**: Binary size may exceed 50MB due to dependencies
- **Mitigation**: Use build flags, profile with go tool nm, consider UPX compression, remove unused dependencies

---

## Dependency & Execution Summary

**Critical Path Sequence**:
1. **WP01** (Project Setup) → Foundation for all work
2. **WP02** (Storage) ∥ **WP03** (EventBus) → Can proceed in parallel after WP01
3. **WP04** (Templates) → Depends on WP01 + WP02
4. **WP05** (MCP Server) → Depends on WP01 + WP02 + WP03
5. **WP06** (Agent Manager) ∥ **WP07** (Orchestrator) → WP06 depends on WP01 + WP03 + WP05; WP07 depends on WP01 + WP02 + WP03 + WP06
6. **WP08** (CLI Commands) ∥ **WP09** (TUI) → Can proceed in parallel after WP07; WP08-T051 depends on WP09
7. **WP10** (Documentation) → Final polish after all work packages complete

**Parallelization Opportunities**:
- **Phase 1 Foundation**: WP01 must complete first (no parallelization)
- **Phase 2 Core Systems**: WP02, WP03, WP04 can partially overlap after WP01
- **Phase 3 Integration**: WP05 after WP01-WP03; WP06 can start after WP05; WP07 can start after WP06
- **Phase 4 User Interface**: WP08 and WP09 can run in parallel after WP07
- **Phase 5 Polish**: WP10 all subtasks can run in parallel

**MVP Scope**:
- **Minimum**: WP01-WP07 + WP08 (run, resume, list, show commands only)
- **Recommended**: WP01-WP09 (includes TUI for full user experience)
- **Complete**: WP01-WP10 (includes documentation and polish for production release)

**Work Package Size Estimates**:
- WP01: 4-6 hours (foundation setup)
- WP02: 6-8 hours (storage implementation and testing)
- WP03: 4-6 hours (EventBus and logging)
- WP04: 4-6 hours (template parsing and validation)
- WP05: 6-8 hours (MCP server and 7 tool handlers)
- WP06: 5-7 hours (agent SDK integration and health checks)
- WP07: 8-10 hours (orchestrator with pause/resume logic)
- WP08: 8-10 hours (9 CLI commands with output formatting)
- WP09: 6-8 hours (TUI with Bubbletea components)
- WP10: 4-6 hours (documentation and polish)
- **Total**: 55-75 hours (approximately 7-10 working days for one developer)

---

## Subtask Index (Reference)

| Subtask ID | Summary | Work Package | Priority | Parallel? |
|------------|---------|--------------|----------|-----------|
| T001 | Initialize Go module and dependencies | WP01 | P0 | No |
| T002 | Create project directory structure | WP01 | P0 | Yes (with T001) |
| T003 | Define core types in pkg/types/ | WP01 | P0 | Yes (after T001) |
| T004 | Implement configuration system | WP01 | P0 | Yes (after T001) |
| T005 | Create Makefile with targets | WP01 | P0 | Yes |
| T006 | Set up golangci-lint configuration | WP01 | P0 | Yes |
| T007 | Implement atomic write pattern | WP02 | P0 | No |
| T008 | Implement workspace management | WP02 | P0 | Yes |
| T009 | Implement session state persistence | WP02 | P0 | Yes |
| T010 | Implement message file operations | WP02 | P0 | Yes |
| T011 | Implement shared context operations | WP02 | P0 | Yes |
| T012 | Implement memory file operations | WP02 | P0 | Yes |
| T013 | Implement deliverable operations | WP02 | P0 | Yes |
| T014 | Implement file watcher with fsnotify | WP02 | P0 | Yes |
| T015 | Define all event types | WP03 | P0 | No |
| T016 | Implement EventBus with channels | WP03 | P0 | Yes (after T015) |
| T017 | Implement subscriber management | WP03 | P0 | Yes (with T016) |
| T018 | Implement structured logger | WP03 | P0 | Yes |
| T019 | Implement event logging subscriber | WP03 | P0 | Yes |
| T020 | Define TaskTemplate struct | WP04 | P0 | No |
| T021 | Implement template parser | WP04 | P0 | Yes (after T020) |
| T022 | Implement template validator | WP04 | P0 | Yes (after T020) |
| T023 | Implement workspace scaffolding | WP04 | P0 | Yes (after T020) |
| T024 | Create example templates | WP04 | P0 | Yes |
| T025 | Implement MCP server with STDIO | WP05 | P1 | No |
| T026 | Implement tool registration | WP05 | P1 | Yes (after T025) |
| T027 | Implement message tools handlers | WP05 | P1 | Yes (after T026) |
| T028 | Implement shared context handlers | WP05 | P1 | Yes (after T026) |
| T029 | Implement memory tools handlers | WP05 | P1 | Yes (after T026) |
| T030 | Implement deliverable handler | WP05 | P1 | Yes (after T026) |
| T031 | Implement Agent struct and methods | WP06 | P1 | No |
| T032 | Implement SpawnAgent with SDK | WP06 | P1 | Yes (after T031) |
| T033 | Implement StartTurn with timeout | WP06 | P1 | Yes (after T031) |
| T034 | Implement health check goroutine | WP06 | P1 | Yes (after T031) |
| T035 | Implement metrics collection | WP06 | P1 | Yes |
| T036 | Implement Session lifecycle | WP07 | P1 | No |
| T037 | Implement Turn execution logic | WP07 | P1 | Yes (with T036) |
| T038 | Implement orchestration loop | WP07 | P1 | Yes (after T036) |
| T039 | Implement pause handling | WP07 | P1 | Yes (after T038) |
| T040 | Implement resume logic | WP07 | P1 | Yes (after T038) |
| T041 | Implement completion detection | WP07 | P1 | Yes |
| T042 | Implement main.go and root command | WP08 | P2 | No |
| T043 | Implement run command | WP08 | P2 | Yes (after T042) |
| T044 | Implement resume command | WP08 | P2 | Yes (after T042) |
| T045 | Implement list sessions command | WP08 | P2 | Yes (after T042) |
| T046 | Implement show command | WP08 | P2 | Yes (after T042) |
| T047 | Implement validate command | WP08 | P2 | Yes (after T042) |
| T048 | Implement init template command | WP08 | P2 | Yes (after T042) |
| T049 | Implement clean command | WP08 | P2 | Yes (after T042) |
| T050 | Implement version command | WP08 | P2 | Yes (after T042) |
| T051 | Implement watch command | WP08 | P2 | Yes (after WP09) |
| T052 | Implement TUI Model | WP09 | P2 | No |
| T053 | Implement TUI Update function | WP09 | P2 | Yes (with T052) |
| T054 | Implement TUI View function | WP09 | P2 | Yes (with T052) |
| T055 | Implement turn list component | WP09 | P2 | Yes (after T052) |
| T056 | Implement file viewer component | WP09 | P2 | Yes (after T052) |
| T057 | Implement status bar component | WP09 | P2 | Yes (after T052) |
| T058 | Implement header component | WP09 | P2 | Yes (after T052) |
| T059 | Implement Lipgloss styles | WP09 | P2 | Yes |
| T060 | Write architecture.md | WP10 | P3 | Yes |
| T061 | Write task-templates.md | WP10 | P3 | Yes |
| T062 | Write mcp-tools.md | WP10 | P3 | Yes |
| T063 | Update README.md | WP10 | P3 | Yes |
| T064 | Optimize binary size | WP10 | P3 | Yes |
| T065 | Add shell completion | WP10 | P3 | Yes |
| T066 | Create install script | WP10 | P3 | Yes |
| T067 | Polish error messages | WP10 | P3 | Yes |
| T068 | Add version embedding | WP10 | P3 | Yes |
| T069 | Create CHANGELOG.md | WP10 | P3 | Yes |

---

> This task breakdown enables complete implementation of the multi-agent collaboration CLI. Each work package is independently deliverable with clear success criteria. Prompt files in `tasks/planned/` will provide detailed implementation guidance for each work package with exhaustive context for autonomous execution.
