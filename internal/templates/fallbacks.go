package templates

// Embedded fallback templates for system resilience
// These are minimal fallback templates that ensure the system works even if template files are missing

const (
	// System prompt fallbacks
	fallbackSpecifyPrompt = `# Specification Agent

You are a specification assistant that creates detailed feature specifications.

## Context
- **Feature**: {{.FriendlyName}}
- **Number**: {{.FeatureNumber}}
- **Slug**: {{.Slug}}
- **Phase**: {{.Phase}}

## Your Role
Create a comprehensive specification that defines what needs to be built, focusing on user value and business requirements rather than implementation details.

## Key Guidelines
- Write for non-technical stakeholders
- Focus on user scenarios and acceptance criteria
- Define measurable success criteria
- Avoid implementation details (technologies, frameworks, APIs)
- Ensure requirements are testable and unambiguous

## Output
Write your specification to the file: {{.SpecFile}}
The specification should include all sections: Overview, User Scenarios, Requirements, Success Criteria, and Constraints.

Begin your specification work.`

	fallbackPlanPrompt = `# Planning Agent

You are a planning assistant that creates detailed implementation plans.

## Context
- **Feature**: {{.FriendlyName}}
- **Specification**: {{.SpecFile}}
- **Plan File**: {{.PlanFile}}
- **Phase**: {{.Phase}}

## Your Role
Create a comprehensive implementation plan that defines how to build the feature, including technical decisions, phases, and architectural considerations.

## Key Guidelines
- Review the specification thoroughly
- Design implementation phases
- Make key technical decisions with rationale
- Consider testing and integration strategies
- Plan for maintainability and extensibility

## Output
Write your implementation plan to: {{.PlanFile}}
The plan should include phases, technical decisions, and implementation strategy.

Begin your planning work.`

	fallbackTasksPrompt = `# Tasks Agent

You are a tasks assistant that creates detailed implementation task breakdowns.

## Context
- **Feature**: {{.FriendlyName}}
- **Specification**: {{.SpecFile}}
- **Plan**: {{.PlanFile}}
- **Tasks File**: {{.TasksFile}}
- **Phase**: {{.Phase}}

## Your Role
Break down the implementation plan into specific, actionable tasks with clear work packages.

## Key Guidelines
- Review specification and plan thoroughly
- Create granular, actionable tasks
- Organize tasks into logical work packages
- Define dependencies between tasks
- Include testing and validation tasks

## Output
Write your task breakdown to: {{.TasksFile}}
The tasks should be specific, testable, and well-organized.

Begin your task breakdown work.`

	fallbackImplementPrompt = `# Implementation Agent

You are an implementation assistant that executes the tasks defined in the implementation plan.

## Context
- **Feature**: {{.FriendlyName}}
- **Specification**: {{.SpecFile}}
- **Plan**: {{.PlanFile}}
- **Tasks**: {{.TasksFile}}
- **Phase**: {{.Phase}}

## Your Role
Execute the implementation tasks defined in tasks.md, writing production-quality code that meets the specification requirements.

## Key Guidelines
- Follow the implementation plan carefully
- Write clean, maintainable code
- Include appropriate error handling
- Add tests for new functionality
- Document complex logic

Begin your implementation work.`

	fallbackBugPrompt = `# Bug Fix Agent

You are a bug fix assistant that diagnoses and resolves issues.

## Context
- **Bug ID**: {{.BugID}}
- **Title**: {{.Title}}
- **Description**: {{.Description}}
- **Phase**: {{.Phase}}

## Your Role
Diagnose and fix the reported bug with minimal impact on existing functionality.

## Key Guidelines
- Understand the bug thoroughly
- Identify root cause
- Implement minimal fix
- Add tests to prevent regression
- Verify fix resolves the issue

Begin your bug fix work.`

	fallbackCollaborationPrompt = `You are {{.AgentName}}, a {{.AgentRole}}.

This is turn {{.TurnNumber}} of {{.MaxTurns}}. You are collaborating with {{.PartnerName}}.

=== TASK ===
{{.Task}}

=== MESSAGES FROM YOUR PARTNER ===
{{.Messages}}

=== SHARED CONTEXT ===
{{.SharedContext}}

=== YOUR PRIVATE MEMORY ===
{{.Memory}}

=== INSTRUCTIONS ===
You have built-in tools (Read, Write, Edit, Glob, Grep, Bash) to work with files in your workspace.

**File-Based Communication Protocol:**

To send a message to your partner:
  Use Write tool to create: messages/{{.AgentID}}_turn_{{.TurnNumberPadded}}.md
  (Your partner will see it on their next turn)

To update shared context (both agents can see):
  Use Write tool to update: shared_context.md

To update your private memory:
  Use Write tool to update: memory/{{.AgentID}}_memory.md

To submit your final deliverable:
  Use Write tool to create: {{.AgentID}}_deliverable.md
  (Session completes when BOTH agents submit matching deliverables)

**On your turn:**
1. Review the task, messages, shared context, and your memory (shown above)
2. Perform your role's responsibilities
3. Send a message to your partner if needed (write to messages/ directory)
4. Update shared context if you have findings to share
5. Update your memory to track your progress
6. When the task is complete, submit your deliverable

Begin your turn.`

	// Output template fallbacks
	fallbackSpecTemplate = `# Feature Specification: {{.FriendlyName}}

**Feature Number**: {{.FeatureNumber}}
**Created**: {{.CreatedDate}}
**Status**: Draft

## Overview

[Brief description of the feature and its purpose]

## User Scenarios & Testing

### User Story 1 - [Primary Use Case]

As a [user type], I want [functionality] so that [benefit/value].

**Acceptance Scenarios**:
1. **Given** [precondition], **When** [action], **Then** [expected result]
2. **Given** [precondition], **When** [action], **Then** [expected result]

### User Story 2 - [Secondary Use Case]

[Additional user stories as needed]

## Requirements

### Functional Requirements

- **FR-001**: [Requirement description]
- **FR-002**: [Requirement description]

### Key Entities

- **[Entity Name]**: [Description of key data structure or concept]

## Success Criteria

### Measurable Outcomes

- **SC-001**: [Measurable success criterion]
- **SC-002**: [Measurable success criterion]

## Assumptions

- **AS-001**: [Assumption about the system or environment]

## Dependencies

- **DEP-001**: [External dependency or requirement]

## Out of Scope

- **OOS-001**: [What is explicitly not included]

## Constraints

- **CON-001**: [Technical or business constraint]`

	fallbackPlanTemplate = `# Implementation Plan: {{.FeatureName}}

**Created**: {{.CreatedDate}}
**Status**: Draft

## Summary

[1-2 paragraph overview of the implementation approach]

## Technical Context

### Current State

[Describe the existing codebase and relevant components]

### Proposed Solution

[High-level description of the technical approach]

## Implementation Phases

### Phase 1: [Phase Name]

**Goal**: [Phase objective]

**Tasks**:
1. [Task description]
2. [Task description]

### Phase 2: [Phase Name]

**Goal**: [Phase objective]

**Tasks**:
1. [Task description]
2. [Task description]

## Key Decisions

### Decision 1: [Title]

**Context**: [What prompted this decision]
**Chosen Approach**: [Selected option]
**Rationale**: [Why]

## Testing Strategy

### Unit Tests

[Testing approach]

### Integration Tests

[Integration testing approach]

## Success Metrics

- [ ] All requirements implemented
- [ ] Tests passing
- [ ] Documentation complete

## References

- [spec.md](./spec.md) - Feature specification`

	fallbackTasksTemplate = `# Implementation Tasks: {{.FeatureName}}

## Overview
[Brief description of the feature and main objectives]

## Work Packages

### WP01: [Work Package Name]
**Priority**: [P0/P1/P2]
**Goal**: [Brief description of work package goal]
**Dependencies**: [None/WP## dependencies]

#### Subtasks
- [ ] T001: [Task description]
- [ ] T002: [Task description]

#### Implementation Notes
[Additional context, technical notes, or considerations]

### WP02: [Work Package Name]
**Priority**: [P0/P1/P2]
**Goal**: [Brief description of work package goal]
**Dependencies**: [None/WP## dependencies]

#### Subtasks
- [ ] T003: [Task description]
- [ ] T004: [Task description]

## Success Criteria

- [ ] All work packages completed
- [ ] Tests passing
- [ ] Manual testing completed`

	fallbackTaskProgressTemplate = `# Implementation Progress: {{.FriendlyName}}

**Feature Number**: {{.FeatureNumber}}
**Created**: {{.CurrentDate}}
**Status**: In Progress

## Overview
Feature: {{.FriendlyName}}
Implementation started: {{.CurrentDate}}

## Current Status
Implementation in progress...

## Work Packages Progress

### Work Package Status
- [ ] WP01: [Work Package Name]
- [ ] WP02: [Work Package Name]
- [ ] WP03: [Work Package Name]

## Recent Activity
- {{.CurrentTime}}: Implementation started

## Next Steps
- Complete current work package
- Run tests and validation
- Update progress tracking

## Notes
Implementation progress will be updated as tasks are completed.`

	fallbackRequirementsChecklist = `# Specification Quality Checklist: {{.FeatureName}}

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: {{.CreatedDate}}
**Feature**: [spec.md](../spec.md)

## Content Quality

- [ ] No implementation details (languages, frameworks, APIs)
- [ ] Focused on user value and business needs
- [ ] Written for non-technical stakeholders
- [ ] All mandatory sections completed

## Requirement Completeness

- [ ] No [NEEDS CLARIFICATION] markers remain
- [ ] Requirements are testable and unambiguous
- [ ] Success criteria are measurable
- [ ] All acceptance scenarios are defined
- [ ] Edge cases are identified
- [ ] Scope is clearly bounded

## Feature Readiness

- [ ] All functional requirements have clear acceptance criteria
- [ ] User scenarios cover primary flows
- [ ] Feature meets measurable outcomes defined in Success Criteria

## Next Steps

Once all items are checked:
- Ready for planning phase
- Consider running clarification if needed`
)

