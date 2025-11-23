---
work_package_id: WP04
title: "Template System"
priority: P0
status: planned
subtasks:
  - T020
  - T021
  - T022
  - T023
  - T024
dependencies:
  - WP01
  - WP02
lane: planned
history:
  - timestamp: "2025-11-23"
    action: created
    status: planned
---

# Work Package WP04: Template System

**Objective**: Parse and validate task templates with YAML frontmatter, markdown body, and workspace structure scaffolding support.

**Priority**: P0 (Required by WP07 orchestrator, WP08 CLI commands)

**Estimated Effort**: 4-6 hours

## Context

Task templates are the user-facing interface for defining agent collaborations. They combine YAML configuration (agent setup, limits) with markdown task descriptions. The parser must handle malformed YAML gracefully with helpful error messages, and the validator must catch all required field violations before expensive agent spawning.

**Template Structure** (per data-model.md:186-248):
```yaml
---
agent1_name: "Architect"
agent1_role: "System designer"
agent1_system_prompt: "You are..."
agent1_model: "claude-sonnet-4"

agent2_name: "Reviewer"
agent2_role: "Design critic"
agent2_system_prompt: "You are..."
agent2_model: "claude-sonnet-4"

max_turns: 10
turn_timeout_seconds: 300  # optional
turn_delay_ms: 0           # optional

workspace_structure:       # optional
  - path: "designs/"
    type: "directory"
  - path: "designs/draft.md"
    type: "file"
    content: "# Draft\n"

cost_limits:               # optional
  per_agent: 5.00
  total: 10.00
---

# Task Description

Collaborate on designing a scalable architecture for...
```

## Detailed Guidance

### Subtask T020: Define TaskTemplate struct

**Goal**: Data structure matching data-model.md:186-248.

**Implementation Steps**:

1. **Create internal/template/template.go**:
   ```go
   package template

   import "github.com/yourusername/collab/internal/storage"

   type TaskTemplate struct {
       // Agent 1 configuration
       Agent1Name         string `yaml:"agent1_name"`
       Agent1Role         string `yaml:"agent1_role"`
       Agent1SystemPrompt string `yaml:"agent1_system_prompt"`
       Agent1Model        string `yaml:"agent1_model"`

       // Agent 2 configuration
       Agent2Name         string `yaml:"agent2_name"`
       Agent2Role         string `yaml:"agent2_role"`
       Agent2SystemPrompt string `yaml:"agent2_system_prompt"`
       Agent2Model        string `yaml:"agent2_model"`

       // Collaboration settings
       MaxTurns           int `yaml:"max_turns"`
       TurnTimeoutSeconds int `yaml:"turn_timeout_seconds"`
       TurnDelayMS        int `yaml:"turn_delay_ms"`

       // Optional workspace structure
       WorkspaceStructure []storage.WorkspaceStructure `yaml:"workspace_structure,omitempty"`

       // Optional completion criteria
       CompletionCriteria string `yaml:"completion_criteria,omitempty"`

       // Optional cost limits
       CostLimits *CostLimits `yaml:"cost_limits,omitempty"`

       // Task body (markdown after frontmatter)
       TaskBody string `yaml:"-"`
   }

   type CostLimits struct {
       PerAgent float64 `yaml:"per_agent"`
       Total    float64 `yaml:"total"`
   }

   // Defaults returns a template with sensible defaults
   func Defaults() *TaskTemplate {
       return &TaskTemplate{
           Agent1Model:        "claude-sonnet-4",
           Agent2Model:        "claude-sonnet-4",
           TurnTimeoutSeconds: 300,
           TurnDelayMS:        0,
       }
   }
   ```

**Acceptance Criteria**:
- Struct matches data-model.md specification
- YAML tags for all fields
- Defaults method provides sensible values
- Compiles without errors

---

### Subtask T021: Implement template parser

**Goal**: Extract YAML frontmatter and markdown body, handle malformed YAML with line numbers.

**Implementation Steps**:

