# Manual Testing Plan - Shiki CLI

**Version**: 6c1455e
**Build Date**: 2025-11-24
**Platform**: darwin/arm64

## Prerequisites

Before starting manual testing, ensure:

1. **Executable Built**: `./shiki` exists in project root (completed)
2. **API Key Available**: Set `ANTHROPIC_API_KEY` environment variable
   ```bash
   export ANTHROPIC_API_KEY="your-api-key-here"
   ```
3. **Clean Workspace**: Remove any previous test sessions
   ```bash
   rm -rf ~/.local/share/shiki-cli/sessions/*
   ```

## Testing Strategy

This plan covers all 8 user stories (P1-P3 priorities) from the spec with manual verification steps. Each test scenario includes:
- Setup instructions
- Expected behavior
- Success criteria
- Commands to run

---

## Test Suite 1: Core Functionality (P1)

### Test 1.1: Start Agent Collaboration with TUI (User Story 1)

**Priority**: P1 - Critical
**Acceptance Scenario**: Start collaboration, monitor in TUI, verify completion with deliverable

**Setup**:
```bash
cd /Users/darinhaener/code/collab/.worktrees/001-multi-agent-collaboration
export ANTHROPIC_API_KEY="your-api-key"
```

**Steps**:
1. Start collaboration with TUI:
   ```bash
   ./shiki run examples/simple-agreement.md --watch
   ```

2. Observe TUI behavior:
   - [ ] TUI launches within 10 seconds
   - [ ] Session header shows session ID, task name "Simple Agreement Task"
   - [ ] Turn history pane (left) shows agent activity
   - [ ] File view pane (right) shows current file content
   - [ ] Status bar shows current agent and progress

3. Monitor real-time updates:
   - [ ] Turn history updates when each agent completes a turn
   - [ ] Token usage and cost displayed per turn
   - [ ] File content updates when agents write messages or update context

4. Verify completion:
   - [ ] Session completes when both agents approve
   - [ ] Deliverable file created in session directory
   - [ ] TUI shows "Completed" status
   - [ ] Final session summary displayed

5. Navigate TUI (before completion):
   - [ ] Press Tab to switch between panes
   - [ ] Use arrow keys to navigate turn history
   - [ ] Press 'm' to view messages file
   - [ ] Press 'c' to view shared context
   - [ ] Press 'd' to view deliverable (after completion)
   - [ ] Press 'q' to exit TUI (session should continue)

**Expected Session Artifacts**:
```bash
ls -la ~/.local/share/shiki-cli/sessions/[session-id]/
```
Should contain:
- `task_template.md` - Original task
- `messages/` - Directory with agent messages
- `shared_context.md` - Shared knowledge
- `agent_1_memory.md` - Proposer's memory
- `agent_2_memory.md` - Approver's memory
- `deliverable.md` - Final agreed color choice
- `session_state.json` - Session metadata
- `orchestrator.log` - Structured logs

**Success Criteria**:
- TUI launches and renders correctly
- Real-time updates visible within 100ms of agent actions
- Session completes with deliverable.md containing color choice
- All artifacts present and human-readable

---

### Test 1.2: Headless Collaboration (User Story 2)

**Priority**: P1 - Critical
**Acceptance Scenario**: Run without TUI, verify completion via stdout and files

**Steps**:
1. Run collaboration without --watch:
   ```bash
   ./shiki run examples/simple-agreement.md
   ```

2. Observe headless output:
   - [ ] Turn-by-turn progress printed to stdout
   - [ ] Each turn shows: agent name, turn number, duration
   - [ ] Tool invocations logged to stdout
   - [ ] Token usage and cost displayed per turn

3. Verify completion:
   - [ ] Final deliverable path printed to stdout
   - [ ] Exit code is 0 (check: `echo $?`)
   - [ ] Session marked as "completed"

4. Check session directory:
   ```bash
   ls -la ~/.local/share/shiki-cli/sessions/[session-id]/
   ```
   - [ ] All expected files present
   - [ ] deliverable.md contains final agreement

**Success Criteria**:
- Headless mode completes without user interaction
- Exit code 0 on success
- Deliverable path printed to stdout
- All artifacts correctly saved

---

### Test 1.3: Pause and Resume (User Story 3)

