# Known Issues

## SDK MCP Server Tool Routing Issue

**Status**: Blocked - SDK Issue
**Date**: 2025-11-24
**Component**: MCP Collaboration Tools

### Problem

The Claude Agent SDK's `CreateSdkMcpServer()` function creates an MCP server but does not route tool calls to the handler functions.

**Symptoms**:
- MCP server is created successfully with all 7 collaboration tools
- Tools are registered and added to AllowedTools list
- Agent attempts to call tools (e.g., `mcp__collaboration__read_messages`)
- Tool handler functions are NEVER executed (confirmed with debug logging at handler entry points)
- Agents fall back to using built-in tools like `Read` and `Bash` to manually access files

**What Works**:
- ✓ MCP server creation (`claude.CreateSdkMcpServer()`)
- ✓ Tool registration with proper names
- ✓ SDK client configuration with MCP server
- ✓ Agent querying and tool invocation
- ✓ Workspace directory setup
- ✓ File structure for messages, memory, shared context

**What Doesn't Work**:
- ✗ Tool handler execution - handlers are never called
- ✗ MCP tool functionality - tools appear to fail silently

### Investigation Summary

**Confirmed Working in Ruby SDK**: The same pattern works correctly in the Ruby implementation (`multi_agent_mvp`):
```ruby
@mcp_server = ClaudeAgentSDK.create_sdk_mcp_server(
  name: "collaboration",
  version: '1.0.0',
  tools: @communication_tools
)
```

**Go Implementation** (not working):
```go
mcpServer := claude.CreateSdkMcpServer(
    "collaboration",
    "1.0.0",
    mcpTools,
)

opts := &claude.Options{
    // ...
    McpServers: map[string]claude.McpServerConfig{
        "collaboration": mcpServer,
    },
    AllowedTools: allowedTools,
    // ...
}
```

### Attempted Solutions

1. ✗ Adjusted workspace directory paths
2. ✗ Updated tool naming (with/without prefixes)
3. ✗ Modified AllowedTools configuration
4. ✗ Added SystemPrompt configuration
5. ✗ Updated agent query to include full tool names
6. ✗ Removed AllowedTools restrictions
7. ✗ Added comprehensive debug logging

### Files Involved

- `internal/mcp/tools.go` - Tool implementations with handlers
- `internal/agent/agent.go` - SDK MCP server creation and configuration
- `internal/orchestrator/orchestrator.go` - Tool distribution to agents
- `internal/agent/manager.go` - Tool invocation monitoring

### Next Steps

**Option 1**: Contact SDK Maintainers
- File issue with `claude-agent-sdk-go` repository
- Share reproduction case and comparison with working Ruby implementation

**Option 2**: Alternative Approach
- Implement stdio-based MCP server instead of SDK MCP server
- This would run as a separate process but might have better tool routing

**Option 3**: Workaround
- Use built-in file tools (Read, Write, Edit, Bash) for now
- Agents can still collaborate by directly reading/writing message files
- Less elegant but functional

### Impact

**Current State**: Multi-agent collaboration is partially functional
- Agents can execute turns
- Agents can use built-in tools to read/write files manually
- Core orchestration loop works correctly

**Blocked Features**:
- Clean MCP-based inter-agent messaging
- Tool-based memory management
- Structured deliverable submission

### Code References

- MCP tool creation: `internal/mcp/tools.go:99-138` (read_messages example)
- SDK server setup: `internal/agent/agent.go:42-84`
- Tool invocation: `internal/agent/manager.go:125-175`
