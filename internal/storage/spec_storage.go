package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/darinhaener/collab/internal/config"
	"gopkg.in/yaml.v3"
)

// FeatureSpec represents a feature specification
type FeatureSpec struct {
	Number      int       `json:"number"`
	Slug        string    `json:"slug"`
	FeatureName string    `json:"feature_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"` // draft, in-progress, complete
	Content     string    `json:"-"`      // spec.md content (not in JSON)
}

// SpecMetadata holds metadata for the spec frontmatter
type SpecMetadata struct {
	FeatureNumber int       `yaml:"feature_number"`
	Slug          string    `yaml:"slug"`
	FeatureName   string    `yaml:"feature_name"`
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	Status        string    `yaml:"status"`
}

// ValidateFeatureName validates a feature name according to requirements
func ValidateFeatureName(name string) error {
	// Check for empty name before trimming (important for whitespace-only detection)
	if name == "" {
		return fmt.Errorf("feature name cannot be empty")
	}

	// Check for whitespace-only names
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("feature name cannot be whitespace-only")
	}

	// Check length limit (100 characters) on the original name
	if len(name) > 100 {
		return fmt.Errorf("feature name cannot exceed 100 characters (got %d)", len(name))
	}

	// Check for directory traversal characters
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("feature name cannot contain directory traversal characters (/, \\, ..)")
	}

	// Check for other problematic characters
	problematicChars := []string{"\x00", "\n", "\r", "\t"}
	for _, char := range problematicChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("feature name contains invalid control characters")
		}
	}

	return nil
}

// CreateSlugFromName converts a feature name to a slug format with enhanced validation
func CreateSlugFromName(featureName string, featureNumber int) string {
	// Trim whitespace first
	name := strings.TrimSpace(featureName)

	// Convert to lowercase
	slug := strings.ToLower(name)

	// Handle Unicode characters by normalizing them
	var result strings.Builder
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			result.WriteRune('-')
		}
		// Skip other Unicode categories (marks, etc.)
	}
	slug = result.String()

	// Replace multiple consecutive hyphens with single hyphen
	reg := regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Ensure we have some content after sanitization
	if slug == "" {
		slug = "feature"
	}

	// Truncate if too long (leave room for number prefix and hyphen)
	maxSlugLength := 50 // reasonable filesystem limit
	if len(slug) > maxSlugLength {
		// Try to truncate at word boundary
		if lastHyphen := strings.LastIndex(slug[:maxSlugLength], "-"); lastHyphen > 10 {
			slug = slug[:lastHyphen]
		} else {
			slug = slug[:maxSlugLength]
		}
	}

	// Add feature number prefix
	return fmt.Sprintf("%03d-%s", featureNumber, slug)
}

// CreateSlug maintains backward compatibility by calling CreateSlugFromName
func CreateSlug(friendlyName string, featureNumber int) string {
	return CreateSlugFromName(friendlyName, featureNumber)
}

// GetSpecsDir returns the base directory for all specs
func GetSpecsDir() string {
	return filepath.Join(config.GetDataDir(), "specs")
}

// GetSpecDir returns the directory for a specific spec
func GetSpecDir(slug string) string {
	return filepath.Join(GetSpecsDir(), slug)
}

// SaveSpec saves a feature specification to disk
func SaveSpec(spec *FeatureSpec) error {
	specDir := GetSpecDir(spec.Slug)

	// Create directory structure
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return fmt.Errorf("create spec directory: %w", err)
	}

	checklistsDir := filepath.Join(specDir, "checklists")
	if err := os.MkdirAll(checklistsDir, 0o755); err != nil {
		return fmt.Errorf("create checklists directory: %w", err)
	}

	// Save meta.json
	metaPath := filepath.Join(specDir, "meta.json")
	metaData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta.json: %w", err)
	}
	if err := AtomicWrite(metaPath, metaData, 0o644); err != nil {
		return fmt.Errorf("write meta.json: %w", err)
	}

	// Save spec.md with YAML frontmatter
	specPath := filepath.Join(specDir, "spec.md")
	metadata := SpecMetadata{
		FeatureNumber: spec.Number,
		Slug:          spec.Slug,
		FeatureName:   spec.FeatureName,
		CreatedAt:     spec.CreatedAt,
		UpdatedAt:     spec.UpdatedAt,
		Status:        spec.Status,
	}

	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal spec metadata: %w", err)
	}

	fullContent := fmt.Sprintf("---\n%s---\n\n%s", string(yamlData), spec.Content)
	if err := AtomicWriteString(specPath, fullContent, 0o644); err != nil {
		return fmt.Errorf("write spec.md: %w", err)
	}

	return nil
}