**Priority**: P1 - Critical
**Acceptance Scenario**: Pause mid-execution, resume from exact state

**Steps**:
1. Start collaboration with TUI:
   ```bash
   ./shiki run examples/code-review.md --watch
   ```

2. Pause mid-execution:
   - [ ] Wait for 2-3 turns to complete
   - [ ] Press Ctrl+C to pause
   - [ ] Verify graceful shutdown message
   - [ ] Note the session ID from output

3. Verify paused state:
   ```bash
   ./shiki list sessions
   ```
   - [ ] Session status shows "paused"
   - [ ] Turn count matches when paused
   - [ ] Timestamp shows last activity

4. Inspect session state:
   ```bash
   ./shiki show [session-id]
   ```
   - [ ] Shows summary with status "paused"
   - [ ] Turn count, duration, and cost displayed
   - [ ] All artifacts accessible

5. Resume session:
   ```bash
   ./shiki resume [session-id] --watch
   ```
   - [ ] TUI shows full turn history from before pause
   - [ ] Collaboration continues from next turn
   - [ ] No duplicate turns or lost data

6. Verify completion:
   - [ ] Session completes successfully
   - [ ] Final deliverable contains complete work
   - [ ] Total turn count includes pre-pause + post-resume

**Success Criteria**:
- Pause preserves exact state in session_state.json
- Resume restores all context and continues correctly
- No data loss or state corruption
- Turn history shows continuous progression

---

## Test Suite 2: Session Management (P2)

### Test 2.1: Inspect Session Artifacts (User Story 4)

**Priority**: P2 - Important
**Acceptance Scenario**: Examine messages, context, memory, logs from completed session

**Prerequisite**: Complete Test 1.1 or 1.2 to have a finished session

**Steps**:
1. List all sessions:
   ```bash
   ./shiki list sessions
   ```
   - [ ] Table shows session ID, status, turns, duration, start time
   - [ ] Completed sessions marked with status "completed"

2. Show session summary:
   ```bash
   ./shiki show [session-id]
   ```
   - [ ] Displays status, duration, turn count
   - [ ] Shows token usage (total and per-agent)
   - [ ] Shows total cost
   - [ ] Lists artifact file paths

3. View messages:
   ```bash
   ./shiki show [session-id] --messages
   ```
   - [ ] All agent messages displayed chronologically
   - [ ] Each message shows: timestamp, agent, turn number
   - [ ] Message content is readable markdown

4. View shared context:
   ```bash
   ./shiki show [session-id] --context
   ```
   - [ ] Shared knowledge accumulated during collaboration
   - [ ] Shows collaborative decisions and agreements

5. View agent memory:
   ```bash
   ./shiki show [session-id] --memory agent_1
   ```
   - [ ] Agent-specific private state displayed
   - [ ] Repeat for agent_2

6. View deliverable:
   ```bash
   ./shiki show [session-id] --deliverable
   ```
   - [ ] Final deliverable content displayed
   - [ ] Shows approval metadata

7. View logs:
   ```bash
   cat ~/.local/share/shiki-cli/sessions/[session-id]/orchestrator.log
   ```
   - [ ] JSON-formatted log entries
   - [ ] Events include: TurnStarted, TurnCompleted, MessageSent, etc.
   - [ ] No sensitive data (API keys) in logs

**Success Criteria**:
- All artifacts accessible via show command
- Artifacts are human-readable markdown
- No data corruption or missing files

---

### Test 2.2: List and Filter Sessions (User Story 5)

**Priority**: P2 - Important
**Acceptance Scenario**: List sessions, filter by status, clean up old ones

**Prerequisite**: Have multiple sessions with different statuses (completed, paused, running)

**Steps**:
1. Create multiple sessions:
   ```bash
   # Run one to completion
   ./shiki run examples/simple-agreement.md

   # Start one and pause it
   ./shiki run examples/code-review.md --watch
   # (Pause with Ctrl+C after 2 turns)
   ```

2. List all sessions:
   ```bash
   ./shiki list sessions
   ```
   - [ ] Table displays all sessions
   - [ ] Columns: ID, Status, Turns, Duration, Started
   - [ ] Sessions sorted by start time (newest first)

