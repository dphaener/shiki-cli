# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-11-23

### Added

**CLI Tool**
- Complete command-line interface for multi-agent AI collaboration
- 9 commands: `run`, `resume`, `list`, `show`, `watch`, `validate`, `init`, `clean`, `version`
- Global flags: `--verbose`, `--no-color`, `--config`, `--workspace`
- Shell completion generation for bash, zsh, fish, and PowerShell
- Exit code conventions (0=success, 1=incomplete, 2=error, 3=validation, 130=interrupt)

**Agent Orchestration**
- Turn-based execution with agent alternation (agent1 → agent2 → agent1...)
- Configurable turn timeout (default 60s)
- Max turns enforcement with incomplete status handling
- Pause/resume with zero data loss (SIGTERM/SIGINT handling)
- Session state persistence after each turn
- Automatic completion detection via deliverable matching

**MCP Integration**
- Single shared MCP server for both agents (STDIO transport)
- 7 collaboration tools:
  - `send_message` / `read_messages` - Agent messaging
  - `write_shared_context` / `read_shared_context` - Shared workspace
  - `update_memory` / `read_memory` - Per-agent private notes
  - `submit_deliverable` - Deliverable submission with approval
- Tool invocation event emission for observability
- Atomic file operations (temp-file-and-rename pattern)

**Terminal UI (TUI)**
- Real-time monitoring with Bubbletea/Lipgloss
- Split-pane layout: turn history | file viewer
- Header: session ID, task name, duration, total cost
- Status bar: current activity, file counts, keybindings
- Keyboard navigation: ↑↓ (turns), Tab (files), PgUp/PgDn (scroll), q (quit), Ctrl+C (pause)
- <100ms update latency via EventBus subscription
- Support for 80x24 terminal minimum

**Storage System**
- XDG-compliant directories (~/.config/shiki-cli, ~/.local/share/shiki-cli)
- Atomic writes for all file operations (prevents corruption)
- File permissions: directories 0700, files 0600 (user-only)
- Workspace structure:
  - `session_state.json` - Session metadata
  - `orchestrator.log` - Structured event log (JSON)
  - `shared_context.md` - Shared workspace context
  - `messages/` - Agent messages with YAML frontmatter
  - `memory/` - Per-agent private memory files
  - `deliverables/` - Submitted deliverables
- File watching with fsnotify (<10ms detection latency)

**Template System**
- YAML frontmatter + Markdown body format
- Required fields: agent names, roles, system prompts, max_turns
- Optional workspace structure scaffolding
- Template validation with helpful error messages
- Example templates:
  - `simple-agreement.md` - Quick consensus task
  - `code-review.md` - Code review collaboration
  - `architecture-design.md` - Complex system design

**Event-Driven Architecture**
- Channel-based EventBus with non-blocking publish
- 14 event types (SessionCreated, TurnStarted, FileUpdated, etc.)
- Multiple subscribers (TUI, logger, file watcher)
- Graceful shutdown with event draining

**Metrics and Cost Tracking**
- Token counting per turn and per agent
- Cost calculation based on model pricing
- Metrics accuracy target: within 1% of SDK values
- Real-time cost display in TUI

**Agent Management**
- Integration with claude-agent-sdk-go
- Health checks every 5 seconds
- Process crash detection within 1s
- Graceful shutdown (SIGTERM → 5s → SIGKILL)
- No orphaned processes

**Configuration**
- Priority hierarchy: CLI flags > env vars > config file > defaults
- Config file: `~/.config/shiki-cli/config.json`
- Configurable fields: workspace_dir, log_level, default_timeout, anthropic_api_key
- Environment variable support: `ANTHROPIC_API_KEY`, `COLLAB_WORKSPACE_DIR`, etc.

**Documentation**
- Comprehensive architecture documentation (docs/architecture.md)
- Task template guide with examples and patterns (docs/task-templates.md)
- MCP tools reference with usage examples (docs/mcp-tools.md)
- Updated README with quick start and troubleshooting