// getEmbeddedSystemPrompt returns fallback system prompt content
func getEmbeddedSystemPrompt(phase string) string {
	switch phase {
	case "specify":
		return fallbackSpecifyPrompt
	case "plan":
		return fallbackPlanPrompt
	case "tasks":
		return fallbackTasksPrompt
	case "implement":
		return fallbackImplementPrompt
	case "bug":
		return fallbackBugPrompt
	case "collaboration":
		return fallbackCollaborationPrompt
	default:
		return "# " + phase + " System Prompt\n\n[Embedded fallback for " + phase + " phase]"
	}
}

// getEmbeddedOutputTemplate returns fallback output template content
func getEmbeddedOutputTemplate(phase string) string {
	switch phase {
	case "spec":
		return fallbackSpecTemplate
	case "plan":
		return fallbackPlanTemplate
	case "tasks":
		return fallbackTasksTemplate
	case "task-progress":
		return fallbackTaskProgressTemplate
	default:
		return "# " + phase + " Template\n\n[Embedded fallback for " + phase + " output template]"
	}
}

// getEmbeddedChecklist returns fallback checklist content
func getEmbeddedChecklist(name string) string {
	switch name {
	case "requirements":
		return fallbackRequirementsChecklist
	default:
		return "# " + name + " Checklist\n\n[Embedded fallback for " + name + " checklist]"
	}
}