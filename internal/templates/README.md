# Template System Documentation

## Overview

The Shiki template system provides a comprehensive, unified approach to template processing across all workflow phases. It replaces the previous scattered templating approach with a centralized system that supports advanced features including template inheritance, conditional logic, and 60+ helper functions.

## Architecture

### Core Components

1. **TemplateProcessor** (`processor.go`) - The main engine for template loading and processing
2. **PhaseContext** (`types.go`) - Unified data structure for template variable substitution
3. **Embedded Fallbacks** (`fallbacks.go`) - System resilience through embedded backup templates

### Directory Structure

```
templates/
├── system-prompts/      # Agent system prompts by phase
│   ├── specify.md       # Specification agent prompt
│   ├── plan.md          # Planning agent prompt
│   ├── tasks.md         # Tasks breakdown agent prompt
│   ├── implement.md     # Implementation agent prompt
│   ├── bug.md           # Bug fix agent prompt
│   └── collaboration.md # Multi-agent collaboration prompt
├── output-templates/    # Output format templates
│   ├── spec.md          # Feature specification template
│   ├── plan.md          # Implementation plan template
│   ├── tasks.md         # Task breakdown template
│   └── task-progress.md # Implementation progress template
└── checklists/          # Quality validation checklists
    └── requirements.md  # Specification quality checklist
```

## Template Processing Features

### Basic Variable Substitution

Templates use Go's `text/template` syntax for variable substitution:

```go
// Basic variables
{{.FriendlyName}}        // Feature name
{{.FeatureNumber}}       // Feature ID number
{{.CurrentDate}}         // Current date (2006-01-02)
{{.CurrentTime}}         // Current time (15:04:05)

// File paths
{{.SpecFile}}           // Path to spec.md
{{.PlanFile}}           // Path to plan.md
{{.TasksFile}}          // Path to tasks.md
```

### Conditional Logic

Templates support conditional rendering:

```go
{{if .HasDescription}}
Description: {{.FeatureDesc}}
{{else}}
No description provided.
{{end}}

{{if .FileExistsCheck "spec.md"}}
Specification exists and is ready for review.
{{end}}
```

### Loop Operations

Process collections with range operations:

```go
{{range .DirectoryList}}
- {{.}}
{{end}}

{{range $index, $item := .DirectoryList}}
{{$index}}: {{$item}}
{{end}}
```

### Template Inheritance

Compose templates using include and partial functions:

```go
{{template "header" .}}

{{include "shared/footer.md" .}}

{{partial "components/status-badge.md" .}}
```

## Helper Functions

The template system provides 60+ helper functions organized into categories:

### String Manipulation

- `upper`, `lower`, `title` - Case conversion
- `trim`, `trimSpace`, `trimPrefix`, `trimSuffix` - String trimming
- `split`, `join`, `replace`, `replaceAll` - String operations
- `contains`, `hasPrefix`, `hasSuffix` - String testing
- `substring`, `pad`, `padLeft`, `padRight` - String formatting

### Logical Operations

- `and`, `or`, `not` - Boolean logic
- `if`, `default` - Conditional operations
- `coalesce` - First non-empty value

### Comparison Operations

- `eq`, `ne`, `lt`, `le`, `gt`, `ge` - Comparison operators
- `compare` - String comparison
- `min`, `max` - Numeric operations

### Array Operations

- `len`, `first`, `last`, `rest` - Array inspection
- `reverse`, `sort` - Array manipulation
- `arrayContains`, `append`, `unique` - Array operations

### File Operations

- `fileExists` - Check file existence
- `readFile` - Load file content
- `basename`, `dirname`, `ext` - Path operations

### Utility Functions

- `now`, `formatTime` - Time operations
- `env` - Environment variables
- `toJSON`, `fromJSON` - JSON serialization
- `debug`, `log` - Debugging support

## PhaseContext

The `PhaseContext` struct provides a unified data model for all template operations:

### Core Fields

```go
type PhaseContext struct {
    // Identification
    FriendlyName   string    // Human-readable feature name
    FeatureNumber  int       // Unique feature identifier
    Slug           string    // URL-safe identifier
    Phase          string    // Current workflow phase

    // File Paths
    SpecFile       string    // spec.md path
    PlanFile       string    // plan.md path
    TasksFile      string    // tasks.md path
    FeatureDir     string    // Feature directory path

    // Timestamps
    CurrentDate    string    // Current date (YYYY-MM-DD)
    CurrentTime    string    // Current time (HH:MM:SS)
    Timestamp      string    // RFC3339 timestamp

    // Environment
    WorkingDir     string    // Current working directory
    ProjectName    string    // Project name from directory

    // Dynamic Data
    FileExists     map[string]bool      // File existence cache
    FileContents   map[string]string    // File content cache
    CustomData     map[string]interface{} // Custom key-value data
}
```

### Builder Methods

PhaseContext uses a fluent builder pattern:

```go
ctx := NewPhaseContext().
    WithCore("User Authentication", 42, "user-auth", "specify").
    WithPaths("/path/to/spec.md", "/path/to/plan.md", "/path/to/tasks.md",
              "/specs", "/contracts", "/checklists", "/features", "/bugs").
    WithSpecifyData(true, "Add OAuth2 authentication system").
    WithTimestamps().
    WithEnvironment().
    WithFileCheck("spec.md", "plan.md").
    WithFileContent("existing-config.json").
    WithCustomData("priority", "P0")
```

### Utility Methods

```go
// File operations
exists := ctx.FileExistsCheck("/path/to/file")
content := ctx.GetFileContent("/path/to/file")

// Custom data
ctx.Set("key", "value")
value := ctx.Get("key")
hasKey := ctx.Has("key")
```

## Advanced Features

### Template Validation

