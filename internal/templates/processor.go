package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// TemplateProcessor handles loading and processing of templates with fallback to embedded defaults
type TemplateProcessor struct {
	templateDir       string
	cache             map[string]string
	fallbackTemplates map[string]string
	templateCache     map[string]*template.Template
}

// NewTemplateProcessor creates a new template processor with the specified template directory
func NewTemplateProcessor(templateDir string) *TemplateProcessor {
	if templateDir == "" {
		templateDir = "templates"
	}

	tp := &TemplateProcessor{
		templateDir:       templateDir,
		cache:             make(map[string]string),
		fallbackTemplates: make(map[string]string),
		templateCache:     make(map[string]*template.Template),
	}

	// Initialize embedded fallback templates
	tp.initializeFallbacks()

	return tp
}

// LoadSystemPrompt loads and processes a system prompt template for the given phase
func (tp *TemplateProcessor) LoadSystemPrompt(phase string, context *PhaseContext) (string, error) {
	templatePath := filepath.Join("system-prompts", phase+".md")
	content, err := tp.loadTemplate(templatePath)
	if err != nil {
		return "", fmt.Errorf("load system prompt template %s: %w", phase, err)
	}

	return tp.ProcessTemplate(content, context)
}

// LoadOutputTemplate loads and processes an output template for the given phase
func (tp *TemplateProcessor) LoadOutputTemplate(phase string, context *PhaseContext) (string, error) {
	templatePath := filepath.Join("output-templates", phase+".md")
	content, err := tp.loadTemplate(templatePath)
	if err != nil {
		return "", fmt.Errorf("load output template %s: %w", phase, err)
	}

	return tp.ProcessTemplate(content, context)
}

// LoadChecklistTemplate loads and processes a checklist template
func (tp *TemplateProcessor) LoadChecklistTemplate(name string, context *PhaseContext) (string, error) {
	templatePath := filepath.Join("checklists", name+".md")
	content, err := tp.loadTemplate(templatePath)
	if err != nil {
		return "", fmt.Errorf("load checklist template %s: %w", name, err)
	}

	return tp.ProcessTemplate(content, context)
}

// ProcessTemplate processes template content with variable substitution and advanced features
func (tp *TemplateProcessor) ProcessTemplate(templateContent string, context *PhaseContext) (string, error) {
	// Create template with helper functions
	tmpl, err := template.New("template").Funcs(tp.getTemplateFunctions()).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	// Execute template with context data
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// ProcessTemplateWithInheritance processes template content with inheritance and composition support
func (tp *TemplateProcessor) ProcessTemplateWithInheritance(templateContent string, context *PhaseContext, templateName string) (string, error) {
	// Create a master template that can parse and reference other templates
	tmpl := template.New(templateName).Funcs(tp.getTemplateFunctions())

	// Add template composition functions specific to this instance
	tmpl = tmpl.Funcs(template.FuncMap{
		"template": func(name string, data interface{}) (string, error) {
			return tp.executeSubTemplate(tmpl, name, data)
		},
		"include": func(path string) (string, error) {
			return tp.includeTemplate(path)
		},
		"partial": func(path string, data interface{}) (string, error) {
			return tp.executePartial(tmpl, path, data)
		},
	})

	// Parse the main template
	tmpl, err := tmpl.Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("parse main template: %w", err)
	}

	// Execute the main template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// LoadTemplateWithInheritance loads and processes a template with inheritance support
func (tp *TemplateProcessor) LoadTemplateWithInheritance(templatePath string, context *PhaseContext) (string, error) {
	content, err := tp.loadTemplate(templatePath)
	if err != nil {
		return "", err
	}

	templateName := filepath.Base(templatePath)
	return tp.ProcessTemplateWithInheritance(content, context, templateName)
}

// executeSubTemplate executes a sub-template by name
func (tp *TemplateProcessor) executeSubTemplate(parentTmpl *template.Template, name string, data interface{}) (string, error) {
	// Try to find the template in the parent template's defined templates
	tmpl := parentTmpl.Lookup(name)
	if tmpl == nil {
		// If not found, try to load it from file
		content, err := tp.loadTemplate(name + ".md")
		if err != nil {
			return "", fmt.Errorf("load sub-template %s: %w", name, err)
		}

		// Parse the sub-template into the parent template
		tmpl, err = parentTmpl.New(name).Parse(content)
		if err != nil {
			return "", fmt.Errorf("parse sub-template %s: %w", name, err)
		}
	}

	// Execute the sub-template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute sub-template %s: %w", name, err)
	}

	return buf.String(), nil
}

