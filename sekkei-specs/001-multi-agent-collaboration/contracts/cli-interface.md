# CLI Interface Contract

This document defines the command-line interface, exit codes, output formats, and flag specifications.

## Command Structure

```
collab [global-flags] <command> [command-flags] [arguments]
```

## Global Flags

Available on all commands:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--verbose` | `-v` | bool | false | Enable verbose logging output |
| `--no-color` | | bool | false | Disable ANSI color output |
| `--config` | `-c` | string | `~/.config/collab-cli/config.json` | Path to configuration file |
| `--workspace` | `-w` | string | `~/.local/share/collab-cli/sessions` | Path to sessions workspace directory |

## Commands

### `collab run <task-file>`

Start a new collaboration session from a task template.

**Arguments**:
- `task-file` (required): Path to task template markdown file

**Flags**:
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--watch` | | bool | false | Launch interactive TUI to monitor progress |
| `--session-id` | | string | auto-generated | Override automatic session ID |

**Exit Codes**:
- `0`: Session completed successfully (both agents approved deliverable)
- `1`: Session incomplete (max turns reached, no deliverable)
- `2`: Session error (agent crash, API failure, validation error)
- `3`: Invalid task template (validation failed)
- `130`: User interrupted (Ctrl+C) - session saved as paused

**Output (headless mode)**:
```
✓ Session created: 20251123-143022-a7f3
✓ Workspace: /Users/user/.local/share/collab-cli/sessions/20251123-143022-a7f3
✓ Starting agents...

Turn 0 [Architect]: 2.3s, 2345 tokens, $0.06, 2 tools
  → Sent message to Reviewer
  → Updated shared_context.md

Turn 1 [Reviewer]: 3.1s, 1872 tokens, $0.05, 1 tool
  → Sent message to Architect

Turn 2 [Architect]: 2.8s, 2103 tokens, $0.05, 3 tools
  → Sent message to Reviewer
  → Updated shared_context.md
  → Submitted deliverable (pending other agent)

Turn 3 [Reviewer]: 2.2s, 1654 tokens, $0.04, 1 tool
  → Submitted deliverable (APPROVED - session complete!)

✓ Session completed in 10.4s
✓ Total cost: $0.20 (Architect: $0.11, Reviewer: $0.09)
✓ Deliverable: /Users/user/.local/share/collab-cli/sessions/20251123-143022-a7f3/deliverable.md
```

**Output (watch mode)**:
Launches full-screen TUI (see TUI specification below)

---

### `collab resume <session-id>`

Resume a paused collaboration session.

**Arguments**:
- `session-id` (required): Session ID to resume, e.g., `20251123-143022-a7f3`

**Flags**:
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--watch` | | bool | false | Launch interactive TUI |

**Exit Codes**:
- `0`: Session completed successfully
- `1`: Session incomplete (max turns reached)
- `2`: Session error or already completed
- `3`: Session not found or cannot resume
- `130`: User interrupted (Ctrl+C)

**Output**:
```
✓ Resuming session: 20251123-143022-a7f3
✓ Status: paused at turn 5/10
✓ Previous cost: $0.45
✓ Continuing from turn 6...

Turn 6 [Reviewer]: 2.5s, 1923 tokens, $0.05
  → Sent message to Architect
...
```

---

### `collab list sessions`

List all collaboration sessions.

**Flags**:
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--status` | `-s` | string | all | Filter by status: `running`, `paused`, `completed`, `incomplete`, `error` |
| `--format` | `-f` | string | table | Output format: `table`, `json`, `csv` |
| `--limit` | `-n` | int | 50 | Maximum sessions to display |

**Exit Codes**:
- `0`: Success
- `2`: Error reading sessions directory

**Output (table format)**:
```
SESSION ID            STATUS     TURNS  DURATION  COST    STARTED
20251123-143022-a7f3  completed  4/10   10.4s     $0.20   2025-11-23 14:30:22
20251123-120145-b2e9  paused     7/15   45.2s     $0.78   2025-11-23 12:01:45
20251122-163521-c9f1  error      2/10   5.1s      $0.12   2025-11-22 16:35:21
20251122-094432-d3a8  running    3/20   12.3s     $0.34   2025-11-22 09:44:32

Total: 4 sessions
```

**Output (json format)**:
```json
{
  "sessions": [
    {
      "id": "20251123-143022-a7f3",
      "status": "completed",
      "task_name": "Architecture Review",
      "turns": {
        "current": 4,
        "max": 10
      },
      "duration_seconds": 10.4,
      "cost": {
        "total": 0.20,
        "agent1": 0.11,
        "agent2": 0.09
      },
      "started_at": "2025-11-23T14:30:22Z",
      "completed_at": "2025-11-23T14:30:32Z"
    }
  ],
  "total": 1
}
```

---