1. **Create internal/template/parser.go**:
   ```go
   package template

   import (
       "fmt"
       "os"
       "strings"

       "gopkg.in/yaml.v3"
   )

   // ParseTemplate reads and parses a task template file
   func ParseTemplate(filePath string) (*TaskTemplate, error) {
       data, err := os.ReadFile(filePath)
       if err != nil {
           return nil, fmt.Errorf("read template file: %w", err)
       }

       return ParseTemplateBytes(data)
   }

   // ParseTemplateBytes parses template from byte slice
   func ParseTemplateBytes(data []byte) (*TaskTemplate, error) {
       content := string(data)

       // Split frontmatter and body
       parts := strings.SplitN(content, "---\n", 3)
       if len(parts) < 3 {
           return nil, fmt.Errorf("invalid template format: missing YAML frontmatter (expected ---\\n...\\n---\\n)")
       }

       frontmatter := parts[1]
       body := strings.TrimSpace(parts[2])

       // Start with defaults
       tmpl := Defaults()

       // Parse YAML frontmatter
       if err := yaml.Unmarshal([]byte(frontmatter), tmpl); err != nil {
           // Extract line number from YAML error for helpful message
           return nil, formatYAMLError(err, frontmatter)
       }

       tmpl.TaskBody = body

       return tmpl, nil
   }

   func formatYAMLError(err error, frontmatter string) error {
       errMsg := err.Error()

       // Try to extract line number from yaml.v3 error
       // Format: "yaml: line X: ..."
       if strings.Contains(errMsg, "line ") {
           return fmt.Errorf("YAML frontmatter error: %s", errMsg)
       }

       // Count lines up to error position if available
       lines := strings.Split(frontmatter, "\n")
       return fmt.Errorf("YAML frontmatter error (total %d lines): %s", len(lines), errMsg)
   }
   ```