3. Filter by status:
   ```bash
   ./shiki list sessions --status completed
   ```
   - [ ] Only completed sessions shown

   ```bash
   ./shiki list sessions --status paused
   ```
   - [ ] Only paused sessions shown

4. Clean old sessions:
   ```bash
   ./shiki clean --older-than 1m
   ```
   - [ ] Confirmation prompt appears
   - [ ] Lists sessions to be deleted
   - [ ] After confirmation, sessions removed
   - [ ] Verify with `./shiki list sessions`

**Success Criteria**:
- List command shows accurate session metadata
- Filtering works correctly by status
- Clean command safely removes old sessions with confirmation

---

## Test Suite 3: Validation & Tools (P2-P3)

### Test 3.1: Validate Task Templates (User Story 6)

**Priority**: P2 - Important
**Acceptance Scenario**: Validate valid/invalid templates, verify error messages

**Steps**:
1. Validate correct template:
   ```bash
   ./shiki validate examples/simple-agreement.md
   ```
   - [ ] Validation passes with checkmarks
   - [ ] All criteria validated: YAML frontmatter, required fields, syntax

2. Test missing required fields:
   Create `test-invalid.md`:
   ```yaml
   ---
   agent1_name: "Test"
   # Missing agent1_role, agent1_system_prompt, agent2_*, max_turns
   ---
   Task description
   ```

   ```bash
   ./shiki validate test-invalid.md
   ```
   - [ ] Validation fails with specific errors
   - [ ] Each missing field reported

3. Test invalid YAML:
   Create `test-bad-yaml.md`:
   ```yaml
   ---
   agent1_name: "Test
   # Unclosed quote
   ---
   Task
   ```

   ```bash
   ./shiki validate test-bad-yaml.md
   ```
   - [ ] Validation fails with YAML syntax error
   - [ ] Line number reported (if possible)

4. Test invalid field types:
   Create `test-bad-types.md`:
   ```yaml
   ---
   agent1_name: "Test"
   agent1_role: "Role"
   agent1_system_prompt: "Prompt"
   agent2_name: "Test2"
   agent2_role: "Role2"
   agent2_system_prompt: "Prompt2"
   max_turns: "not-a-number"
   ---
   Task
   ```

   ```bash
   ./shiki validate test-bad-types.md
   ```
   - [ ] Validation fails with type error for max_turns

**Success Criteria**:
- Valid templates pass with clear success message
- Invalid templates fail with specific, actionable errors
- All required fields validated before agent spawning

---

### Test 3.2: Create Task Templates (User Story 7)

**Priority**: P3 - Nice to have
**Acceptance Scenario**: Use init command to create valid template

**Steps**:
1. Initialize new template:
   ```bash
   ./shiki init template my-test-task
   ```
   - [ ] Interactive wizard prompts for:
     - Agent 1 name
     - Agent 1 role
     - Agent 1 system prompt
     - Agent 2 name
     - Agent 2 role
     - Agent 2 system prompt
     - Max turns
     - Task description

2. Verify generated template:
   ```bash
   cat my-test-task.md
   ```
   - [ ] Contains valid YAML frontmatter
   - [ ] All required fields present
   - [ ] Task description included

3. Validate generated template:
   ```bash
   ./shiki validate my-test-task.md
   ```
   - [ ] Validation passes

4. Run generated template:
   ```bash
   ./shiki run my-test-task.md
   ```
   - [ ] Session starts successfully
   - [ ] Agents execute as configured

**Success Criteria**:
- Init wizard creates valid, runnable templates
- Generated templates pass validation
- Templates work immediately without manual editing

---

### Test 3.3: Watch Running Session (User Story 8)

**Priority**: P3 - Nice to have
**Acceptance Scenario**: Attach TUI to headless session

**Steps**:
1. Start headless session:
   ```bash
   ./shiki run examples/architecture-design.md &
   # Note the session ID from output
   ```

2. Attach TUI to running session:
   ```bash
   ./shiki watch [session-id]
   ```
   - [ ] TUI launches showing current session
   - [ ] All past turns visible in history
   - [ ] Real-time updates for new turns

3. Detach TUI:
   - [ ] Press 'q' to exit TUI
   - [ ] Verify session continues running:
     ```bash
     ./shiki list sessions
     # Should show status "running"
     ```

