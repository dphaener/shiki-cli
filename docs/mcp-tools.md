# MCP Tools Reference

## Overview

Collab exposes **7 collaboration tools** to agents via the Model Context Protocol (MCP). Agents invoke these tools to communicate, share context, maintain memory, and submit deliverables.

**Transport**: STDIO (single shared MCP server for both agents)

**Protocol Version**: MCP 1.0

**Event Emission**: All tool invocations emit `ToolInvoked` events to the EventBus for logging and TUI updates.

## Tool Categories

| Tool | Purpose | Shared/Private |
|------|---------|----------------|
| `send_message` | Send message to other agent | Shared (visible to both) |
| `read_messages` | Read messages from other agent | Shared (reads all messages) |
| `write_shared_context` | Update shared workspace context | Shared (both can read/write) |
| `read_shared_context` | Read shared workspace context | Shared (both can read) |
| `update_memory` | Update agent's private notes | Private (per-agent) |
| `read_memory` | Read agent's private notes | Private (per-agent) |
| `submit_deliverable` | Submit final deliverable | Completion trigger |

## Tool Specifications

### 1. send_message

**Purpose**: Send a markdown message to the other agent in the collaboration.

**Use Case**: Agent wants to communicate findings, ask questions, or provide feedback.

**Input**:
```json
{
  "content": "string (markdown, min length 1)"
}
```

**Output**:
```json
{
  "message_path": "/path/to/messages/001-agent1.md",
  "message_id": "001"
}
```

**Behavior**:
1. Creates message file in `messages/` directory
2. Message filename: `<id>-<agent_name>.md` (e.g., `001-agent1.md`)
3. Message includes YAML frontmatter with metadata:
   ```yaml
   ---
   from: agent1
   to: agent2
   timestamp: 2025-11-23T10:30:00Z
   turn: 3
   ---
   <content here>
   ```
4. Emits `MessageSent` event
5. Emits `ToolInvoked` event

**Errors**:
- `VALIDATION_ERROR`: Content is empty or invalid UTF-8
- `IO_ERROR`: Failed to write message file

**Example**:
```json
// Input
{
  "content": "# Design Proposal\n\nI propose a microservices architecture..."
}

// Output
{
  "message_path": "/Users/user/.local/share/collab-cli/sessions/abc123/messages/003-architect.md",
  "message_id": "003"
}
```

**From Agent Perspective**:
```python
# Agent wants to send a design proposal
response = mcp.invoke_tool("send_message", {
    "content": """# Design Proposal

## Architecture
Microservices with API gateway...

## Trade-offs
- Pro: Scalability
- Con: Complexity

Thoughts?"""
})

print(f"Message sent: {response['message_id']}")
```

---

### 2. read_messages

**Purpose**: Read all messages from the other agent, optionally filtered by turn number.

**Use Case**: Agent wants to review conversation history or catch up on latest messages.

**Input**:
```json
{
  "since_turn": 0  // optional, filter messages >= this turn
}
```

**Output**:
```json
{
  "messages": [
    {
      "id": "001",
      "from": "agent2",
      "to": "agent1",
      "timestamp": "2025-11-23T10:15:00Z",
      "turn": 1,
      "content": "Message content...",
      "file_path": "/path/to/messages/001-agent2.md"
    },
    ...
  ]
}
```

**Behavior**:
1. Scans `messages/` directory
2. Parses YAML frontmatter from each message file
3. Filters messages from other agent (not self)
4. Optionally filters by `since_turn`
5. Returns messages sorted by turn number
6. Emits `ToolInvoked` event

**Errors**:
- `IO_ERROR`: Failed to read messages directory
- `PARSE_ERROR`: Message file has invalid YAML frontmatter

**Example**:
```json
// Input: Get all messages
{}

// Output
{
  "messages": [
    {
      "id": "001",
      "from": "reviewer",
      "to": "architect",
      "timestamp": "2025-11-23T10:15:00Z",
      "turn": 2,
      "content": "I have concerns about the database design...",
      "file_path": "/Users/user/.local/share/collab-cli/sessions/abc123/messages/001-reviewer.md"
    },
    {
      "id": "002",
      "from": "reviewer",
      "to": "architect",
      "timestamp": "2025-11-23T10:25:00Z",
      "turn": 4,
      "content": "Thanks for addressing my concerns. One more question...",
      "file_path": "/Users/user/.local/share/collab-cli/sessions/abc123/messages/002-reviewer.md"
    }
  ]
}
```