// LoadSpec loads a feature specification from disk
func LoadSpec(slug string) (*FeatureSpec, error) {
	specDir := GetSpecDir(slug)

	// Check if directory exists
	if _, err := os.Stat(specDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("spec not found: %s", slug)
	}

	// Load meta.json
	metaPath := filepath.Join(specDir, "meta.json")
	metaData, err := os.ReadFile(metaPath) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		return nil, fmt.Errorf("read meta.json: %w", err)
	}

	var spec FeatureSpec
	if err := json.Unmarshal(metaData, &spec); err != nil {
		return nil, fmt.Errorf("parse meta.json: %w", err)
	}

	// Load spec.md content (strip frontmatter)
	specPath := filepath.Join(specDir, "spec.md")
	content, _, err := readSpecContent(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec.md: %w", err)
	}

	spec.Content = content
	return &spec, nil
}

// LoadSpecByNumber loads a spec by its feature number
func LoadSpecByNumber(number int) (*FeatureSpec, error) {
	specs, err := ListSpecs()
	if err != nil {
		return nil, err
	}

	for _, spec := range specs {
		if spec.Number == number {
			return LoadSpec(spec.Slug)
		}
	}

	return nil, fmt.Errorf("spec not found: %03d", number)
}

// ListSpecs returns all feature specifications
func ListSpecs() ([]*FeatureSpec, error) {
	specsDir := GetSpecsDir()

	// Check if specs directory exists
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return []*FeatureSpec{}, nil
	}

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return nil, fmt.Errorf("read specs directory: %w", err)
	}

	var specs []*FeatureSpec
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden files like .counter
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		metaPath := filepath.Join(specsDir, entry.Name(), "meta.json")
		metaData, err := os.ReadFile(metaPath) //nolint:gosec // G304: Reading from validated path
		if err != nil {
			// Skip directories without meta.json
			continue
		}

		var spec FeatureSpec
		if err := json.Unmarshal(metaData, &spec); err != nil {
			continue
		}

		specs = append(specs, &spec)
	}

	return specs, nil
}

// DeleteSpec deletes a feature specification
func DeleteSpec(slug string) error {
	specDir := GetSpecDir(slug)
	return os.RemoveAll(specDir)
}

// UpdateSpecStatus updates the status of a spec
func UpdateSpecStatus(slug, status string) error {
	spec, err := LoadSpec(slug)
	if err != nil {
		return err
	}

	spec.Status = status
	spec.UpdatedAt = time.Now()
	return SaveSpec(spec)
}

// UpdateSpecContent updates the content of a spec
func UpdateSpecContent(slug, content string) error {
	spec, err := LoadSpec(slug)
	if err != nil {
		return err
	}

	spec.Content = content
	spec.UpdatedAt = time.Now()
	return SaveSpec(spec)
}

// readSpecContent reads spec.md and separates frontmatter from content
func readSpecContent(path string) (string, *SpecMetadata, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		return "", nil, err
	}

	// Split frontmatter and content
	parts := strings.SplitN(string(data), "---\n", 3)
	if len(parts) < 3 {
		// No metadata, return content only
		return string(data), nil, nil
	}

	var meta SpecMetadata
	if err := yaml.Unmarshal([]byte(parts[1]), &meta); err != nil {
		return "", nil, fmt.Errorf("parse spec metadata: %w", err)
	}

	return strings.TrimSpace(parts[2]), &meta, nil
}

// PromptForFeatureName prompts the user for a feature name with validation and retry logic
func PromptForFeatureName() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("What would you like to name this feature? ")

		if !scanner.Scan() {
			// Handle EOF or interrupted input
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("input error: %w", err)
			}
			return "", fmt.Errorf("input interrupted")
		}

		name := scanner.Text()

		// Validate the input
		if err := ValidateFeatureName(name); err != nil {
			fmt.Printf("Invalid feature name: %v\nPlease try again.\n\n", err)
			continue
		}

		return strings.TrimSpace(name), nil
	}
}
