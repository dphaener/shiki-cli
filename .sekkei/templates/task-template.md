---
description: "Work package task list template"
---
*Path: [sekkei-specs/###-feature-name/tasks.md]*

# Work Packages: [Feature Name]

**Feature**: [Brief feature description]
**Inputs**: Design documents from `sekkei-specs/###-feature-name/`
**Prerequisites**: plan.md, spec.md, [other prerequisites]

**Tests**: [Testing approach - e.g., "Testing is explicitly required per spec success criteria"]

**Organization**: [X fine-grained subtasks rolled into Y work packages. Each work package is independently deliverable and testable.]

**Prompt Files**: Each work package references a matching prompt file in `sekkei-specs/###-feature-name/tasks/planned/`

## Path Conventions
- [Document any path conventions specific to this feature]

---

## Work Package WP01: [Package Name] (Priority: P0)

**Goal**: [What this package delivers]
**Independent Test**: [How to verify this package works in isolation]
**Prompt**: `tasks/planned/WP01-slug.md`

### Included Subtasks
- [ ] T001 [Subtask description]
- [ ] T002 [Subtask description]
- [ ] T003 [Subtask description]

### Implementation Notes
1. [Key implementation guidance]
2. [Important considerations]
3. [Dependencies or constraints]

### Parallel Opportunities
- [Which tasks can run in parallel]
- [Which tasks must be sequential]

### Dependencies
- [Dependencies on other work packages or external systems]

### Risks & Mitigations
- **Risk**: [Risk description]
- **Mitigation**: [How to mitigate]

---

## Work Package WP02: [Package Name] (Priority: P1)

[Same structure as WP01]

---

## Dependency & Execution Summary

**Critical Path Sequence**:
1. **WP01** ([Name]) → Foundation
2. **WP02** ([Name]) → Depends on WP01
3. **WP03** ([Name]) ∥ **WP04** ([Name]) → Can proceed in parallel after WP02
4. **WP05** ([Name]) → Depends on WP03 + WP04

**Parallelization Opportunities**:
- [Which work packages can run in parallel]
- [Which subtasks within packages can run in parallel]

**MVP Scope**:
- **Minimum**: [Minimal viable packages]
- **Recommended**: [Full recommended scope]

**Estimated Timeline**:
- WP01: [X-Y hours]
- WP02: [X-Y hours]
- **Total**: [Total estimate]

---

## Subtask Index (Reference)

| Subtask ID | Summary | Work Package | Priority | Parallel? |
|------------|---------|--------------|----------|-----------|
| T001 | [Description] | WP01 | P0 | No |
| T002 | [Description] | WP01 | P0 | Yes |
| T003 | [Description] | WP02 | P1 | Yes |

---

> This task breakdown enables complete implementation of [feature name]. Each work package is independently deliverable with clear success criteria. Prompt files in `tasks/planned/` provide detailed implementation guidance for each work package.