// includeTemplate includes raw template content without processing
func (tp *TemplateProcessor) includeTemplate(path string) (string, error) {
	content, err := tp.loadTemplate(path)
	if err != nil {
		return "", fmt.Errorf("include template %s: %w", path, err)
	}
	return content, nil
}

// executePartial executes a partial template with data
func (tp *TemplateProcessor) executePartial(parentTmpl *template.Template, path string, data interface{}) (string, error) {
	content, err := tp.loadTemplate(path)
	if err != nil {
		return "", fmt.Errorf("load partial %s: %w", path, err)
	}

	// Create a new template for the partial
	partialName := filepath.Base(path)
	tmpl, err := parentTmpl.New(partialName).Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse partial %s: %w", path, err)
	}

	// Execute the partial
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute partial %s: %w", path, err)
	}

	return buf.String(), nil
}

// ValidateTemplate validates a template for syntax errors and missing variables
func (tp *TemplateProcessor) ValidateTemplate(templateContent string, expectedVars []string) []error {
	var errors []error

	// Parse template to check for syntax errors
	tmpl, err := template.New("validation").Funcs(tp.getTemplateFunctions()).Parse(templateContent)
	if err != nil {
		errors = append(errors, fmt.Errorf("template syntax error: %w", err))
		return errors // Can't continue validation if it doesn't parse
	}

	// Check for undefined variables by executing with empty context
	var buf bytes.Buffer
	emptyContext := NewPhaseContext()
	if err := tmpl.Execute(&buf, emptyContext); err != nil {
		if strings.Contains(err.Error(), "undefined") || strings.Contains(err.Error(), "nil pointer") {
			errors = append(errors, fmt.Errorf("template validation warning: %w", err))
		}
	}

	// Validate expected variables exist in template
	templateStr := templateContent
	for _, varName := range expectedVars {
		varPattern := "{{.*" + varName + ".*}}"
		if matched, _ := regexp.MatchString(varPattern, templateStr); !matched {
			errors = append(errors, fmt.Errorf("expected variable '%s' not found in template", varName))
		}
	}

	return errors
}

// ValidateTemplateFile validates a template file
func (tp *TemplateProcessor) ValidateTemplateFile(templatePath string, expectedVars []string) []error {
	content, err := tp.loadTemplate(templatePath)
	if err != nil {
		return []error{fmt.Errorf("failed to load template %s: %w", templatePath, err)}
	}

	errors := tp.ValidateTemplate(content, expectedVars)

	// Add file path context to errors
	for i, err := range errors {
		errors[i] = fmt.Errorf("in template %s: %w", templatePath, err)
	}

	return errors
}

// SafeProcessTemplate processes a template with enhanced error handling
func (tp *TemplateProcessor) SafeProcessTemplate(templateContent string, context *PhaseContext) (string, []error) {
	var warnings []error

	// Validate template first
	validationErrors := tp.ValidateTemplate(templateContent, nil)
	for _, err := range validationErrors {
		if strings.Contains(err.Error(), "warning") {
			warnings = append(warnings, err)
		} else {
			return "", []error{err}
		}
	}

	// Process template with recovery
	defer func() {
		if r := recover(); r != nil {
			warnings = append(warnings, fmt.Errorf("template execution panic: %v", r))
		}
	}()

	result, err := tp.ProcessTemplate(templateContent, context)
	if err != nil {
		return "", []error{fmt.Errorf("template processing error: %w", err)}
	}

	return result, warnings
}

// TemplateDebugInfo contains debugging information about template processing
type TemplateDebugInfo struct {
	TemplateName     string
	ProcessingTime   time.Duration
	Variables        map[string]interface{}
	UsedFunctions    []string
	ParsedTemplates  []string
	ExecutionSteps   []string
	Warnings         []string
	VariableAccess   map[string]int
}