**Development**
- Makefile with build, test, lint, install, clean targets
- Version embedding via ldflags (version, commit, build date)
- Binary size optimization (-s -w flags)
- golangci-lint configuration
- Unit tests with >80% coverage
- Integration tests for CLI commands

### Features

**Collaboration Patterns**
- Proposal-Approval: One agent proposes, other approves/rejects
- Debate-Consensus: Equal authority agents reach agreement
- Builder-Validator: Incremental building with validation
- Researcher-Synthesizer: Research gathering and synthesis

**Session Management**
- Create new sessions from templates
- Pause sessions at any turn with state preservation
- Resume sessions from exact point of pause
- List sessions with filtering (status, date)
- Show session details (messages, deliverables, logs, context)
- Clean old sessions with age and status filters

**File-Based Collaboration**
- All artifacts in human-readable Markdown
- Message files with YAML frontmatter (from, to, timestamp, turn)
- Shared context for collaboration state
- Per-agent memory for private notes
- Deliverable comparison for completion detection

**Real-Time Monitoring**
- Live turn execution updates
- File change detection and display
- Cost and token tracking
- Turn history with metrics
- File content viewer with markdown support

### Performance

- Session startup: <10s (agent spawning + MCP server initialization)
- TUI update latency: <100ms (event-driven updates)
- File write detection: <10ms (fsnotify)
- EventBus throughput: 10,000+ events/sec
- Binary size: <50MB (optimized build with stripped debug symbols)
- Graceful shutdown: <5s (all components)

### Security

- API key protection (never logged, environment variable or config file)
- File permissions: user-only access (0700 directories, 0600 files)
- Process isolation (agents run as user, no elevated privileges)
- Input validation (template field validation, UTF-8 checks)
- Path traversal prevention (workspace_structure paths sanitized)

### Requirements Met

- All 59 functional requirements (FR-001 to FR-059)
- All 10 success criteria (SC-001 to SC-010)
- All 8 user scenarios (P1-P3 priority levels)
- Zero data loss on pause/resume
- Sub-second response time for CLI commands
- Single-binary deployment

### Technology Stack

- **Language**: Go 1.24
- **CLI Framework**: spf13/cobra
- **TUI Framework**: charmbracelet/bubbletea + lipgloss + bubbles
- **Agent SDK**: github.com/connerohnesorge/claude-agent-sdk-go
- **File Watching**: fsnotify/fsnotify
- **Template Parsing**: gopkg.in/yaml.v3
- **Build System**: Make
- **Linting**: golangci-lint

### Deployment

- Single binary (<50MB)
- No external dependencies
- Cross-platform (Linux, macOS, Windows via PowerShell)
- Install script (scripts/install.sh)
- Shell completions (bash, zsh, fish, PowerShell)

## [Unreleased]

### Planned (Future Releases)

**Multi-Agent Support** (v0.2.0)
- Support for >2 agents
- Configurable turn order (round-robin, priority-based)
- Agent role specialization

**Advanced Features** (v0.3.0)
- Streaming turn execution (partial results)
- Cost budgets (halt when limit reached)
- Session replay (visualize completed sessions)
- Web UI (alternative to TUI)

**Collaboration Enhancements** (v0.4.0)
- Template marketplace (share templates)
- Remote agent execution (cloud agents)
- Agent memory persistence across sessions
- Custom MCP tool plugins

**Performance Optimizations** (v0.5.0)
- Parallel agent execution (optional, for independent tasks)
- Incremental file watching (reduce CPU)
- Session compression (reduce disk usage)
- Metrics export (Prometheus, InfluxDB)

---

## Version History

- **0.1.0** (2025-11-23) - Initial release with core collaboration features

---

**Legend**:
- **Added** - New features
- **Changed** - Changes to existing functionality
- **Deprecated** - Soon-to-be removed features
- **Removed** - Removed features
- **Fixed** - Bug fixes
- **Security** - Security improvements

---

Generated with Sekkei specification-driven development.