```json
// Input: Get messages since turn 3
{
  "since_turn": 3
}

// Output
{
  "messages": [
    {
      "id": "002",
      "from": "reviewer",
      "to": "architect",
      "timestamp": "2025-11-23T10:25:00Z",
      "turn": 4,
      "content": "Thanks for addressing my concerns...",
      "file_path": "/..."
    }
  ]
}
```

**From Agent Perspective**:
```python
# Read latest messages
response = mcp.invoke_tool("read_messages", {
    "since_turn": current_turn - 1
})

for msg in response['messages']:
    print(f"Turn {msg['turn']} - {msg['from']}: {msg['content'][:50]}...")
```

---

### 3. write_shared_context

**Purpose**: Overwrite the shared collaboration context file with new content.

**Use Case**: Agent wants to update the workspace state, document decisions, or maintain shared notes.

**Visibility**: Both agents can read and write this file.

**Input**:
```json
{
  "content": "string (markdown, min length 0)"
}
```

**Output**:
```json
{
  "success": true,
  "path": "/path/to/shared_context.md"
}
```

**Behavior**:
1. Overwrites `shared_context.md` with new content
2. Uses atomic write (temp file + rename)
3. Emits `FileUpdated` event (TUI updates)
4. Emits `ToolInvoked` event

**Errors**:
- `VALIDATION_ERROR`: Content is invalid UTF-8
- `IO_ERROR`: Failed to write shared_context.md

**Example**:
```json
// Input
{
  "content": "# Current Design\n\n## Decisions\n- Database: PostgreSQL\n- API: REST\n\n## Open Questions\n- Caching strategy?"
}

// Output
{
  "success": true,
  "path": "/Users/user/.local/share/collab-cli/sessions/abc123/shared_context.md"
}
```

**From Agent Perspective**:
```python
# Update shared context with current state
mcp.invoke_tool("write_shared_context", {
    "content": """# Design Session

## Agreed Decisions
- Architecture: Microservices
- Database: PostgreSQL with sharding
- API: GraphQL

## Pending Decisions
- Caching: Redis vs Memcached?
- Auth: JWT vs session cookies?

## Next Steps
1. Finalize caching strategy
2. Design auth flow
"""
})
```

**Best Practices**:
- Use shared_context as a "working document"
- Update incrementally (read first, modify, write back)
- Structure with headers for clarity
- Document decisions + rationale
- Track open questions

---

### 4. read_shared_context

**Purpose**: Read the current content of the shared collaboration context file.

**Use Case**: Agent wants to check current workspace state or review documented decisions.

**Input**:
```json
{}  // No parameters
```

**Output**:
```json
{
  "content": "Markdown content of shared_context.md",
  "path": "/path/to/shared_context.md"
}
```

**Behavior**:
1. Reads `shared_context.md`
2. Returns raw markdown content
3. Emits `ToolInvoked` event

**Errors**:
- `IO_ERROR`: Failed to read shared_context.md (file may not exist yet)

**Example**:
```json
// Input
{}

// Output
{
  "content": "# Design Session\n\n## Decisions\n- Database: PostgreSQL\n...",
  "path": "/Users/user/.local/share/collab-cli/sessions/abc123/shared_context.md"
}
```

**From Agent Perspective**:
```python
# Check current shared context
response = mcp.invoke_tool("read_shared_context", {})
context = response['content']

# Parse and update
if "## Open Questions" in context:
    # Add answer to open questions
    updated = context.replace(
        "## Open Questions\n- Caching strategy?",
        "## Open Questions\n- ~~Caching strategy?~~ ANSWERED: Use Redis"
    )
    mcp.invoke_tool("write_shared_context", {"content": updated})
```

---

### 5. update_memory

**Purpose**: Overwrite this agent's private memory file with new content.

**Use Case**: Agent wants to maintain private notes, track strategy, or remember context across turns.

**Visibility**: Only this agent can read/write its memory file. Other agent cannot access.

**Input**:
```json
{
  "content": "string (markdown, min length 0)"
}
```

**Output**:
```json
{
  "success": true,
  "path": "/path/to/memory/agent1.md"
}
```

**Behavior**:
1. Overwrites `memory/<agent_name>.md` with new content
2. Uses atomic write
3. Per-agent isolation (agent1 → memory/agent1.md, agent2 → memory/agent2.md)
4. Emits `ToolInvoked` event

**Errors**:
- `VALIDATION_ERROR`: Content is invalid UTF-8
- `IO_ERROR`: Failed to write memory file

**Example**:
```json
// Input
{
  "content": "# My Strategy\n\n- Be constructive in feedback\n- Focus on security\n\n## Notes\n- Reviewer seems concerned about scalability"
}

// Output
{
  "success": true,
  "path": "/Users/user/.local/share/collab-cli/sessions/abc123/memory/architect.md"
}
```