// ProcessTemplateWithDebug processes a template with detailed debugging information
func (tp *TemplateProcessor) ProcessTemplateWithDebug(templateContent string, context *PhaseContext, templateName string) (string, *TemplateDebugInfo, error) {
	startTime := time.Now()
	debugInfo := &TemplateDebugInfo{
		TemplateName:    templateName,
		Variables:       make(map[string]interface{}),
		UsedFunctions:   []string{},
		ParsedTemplates: []string{},
		ExecutionSteps:  []string{},
		Warnings:        []string{},
		VariableAccess:  make(map[string]int),
	}

	debugInfo.ExecutionSteps = append(debugInfo.ExecutionSteps, "Starting template processing")

	// Extract variables from context using reflection
	debugInfo.Variables = tp.extractContextVariables(context)
	debugInfo.ExecutionSteps = append(debugInfo.ExecutionSteps, fmt.Sprintf("Extracted %d context variables", len(debugInfo.Variables)))

	// Create template with debug-enabled functions
	tmpl, err := template.New(templateName).Funcs(tp.getDebugTemplateFunctions(debugInfo)).Parse(templateContent)
	if err != nil {
		debugInfo.ProcessingTime = time.Since(startTime)
		return "", debugInfo, fmt.Errorf("parse template: %w", err)
	}

	debugInfo.ExecutionSteps = append(debugInfo.ExecutionSteps, "Template parsed successfully")
	debugInfo.ParsedTemplates = append(debugInfo.ParsedTemplates, templateName)

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, context); err != nil {
		debugInfo.ProcessingTime = time.Since(startTime)
		return "", debugInfo, fmt.Errorf("execute template: %w", err)
	}

	debugInfo.ExecutionSteps = append(debugInfo.ExecutionSteps, "Template executed successfully")
	debugInfo.ProcessingTime = time.Since(startTime)

	return buf.String(), debugInfo, nil
}

// IntrospectTemplate analyzes a template and returns information about its structure
func (tp *TemplateProcessor) IntrospectTemplate(templateContent string) (*TemplateIntrospection, error) {
	introspection := &TemplateIntrospection{
		Variables:     []string{},
		Functions:     []string{},
		Conditionals:  []string{},
		Loops:         []string{},
		Includes:      []string{},
		Dependencies:  []string{},
		Complexity:    0,
	}

	// Parse template to extract structure
	tmpl, err := template.New("introspection").Funcs(tp.getTemplateFunctions()).Parse(templateContent)
	if err != nil {
		return introspection, fmt.Errorf("parse template for introspection: %w", err)
	}

	// Analyze template content using regex patterns
	introspection.Variables = tp.extractVariableNames(templateContent)
	introspection.Functions = tp.extractFunctionCalls(templateContent)
	introspection.Conditionals = tp.extractConditionals(templateContent)
	introspection.Loops = tp.extractLoops(templateContent)
	introspection.Includes = tp.extractIncludes(templateContent)

	// Calculate complexity based on template features
	introspection.Complexity = len(introspection.Variables) +
		len(introspection.Functions)*2 +
		len(introspection.Conditionals)*3 +
		len(introspection.Loops)*4 +
		len(introspection.Includes)*2

	// Determine dependencies
	for _, include := range introspection.Includes {
		introspection.Dependencies = append(introspection.Dependencies, include)
	}

	// Add template name to parsed templates list
	_ = tmpl // Use the parsed template variable

	return introspection, nil
}

// TemplateIntrospection contains analysis information about a template
type TemplateIntrospection struct {
	Variables     []string
	Functions     []string
	Conditionals  []string
	Loops         []string
	Includes      []string
	Dependencies  []string
	Complexity    int
}

// extractContextVariables extracts variable names and values from PhaseContext using reflection
func (tp *TemplateProcessor) extractContextVariables(context *PhaseContext) map[string]interface{} {
	variables := make(map[string]interface{})

	v := reflect.ValueOf(context).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if field.CanInterface() {
			variables[fieldType.Name] = field.Interface()
		}
	}

	return variables
}

// getDebugTemplateFunctions returns template functions with debugging capabilities
func (tp *TemplateProcessor) getDebugTemplateFunctions(debugInfo *TemplateDebugInfo) template.FuncMap {
	baseFuncs := tp.getTemplateFunctions()

	// Wrap functions to track usage
	for name, fn := range baseFuncs {
		wrappedFn := tp.wrapFunctionForDebugging(name, fn, debugInfo)
		baseFuncs[name] = wrappedFn
	}

	return baseFuncs
}

