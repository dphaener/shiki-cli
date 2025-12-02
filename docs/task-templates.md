# Task Template Guide

## Overview

Task templates define the collaboration parameters for a two-agent session. Templates use **YAML frontmatter** to specify agent configuration and **Markdown** to describe the task.

## Template Format

Templates are markdown files (`.md`) with YAML frontmatter:

```markdown
---
agent1_name: "Agent Name"
agent1_role: "Agent role description"
agent1_system_prompt: "System prompt for agent"
agent2_name: "Agent Name"
agent2_role: "Agent role description"
agent2_system_prompt: "System prompt for agent"
max_turns: 10
workspace_structure:
  - path: "file.md"
    content: "Initial content"
  - path: "dir/"
    type: "directory"
---

# Task Title

Task description in markdown...
```

## Required Fields

All templates must include these fields in the YAML frontmatter:

### Agent 1 Configuration

- **`agent1_name`** (string): Display name for the first agent
  - Example: `"Architect"`, `"Proposer"`, `"Designer"`
  - Used in TUI and logs

- **`agent1_role`** (string): Brief role description
  - Example: `"Design systems"`, `"Propose solutions"`
  - Helps users understand agent purpose

- **`agent1_system_prompt`** (string): System prompt sent to Claude
  - Defines agent's behavior and constraints
  - Can be multi-line YAML string
  - Example:
    ```yaml
    agent1_system_prompt: |
      You are an experienced system architect.
      Design scalable, maintainable solutions.
      Focus on trade-offs and constraints.
    ```

### Agent 2 Configuration

- **`agent2_name`** (string): Display name for the second agent

- **`agent2_role`** (string): Brief role description

- **`agent2_system_prompt`** (string): System prompt for second agent

### Execution Parameters

- **`max_turns`** (integer): Maximum number of turns before session ends
  - Must be > 0
  - Typical values: 4-20
  - Session can end early if agents submit matching deliverables
  - If max_turns reached without agreement, session status becomes `incomplete`

## Optional Fields

### Workspace Structure

**`workspace_structure`** (array): Define initial files and directories in the workspace.

If omitted, workspace starts empty except for standard files:
- `session_state.json` (session metadata)
- `orchestrator.log` (event log)
- `shared_context.md` (empty by default)
- `messages/` (directory for agent messages)
- `memory/` (directory for agent memory files)
- `deliverables/` (directory for submitted deliverables)

To pre-populate workspace with custom files:

```yaml
workspace_structure:
  - path: "design.md"
    content: |
      # System Design

      ## Requirements
      -

  - path: "notes/"
    type: "directory"

  - path: "api/spec.yaml"
    content: |
      openapi: 3.0.0
      info:
        title: API
```

**Field Specifications**:
- `path` (string, required): Relative path within workspace
  - For files: `"file.md"`, `"subdir/file.txt"`
  - For directories: Must end with `/` (e.g., `"notes/"`)
  - Security: Cannot use `../` (path traversal prevented)

- `type` (string, optional): `"file"` (default) or `"directory"`
  - Explicit for directories: `type: "directory"`
  - Files don't need to specify type

- `content` (string, optional): Initial file content
  - Only for files (ignored for directories)
  - Can be multi-line YAML string using `|` or `>`
  - If omitted, file created empty

## Markdown Body

The markdown body (after frontmatter) describes the task:

```markdown
---
# frontmatter here
---

# Task Title

Task description with context, requirements, and deliverable expectations.

## Background
Context about why this task exists...

## Requirements
- Requirement 1
- Requirement 2

## Deliverable
Describe what the agents should submit...
```

**Best Practices**:
- Clear task description
- Explicit requirements
- Define what success looks like
- Provide context (background, constraints)
- Specify deliverable format

## Template Validation

Use `shiki validate` to check template correctness:

```bash
shiki validate task.md
```