**From Agent Perspective**:
```python
# Update private memory
mcp.invoke_tool("update_memory", {
    "content": """# Review Strategy

## Focus Areas
- Security (SQL injection, XSS)
- Performance (N+1 queries)
- Maintainability (code duplication)

## Session Notes
- Turn 3: Developer addressed security concerns
- Turn 5: Still concerned about caching strategy
- Turn 7: Need to push for load testing plan

## Next Turn
- Ask about monitoring/alerting
- Request error handling details
"""
})
```

**Best Practices**:
- Use memory as a "scratch pad"
- Track your strategy and focus areas
- Note what you've learned from other agent
- Plan next turn's talking points
- Memory persists across pause/resume

---

### 6. read_memory

**Purpose**: Read the current content of this agent's private memory file.

**Use Case**: Agent wants to recall previous notes or strategy.

**Input**:
```json
{}  // No parameters
```

**Output**:
```json
{
  "content": "Markdown content of this agent's memory.md",
  "path": "/path/to/memory/agent1.md"
}
```

**Behavior**:
1. Reads `memory/<agent_name>.md`
2. Returns raw markdown content
3. Emits `ToolInvoked` event

**Errors**:
- `IO_ERROR`: Failed to read memory file (file may not exist yet)

**Example**:
```json
// Input
{}

// Output
{
  "content": "# My Strategy\n\n- Be constructive...",
  "path": "/Users/user/.local/share/collab-cli/sessions/abc123/memory/architect.md"
}
```

**From Agent Perspective**:
```python
# Recall previous notes
response = mcp.invoke_tool("read_memory", {})
memory = response['content']

# Check if we noted something important
if "TODO" in memory:
    # Extract TODOs and address them
    ...
```

---

### 7. submit_deliverable

**Purpose**: Submit the final collaboration deliverable with approval.

**Completion Trigger**: Session completes when **both agents** submit deliverables with:
1. `approved: true`
2. **Identical content** (byte-for-byte match)

**Input**:
```json
{
  "content": "string (markdown, min length 1)",
  "approved": true  // must be true
}
```

**Output**:
```json
{
  "status": "pending" | "completed",
  "message": "Human-readable status message"
}
```

**Behavior**:
1. Validates `approved == true` and content non-empty
2. Writes deliverable to `deliverables/<agent_name>.md`
3. If other agent hasn't submitted: returns `status: "pending"`
4. If other agent submitted:
   - Compare content byte-for-byte
   - If match: Emit `SessionCompleted` event, return `status: "completed"`
   - If mismatch: Return error `CONTENT_MISMATCH`
5. Emits `ToolInvoked` event

**Errors**:
- `VALIDATION_ERROR`: Content empty or `approved` is false
- `CONTENT_MISMATCH`: Both agents submitted but content doesn't match
- `IO_ERROR`: Failed to write deliverable file

**Notes**:
- Agents can submit multiple times (latest replaces previous)
- Content must match **exactly** (including whitespace, punctuation)
- If `max_turns` reached without matching submissions, session becomes `incomplete`

**Example**:

**Agent 1 submits first**:
```json
// Input
{
  "content": "# Final Design\n\nArchitecture: Microservices...",
  "approved": true
}

// Output
{
  "status": "pending",
  "message": "Deliverable submitted. Waiting for other agent to submit matching deliverable."
}
```

**Agent 2 submits matching deliverable**:
```json
// Input
{
  "content": "# Final Design\n\nArchitecture: Microservices...",
  "approved": true
}

// Output
{
  "status": "completed",
  "message": "Session completed! Both agents submitted matching deliverables."
}
```

**Agent 2 submits different deliverable**:
```json
// Input
{
  "content": "# Final Design\n\nArchitecture: Monolith...",  // Different!
  "approved": true
}

// Error Response
{
  "error": "CONTENT_MISMATCH",
  "message": "Deliverables don't match. Please coordinate with other agent to submit identical content."
}
```

**From Agent Perspective**:
```python
# Agent 1: Submit deliverable
response = mcp.invoke_tool("submit_deliverable", {
    "content": final_design,
    "approved": True
})

if response['status'] == "pending":
    # Send message to other agent
    mcp.invoke_tool("send_message", {
        "content": f"I've submitted the final design. Please review and submit if you approve:\n\n{final_design}"
    })
elif response['status'] == "completed":
    print("Session complete!")
```

