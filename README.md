# Shiki 指揮

A CLI for spec-driven development with AI agent orchestration. Think of it as a conductor (指揮 = "shiki") for your development workflow.

## What is this?

Shiki helps you build software using a structured, spec-first approach. Instead of diving straight into code, you define what you're building, plan how to build it, then execute with AI assistance.

The CLI also includes a multi-agent collaboration system where two AI agents can work together on complex tasks - debating architecture decisions, reviewing each other's work, or iterating on designs until they reach consensus.

## Prerequisites

- **Go 1.24+**
- **[Claude Code](https://claude.ai/code)** installed and logged in

That's it! Shiki uses your Claude Code credentials automatically - no API keys to manage.

## Installation

```bash
# Clone and build
git clone https://github.com/dphaener/shiki-cli.git
cd shiki-cli
make build

# Install to ~/.local/bin
./scripts/install.sh

# Verify it works
shiki version
```

## Quick Start

### Multi-Agent Collaboration

The coolest feature - watch two AI agents collaborate on a task in real-time:

```bash
# Run a collaboration session with the TUI
shiki run examples/simple-agreement.md --watch
```

You'll see a split-pane terminal UI showing:
- Left: Turn-by-turn history of agent interactions
- Right: Live file viewer showing shared context, messages, deliverables

```
┌─────────────────────────────────────────────────────────┐
│ Session: abc123 │ Architecture Review │ 00:05:32 │ $0.42│
├──────────────────────────┬──────────────────────────────┤
│  Turn History            │  shared_context.md           │
│  #1 Architect  5s $0.12  │                              │
│  #2 Reviewer   3s $0.08  │  # Current Design            │
│  #3 Architect  4s $0.10  │  We've agreed on...          │
│  ► #4 Reviewer ...       │                              │
├──────────────────────────┴──────────────────────────────┤
│ ⏵ Running │ Tab=files ↑↓=nav q=quit Ctrl+C=pause       │
└─────────────────────────────────────────────────────────┘
```

**Keyboard shortcuts:**
- `↑`/`↓` - Navigate turn history
- `Tab` - Cycle through workspace files
- `PgUp`/`PgDn` - Scroll file content
- `Ctrl+C` - Pause session (resume later with `shiki resume`)
- `q` - Quit TUI (session keeps running)

### Session Management

```bash
# List all sessions
shiki list

# Resume a paused session
shiki resume abc123 --watch

# See what happened in a session
shiki show abc123 --messages

# Clean up old sessions
shiki clean --older-than=7d
```

## How Agent Collaboration Works

You define a task template with two agents, their roles, and what they're working on:

```markdown
---
agent1_name: "Architect"
agent1_role: "Design systems"
agent1_system_prompt: |
  You design scalable systems. Focus on trade-offs.

agent2_name: "Critic"
agent2_role: "Challenge decisions"
agent2_system_prompt: |
  You poke holes in designs. Find edge cases.

max_turns: 10
---

# Task: Design a Rate Limiting System

Requirements:
- 10k requests/second
- Fair across tenants
- Graceful degradation

The agents should debate approaches and submit a final design.
```

The agents take turns, communicating through:
- **Messages** - Direct agent-to-agent chat
- **Shared Context** - A workspace file both can read/write
- **Memory** - Private notes each agent keeps
- **Deliverables** - Final output (session ends when both submit matching deliverables)

## Commands

| Command | What it does |
|---------|--------------|
| `shiki run task.md --watch` | Start a new collaboration session |
| `shiki resume <id>` | Continue a paused session |
| `shiki list` | Show all sessions |
| `shiki show <id>` | Session details, messages, logs |
| `shiki watch <id>` | Attach TUI to running session |
| `shiki validate task.md` | Check if a template is valid |
| `shiki clean --older-than=7d` | Remove old sessions |

## Project Structure

```
shiki-cli/
├── cmd/shiki/           # CLI entry point
├── internal/
│   ├── cli/             # Commands (Cobra)
│   ├── orchestrator/    # Session & turn management
│   ├── agent/           # Claude SDK integration
│   ├── mcp/             # MCP server (7 collaboration tools)
│   ├── tui/             # Terminal UI (Bubbletea)
│   ├── storage/         # Atomic file operations
│   └── events/          # Event bus for real-time updates
├── examples/            # Example task templates
└── docs/                # More documentation
```

## Docs

- [Architecture Overview](docs/architecture.md) - How it all fits together
- [Task Templates Guide](docs/task-templates.md) - Writing collaboration templates
- [MCP Tools Reference](docs/mcp-tools.md) - The 7 tools agents use to collaborate

## Troubleshooting

**"Failed to start agent"**

Make sure Claude Code is installed and you're logged in:
```bash
claude --version  # Should show version
claude login      # If not logged in
```

**"Template validation failed"**

Run `shiki validate task.md` to see what's wrong. Usually it's:
- Missing required fields (agent names, roles, prompts, max_turns)
- YAML syntax errors
- max_turns set to 0 or negative

**"Session not found"**

Check the session ID with `shiki list`. IDs are short hashes like `abc123`.

## Contributing

1. Fork it
2. Create a branch
3. Make your changes
4. Open a PR

## License

MIT

---

Built with [Cobra](https://github.com/spf13/cobra), [Bubbletea](https://github.com/charmbracelet/bubbletea), [Lipgloss](https://github.com/charmbracelet/lipgloss), and [claude-agent-sdk-go](https://github.com/connerohnesorge/claude-agent-sdk-go).
