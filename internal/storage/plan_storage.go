package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dphaener/shiki-cli/internal/templates"
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

// LoadPlanTemplate loads the plan template using the new template system
func LoadPlanTemplate() (string, error) {
	// Use new template processor for plan templates
	processor := templates.NewTemplateProcessor("templates")

	// Create basic context for template processing
	context := templates.NewPhaseContext().
		WithCore("Template Preview", 0, "", "plan")

	// Load plan output template
	content, err := processor.LoadOutputTemplate("plan", context)
	if err != nil {
		return "", fmt.Errorf("load plan template: %w", err)
	}

	return content, nil
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

