---
description: "Research decision log for multi-agent collaboration CLI"
---

# Research Decision Log

Document the outcomes of Phase 0 discovery work. Capture every clarification resolved and the supporting evidence that backs each decision.

## Summary

- **Feature**: 001-multi-agent-collaboration
- **Date**: 2025-11-23
- **Researchers**: AI Agent (Claude) with stakeholder validation
- **Open Questions**: None - all critical architecture decisions validated

## Decisions & Rationale

For each decision, include the supporting sources and why the team aligned on this direction.

| Decision | Rationale | Evidence | Status |
|----------|-----------|----------|--------|
| Use claude-agent-sdk-go for agent management | External SDK wraps Claude CLI with channel-based async, eliminating need to build subprocess management from scratch | E-001, E-002, E-003 | final |
| Single shared MCP server architecture | Provides single source of truth for file operations, simplifies event emission, prevents race conditions | E-008, stakeholder validation | final |
| Template-driven workspace scaffolding | Pre-populate session workspaces based on template `workspace_structure` field for consistent agent starting state | Stakeholder validation | final |
| Channel-based EventBus pattern | Idiomatic Go concurrency pattern, non-blocking delivery, supports 100ms TUI update requirement | E-005, E-006 | final |
| Graceful degradation error strategy | Continue on non-critical errors (log writes), preserve state on critical errors (agent crashes, API failures) | Stakeholder validation | final |
| Cobra for CLI framework | Industry standard, provides command hierarchy, flag parsing, help generation | E-009 | final |
| Bubbletea for TUI framework | Elm architecture proven for terminal UIs, integrates with Lipgloss/Bubbles for rich rendering | E-007 | final |
| fsnotify for file watching | Cross-platform, event channels align with EventBus, meets latency requirements | E-006 | final |

## Evidence Highlights

Summarize the most impactful findings from the evidence log. Link back to specific rows so the trail is auditable.

- **SDK Architecture (E-001, E-003)** – claude-agent-sdk-go spawns Claude CLI as subprocess and manages bidirectional communication via goroutines and channels. This eliminates the need to implement process lifecycle management, STDIO protocol, or retry logic from scratch.

- **MCP STDIO Transport (E-004, E-008)** – MCP supports STDIO transport where a single server can handle multiple client connections. This validates our single-server architecture where both agents connect to one MCP tool server.

- **Go Concurrency Patterns (E-005)** – Channel-based pub/sub is the idiomatic Go approach for EventBus. Subscribers receive events via channels, allowing non-blocking delivery and clean shutdown semantics.

- **File Watcher Performance (E-006)** – fsnotify uses OS-native file watching (inotify on Linux, FSEvents on macOS) and delivers events via channels with sub-millisecond latency, well within the 100ms TUI update requirement.

- **TUI Framework Maturity (E-007)** – Bubbletea's Elm architecture (Model/View/Update) provides declarative state management and has proven track record in production terminal applications (gh, glow, soft-serve).

## Technical Architecture Decisions

### Agent SDK Integration

**Chosen Approach**: External dependency on `github.com/connerohnesorge/claude-agent-sdk-go`

**Alternatives Considered**:
- **Build SDK from scratch**: Full control but requires implementing subprocess management, STDIO protocol, error handling, retry logic
- **Direct Claude CLI invocation**: Simpler but loses streaming, async communication, structured message types
- **Use existing SDK**: Leverage tested implementation, focus on orchestration logic

**Rationale**: Using the external SDK allows us to focus on orchestration, session management, and TUI rather than low-level agent process management. The SDK provides channel-based async communication which integrates cleanly with our EventBus pattern.

**Supporting Evidence**: E-001 (subprocess architecture), E-002 (requirements), E-003 (async patterns)

---

### MCP Tool Server Architecture

**Chosen Approach**: Single shared MCP server that both agents connect to via STDIO

**Alternatives Considered**:
- **Per-agent MCP servers**: Each agent has dedicated server, backend storage layer synchronizes
- **Orchestrator-embedded tools**: Orchestrator exposes tools directly without MCP abstraction
- **Shared MCP server**: One server, both agents connect, single source of truth

**Rationale**: Single server eliminates synchronization complexity, provides atomic file operations, centralizes event emission for tool invocations, and aligns with MCP STDIO transport capabilities.

**Supporting Evidence**: E-008 (STDIO multi-client support), stakeholder validation

---

### Workspace Initialization Strategy

**Chosen Approach**: Template-driven scaffolding using `workspace_structure` field in task templates

**Alternatives Considered**:
- **Empty workspace**: Agents create all files via MCP tools
- **Template scaffolding**: Pre-populate directories/files from template definition
- **Source directory copy**: Clone existing directory structure

**Rationale**: Template-driven approach provides consistency (agents start with known files), reduces turn-0 boilerplate (no need to create shared_context.md, memory files), and makes templates self-contained.

**Supporting Evidence**: Stakeholder validation, aligns with FR-058 workspace_structure field

---

### Event Distribution Pattern

**Chosen Approach**: Channel-based EventBus with subscriber channels

