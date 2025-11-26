package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// PlanSetupResult contains the result of plan setup
type PlanSetupResult struct {
	PlanFile     string
	ContractsDir string
	SpecFile     string
	SpecDir      string
}

// SetupPlanSession sets up a new planning session for a spec
// This is the native Go equivalent of setup-plan.sh
func SetupPlanSession(specSlug string) (*PlanSetupResult, error) {
	// Get spec directory
	specDir := GetSpecDir(specSlug)

	// Validate spec directory exists
	if _, err := os.Stat(specDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("spec directory not found: %s", specDir)
	}

	// Validate spec.md exists
	specFile := filepath.Join(specDir, "spec.md")
	if _, err := os.Stat(specFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("spec.md not found in %s", specDir)
	}

	// Define paths
	planFile := filepath.Join(specDir, "plan.md")
	contractsDir := filepath.Join(specDir, "contracts")

	// Create plan.md from template if it doesn't exist
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		template, err := LoadPlanTemplate()
		if err != nil {
			return nil, fmt.Errorf("load plan template: %w", err)
		}

		if err := AtomicWriteString(planFile, template, 0o644); err != nil {
			return nil, fmt.Errorf("write plan.md: %w", err)
		}
	}

	// Create contracts directory
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create contracts directory: %w", err)
	}

	return &PlanSetupResult{
		PlanFile:     planFile,
		ContractsDir: contractsDir,
		SpecFile:     specFile,
		SpecDir:      specDir,
	}, nil
}

// LoadPlanTemplate loads the plan template from .sekkei/templates/plan-template.md
func LoadPlanTemplate() (string, error) {
	// Try to load from .sekkei/templates/plan-template.md
	templatePath := filepath.Join(".sekkei", "templates", "plan-template.md")
	if content, err := os.ReadFile(templatePath); err == nil {
		return string(content), nil
	}

	// Fall back to a default template
	return defaultPlanTemplate, nil
}

// PlanExists checks if a plan.md file exists for the given spec
func PlanExists(specSlug string) bool {
	planPath := GetPlanPath(specSlug)
	_, err := os.Stat(planPath)
	return err == nil
}

// GetPlanPath returns the path to plan.md for a spec
func GetPlanPath(specSlug string) string {
	return filepath.Join(GetSpecDir(specSlug), "plan.md")
}

// GetContractsDir returns the path to the contracts directory for a spec
func GetContractsDir(specSlug string) string {
	return filepath.Join(GetSpecDir(specSlug), "contracts")
}

// LoadPlanContent loads the content of plan.md
func LoadPlanContent(specSlug string) (string, error) {
	planPath := GetPlanPath(specSlug)
	content, err := os.ReadFile(planPath) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		return "", fmt.Errorf("read plan.md: %w", err)
	}
	return string(content), nil
}

// GetCompletedSpecs returns all specs with status="complete"
func GetCompletedSpecs() ([]*FeatureSpec, error) {
	allSpecs, err := ListSpecs()
	if err != nil {
		return nil, err
	}

	var completedSpecs []*FeatureSpec
	for _, spec := range allSpecs {
		if spec.Status == "complete" {
			completedSpecs = append(completedSpecs, spec)
		}
	}

	return completedSpecs, nil
}

// ValidateSpecForPlanning validates that a spec is ready for planning
func ValidateSpecForPlanning(specSlug string) (*FeatureSpec, error) {
	spec, err := LoadSpec(specSlug)
	if err != nil {
		return nil, fmt.Errorf("spec not found: %s", specSlug)
	}

	if spec.Status != "complete" {
		return nil, fmt.Errorf("spec %s is not complete (status: %s). Complete the specification first with 'collab specify resume %s'",
			specSlug, spec.Status, specSlug)
	}

	return spec, nil
}

// defaultPlanTemplate is the fallback plan template
var defaultPlanTemplate = `# Implementation Plan: [Feature Name]

**Created**: [YYYY-MM-DD]
**Status**: Draft

## Summary

[1-2 paragraph overview of the implementation approach]

## Technical Context

### Current State

[Describe the existing codebase and relevant components]

### Proposed Solution

[High-level description of the technical approach]

## Implementation Phases

### Phase 0: Setup & Infrastructure

**Goal**: [Setup goals]

**Tasks**:
1. [Setup task]

### Phase 1: Core Implementation

**Goal**: [Implementation goals]

**Tasks**:
1. [Implementation task]

### Phase 2: Testing & Polish

**Goal**: [Testing goals]

**Tasks**:
1. [Testing task]

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

## Open Questions

- [ ] [Question]

## References

- [spec.md](./spec.md) - Feature specification
`