2. **Write parser tests**:
   ```go
   func TestParseTemplate(t *testing.T) {
       template := `---
   agent1_name: "Alice"
   agent1_role: "Designer"
   agent1_system_prompt: "You are a designer"
   agent2_name: "Bob"
   agent2_role: "Reviewer"
   agent2_system_prompt: "You are a reviewer"
   max_turns: 5
   ---

   # Task

   Design a system.
   `

       tmpl, err := ParseTemplateBytes([]byte(template))
       require.NoError(t, err)

       assert.Equal(t, "Alice", tmpl.Agent1Name)
       assert.Equal(t, "Bob", tmpl.Agent2Name)
       assert.Equal(t, 5, tmpl.MaxTurns)
       assert.Equal(t, "# Task\n\nDesign a system.", tmpl.TaskBody)
       assert.Equal(t, "claude-sonnet-4", tmpl.Agent1Model) // Default
   }

   func TestParseTemplateMalformedYAML(t *testing.T) {
       template := `---
   agent1_name: "Alice
   invalid yaml here
   ---

   # Task
   `

       _, err := ParseTemplateBytes([]byte(template))
       require.Error(t, err)
       assert.Contains(t, err.Error(), "YAML frontmatter error")
   }

   func TestParseTemplateMissingFrontmatter(t *testing.T) {
       template := `# Task without frontmatter`

       _, err := ParseTemplateBytes([]byte(template))
       require.Error(t, err)
       assert.Contains(t, err.Error(), "missing YAML frontmatter")
   }
   ```

**Acceptance Criteria**:
- Parses valid templates correctly
- Applies defaults for optional fields
- Malformed YAML returns helpful error with line info
- Missing frontmatter detected with clear error
- Task body preserved exactly (including whitespace)

---

### Subtask T022: Implement template validator

**Goal**: Check required fields, validate types/ranges, return detailed errors.

**Implementation Steps**:

1. **Create internal/template/validator.go**:
   ```go
   package template

   import (
       "fmt"
       "strings"
   )

   type ValidationError struct {
       Field   string
       Message string
   }

   func (ve ValidationError) Error() string {
       return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
   }

   type ValidationErrors []ValidationError

   func (ve ValidationErrors) Error() string {
       var msgs []string
       for _, err := range ve {
           msgs = append(msgs, err.Error())
       }
       return strings.Join(msgs, "\n")
   }

   // ValidateTemplate checks all requirements
   func ValidateTemplate(tmpl *TaskTemplate) error {
       var errors ValidationErrors

       // Required fields (FR-057)
       if tmpl.Agent1Name == "" {
           errors = append(errors, ValidationError{"agent1_name", "required field missing"})
       }
       if tmpl.Agent1Role == "" {
           errors = append(errors, ValidationError{"agent1_role", "required field missing"})
       }
       if tmpl.Agent1SystemPrompt == "" {
           errors = append(errors, ValidationError{"agent1_system_prompt", "required field missing"})
       }

       if tmpl.Agent2Name == "" {
           errors = append(errors, ValidationError{"agent2_name", "required field missing"})
       }
       if tmpl.Agent2Role == "" {
           errors = append(errors, ValidationError{"agent2_role", "required field missing"})
       }
       if tmpl.Agent2SystemPrompt == "" {
           errors = append(errors, ValidationError{"agent2_system_prompt", "required field missing"})
       }

       if tmpl.MaxTurns <= 0 {
           errors = append(errors, ValidationError{"max_turns", "must be greater than 0"})
       }

       // Validate optional fields if present
       if tmpl.TurnTimeoutSeconds < 0 {
           errors = append(errors, ValidationError{"turn_timeout_seconds", "cannot be negative"})
       }

       if tmpl.TurnDelayMS < 0 {
           errors = append(errors, ValidationError{"turn_delay_ms", "cannot be negative"})
       }

       // Validate workspace structure if provided
       for i, item := range tmpl.WorkspaceStructure {
           if item.Path == "" {
               errors = append(errors, ValidationError{
                   fmt.Sprintf("workspace_structure[%d].path", i),
                   "path cannot be empty",
               })
           }
           if item.Type != "file" && item.Type != "directory" {
               errors = append(errors, ValidationError{
                   fmt.Sprintf("workspace_structure[%d].type", i),
                   fmt.Sprintf("must be 'file' or 'directory', got '%s'", item.Type),
               })
           }
       }

       // Validate cost limits if provided
       if tmpl.CostLimits != nil {
           if tmpl.CostLimits.PerAgent < 0 {
               errors = append(errors, ValidationError{"cost_limits.per_agent", "cannot be negative"})
           }
           if tmpl.CostLimits.Total < 0 {
               errors = append(errors, ValidationError{"cost_limits.total", "cannot be negative"})
           }
           if tmpl.CostLimits.Total > 0 && tmpl.CostLimits.PerAgent > tmpl.CostLimits.Total {
               errors = append(errors, ValidationError{"cost_limits", "per_agent cannot exceed total"})
           }
       }

       if len(errors) > 0 {
           return errors
       }

       return nil
   }
   ```

2. **Write validation tests**:
   ```go
   func TestValidateTemplateValid(t *testing.T) {
       tmpl := &TaskTemplate{
           Agent1Name:         "Alice",
           Agent1Role:         "Designer",
           Agent1SystemPrompt: "Prompt 1",
           Agent2Name:         "Bob",
           Agent2Role:         "Reviewer",
           Agent2SystemPrompt: "Prompt 2",
           MaxTurns:           10,
       }

       err := ValidateTemplate(tmpl)
       assert.NoError(t, err)
   }

   func TestValidateTemplateMissingRequired(t *testing.T) {
       tmpl := &TaskTemplate{
           Agent1Name: "Alice",
           // Missing other required fields
           MaxTurns: 10,
       }

       err := ValidateTemplate(tmpl)
       require.Error(t, err)

       verrs, ok := err.(ValidationErrors)
       require.True(t, ok)
       assert.GreaterOrEqual(t, len(verrs), 5) // Missing 5+ required fields
   }

   func TestValidateTemplateInvalidMaxTurns(t *testing.T) {
       tmpl := &TaskTemplate{
           Agent1Name:         "Alice",
           Agent1Role:         "Designer",
           Agent1SystemPrompt: "Prompt",
           Agent2Name:         "Bob",
           Agent2Role:         "Reviewer",
           Agent2SystemPrompt: "Prompt",
           MaxTurns:           0, // Invalid
       }

       err := ValidateTemplate(tmpl)
       require.Error(t, err)
       assert.Contains(t, err.Error(), "max_turns")
   }
   ```

**Acceptance Criteria**:
- Catches all missing required fields (7 fields per FR-057)
- Validates max_turns > 0
- Validates workspace structure types
- Validates cost limits consistency
- Returns detailed error messages with field names
- Returns nil for valid templates

**Reference**:
- spec.md:243-248 (FR-056 to FR-059 template requirements)
- data-model.md:242-248 (validation rules)

---

### Subtask T023: Implement workspace scaffolding

**Goal**: Create directories and files from workspace_structure field.

**Implementation Steps**:

1. **Create internal/template/scaffold.go**:
   ```go
   package template

   import (
       "fmt"

       "github.com/yourusername/collab/internal/storage"
   )

   // ScaffoldWorkspace creates workspace from template structure
   func ScaffoldWorkspace(sessionID, baseDir string, tmpl *TaskTemplate) (string, error) {
       // Use storage layer's CreateWorkspace
       workspaceDir, err := storage.CreateWorkspace(
           sessionID,
           baseDir,
           tmpl.WorkspaceStructure,
       )
       if err != nil {
           return "", fmt.Errorf("scaffold workspace: %w", err)
       }

       // Copy task template to workspace for reference
       templateContent := serializeTemplate(tmpl)
       if err := storage.AtomicWriteString(
           filepath.Join(workspaceDir, "task.md"),
           templateContent,
           0600,
       ); err != nil {
           return "", fmt.Errorf("write task template: %w", err)
       }

       return workspaceDir, nil
   }

   func serializeTemplate(tmpl *TaskTemplate) string {
       // Reconstruct template file for reference
       yamlData, _ := yaml.Marshal(tmpl)
       return fmt.Sprintf("---\n%s---\n\n%s", string(yamlData), tmpl.TaskBody)
   }
   ```

**Acceptance Criteria**:
- Workspace created with template structure
- task.md copied to workspace for reference
- Nested directories supported
- File content from template preserved

---

### Subtask T024: Create example templates

**Goal**: Provide 3 example templates covering simple to complex scenarios.

**Implementation Steps**:

1. **Create examples/simple-agreement.md**:
   ```yaml
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