**Validation Checks**:
- ✓ Valid YAML frontmatter syntax
- ✓ All required fields present
- ✓ Field types correct (string, integer, boolean)
- ✓ `max_turns` > 0
- ✓ No path traversal in `workspace_structure`
- ✓ UTF-8 encoding

**Example Output**:
```
✓ Template is valid
  - agent1: Architect (Design systems)
  - agent2: Reviewer (Review and critique)
  - max_turns: 10
  - workspace: 3 files, 1 directory
```

**Validation Errors**:
```
✗ Template validation failed:
  Line 5: agent1_system_prompt is required but missing
  Line 8: max_turns must be a positive integer, got: "ten"
  Line 12: workspace_structure[0].path contains path traversal: ../etc/passwd
```

## Example Templates

### 1. Simple Agreement

Use case: Quick consensus on a simple decision.

```markdown
---
agent1_name: "Proposer"
agent1_role: "Propose solutions"
agent1_system_prompt: "You propose simple solutions to problems. Keep responses concise."
agent2_name: "Approver"
agent2_role: "Review and approve"
agent2_system_prompt: "You review proposals and approve if they meet requirements. Be critical but fair."
max_turns: 4
---

# Simple Agreement Task

Work together to agree on the best color for a product logo.

Requirements:
- Must be a primary or secondary color
- Must have good contrast on white background
- Must convey professionalism

Submit your agreed color choice as the deliverable.
```

**Characteristics**:
- Short max_turns (4)
- Simple, focused task
- No workspace structure (defaults)
- Clear success criteria

### 2. Code Review

Use case: Architect designs API, reviewer critiques and approves.

```markdown
---
agent1_name: "Developer"
agent1_role: "Write code"
agent1_system_prompt: |
  You write clean, well-tested code.
  Follow best practices for the language.
  Document public APIs.
agent2_name: "Reviewer"
agent2_role: "Review code quality"
agent2_system_prompt: |
  You review code for correctness, clarity, and maintainability.
  Check for bugs, security issues, and style violations.
  Suggest improvements but be constructive.
max_turns: 8
workspace_structure:
  - path: "code/"
    type: "directory"
  - path: "tests/"
    type: "directory"
  - path: "README.md"
    content: |
      # Code Review Session

      ## Guidelines
      - Write tests before code
      - Document public APIs
      - Follow language style guide
---

# Code Review: User Authentication

Implement a simple user authentication system with the following:

## Requirements
- Email/password registration
- Login with JWT tokens
- Password hashing (bcrypt)
- Input validation

## Deliverable
Submit the agreed-upon implementation plan as the deliverable.
```

**Characteristics**:
- Moderate max_turns (8)
- Workspace structure (code/, tests/, README.md)
- Multi-line system prompts
- Pre-populated README with guidelines

### 3. Architecture Design

Use case: Complex design requiring multiple iterations.