4. Replay completed session:
   ```bash
   # Wait for session to complete or use a completed session ID
   ./shiki watch [session-id] --replay
   ```
   - [ ] TUI shows full turn history
   - [ ] Can navigate through all turns
   - [ ] Status shows "completed" (not "running")

**Success Criteria**:
- TUI can attach to running sessions without interruption
- Detaching TUI doesn't stop session
- Replay mode shows complete history for analysis

---

## Test Suite 4: Advanced Scenarios

### Test 4.1: Error Handling

**Scenario**: Missing API key

**Steps**:
```bash
unset ANTHROPIC_API_KEY
./shiki run examples/simple-agreement.md
```
- [ ] Fails fast with clear error message
- [ ] Error mentions ANTHROPIC_API_KEY requirement
- [ ] Exit code non-zero

**Scenario**: Invalid workspace permissions

**Steps**:
```bash
mkdir -p /tmp/readonly-workspace
chmod 000 /tmp/readonly-workspace
./shiki run examples/simple-agreement.md --workspace /tmp/readonly-workspace
```
- [ ] Fails with permission error
- [ ] Clear message about workspace not writable
- [ ] Exit code non-zero

**Scenario**: Max turns reached

**Steps**:
Create task with low max_turns and impossible agreement:
```yaml
---
agent1_name: "Yes"
agent1_role: "Always say yes"
agent1_system_prompt: "Only respond with yes"
agent2_name: "No"
agent2_role: "Always say no"
agent2_system_prompt: "Only respond with no"
max_turns: 3
---
Agree on a color.
```

```bash
./shiki run test-max-turns.md
```
- [ ] Session stops at max_turns
- [ ] Status marked as "incomplete"
- [ ] Clear message about max turns reached

---

### Test 4.2: Cost Tracking

**Scenario**: Verify accurate cost and token tracking

**Steps**:
1. Run a session:
   ```bash
   ./shiki run examples/code-review.md --watch
   ```

2. Monitor during execution:
   - [ ] TUI shows token count per turn
   - [ ] Cost calculated per agent
   - [ ] Total cost updated in header

3. After completion:
   ```bash
   ./shiki show [session-id]
   ```
   - [ ] Total tokens matches sum of all turns
   - [ ] Cost calculation accurate (within 1%)
   - [ ] Per-agent costs tracked separately

**Success Criteria**:
- Token usage tracked accurately
- Cost calculated correctly per pricing model
- Displayed in real-time and in session summary

---

### Test 4.3: File System Watching

**Scenario**: Verify real-time file updates in TUI

**Steps**:
1. Start session with TUI:
   ```bash
   ./shiki run examples/architecture-design.md --watch
   ```

2. In another terminal, modify shared context:
   ```bash
   echo "## External Note\nManually added content" >> \
     ~/.local/share/shiki-cli/sessions/[session-id]/shared_context.md
   ```

3. Observe TUI:
   - [ ] File view updates within 100ms
   - [ ] New content visible
   - [ ] FileUpdated event may appear in logs

**Success Criteria**:
- TUI updates reflect file changes in real-time
- fsnotify detects modifications
- No crashes or errors from external file changes

---

### Test 4.4: Workspace Structure

**Scenario**: Verify custom workspace structure from template

**Steps**:
1. Run code-review template (has workspace_structure):
   ```bash
   ./shiki run examples/code-review.md
   ```

2. Check session directory:
   ```bash
   ls -la ~/.local/share/shiki-cli/sessions/[session-id]/
   ```
   - [ ] `src/` directory created
   - [ ] `tests/` directory created
   - [ ] `README.md` file created with specified content

**Success Criteria**:
- Workspace structure from template created before agents start
- Files and directories accessible to agents
- Initial file content matches template specification

---

## Test Suite 5: Integration & Performance

### Test 5.1: End-to-End Workflow

**Scenario**: Complete workflow from template creation to deliverable

**Steps**:
1. Create template:
   ```bash
   ./shiki init template e2e-test
   # Fill in wizard prompts
   ```

2. Validate template:
   ```bash
   ./shiki validate e2e-test.md
   ```

3. Run with TUI:
   ```bash
   ./shiki run e2e-test.md --watch
   ```

