# Collab - Sekkei Development Guidelines

Auto-generated from Sekkei initialization. Last updated: 2025-11-23

**Project**: Collab
**Mission**: software_development
**AI Assistant**: claude

## Sekkei Workflow

This project follows specification-driven development:

1. **/sekkei.specify** - Create feature specifications in `sekkei-specs/`
2. **/sekkei.plan** - Generate implementation plans with architecture decisions
3. **/sekkei.tasks** - Break down features into work packages and subtasks
4. **/sekkei.implement** - Execute tasks using generated prompts
5. **/sekkei.review** - Review completed work for quality
6. **/sekkei.accept** - Validate and merge features

### Workflow Details

**Specification Phase**:
- Define WHAT (functionality) and WHY (value) without HOW (implementation)
- Focus on user scenarios and acceptance criteria
- Document requirements, assumptions, and constraints
- Output: `sekkei-specs/###-feature/spec.md`

**Planning Phase**:
- Design architecture and technical approach
- Make technology and pattern decisions
- Document rationale with evidence
- Output: `plan.md`, `contracts/`, optional `research.md`

**Task Breakdown**:
- Decompose into independently testable work packages
- Create detailed prompts for each work package
- Define dependencies and execution order
- Output: `tasks.md` and `tasks/planned/WP##-*.md`

**Implementation**:
- Follow Kanban workflow: planned → doing → for_review → done
- Move work package prompts through lanes as you progress
- Commit after each lane transition
- Output: Working code, passing tests

## File Structure

```
Collab/
├── .sekkei/
│   ├── templates/           # Markdown templates for workflows
│   ├── scripts/             # Workflow automation scripts
│   │   ├── bash/            # Bash workflow scripts
│   │   └── powershell/      # PowerShell workflow scripts
│   └── memory/
│       └── constitution.md  # Project principles
├── sekkei-specs/            # Feature specifications
│   └── ###-feature/
│       ├── spec.md          # Feature specification
│       ├── plan.md          # Implementation plan
│       ├── tasks.md         # Work breakdown
│       ├── contracts/       # API contracts and interfaces
│       ├── research/        # Technical research (optional)
│       │   ├── evidence-log.csv
│       │   └── source-register.csv
│       └── tasks/           # Kanban lanes
│           ├── planned/     # Ready to start
│           ├── doing/       # In progress
│           ├── for_review/  # Awaiting review
│           └── done/        # Complete
├── .claude/commands/        # Claude Code slash commands
└── .worktrees/              # Git worktrees for parallel features
```

## Development Guidelines

### Software Development Standards

- **Test-Driven Development**: Write tests before implementation
- **Comprehensive Testing**: Unit, integration, and end-to-end tests
- **Documentation**: Document public APIs and complex logic
- **Semantic Versioning**: Follow semver for releases
- **Atomic Commits**: Keep commits focused and well-described
- **Code Review**: All work packages reviewed before merging

### Git Workflow

- **Feature Branches**: One branch per feature (e.g., `003-feature-name`)
- **Worktrees**: Use `.worktrees/` for parallel development
- **Commit Messages**: Clear, descriptive, with context
- **Lane Transitions**: Commit when moving work packages between lanes
- **Pull Requests**: Review before merging to main

### Task Management

Work packages follow this lifecycle:

1. **planned** - Work package prompt created, ready to start
2. **doing** - Implementation in progress
3. **for_review** - Implementation complete, awaiting validation
4. **done** - Reviewed and accepted

Move prompts between lanes by relocating the markdown file:
```bash
# Start work package
mv tasks/planned/WP01-name.md tasks/doing/

# Complete work package
mv tasks/doing/WP01-name.md tasks/for_review/

# Accept after review
mv tasks/for_review/WP01-name.md tasks/done/
```

## Common Commands

### Bash

```bash
# Create new feature
.sekkei/scripts/bash/setup-spec.sh "feature-name"

# Generate plan
.sekkei/scripts/bash/setup-plan.sh "###-feature-name"

# Break down tasks
.sekkei/scripts/bash/setup-tasks.sh "###-feature-name"

# Start implementation
.sekkei/scripts/bash/setup-implement.sh "###-feature-name"

# Update agent context
.sekkei/scripts/bash/update-agent-context.sh
```

### Git Commands

```bash
# List active feature worktrees
git worktree list

# Create new worktree for feature
git worktree add .worktrees/###-feature-name ###-feature-name

# Remove worktree after merging
git worktree remove .worktrees/###-feature-name
```

## Constitution

Project-specific principles and standards are defined in `.sekkei/memory/constitution.md`.

**Core Principles**:
- [Will be defined during project initialization or first feature]

**Architecture Standards**:
- [Will be established based on technology stack]

**Code Style**:
- [Language-specific conventions to be documented]

**Testing Requirements**:
- [Coverage goals and test types to be defined]

Use `/sekkei.constitution` to update these as the project evolves.

## Context Updates

This file should be updated when:
- New architectural patterns are introduced
- Project conventions change
- New workflows or commands are added
- Testing or quality standards are updated

Run `.sekkei/scripts/{bash|powershell}/update-agent-context.{sh|ps1}` to refresh this context.

---

<!-- MANUAL ADDITIONS START -->
<!-- Add project-specific context below this line -->

<!-- MANUAL ADDITIONS END -->