Validate templates before processing:

```go
processor := NewTemplateProcessor()
errors := processor.ValidateTemplate(templateContent, context)
for _, err := range errors {
    log.Printf("Validation error: %v", err)
}
```

### Safe Processing

Process templates with error recovery:

```go
result, errors := processor.SafeProcessTemplate(templateContent, context)
if len(errors) > 0 {
    // Handle partial success with errors
}
```

### Debug Mode

Debug template processing with detailed output:

```go
result, debug := processor.ProcessTemplateWithDebug(templateContent, context)
log.Printf("Debug info: %s", debug)
```

### Template Introspection

Analyze template structure:

```go
funcs := getTemplateFunctions()
vars := extractTemplateVariables(templateContent)
deps := analyzeDependencies(templateContent)
```

## Integration with TUI Models

### System Prompt Loading

Each phase model loads its system prompt:

```go
// In SpecifyModel
prompt, err := m.templates.LoadSystemPrompt("specify", context)
if err != nil {
    return fmt.Errorf("load specify prompt: %w", err)
}
```

### Output Template Processing

Models with output templates process them for preview:

```go
// In ImplementModel
func (m *ImplementModel) loadImplementTemplateContent(session *session.WorkflowSession) {
    context := templates.NewPhaseContext().
        WithCore(session.FriendlyName, session.FeatureNumber, session.Slug, "implement").
        WithPaths(session.SpecFile, session.PlanFile, session.TasksFile,
                  session.SpecDir, session.ContractsDir, session.ChecklistDir,
                  session.FeatureDir, session.BugDir).
        WithTimestamps()

    content, err := m.templates.LoadOutputTemplate("task-progress", context)
    if err != nil {
        m.templateContent = "Error loading template: " + err.Error()
        return
    }
    m.templateContent = content
}
```

## Error Handling and Fallbacks

### Embedded Fallbacks

The system includes embedded fallback templates for system resilience:

- If template files are missing, embedded versions are used
- Fallbacks ensure the system works even without template directory
- All core templates have embedded equivalents

### Error Recovery

Template processing includes comprehensive error handling:

```go
// File loading with fallback
func (tp *TemplateProcessor) loadTemplate(relativePath string) (string, error) {
    // Try loading from file system
    content, err := tp.loadFromFile(relativePath)
    if err == nil {
        return content, nil
    }

    // Fall back to embedded templates
    if fallback, exists := tp.fallbackTemplates[relativePath]; exists {
        return fallback, nil
    }

    return "", fmt.Errorf("template not found: %s", relativePath)
}
```

## Performance Considerations

### Template Caching

- Parsed templates are cached to avoid repeated parsing
- Cache is invalidated when templates change
- Performance tests show ~8μs per template processing

### Memory Usage

- PhaseContext uses lazy loading for file content
- Maps are initialized only when needed
- Embedded fallbacks are loaded once at startup

## Testing

### Comprehensive Test Suite

The template system includes extensive testing:

```go
func TestTemplateProcessor_BasicFunctionality(t *testing.T)
func TestTemplateProcessor_ConditionalLogic(t *testing.T)
func TestTemplateProcessor_LoopFunctionality(t *testing.T)
func TestTemplateProcessor_HelperFunctions(t *testing.T)
func TestTemplateProcessor_StringOperations(t *testing.T)
func TestTemplateProcessor_ArrayOperations(t *testing.T)
func TestTemplateProcessor_LogicalOperations(t *testing.T)
func TestTemplateProcessor_ComparisonOperations(t *testing.T)
func TestTemplateProcessor_FileOperations(t *testing.T)
func TestTemplateProcessor_TemplateInheritance(t *testing.T)
func TestTemplateProcessor_TemplateValidation(t *testing.T)
func TestTemplateProcessor_SafeProcessing(t *testing.T)
func TestTemplateProcessor_DebugMode(t *testing.T)
func TestTemplateProcessor_Introspection(t *testing.T)
func TestTemplateProcessor_PerformanceBenchmark(t *testing.T)
func TestTemplateProcessor_ErrorHandling(t *testing.T)
func TestTemplateProcessor_Integration(t *testing.T)
```

### Running Tests

```bash
go test ./internal/templates/ -v
go test ./internal/templates/ -bench=.
```

## Migration Guide

### From Previous System

The new template system replaces several scattered approaches:

1. **Agent SDK Integration**: Use `SystemPromptPreset: "claude_code"` with `LoadSystemPrompt()`
2. **Unified Data**: Replace custom data structures with `PhaseContext`
3. **Template Loading**: Use `TemplateProcessor` instead of direct file operations
4. **Error Handling**: Leverage embedded fallbacks for reliability

### Breaking Changes

- Custom template data structures are deprecated
- Direct template file reading is replaced by processor methods
- Template syntax remains compatible but new helper functions are available

## Best Practices

1. **Use Builder Pattern**: Leverage PhaseContext builder methods for clean initialization
2. **Validate Templates**: Use validation functions during development
3. **Handle Errors**: Implement proper error handling with fallback strategies
4. **Cache Appropriately**: Let the processor handle template caching
5. **Test Templates**: Write tests for custom templates and template logic
6. **Document Variables**: Document expected template variables in template comments

## Troubleshooting

### Common Issues

1. **Template Not Found**: Check file paths and ensure fallbacks are available
2. **Variable Undefined**: Verify PhaseContext contains expected data
3. **Syntax Errors**: Use validation functions to check template syntax
4. **Performance Issues**: Review template complexity and helper function usage

### Debug Tools

- Use `ProcessTemplateWithDebug()` for detailed processing information
- Enable template validation in development environments
- Check template processor logs for detailed error information

This template system provides a robust, feature-rich foundation for all templating needs in the Shiki CLI tool.