2. **Create examples/code-review.md**:
   ```yaml
   ---
   agent1_name: "Developer"
   agent1_role: "Implement feature"
   agent1_system_prompt: "You write clean, well-documented code. Focus on simplicity and testability."
   agent2_name: "Reviewer"
   agent2_role: "Code reviewer"
   agent2_system_prompt: "You review code for correctness, style, and best practices. Provide constructive feedback."
   max_turns: 10
   turn_timeout_seconds: 600

   workspace_structure:
     - path: "src/"
       type: "directory"
     - path: "tests/"
       type: "directory"
     - path: "README.md"
       type: "file"
       content: "# Feature Implementation\n\n"

   cost_limits:
     per_agent: 2.00
     total: 4.00
   ---

   # Code Review Collaboration

   Implement a simple user authentication feature.

   Requirements:
   - Password hashing with bcrypt
   - Session token generation
   - Login and logout endpoints
   - Unit tests with >80% coverage

   Deliverable: Working code with passing tests and reviewer approval.
   ```

3. **Create examples/architecture-design.md**:
   ```yaml
   ---
   agent1_name: "Architect"
   agent1_role: "System architect"
   agent1_system_prompt: "You are an experienced system architect. Design scalable, maintainable systems with clear justification for technical choices."
   agent2_name: "Critic"
   agent2_role: "Architecture critic"
   agent2_system_prompt: "You critically analyze architecture proposals. Identify weaknesses, edge cases, and potential failures. Push for robust designs."
   max_turns: 15
   turn_timeout_seconds: 900

   workspace_structure:
     - path: "diagrams/"
       type: "directory"
     - path: "designs/architecture.md"
       type: "file"
       content: "# System Architecture\n\n## Overview\n\n"
     - path: "designs/data-model.md"
       type: "file"
       content: "# Data Model\n\n"

   completion_criteria: "Both agents approve a complete architecture design document with diagrams"

   cost_limits:
     per_agent: 5.00
     total: 10.00
   ---

   # Design Scalable E-Commerce Platform

   Design the architecture for a high-traffic e-commerce platform.

   Requirements:
   - Handle 10,000 concurrent users
   - 99.9% uptime SLA
   - Global distribution (multi-region)
   - Real-time inventory management
   - Payment processing integration
   - Search with autocomplete

   Deliverable: Complete architecture document with:
   - System diagram
   - Data model
   - Technology stack decisions with rationale
   - Scalability strategy
   - Failure modes and mitigations
   ```

**Acceptance Criteria**:
- All 3 examples parse and validate successfully
- Examples demonstrate progression: simple → medium → complex
- Examples use different template features (workspace_structure, cost_limits, etc.)
- Examples stored in examples/ directory

**Reference**:
- plan.md:187-189 (examples directory)

---

## Test Strategy

**Unit Tests**:
- Template parsing (valid, malformed, missing frontmatter)
- Validation (missing fields, invalid values, edge cases)
- Scaffolding (nested structures, file content)

**Integration Test**:
```go
func TestTemplateIntegration(t *testing.T) {
    // Parse example template
    tmpl, err := ParseTemplate("../../examples/simple-agreement.md")
    require.NoError(t, err)

    // Validate
    err = ValidateTemplate(tmpl)
    require.NoError(t, err)

    // Scaffold workspace
    baseDir := t.TempDir()
    wsDir, err := ScaffoldWorkspace("test-session", baseDir, tmpl)
    require.NoError(t, err)

    // Verify workspace structure
    assert.DirExists(t, wsDir)
    assert.FileExists(t, filepath.Join(wsDir, "task.md"))
    assert.FileExists(t, filepath.Join(wsDir, "shared_context.md"))
}
```

## Definition of Done

- [ ] All 5 subtasks completed
- [ ] Parser handles malformed YAML gracefully
- [ ] Validator catches 100% of required field violations (SC-009)
- [ ] 3 example templates created and validated
- [ ] Integration test exercises parse → validate → scaffold
- [ ] Error messages are helpful with field names

## References

- [spec.md](../spec.md): FR-056 to FR-059, SC-009
- [plan.md](../plan.md): Lines 353-387 (Phase 3)
- [data-model.md](../data-model.md): Lines 186-253
