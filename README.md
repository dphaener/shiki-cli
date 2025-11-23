# Collab

A production-ready Go CLI tool for orchestrating two-agent AI collaboration via file-based communication, turn-based execution, and real-time TUI monitoring.

## About This Project

Collab enables Claude AI agents to work together on complex tasks through structured collaboration. The system uses file-based communication, a shared MCP tool server, and supports pause/resume capabilities with zero data loss.

This project uses [Sekkei](https://github.com/yourusername/sekkei) for AI-assisted specification-driven development workflows. All features start with a specification, are broken down into testable work packages, and follow a systematic development process.

## Features

- **Two-Agent Collaboration**: Orchestrate agents working together on complex tasks
- **File-Based Communication**: All artifacts stored as human-readable Markdown
- **Pause/Resume**: Save session state at any time and resume later
- **Real-Time Monitoring**: Terminal UI shows live progress and metrics
- **Cost Tracking**: Monitor token usage and costs per agent and session
- **MCP Tool Integration**: Agents communicate via Model Context Protocol

## Prerequisites

- Go 1.24+ (project uses Go 1.24)
- Anthropic API key (set `ANTHROPIC_API_KEY` environment variable)
- golangci-lint (for development)

## Installation

```bash
# Build the binary
make build

# Run tests
make test

# Install to $GOPATH/bin
make install
```

## Quick Start

### Create a New Feature

```
/sekkei.specify "Add user authentication"
```

This creates a feature specification in `sekkei-specs/###-feature-name/spec.md`.

### Generate Implementation Plan

```
/sekkei.plan
```

Creates `plan.md` with architecture decisions, technology choices, and implementation phases.

### Break Down Into Tasks

```
/sekkei.tasks
```

Generates `tasks.md` with work packages and individual task prompts.

### Implement the Feature

```
/sekkei.implement
```

Executes work packages following the Kanban workflow: planned → doing → for_review → done.

## Workflows

### Claude Code Integration

This project includes Claude Code slash commands in `.claude/commands/`:

- `/sekkei.specify` - Create feature specifications
- `/sekkei.plan` - Generate implementation plans
- `/sekkei.tasks` - Break down work into tasks
- `/sekkei.research` - Conduct technical research
- `/sekkei.implement` - Execute implementation
- `/sekkei.review` - Review completed work
- `/sekkei.accept` - Validate and merge features
- `/sekkei.constitution` - Manage project principles

### Available Scripts

**Bash scripts** (`.sekkei/scripts/bash/`):
- `setup-spec.sh` - Create new feature specification
- `setup-plan.sh` - Generate implementation plan
- `setup-tasks.sh` - Break down into work packages
- `setup-research.sh` - Set up technical research
- `setup-implement.sh` - Start implementation workflow
- `update-agent-context.sh` - Update CLAUDE.md with latest context

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

## License

[Your License Here]

---

*Generated by Sekkei 2025-11-23*