```python
# Agent 2: Read deliverable from other agent and submit if approved
# (Note: Must coordinate to ensure exact match)
messages = mcp.invoke_tool("read_messages", {})
last_message = messages['messages'][-1]

if "submitted the final design" in last_message['content']:
    # Extract proposed deliverable from message
    deliverable = extract_deliverable(last_message['content'])

    # Review and approve
    if looks_good(deliverable):
        mcp.invoke_tool("submit_deliverable", {
            "content": deliverable,  # Exact copy
            "approved": True
        })
```

---

## Tool Usage Patterns

### Pattern 1: Iterative Feedback Loop

```
Agent 1 (Turn 1):
  - send_message("Here's my initial proposal...")
  - write_shared_context("# Proposal v1\n...")

Agent 2 (Turn 2):
  - read_messages()
  - read_shared_context()
  - send_message("Good start, but I have concerns about...")
  - update_memory("# Notes\n- Agent1 proposed X, I'm concerned about Y")

Agent 1 (Turn 3):
  - read_messages(since_turn=2)
  - read_memory()  # Recall strategy
  - send_message("Addressed your concerns by...")
  - write_shared_context("# Proposal v2\n...")

... repeat until agreement ...

Agent 1 (Turn N):
  - submit_deliverable(content=final_design, approved=True)

Agent 2 (Turn N+1):
  - read_messages()
  - submit_deliverable(content=final_design, approved=True)
  # Session completes!
```

### Pattern 2: Shared Context as Source of Truth

```
Agent 1 (Turn 1):
  - write_shared_context("""
      # Design Session
      ## Status: In Progress
      ## Current Proposal
      - Architecture: Microservices
      ## Open Questions
      - Caching strategy?
    """)

Agent 2 (Turn 2):
  - read_shared_context()
  - write_shared_context("""
      # Design Session
      ## Status: In Progress
      ## Current Proposal
      - Architecture: Microservices
      - Caching: Redis (answered)
      ## Open Questions
      - Auth mechanism?
    """)

Agent 1 (Turn 3):
  - read_shared_context()
  - write_shared_context("""
      # Design Session
      ## Status: Ready for Approval
      ## Final Design
      - Architecture: Microservices
      - Caching: Redis
      - Auth: JWT
    """)

... both agents submit ...
```

### Pattern 3: Memory for Strategy

```
Agent 2 (Reviewer, Turn 2):
  - update_memory("""
      # Review Strategy
      - Be critical but constructive
      - Focus on security and scalability

      ## Session Progress
      - Turn 2: Agent1 proposed microservices, looks good
      - Concern: No mention of security

      ## Next Turn TODO
      - Ask about auth strategy
      - Request error handling plan
    """)

Agent 2 (Turn 4):
  - read_memory()  # Recall TODOs
  - send_message("Before approving, I need clarity on...")
  - update_memory("""
      ... (previous content) ...

      - Turn 4: Asked about auth and errors

      ## Next Turn TODO
      - Review Agent1's response
      - Check if addresses security concerns
    """)
```

## Event Emission

All tool invocations emit events:

### ToolInvoked Event

Emitted for every tool call:

```json
{
  "type": "ToolInvoked",
  "timestamp": "2025-11-23T10:30:00Z",
  "payload": {
    "agent_id": "agent1",
    "tool_name": "send_message",
    "turn": 3,
    "duration_ms": 45
  }
}
```

**Subscribers**:
- Logger: Writes to `orchestrator.log`
- TUI: Updates tool call statistics

### Additional Events

Specific tools emit additional events:

**send_message** → `MessageSent`
```json
{
  "type": "MessageSent",
  "payload": {
    "from": "agent1",
    "to": "agent2",
    "message_id": "003",
    "turn": 3
  }
}
```

**write_shared_context, update_memory, submit_deliverable** → `FileUpdated`
```json
{
  "type": "FileUpdated",
  "payload": {
    "path": "/path/to/shared_context.md",
    "agent_id": "agent1",
    "turn": 3
  }
}
```

**submit_deliverable (when both submitted)** → `SessionCompleted`
```json
{
  "type": "SessionCompleted",
  "payload": {
    "status": "completed",
    "final_turn": 7,
    "deliverable_path": "/path/to/deliverables/"
  }
}
```

## File Organization

Tools write to standardized workspace structure:

```
~/.local/share/collab-cli/sessions/<session-id>/
├── session_state.json       # Not accessed by tools
├── orchestrator.log          # Event log (not accessed by tools)
├── shared_context.md         # read/write_shared_context
├── messages/
│   ├── 001-agent1.md         # send_message, read_messages
│   ├── 002-agent2.md
│   └── 003-agent1.md
├── memory/
│   ├── agent1.md             # update/read_memory (agent1 only)
│   └── agent2.md             # update/read_memory (agent2 only)
└── deliverables/
    ├── agent1.md             # submit_deliverable (agent1)
    └── agent2.md             # submit_deliverable (agent2)
```

