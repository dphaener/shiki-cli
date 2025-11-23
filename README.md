# Collab

A production-ready Go CLI tool for orchestrating two-agent AI collaboration via file-based communication, turn-based execution, and real-time TUI monitoring.

## Overview

Collab enables two Claude AI agents to work together on complex tasks through structured collaboration. The system uses:

- **File-based communication** - All artifacts stored as human-readable Markdown
- **Turn-based execution** - Agents alternate turns, preventing race conditions
- **MCP tool integration** - 7 collaboration tools for messaging, context, memory, and deliverables
- **Real-time TUI** - Monitor agent collaboration live with split-pane terminal UI
- **Pause/resume** - Save session state at any time with zero data loss
- **Cost tracking** - Monitor token usage and costs per agent and session

This project uses [Sekkei](https://github.com/yourusername/sekkei) for AI-assisted specification-driven development workflows.

## Features

- **Template-driven collaboration** - Define agent roles, system prompts, and workspace structure via YAML frontmatter
- **Turn-based orchestration** - Agents alternate turns with configurable timeouts and max turn limits
- **7 MCP collaboration tools**:
  - `send_message` / `read_messages` - Agent-to-agent communication
  - `write_shared_context` / `read_shared_context` - Shared workspace state
  - `update_memory` / `read_memory` - Per-agent private notes
  - `submit_deliverable` - Completion with approval mechanism
- **Session management** - Create, pause, resume, list, show, clean sessions
- **Real-time TUI** - Split-pane view with turn history, file viewer, and keyboard navigation
- **Atomic file operations** - Temp-file-and-rename pattern prevents corruption
- **Event-driven architecture** - EventBus with <100ms TUI update latency
- **Single-binary deployment** - No dependencies, <50MB binary size

## Prerequisites

- **Go 1.24+** (project uses Go 1.24)
- **Anthropic API key** - Set `ANTHROPIC_API_KEY` environment variable
- **golangci-lint** - For development (optional)

## Quick Start

### Installation

**Option 1: Build from source**
```bash
# Clone repository
git clone https://github.com/yourusername/collab.git
cd collab

# Build the binary
make build

# Install to $HOME/.local/bin
make install

# Verify installation
collab version
```

**Option 2: Use install script**
```bash
# Build and run install script
make build
./scripts/install.sh

# Add to PATH if needed
export PATH="$HOME/.local/bin:$PATH"
```

### Set API Key

```bash
export ANTHROPIC_API_KEY="your-api-key-here"

# Or add to ~/.config/collab-cli/config.json:
{
  "anthropic_api_key": "your-api-key-here"
}
```

### Run Your First Collaboration

```bash
# 1. Validate example template
collab validate examples/simple-agreement.md

# 2. Run collaboration with TUI
collab run examples/simple-agreement.md --watch

# 3. Watch agents collaborate in real-time!
```

**TUI Interface**:
```
┌─────────────────────────────────────────────────────────┐
│ Session: abc123 │ Task │ 00:05:32 │ $0.42              │
├──────────────────────────┬──────────────────────────────┤
│                          │                              │
│  Turn History            │  File Viewer                 │
│  #1 agent1  5s  $0.12    │  shared_context.md           │
│  #2 agent2  3s  $0.08    │                              │
│  #3 agent1  4s  $0.10    │  Current file content...     │
│                          │                              │
├──────────────────────────┴──────────────────────────────┤
│ ⏵ Running │ 5 files │ q=quit Tab=files ↑↓=nav        │
└─────────────────────────────────────────────────────────┘
```

**Keyboard Shortcuts**:
- `↑`/`↓` - Navigate turn history
- `Tab` - Cycle through files
- `PgUp`/`PgDn` - Scroll file viewer
- `Ctrl+C` - Pause session
- `q` - Quit TUI (session continues)

### Pause and Resume

```bash
# Pause with Ctrl+C during execution
# Session state saved automatically

# Resume later
collab resume <session-id>

# List all sessions
collab list

# Show session details
collab show <session-id>
```

## Commands

### Core Commands

| Command | Description | Example |
|---------|-------------|---------|
| `run` | Start new collaboration session | `collab run task.md --watch` |
| `resume` | Continue paused session | `collab resume abc123` |
| `list` | Show all sessions | `collab list --status=paused` |
| `show` | Display session details | `collab show abc123 --messages` |
| `watch` | Attach TUI to session | `collab watch abc123` |

### Template Commands

| Command | Description | Example |
|---------|-------------|---------|
| `validate` | Check template validity | `collab validate task.md` |
| `init` | Create new template | `collab init` |

### Utility Commands

| Command | Description | Example |
|---------|-------------|---------|
| `clean` | Remove old sessions | `collab clean --older-than=7d` |
| `version` | Show version info | `collab version` |

### Examples

**Run collaboration with TUI**:
```bash
collab run examples/architecture-design.md --watch
```

**Run headless (no TUI)**:
```bash
collab run task.md
# Monitor in another terminal:
tail -f ~/.local/share/collab-cli/sessions/<session-id>/orchestrator.log
```

**Resume paused session**:
```bash
collab resume abc123 --watch
```

**List sessions**:
```bash
# All sessions
collab list

# Paused sessions only
collab list --status=paused --format=table

# JSON output
collab list --format=json
```

**Show session details**:
```bash
# Summary
collab show abc123

# With messages
collab show abc123 --messages

# With deliverable
collab show abc123 --deliverable

# With logs
collab show abc123 --logs
```

**Clean old sessions**:
```bash
# Interactive (confirms before deletion)
collab clean --older-than=7d

# Force (no confirmation)
collab clean --older-than=30d --force

# Clean completed sessions only
collab clean --status=completed --older-than=7d
```

## Documentation

- **[Architecture Overview](docs/architecture.md)** - System architecture, components, data flow
- **[Task Templates Guide](docs/task-templates.md)** - Template format, examples, best practices
- **[MCP Tools Reference](docs/mcp-tools.md)** - Tool specifications, usage patterns, examples

## Creating Task Templates

Templates define agent roles, system prompts, and workspace structure using YAML frontmatter:

```markdown
---
agent1_name: "Architect"
agent1_role: "Design systems"
agent1_system_prompt: |
  You design scalable, maintainable systems.
  Focus on trade-offs and constraints.
agent2_name: "Reviewer"
agent2_role: "Challenge design decisions"
agent2_system_prompt: |
  You review designs critically.
  Identify weaknesses and edge cases.
max_turns: 10
workspace_structure:
  - path: "design.md"
    content: "# System Design\n\n"
---

# Task: Design a Multi-Tenant SaaS Platform

## Requirements
- Support 10,000+ organizations
- API response time < 200ms p95
- GDPR, SOC2 compliance

## Deliverable
Submit architecture document with diagrams, schema, and cost estimates.
```

**See** [Task Templates Guide](docs/task-templates.md) for complete documentation.

## Development Workflow (Sekkei)

This project uses [Sekkei](https://github.com/yourusername/sekkei) for specification-driven development:

### Claude Code Integration

Slash commands in `.claude/commands/`:

- `/sekkei.specify` - Create feature specifications
- `/sekkei.plan` - Generate implementation plans
- `/sekkei.tasks` - Break down work into tasks
- `/sekkei.implement` - Execute implementation
- `/sekkei.review` - Review completed work
- `/sekkei.accept` - Validate and merge features

## Project Structure

### Go Application Structure
```
collab/
├── cmd/collab/          # Entry point
├── internal/            # Private packages
│   ├── cli/             # CLI commands (Cobra)
│   ├── orchestrator/    # Session & turn execution
│   ├── agent/           # Agent manager (SDK integration)
│   ├── mcp/             # MCP server
│   ├── storage/         # File operations (atomic writes)
│   ├── events/          # EventBus (channel-based pub/sub)
│   ├── tui/             # Terminal UI (Bubbletea)
│   ├── template/        # Task template parser
│   ├── config/          # Configuration (XDG-compliant)
│   └── logging/         # Structured logging
├── pkg/types/           # Shared types
├── examples/            # Example task templates
├── docs/                # Documentation
├── scripts/             # Build scripts
├── Makefile             # Build automation
└── .golangci.yml        # Linter configuration
```

### Sekkei Project Structure
```
Collab/
├── .sekkei/                  # Sekkei configuration and templates
│   ├── templates/            # Markdown templates for workflows
│   ├── scripts/              # Workflow automation scripts
│   └── memory/
│       └── constitution.md   # Project principles and standards
├── sekkei-specs/             # Feature specifications
│   └── ###-feature-name/
│       ├── spec.md           # Feature specification
│       ├── plan.md           # Implementation plan
│       ├── tasks.md          # Work breakdown
│       ├── contracts/        # API contracts and interfaces
│       ├── research/         # Technical research (optional)
│       └── tasks/            # Kanban-style task management
│           ├── planned/      # Work packages ready to start
│           ├── doing/        # Active work packages
│           ├── for_review/   # Completed work packages awaiting review
│           └── done/         # Reviewed and accepted work
├── .claude/commands/         # Claude Code slash commands
├── .worktrees/               # Git worktrees for parallel development
└── README.md                 # This file
```

## Development Workflow

Sekkei follows a structured development process:

1. **Specify** → Create feature specification (WHAT and WHY)
2. **Plan** → Design architecture and implementation approach (HOW)
3. **Research** → Investigate technical options (optional, before planning)
4. **Tasks** → Break down into concrete work packages
5. **Implement** → Execute work packages with clear prompts
6. **Review** → Validate quality and correctness
7. **Accept** → Merge completed features

Each phase has clear deliverables and validation criteria.

## Contributing

All features follow the Sekkei workflow:

1. Start with `/sekkei.specify` to document the feature
2. Use `/sekkei.plan` to design the implementation
3. Run `/sekkei.tasks` to break down into work packages
4. Execute with `/sekkei.implement`
5. Review and iterate as needed
6. Merge with `/sekkei.accept`

## Project Constitution

Project-specific principles and standards are documented in `.sekkei/memory/constitution.md`. This living document defines:

- Core principles that guide all development
- Architecture standards and patterns
- Code style conventions
- Testing requirements
- Decision log with rationale

Use `/sekkei.constitution` to update as the project evolves.

## Contributing

All features follow the Sekkei workflow:

1. Start with `/sekkei.specify` to document the feature
2. Use `/sekkei.plan` to design the implementation
3. Run `/sekkei.tasks` to break down into work packages
4. Execute with `/sekkei.implement`
5. Review and iterate as needed
6. Merge with `/sekkei.accept`

## Troubleshooting

### "ANTHROPIC_API_KEY not set"

**Solution**: Export API key or add to config file:
```bash
export ANTHROPIC_API_KEY="your-api-key"
# or
echo '{"anthropic_api_key": "your-api-key"}' > ~/.config/collab-cli/config.json
```

### "Template validation failed"

**Solution**: Run `collab validate task.md` to see specific errors. Common issues:
- Missing required fields (agent names, roles, system prompts, max_turns)
- Invalid YAML syntax
- max_turns <= 0

### "Session not found"

**Solution**: Check session ID with `collab list`. Session IDs are short hashes (e.g., `abc123`).

### "Failed to start agent"

**Possible causes**:
- ANTHROPIC_API_KEY not set
- Network connectivity issues
- Anthropic API rate limits

**Solution**: Check logs with `collab show <session-id> --logs` for details.

## Performance

- **Session startup**: <10s (agent spawning + MCP server)
- **TUI update latency**: <100ms (event-driven updates)
- **File write detection**: <10ms (fsnotify)
- **Binary size**: <50MB (optimized build)

## License

MIT License

Copyright (c) 2025

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

---

**Built with**:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [claude-agent-sdk-go](https://github.com/connerohnesorge/claude-agent-sdk-go) - Agent SDK
- [fsnotify](https://github.com/fsnotify/fsnotify) - File watching

*Generated with Sekkei specification-driven development*