```markdown
---
agent1_name: "Architect"
agent1_role: "Design systems"
agent1_system_prompt: |
  You are an experienced system architect.

  Your responsibilities:
  - Design scalable, maintainable systems
  - Consider trade-offs (performance vs simplicity, cost vs features)
  - Document decisions with rationale
  - Use industry-standard patterns

  Focus on practical, implementable designs.

agent2_name: "Critic"
agent2_role: "Challenge design decisions"
agent2_system_prompt: |
  You are a critical reviewer of system designs.

  Your responsibilities:
  - Identify weaknesses and edge cases
  - Challenge assumptions
  - Suggest alternatives
  - Ensure designs meet requirements

  Be constructive and specific in feedback.
  Push for clarity and completeness.

max_turns: 15
workspace_structure:
  - path: "design.md"
    content: |
      # System Design

      ## Overview
      (Architect fills this in)

      ## Components
      (Architect fills this in)

      ## Data Flow
      (Architect fills this in)

      ## Trade-offs
      (Architect fills this in)

  - path: "feedback.md"
    content: |
      # Design Feedback

      ## Questions
      (Critic fills this in)

      ## Concerns
      (Critic fills this in)

      ## Suggestions
      (Critic fills this in)

  - path: "decisions/"
    type: "directory"
---

# Architecture Design: Multi-Tenant SaaS Platform

Design a scalable architecture for a multi-tenant SaaS platform.

## Background
We're building a SaaS product that will serve 100+ organizations, each with their own users, data, and customizations.

## Requirements
- **Scalability**: Support 10,000+ organizations
- **Isolation**: Each organization's data must be isolated
- **Performance**: API response time < 200ms p95
- **Cost**: Optimize for operational costs
- **Compliance**: GDPR, SOC2 requirements

## Constraints
- Use cloud-native architecture (AWS/GCP)
- Budget: $50,000/month infrastructure
- Launch timeline: 6 months

## Deliverable
Submit a comprehensive architecture document covering:
1. High-level architecture diagram
2. Database design (schema + partitioning strategy)
3. Service architecture (microservices/monolith decision)
4. Scaling strategy
5. Security model
6. Cost estimates

Both agents must approve the final design.
```

**Characteristics**:
- Long max_turns (15) for complex design
- Rich workspace structure (design.md, feedback.md, decisions/)
- Detailed system prompts with responsibilities
- Pre-populated templates guide collaboration
- Complex requirements with constraints

## Agent Collaboration Patterns

### Pattern 1: Proposal-Approval

**Use Case**: One agent proposes, the other approves/rejects.

**Agent Roles**:
- Agent 1: Proposer (generates solutions)
- Agent 2: Approver (validates, approves, or requests changes)

**System Prompts**:
```yaml
agent1_system_prompt: "Propose solutions. Listen to feedback and iterate."
agent2_system_prompt: "Review proposals critically. Approve only when requirements met."
```

**Turn Flow**:
1. Proposer submits initial proposal
2. Approver reviews, provides feedback
3. Proposer refines based on feedback
4. Repeat until approval

**Example**: Code review, design approval, color selection

### Pattern 2: Debate-Consensus

**Use Case**: Both agents have equal authority, must reach agreement.

**Agent Roles**:
- Agent 1: Perspective A
- Agent 2: Perspective B

**System Prompts**:
```yaml
agent1_system_prompt: "Argue for your perspective. Listen to counterarguments. Seek common ground."
agent2_system_prompt: "Argue for your perspective. Challenge assumptions. Find compromise."
```

**Turn Flow**:
1. Both agents present their positions
2. Agents debate trade-offs
3. Agents explore middle ground
4. Reach consensus

**Example**: Architecture decisions, naming conventions, priority ranking

### Pattern 3: Builder-Validator

**Use Case**: One agent builds, the other validates incrementally.

**Agent Roles**:
- Agent 1: Builder (creates artifacts)
- Agent 2: Validator (checks correctness)

**System Prompts**:
```yaml
agent1_system_prompt: "Build incrementally. Test your work. Document decisions."
agent2_system_prompt: "Validate each increment. Run tests. Verify requirements."
```

**Turn Flow**:
1. Builder creates artifact
2. Validator checks correctness
3. Builder fixes issues
4. Repeat until validation passes

**Example**: Code implementation, test writing, documentation

### Pattern 4: Researcher-Synthesizer

**Use Case**: One agent researches, the other synthesizes findings.

**Agent Roles**:
- Agent 1: Researcher (gather information)
- Agent 2: Synthesizer (combine into coherent whole)

**System Prompts**:
```yaml
agent1_system_prompt: "Research thoroughly. Gather evidence. Document sources."
agent2_system_prompt: "Synthesize research into actionable insights. Find patterns."
```

**Turn Flow**:
1. Researcher gathers information
2. Synthesizer identifies gaps
3. Researcher fills gaps
4. Synthesizer creates final synthesis

**Example**: Competitive analysis, literature review, requirements gathering