### `collab show <session-id>`

Display detailed information about a session.

**Arguments**:
- `session-id` (required): Session ID to inspect

**Flags**:
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--messages` | `-m` | bool | false | Show all messages between agents |
| `--deliverable` | `-d` | bool | false | Show final deliverable content |
| `--logs` | `-l` | bool | false | Show orchestrator logs |
| `--context` | | bool | false | Show shared_context.md content |
| `--format` | `-f` | string | text | Output format: `text`, `json` |

**Exit Codes**:
- `0`: Success
- `2`: Session not found
- `3`: Invalid session state file

**Output (default)**:
```
Session: 20251123-143022-a7f3
Task: Architecture Review
Status: completed
Created: 2025-11-23 14:30:22
Duration: 10.4s
Turns: 4/10

Agents:
  agent_1 [Architect]
    Turns: 2
    Tokens: 4448
    Cost: $0.11

  agent_2 [Reviewer]
    Turns: 2
    Tokens: 3526
    Cost: $0.09

Total Cost: $0.20
Total Tokens: 7974

Workspace: /Users/user/.local/share/collab-cli/sessions/20251123-143022-a7f3
Deliverable: /Users/user/.local/share/collab-cli/sessions/20251123-143022-a7f3/deliverable.md
Messages: 4 files in messages/
Logs: orchestrator.log (1247 lines)
```

**Output (--messages)**:
```
Messages (4 total):

[Turn 0] Architect → Reviewer (2025-11-23 14:30:27)
---
I've reviewed the system requirements and drafted an initial architecture.
Please see shared_context.md for details. Let me know your thoughts.
---

[Turn 1] Reviewer → Architect (2025-11-23 14:30:30)
---
The architecture looks solid. I have concerns about the database
scaling approach. See my notes in shared_context.md.
---
...
```

**Output (--deliverable)**:
```
Deliverable (approved by both agents):
───────────────────────────────────────
# System Architecture Design

## Overview
...

[Full deliverable markdown content]
───────────────────────────────────────
```

---

### `collab watch <session-id>`

Attach a TUI to a running session or replay a completed session.

**Arguments**:
- `session-id` (required): Session ID to watch

**Flags**:
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--replay` | `-r` | bool | false | Replay mode (for completed sessions) |

**Exit Codes**:
- `0`: User exited TUI normally (session continues if running)
- `2`: Session not found
- `130`: User interrupted (Ctrl+C)

**Behavior**:
- For running sessions: Displays past turns + real-time updates
- For completed sessions: Requires `--replay` flag, displays full history
- Press `q` to exit TUI without stopping session

---

### `collab validate <task-file>`

Validate a task template file without running it.

**Arguments**:
- `task-file` (required): Path to task template to validate

**Exit Codes**:
- `0`: Template is valid
- `3`: Template validation failed

**Output (valid)**:
```
✓ YAML frontmatter valid
✓ Required fields present
  ✓ agent1_name: "Architect"
  ✓ agent1_role: "System designer"
  ✓ agent1_system_prompt: (245 chars)
  ✓ agent2_name: "Reviewer"
  ✓ agent2_role: "Design critic"
  ✓ agent2_system_prompt: (198 chars)
  ✓ max_turns: 10
✓ Optional fields valid
  ✓ workspace_structure: 2 items
  ✓ turn_timeout_seconds: 300
✓ Task body present (1420 chars)

Template is valid and ready to use.
```

**Output (invalid)**:
```
✗ YAML frontmatter parsing failed
  Line 8: unexpected character '>' in mapping value

✗ Required fields missing
  ✗ agent2_system_prompt (required)
  ✗ max_turns (required)

✗ Validation failed

Fix the errors above and run validation again.
```

---

### `collab init template <name>`

Create a new task template interactively.

**Arguments**:
- `name` (required): Name for the template file (will create `<name>.md`)

**Flags**:
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `./<name>.md` | Output path for template |
| `--example` | | string | | Use example template: `simple`, `code-review`, `architecture` |

**Exit Codes**:
- `0`: Template created successfully
- `2`: File already exists (use --force to overwrite)
- `3`: Invalid template configuration

**Output (interactive)**:
```
Creating task template: design-review.md

Agent 1 Configuration:
  Name: Architect
  Role: System designer
  System Prompt: You are an expert system architect...
  Model [claude-sonnet-4]:

Agent 2 Configuration:
  Name: Reviewer
  Role: Design critic
  System Prompt: You are a meticulous design reviewer...
  Model [claude-sonnet-4]:

Collaboration Settings:
  Max turns [10]: 15
  Turn timeout (seconds) [300]:
  Turn delay (ms) [0]:

Task Description:
  (Opens $EDITOR for task body)

✓ Template created: design-review.md
✓ Run `collab validate design-review.md` to verify
```

---

### `collab clean`