4. Pause mid-execution:
   - [ ] Ctrl+C to pause

5. List sessions:
   ```bash
   ./shiki list sessions
   ```

6. Resume:
   ```bash
   ./shiki resume [session-id] --watch
   ```

7. Complete and inspect:
   ```bash
   ./shiki show [session-id] --deliverable
   ```

8. Clean up:
   ```bash
   ./shiki clean --older-than 1m
   ```

**Success Criteria**:
- Complete workflow executes without errors
- All commands work as expected
- Data consistency maintained throughout

---

### Test 5.2: Performance Benchmarks

**Scenario**: Verify performance meets success criteria

**Metrics to Measure**:

1. **Startup Time** (SC-001: < 10 seconds):
   ```bash
   time ./shiki run examples/simple-agreement.md --watch
   # Measure time until TUI renders
   ```
   - [ ] TUI appears in < 10 seconds

2. **File Update Latency** (SC-004: < 100ms):
   - Manually modify file during session
   - Observe TUI update time
   - [ ] Updates appear within 100ms

3. **State Restoration** (SC-002: zero data loss):
   - Pause session
   - Compare session_state.json turn count
   - Resume and verify continuation
   - [ ] Exact state restored

**Success Criteria**:
- All performance targets met
- No degradation under normal load
- Responsive UI even with long turn history

---

## Test Suite 6: Regression Testing

### Test 6.1: Example Templates

**Scenario**: Verify all included examples work

**Steps**:
1. Simple agreement:
   ```bash
   ./shiki run examples/simple-agreement.md
   ```
   - [ ] Completes successfully
   - [ ] Deliverable contains color choice

2. Code review:
   ```bash
   ./shiki run examples/code-review.md
   ```
   - [ ] Completes successfully
   - [ ] Deliverable contains code with tests

3. Architecture design:
   ```bash
   ./shiki run examples/architecture-design.md
   ```
   - [ ] Completes successfully
   - [ ] Deliverable contains architecture document

**Success Criteria**:
- All examples execute without errors
- Deliverables meet template requirements
- No crashes or hangs

---

## Testing Checklist Summary

### P1 - Critical (Must Pass)
- [ ] Test 1.1: Start with TUI
- [ ] Test 1.2: Headless execution
- [ ] Test 1.3: Pause and resume

### P2 - Important (Should Pass)
- [ ] Test 2.1: Inspect artifacts
- [ ] Test 2.2: List and filter sessions
- [ ] Test 3.1: Validate templates

### P3 - Nice to Have (Good to Pass)
- [ ] Test 3.2: Create templates
- [ ] Test 3.3: Watch running sessions

### Advanced (Robustness)
- [ ] Test 4.1: Error handling
- [ ] Test 4.2: Cost tracking
- [ ] Test 4.3: File system watching
- [ ] Test 4.4: Workspace structure

### End-to-End
- [ ] Test 5.1: Complete workflow
- [ ] Test 5.2: Performance benchmarks
- [ ] Test 6.1: Example templates

---

## Bug Reporting Template

If you find issues during testing, document them using this format:

```markdown
**Bug ID**: BUG-[number]
**Severity**: Critical | High | Medium | Low
**Test Case**: [Test number and name]
**Steps to Reproduce**:
1.
2.
3.

**Expected Behavior**:

**Actual Behavior**:

**Environment**:
- Version: 6c1455e
- Platform: darwin/arm64
- Go version: go1.24.10

**Logs/Screenshots**:

**Workaround** (if any):
```

---

## Next Steps After Testing

1. **Document Results**: Record pass/fail for each test case
2. **File Issues**: Create GitHub issues for any bugs found
3. **Update Docs**: Add findings to README.md or docs/
4. **Performance Tuning**: If benchmarks don't meet targets, profile and optimize
5. **User Acceptance**: Get feedback from real users on usability

## Quick Start for Testing

For a rapid smoke test of core functionality:

```bash
# 1. Build
make clean && make build

# 2. Set API key
export ANTHROPIC_API_KEY="your-key"

# 3. Clean workspace
rm -rf ~/.local/share/shiki-cli/sessions/*

# 4. Run simple test with TUI
./shiki run examples/simple-agreement.md --watch

# 5. Verify completion
./shiki list sessions
```

Good luck with testing!