## Error Handling

### Validation Errors

**Cause**: Invalid input parameters.

**Example**:
```json
// send_message with empty content
{
  "error": "VALIDATION_ERROR",
  "message": "Content cannot be empty",
  "field": "content"
}
```

**Agent should**: Fix input and retry.

### IO Errors

**Cause**: File system operation failed.

**Example**:
```json
{
  "error": "IO_ERROR",
  "message": "Failed to write shared_context.md: disk full",
  "path": "/path/to/shared_context.md"
}
```

**Agent should**: Report to user, cannot proceed.

### Content Mismatch

**Cause**: Both agents submitted deliverables but content doesn't match.

**Example**:
```json
{
  "error": "CONTENT_MISMATCH",
  "message": "Deliverables don't match. Agent1 submitted 1234 bytes, Agent2 submitted 1256 bytes.",
  "agent1_deliverable": "/path/to/deliverables/agent1.md",
  "agent2_deliverable": "/path/to/deliverables/agent2.md"
}
```

**Agent should**: Read messages, coordinate with other agent, submit matching content.

## Security Considerations

1. **No Path Traversal**: Tool implementations prevent `../` in file paths
2. **UTF-8 Validation**: All content validated as UTF-8
3. **File Permissions**: Files created with 0600 (user-only read/write)
4. **No Code Execution**: Tools do not execute code or scripts
5. **Isolated Workspaces**: Each session has isolated directory

## Performance

| Tool | Target Latency | Notes |
|------|----------------|-------|
| send_message | <50ms | File write + event emission |
| read_messages | <100ms | Scans directory, parses YAML |
| write_shared_context | <50ms | Atomic write |
| read_shared_context | <10ms | Simple file read |
| update_memory | <50ms | Atomic write |
| read_memory | <10ms | Simple file read |
| submit_deliverable | <100ms | File write + comparison + event |

## Best Practices for Agents

### 1. Message Etiquette

**Do**:
- Be clear and concise
- Use markdown formatting
- Ask specific questions
- Reference previous messages

**Don't**:
- Send empty messages
- Repeat previous messages verbatim
- Send multiple messages per turn (consolidate)

### 2. Shared Context Hygiene

**Do**:
- Read before writing (to avoid clobbering)
- Use structured format (headers, lists)
- Update incrementally
- Document decisions with rationale

**Don't**:
- Overwrite without reading
- Use unstructured text
- Delete information without agreement

### 3. Memory Management

**Do**:
- Track your strategy and goals
- Note important context from other agent
- Plan next turn's actions
- Review memory at turn start

**Don't**:
- Store duplicate information (use shared_context)
- Forget to update memory (lose context)

### 4. Deliverable Coordination

**Do**:
- Propose deliverable in message first
- Get explicit approval from other agent
- Submit exact copy (byte-for-byte)
- Verify content before submitting

**Don't**:
- Submit without coordination
- Assume agreement
- Submit different content

## Troubleshooting

### "Deliverables don't match"

**Problem**: Both agents submitted but content differs.

**Solution**:
1. Read messages to see proposed deliverable
2. Extract exact content (watch for whitespace)
3. Submit identical copy

**Example**:
```python
# Agent 2: Read Agent 1's proposed deliverable
messages = mcp.invoke_tool("read_messages", {})
for msg in messages['messages']:
    if "PROPOSED DELIVERABLE" in msg['content']:
        # Extract content between markers
        deliverable = extract_between(msg['content'], "```", "```")

        # Submit exact copy
        mcp.invoke_tool("submit_deliverable", {
            "content": deliverable,
            "approved": True
        })
```

### "File may not exist yet"

**Problem**: read_shared_context or read_memory returns IO_ERROR.

**Solution**: File doesn't exist yet (first turn), create it with write tool.

```python
try:
    response = mcp.invoke_tool("read_shared_context", {})
    context = response['content']
except IOError:
    # File doesn't exist, create initial content
    mcp.invoke_tool("write_shared_context", {
        "content": "# Design Session\n\n"
    })
```

### "Invalid YAML frontmatter"

**Problem**: read_messages returns PARSE_ERROR.

**Solution**: Message file corrupted. This shouldn't happen (atomic writes), but if it does, notify user.

## Related Documentation

- [Architecture Overview](architecture.md) - MCP server implementation details
- [Task Templates](task-templates.md) - How agents are configured
- [CLI Commands](../README.md#commands) - User commands for session management