Remove old or completed sessions.

**Flags**:
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--older-than` | | duration | | Remove sessions older than duration (e.g., `7d`, `30d`, `1h`) |
| `--status` | `-s` | string | | Only clean sessions with this status |
| `--force` | `-f` | bool | false | Skip confirmation prompt |
| `--dry-run` | | bool | false | Show what would be deleted without deleting |

**Exit Codes**:
- `0`: Success
- `1`: User cancelled
- `2`: Error during cleanup

**Output**:
```
Sessions to remove:
  20251116-120000-x1y2  completed  7 days ago  $0.45
  20251115-093000-a3b4  error      8 days ago  $0.12
  20251114-140000-c5d6  completed  9 days ago  $0.78

Total: 3 sessions, 45.2 MB

Remove these sessions? [y/N]: y

✓ Removed 3 sessions
✓ Freed 45.2 MB
```

---

### `collab version`

Display version information.

**Exit Codes**: `0`

**Output**:
```
collab version 0.1.0
Go version: go1.23.1
Build date: 2025-11-23
Commit: a7f3c2e
```

---

### `collab help [command]`

Display help information.

**Arguments**:
- `command` (optional): Show help for specific command

**Exit Codes**: `0`

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `COLLAB_CLI_WORKSPACE` | Sessions workspace directory | `~/.local/share/collab-cli/sessions` |
| `COLLAB_CLI_CONFIG` | Configuration file path | `~/.config/collab-cli/config.json` |
| `COLLAB_CLI_NO_COLOR` | Disable color output | `false` |
| `ANTHROPIC_API_KEY` | Anthropic API key (required for agents) | (none) |
| `EDITOR` | Editor for init template command | `vim` |

---

## Configuration File Format

Located at `~/.config/collab-cli/config.json`:

```json
{
  "workspace_dir": "/Users/user/.local/share/collab-cli/sessions",
  "default_model": "claude-sonnet-4",
  "turn_timeout_seconds": 300,
  "cost_limits": {
    "per_session": 10.00,
    "per_agent": 5.00
  },
  "ui": {
    "color": true,
    "refresh_rate_fps": 10
  },
  "logging": {
    "level": "info",
    "format": "json"
  }
}
```

---

## Priority Configuration

Settings are resolved in this order (highest priority first):

1. CLI flags (`--workspace`, `--config`, etc.)
2. Environment variables (`COLLAB_CLI_*`)
3. Configuration file (`config.json`)
4. Built-in defaults

---

## TUI Specification

### Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│ Session: 20251123-143022-a7f3 │ Architecture Review │ 10.4s │ $0.20 │
├───────────────────────────────┬─────────────────────────────────────┤
│                               │                                     │
│  Turn History                 │  File Viewer                        │
│  (scrollable list)            │  (scrollable content)               │
│                               │                                     │
│  0 [Architect]     2.3s  $0.06│  ┌─ shared_context.md ────────────┐│
│    2345 tok  2 tools          │  │ # Architecture Design           ││
│  1 [Reviewer]      3.1s  $0.05│  │                                 ││
│    1872 tok  1 tool           │  │ ## Requirements                 ││
│  2 [Architect]     2.8s  $0.05│  │ - High availability             ││
│    2103 tok  3 tools          │  │ - Low latency                   ││
│  3 [Reviewer]      2.2s  $0.04│  │                                 ││
│    1654 tok  1 tool           │  │ ## Proposed Architecture        ││
│  → Deliverable approved!      │  │ ...                             ││
│                               │  └─────────────────────────────────┘│
├───────────────────────────────┴─────────────────────────────────────┤
│ Status: Completed │ Files: 4 messages, deliverable.md │ [q] quit    │
└─────────────────────────────────────────────────────────────────────┘
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑`/`↓` | Navigate turn history |
| `←`/`→` | Switch between panes |
| `Tab` | Cycle through files |
| `PgUp`/`PgDn` | Scroll file viewer |
| `Home`/`End` | Jump to top/bottom |
| `?` | Show help overlay |
| `q` | Quit TUI (session continues) |
| `Ctrl+C` | Pause session and quit |

### File Tabs

- `shared_context.md`: Shared collaboration context
- `messages/`: All messages (latest shown)
- `memory/agent_1`: Agent 1 private memory
- `memory/agent_2`: Agent 2 private memory
- `deliverable.md`: Final deliverable (if submitted)

### Real-time Updates

- Turn list updates as new turns complete
- File viewer refreshes when files change (via fsnotify)
- Status bar shows current agent activity
- Cost/token counters update live

---

## Exit Code Summary

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Session incomplete (max turns) |
| `2` | Error (agent crash, API failure, IO error) |
| `3` | Validation error (invalid template, session not found) |
| `130` | User interrupted (Ctrl+C, SIGTERM) |