## Advanced Features

### Multi-line YAML Strings

Use `|` (literal) or `>` (folded) for multi-line content:

```yaml
# Literal: preserves newlines
agent1_system_prompt: |
  You are an architect.

  Responsibilities:
  - Design systems
  - Document decisions

# Folded: collapses newlines to spaces
task_description: >
  This is a long description
  that will be collapsed
  into a single paragraph.
```

### Template Variables (Future)

Templates currently don't support variables, but you can use descriptive placeholders:

```yaml
workspace_structure:
  - path: "api-spec.yaml"
    content: |
      # Replace <service-name> with actual name
      service: <service-name>
```

### Conditional Logic (Not Supported)

Templates don't support conditional logic. Create separate templates for different scenarios:

```
templates/
├── code-review-backend.md
├── code-review-frontend.md
└── code-review-fullstack.md
```

## Best Practices

### 1. System Prompt Guidelines

**Do**:
- Be specific about responsibilities
- Provide examples of desired behavior
- Set clear boundaries
- Define success criteria

**Don't**:
- Be vague ("be helpful")
- Contradict between agents
- Overload with too many instructions
- Include task details (use markdown body)

### 2. Workspace Structure Guidelines

**Do**:
- Pre-populate with templates for complex tasks
- Create directory structure for organization
- Provide scaffolding to guide collaboration
- Use clear, descriptive file names

**Don't**:
- Create too many files (cognitive overload)
- Duplicate information
- Pre-fill complete solutions (defeats collaboration)
- Use deep nesting (keep flat when possible)

### 3. max_turns Guidelines

**Too Few** (< 4):
- Risk: Not enough iterations to reach quality
- Use for: Simple decisions, quick approvals

**Ideal** (4-10):
- Sweet spot for most collaborations
- Enough time to iterate without drag

**Too Many** (> 15):
- Risk: Agents may lose focus, increase cost
- Use for: Complex designs, multi-phase tasks

**Rule of Thumb**:
- Simple agreement: 4-6 turns
- Code review: 6-10 turns
- Architecture design: 10-15 turns
- Research synthesis: 10-20 turns

### 4. Task Description Guidelines

**Structure**:
```markdown
# Task Title (clear, specific)

## Background (why does this matter?)

## Requirements (what must be true?)

## Constraints (what limits exist?)

## Deliverable (what should agents submit?)
```

**Writing Tips**:
- Start with context (why this task exists)
- Make requirements testable
- Be explicit about deliverable format
- Provide examples when possible
- Define "done" clearly

### 5. Testing Templates

Before running real collaboration:

1. **Validate syntax**: `shiki validate task.md`
2. **Review workspace**: Check scaffolded files make sense
3. **Read prompts aloud**: Do they sound natural?
4. **Simulate turns**: Walk through expected flow
5. **Test with real agents**: Run a trial session

## Common Mistakes

### Mistake 1: Vague System Prompts

**Bad**:
```yaml
agent1_system_prompt: "You are helpful."
agent2_system_prompt: "You review things."
```

**Good**:
```yaml
agent1_system_prompt: |
  You propose API designs.
  Focus on REST principles and simplicity.
  Document endpoints with examples.

agent2_system_prompt: |
  You review API designs for consistency and completeness.
  Check for missing error cases.
  Ensure documentation is clear.
```

### Mistake 2: Conflicting Agent Goals

**Bad**:
```yaml
agent1_system_prompt: "Design for maximum performance at any cost."
agent2_system_prompt: "Ensure simplicity above all else."
```

Result: Endless debate, no convergence.

**Good**:
```yaml
agent1_system_prompt: "Design for performance while maintaining reasonable complexity."
agent2_system_prompt: "Review for balance between performance and maintainability."
```

### Mistake 3: Unclear Deliverable

**Bad**:
```markdown
# Task
Design a system.

Submit your design.
```