**Alternatives Considered**:
- **Callback-based**: Subscribers register functions, orchestrator invokes
- **Channel-based**: Subscribers receive via Go channels
- **Hybrid buffered**: Buffered channels with timeout/drop semantics

**Rationale**: Channel-based is idiomatic Go, enables non-blocking publish (select with default), integrates with goroutine lifecycle management, supports graceful shutdown via channel close.

**Supporting Evidence**: E-005 (Go patterns), E-006 (fsnotify integration)

---

### Error Handling Philosophy

**Chosen Approach**: Graceful degradation - continue on non-critical errors, fail-safe on critical errors

**Alternatives Considered**:
- **Fail-fast**: Any error stops session immediately
- **Auto-retry**: Automatic retry with exponential backoff
- **Graceful degradation**: Classify errors, continue when safe

**Rationale**: Provides best user experience (sessions continue despite log write failures), preserves state for resume on critical failures (agent crashes, API auth), allows cost-conscious pause points.

**Supporting Evidence**: Stakeholder validation, aligns with FR-019 (crash handling)

**Error Classification**:
- **Critical (fail and preserve state)**: Agent process crash, API authentication failure, MCP server crash, disk full preventing session_state.json write
- **Non-critical (log and continue)**: Log file write failure, TUI render error, metrics collection failure, file watcher error

---

### CLI Framework

**Chosen Approach**: spf13/cobra for command structure and flag parsing

**Alternatives Considered**:
- **Standard library flag**: Minimal but lacks subcommands, help generation
- **urfave/cli**: Good alternative but less widely adopted
- **spf13/cobra**: Industry standard, used by kubectl, gh, hugo

**Rationale**: Cobra provides command hierarchy (run, resume, list, show, etc.), automatic help generation, flag inheritance, persistent flags, and shell completion support.

**Supporting Evidence**: E-009 (industry adoption), aligns with FR-001 to FR-010 (CLI commands)

---

### TUI Framework

**Chosen Approach**: Charmbracelet Bubbletea with Lipgloss and Bubbles

**Alternatives Considered**:
- **tview/tcell**: Full-featured but more complex API
- **termui**: Good for dashboards but lacks event handling
- **Bubbletea**: Elm architecture, composable, well-maintained

**Rationale**: Bubbletea's Model/View/Update pattern separates state from rendering, enabling testable TUI logic. Lipgloss provides declarative styling. Bubbles offers reusable components (viewport, list). Ecosystem is actively maintained by Charmbracelet team.

**Supporting Evidence**: E-007 (Elm architecture), aligns with FR-042 to FR-051 (TUI requirements)

---

### File System Watching

**Chosen Approach**: fsnotify for cross-platform file change detection

**Alternatives Considered**:
- **Polling**: Simple but CPU-intensive and high latency
- **OS-specific APIs**: Optimal performance but not portable
- **fsnotify**: Cross-platform wrapper over inotify/FSEvents/etc

**Rationale**: fsnotify provides OS-native performance with portable API, delivers events via channels (integrates with EventBus), and supports the 100ms update latency requirement.

**Supporting Evidence**: E-006 (performance), aligns with SC-004 (100ms file updates)

## Risks / Concerns

- **SDK Dependency Risk**: External SDK (`claude-agent-sdk-go`) is early-stage (20 commits). If SDK has bugs or breaking changes, we're blocked.
  - **Mitigation**: Fork SDK early in implementation. Vendor dependencies. Monitor upstream for critical fixes. Be prepared to contribute patches.

- **MCP Server Concurrency**: Single MCP server handling two agents concurrently requires careful synchronization of file writes.
  - **Mitigation**: Use file locking or atomic writes (write to temp file, rename). Emit events after file operations complete. Test concurrent tool invocations thoroughly.

- **TUI Performance Under Load**: Rendering updates at 10 FPS with large message history could consume CPU.
  - **Mitigation**: Implement virtual scrolling (render visible viewport only). Debounce rapid file change events. Profile and optimize hot paths.

- **Session State Serialization**: Complex session state (goroutines, channels, file watchers) must serialize to JSON for pause/resume.
  - **Mitigation**: Separate transient runtime state from persistent state. Document what gets serialized (turn history, metrics, agent config) vs. what gets reconstructed on resume (event channels, file watchers).

- **Agent Process Crashes**: If agent crashes mid-turn, state may be inconsistent (partial file writes, orphaned processes).
  - **Mitigation**: Implement health check goroutine that monitors agent process. Use atomic file writes. Save session state before each turn starts. Provide clear error messages for manual intervention.

- **Binary Size**: Dependencies (Bubbletea, Cobra, SDK) could inflate binary beyond 50MB constraint.
  - **Mitigation**: Use Go build flags (-ldflags="-s -w" for stripping). Profile binary size with `go tool nm`. Consider UPX compression if needed.

## Next Actions

1. ✓ Validate architecture decisions with stakeholder
2. ✓ Document evidence trail in CSV logs
3. → Generate data-model.md with entities and relationships
4. → Create API contracts for MCP tools in contracts/ directory
5. → Fill in plan.md implementation phases
6. → Update agent context (CLAUDE.md) with technology stack

> Keep this document living. As more evidence arrives, update decisions and rationale so downstream implementers can trust the history.