// wrapFunctionForDebugging wraps a template function to track its usage
func (tp *TemplateProcessor) wrapFunctionForDebugging(name string, fn interface{}, debugInfo *TemplateDebugInfo) interface{} {
	return func(args ...interface{}) interface{} {
		debugInfo.UsedFunctions = append(debugInfo.UsedFunctions, name)
		debugInfo.ExecutionSteps = append(debugInfo.ExecutionSteps, fmt.Sprintf("Called function: %s", name))

		// Call original function using reflection
		fnValue := reflect.ValueOf(fn)
		argValues := make([]reflect.Value, len(args))
		for i, arg := range args {
			argValues[i] = reflect.ValueOf(arg)
		}

		result := fnValue.Call(argValues)
		if len(result) > 0 {
			return result[0].Interface()
		}
		return nil
	}
}

// Template analysis helper functions
func (tp *TemplateProcessor) extractVariableNames(content string) []string {
	// More comprehensive regex to catch variables in different contexts
	re := regexp.MustCompile(`\.(\w+)`)
	matches := re.FindAllStringSubmatch(content, -1)
	var variables []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			variables = append(variables, match[1])
			seen[match[1]] = true
		}
	}

	return variables
}

func (tp *TemplateProcessor) extractFunctionCalls(content string) []string {
	re := regexp.MustCompile(`\{\{(\w+)\s`)
	matches := re.FindAllStringSubmatch(content, -1)
	var functions []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			functions = append(functions, match[1])
			seen[match[1]] = true
		}
	}

	return functions
}