**Good**:
```markdown
# Task
Design a user authentication system.

## Deliverable Format
Submit a markdown document with:
1. Architecture diagram (ASCII art or description)
2. Component responsibilities
3. Data flow for login/registration
4. Security considerations
5. Estimated cost breakdown
```

### Mistake 4: Workspace Overload

**Bad**:
```yaml
workspace_structure:
  - path: "design/architecture.md"
  - path: "design/diagrams.md"
  - path: "design/decisions.md"
  - path: "feedback/round1.md"
  - path: "feedback/round2.md"
  - path: "notes/agent1.md"
  - path: "notes/agent2.md"
  # ... 20 more files
```

**Good**:
```yaml
workspace_structure:
  - path: "design.md"
    content: "# Design\n\n"
  - path: "feedback.md"
    content: "# Feedback\n\n"
  - path: "notes/"
    type: "directory"
```

## Creating Templates

### Method 1: Use `shiki init`

Interactive wizard creates template from questions:

```bash
collab init

? Template name: architecture-design
? Agent 1 name: Architect
? Agent 1 role: Design systems
? Agent 2 name: Reviewer
? Agent 2 role: Review designs
? Max turns: 10
? Add workspace structure? Yes
  ? File path: design.md
  ? Initial content? Yes
  ...

✓ Template created: architecture-design.md
```

### Method 2: Copy Example

```bash
cp examples/code-review.md my-task.md
$EDITOR my-task.md
shiki validate my-task.md
```

### Method 3: Write from Scratch

```bash
cat > my-task.md << 'EOF'
---
agent1_name: "Agent1"
agent1_role: "Role description"
agent1_system_prompt: "System prompt..."
agent2_name: "Agent2"
agent2_role: "Role description"
agent2_system_prompt: "System prompt..."
max_turns: 10
---

# Task Title

Task description...
EOF

shiki validate my-task.md
```

## Template Library

See `examples/` directory for templates:

- `simple-agreement.md` - Quick consensus task
- `code-review.md` - Developer and reviewer collaboration
- `architecture-design.md` - Complex system design

**Community Templates**: [Coming Soon]

## Troubleshooting

### "YAML frontmatter invalid"

**Cause**: Syntax error in YAML.

**Fix**:
- Check for unmatched quotes
- Ensure proper indentation (2 spaces)
- Validate YAML at https://www.yamllint.com/

### "max_turns must be positive"

**Cause**: `max_turns: 0` or negative.

**Fix**: Set to positive integer (e.g., `max_turns: 10`)

### "Path traversal detected"

**Cause**: workspace_structure path contains `../`

**Fix**: Use relative paths without parent directory references:
```yaml
# Bad
workspace_structure:
  - path: "../etc/passwd"

# Good
workspace_structure:
  - path: "config/settings.json"
```

### "Required field missing"

**Cause**: Missing required YAML field.

**Fix**: Add all required fields:
- `agent1_name`, `agent1_role`, `agent1_system_prompt`
- `agent2_name`, `agent2_role`, `agent2_system_prompt`
- `max_turns`

## Reference

### Complete Template Schema

```yaml
# Required fields
agent1_name: string (non-empty)
agent1_role: string (non-empty)
agent1_system_prompt: string (non-empty)
agent2_name: string (non-empty)
agent2_role: string (non-empty)
agent2_system_prompt: string (non-empty)
max_turns: integer (> 0)

# Optional fields
workspace_structure:
  - path: string (required, no ../)
    type: "file" | "directory" (default: "file")
    content: string (optional, file only)
```

### File Locations

- **Templates**: `examples/` or user-defined location
- **Validation**: `shiki validate <path>`
- **Execution**: `shiki run <path>`

### Related Commands

- `shiki init` - Create new template interactively
- `shiki validate <template>` - Validate template
- `shiki run <template>` - Execute collaboration
- `shiki show <session-id> --template` - View session template