func (tp *TemplateProcessor) extractConditionals(content string) []string {
	re := regexp.MustCompile(`\{\{if\s+(.+?)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)
	var conditionals []string

	for _, match := range matches {
		if len(match) > 1 {
			conditionals = append(conditionals, match[1])
		}
	}

	return conditionals
}

func (tp *TemplateProcessor) extractLoops(content string) []string {
	re := regexp.MustCompile(`\{\{range\s+(.+?)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)
	var loops []string

	for _, match := range matches {
		if len(match) > 1 {
			loops = append(loops, match[1])
		}
	}

	return loops
}

func (tp *TemplateProcessor) extractIncludes(content string) []string {
	re := regexp.MustCompile(`\{\{(?:template|include|partial)\s+"(.+?)"\s*.*?\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)
	var includes []string

	for _, match := range matches {
		if len(match) > 1 {
			includes = append(includes, match[1])
		}
	}

	return includes
}

// getTemplateFunctions returns a map of helper functions available in templates
func (tp *TemplateProcessor) getTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		// String manipulation functions
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     strings.Title,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"join":      strings.Join,
		"split":     strings.Split,
		"trim":      strings.TrimSpace,

		// Logical functions
		"and": func(a, b bool) bool { return a && b },
		"or":  func(a, b bool) bool { return a || b },
		"not": func(a bool) bool { return !a },

		// Comparison functions
		"eq": func(a, b interface{}) bool { return a == b },
		"ne": func(a, b interface{}) bool { return a != b },
		"lt": func(a, b int) bool { return a < b },
		"le": func(a, b int) bool { return a <= b },
		"gt": func(a, b int) bool { return a > b },
		"ge": func(a, b int) bool { return a >= b },

		// Utility functions
		"default": func(defaultVal, val interface{}) interface{} {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},
		"coalesce": func(vals ...interface{}) interface{} {
			for _, val := range vals {
				if val != nil && val != "" {
					return val
				}
			}
			return ""
		},

		// Array/slice functions
		"len":   func(v interface{}) int { return reflect.ValueOf(v).Len() },
		"empty": func(v interface{}) bool {
			if v == nil {
				return true
			}
			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
				return rv.Len() == 0
			case reflect.Bool:
				return !rv.Bool()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return rv.Int() == 0
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				return rv.Uint() == 0
			case reflect.Float32, reflect.Float64:
				return rv.Float() == 0
			case reflect.Interface, reflect.Ptr:
				return rv.IsNil()
			}
			return false
		},

		// Enhanced array/slice iteration functions
		"first": func(v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				if rv.Len() > 0 {
					return rv.Index(0).Interface()
				}
			}
			return nil
		},
		"last": func(v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				if rv.Len() > 0 {
					return rv.Index(rv.Len() - 1).Interface()
				}
			}
			return nil
		},
		"index": func(i int, v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				if i >= 0 && i < rv.Len() {
					return rv.Index(i).Interface()
				}
			}
			return nil
		},
		"slice": func(start, end int, v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				length := rv.Len()
				if start < 0 {
					start = 0
				}
				if end > length {
					end = length
				}
				if start < end {
					return rv.Slice(start, end).Interface()
				}
			}
			return v
		},
		"reverse": func(v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				length := rv.Len()
				result := reflect.MakeSlice(rv.Type(), length, length)
				for i := 0; i < length; i++ {
					result.Index(i).Set(rv.Index(length - 1 - i))
				}
				return result.Interface()
			}
			return v
		},
		"sort": func(v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				// Simple string sort implementation
				if rv.Type().Elem().Kind() == reflect.String {
					length := rv.Len()
					result := reflect.MakeSlice(rv.Type(), length, length)
					reflect.Copy(result, rv)

					// Bubble sort for simplicity
					for i := 0; i < length-1; i++ {
						for j := 0; j < length-i-1; j++ {
							a := result.Index(j).String()
							b := result.Index(j + 1).String()
							if a > b {
								temp := result.Index(j).Interface()
								result.Index(j).Set(result.Index(j + 1))
								result.Index(j + 1).Set(reflect.ValueOf(temp))
							}
						}
					}
					return result.Interface()
				}
			}
			return v
		},
		"arrayContains": func(item, v interface{}) bool {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				for i := 0; i < rv.Len(); i++ {
					if reflect.DeepEqual(rv.Index(i).Interface(), item) {
						return true
					}
				}
			}
			return false
		},
		"append": func(v interface{}, items ...interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				result := reflect.MakeSlice(rv.Type(), rv.Len()+len(items), rv.Len()+len(items))
				reflect.Copy(result, rv)
				for i, item := range items {
					result.Index(rv.Len() + i).Set(reflect.ValueOf(item))
				}
				return result.Interface()
			}
			return v
		},
		"unique": func(v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				seen := make(map[interface{}]bool)
				result := reflect.MakeSlice(rv.Type(), 0, rv.Len())
				for i := 0; i < rv.Len(); i++ {
					item := rv.Index(i).Interface()
					if !seen[item] {
						seen[item] = true
						result = reflect.Append(result, rv.Index(i))
					}
				}
				return result.Interface()
			}
			return v
		},
	}
}

// loadTemplate loads a template from file or falls back to embedded content
func (tp *TemplateProcessor) loadTemplate(relativePath string) (string, error) {
	// Check cache first
	if cached, exists := tp.cache[relativePath]; exists {
		return cached, nil
	}

	// Try to load from file system first
	fullPath := filepath.Join(tp.templateDir, relativePath)
	if content, err := os.ReadFile(fullPath); err == nil {
		contentStr := string(content)
		tp.cache[relativePath] = contentStr
		return contentStr, nil
	}

	// Fall back to embedded template
	if fallback, exists := tp.fallbackTemplates[relativePath]; exists {
		tp.cache[relativePath] = fallback
		return fallback, nil
	}

	return "", fmt.Errorf("template not found: %s (checked %s and embedded fallbacks)", relativePath, fullPath)
}

// initializeFallbacks sets up embedded fallback templates
func (tp *TemplateProcessor) initializeFallbacks() {
	// System prompt fallbacks using actual embedded content
	tp.fallbackTemplates["system-prompts/specify.md"] = getEmbeddedSystemPrompt("specify")
	tp.fallbackTemplates["system-prompts/plan.md"] = getEmbeddedSystemPrompt("plan")
	tp.fallbackTemplates["system-prompts/tasks.md"] = getEmbeddedSystemPrompt("tasks")
	tp.fallbackTemplates["system-prompts/implement.md"] = getEmbeddedSystemPrompt("implement")
	tp.fallbackTemplates["system-prompts/bug.md"] = getEmbeddedSystemPrompt("bug")
	tp.fallbackTemplates["system-prompts/collaboration.md"] = getEmbeddedSystemPrompt("collaboration")

	// Output template fallbacks using actual embedded content
	tp.fallbackTemplates["output-templates/spec.md"] = GetSpecTemplate()
	tp.fallbackTemplates["output-templates/plan.md"] = getEmbeddedOutputTemplate("plan")
	tp.fallbackTemplates["output-templates/tasks.md"] = getEmbeddedOutputTemplate("tasks")
	tp.fallbackTemplates["output-templates/task-progress.md"] = getEmbeddedOutputTemplate("task-progress")

	// Checklist fallbacks using actual embedded content
	tp.fallbackTemplates["checklists/requirements.md"] = getEmbeddedChecklist("requirements")
}