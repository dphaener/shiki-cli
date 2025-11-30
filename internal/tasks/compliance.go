package tasks

import (
	"regexp"
	"strings"
	"time"
)

// CompliancePattern represents a pattern for detecting agent compliance
type CompliancePattern struct {
	Name        string         // Human-readable name
	Pattern     *regexp.Regexp // Regex pattern to match
	Required    bool           // Whether this pattern is required for compliance
	Weight      float64        // Weight in compliance score calculation (0.0-1.0)
	Description string         // Description of what this pattern validates
}

// ComplianceDetector validates agent responses for task progress tracking compliance
type ComplianceDetector struct {
	patterns []CompliancePattern
}

// NewComplianceDetector creates a new compliance detector with standard patterns
func NewComplianceDetector() *ComplianceDetector {
	detector := &ComplianceDetector{
		patterns: make([]CompliancePattern, 0),
	}

	detector.initializeStandardPatterns()
	return detector
}

// initializeStandardPatterns sets up the standard compliance patterns
func (cd *ComplianceDetector) initializeStandardPatterns() {
	// Pattern for explicit task start announcements
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        "TaskStartAnnouncement",
		Pattern:     regexp.MustCompile(`(?i)starting\s+(T\d+)[:]\s*(.+)`),
		Required:    true,
		Weight:      0.2,
		Description: "Agent explicitly announces when starting a task",
	})

	// Pattern for explicit task completion announcements
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        "TaskCompletionAnnouncement",
		Pattern:     regexp.MustCompile(`(?i)completed\s+(T\d+)\s+(successfully|completely)`),
		Required:    true,
		Weight:      0.3,
		Description: "Agent explicitly announces when completing a task",
	})

	// Pattern for blocked task reporting with reason
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        "BlockedTaskReporting",
		Pattern:     regexp.MustCompile(`(?i)(T\d+)\s+blocked[:]\s*(.{10,})`),
		Required:    false,
		Weight:      0.2,
		Description: "Agent reports blocked tasks with clear reasons",
	})

	// Pattern for work package completion announcements
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        "WorkPackageCompletion",
		Pattern:     regexp.MustCompile(`(?i)(WP\d+)\s+complete[d]?[,]?\s*(moving to|starting)\s+(WP\d+)`),
		Required:    false,
		Weight:      0.15,
		Description: "Agent announces work package transitions",
	})

	// Pattern for progress file creation announcement
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        "ProgressFileCreation",
		Pattern:     regexp.MustCompile(`(?i)(creat|initializ|generat).*(task-progress\.md|progress.*file)`),
		Required:    true,
		Weight:      0.25,
		Description: "Agent announces creation of progress tracking file",
	})

	// Pattern for task ID usage (general task mentions)
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        "TaskIDUsage",
		Pattern:     regexp.MustCompile(`\b(T\d{3,4}|WP\d{2})\b`),
		Required:    true,
		Weight:      0.1,
		Description: "Agent uses explicit task and work package IDs",
	})

	// Pattern for status update language
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        "StatusUpdateLanguage",
		Pattern:     regexp.MustCompile(`(?i)(updating|marked as|setting.*status|progress.*update)`),
		Required:    false,
		Weight:      0.1,
		Description: "Agent uses explicit status update language",
	})

	// Pattern for timestamp mentions
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        "TimestampMentions",
		Pattern:     regexp.MustCompile(`(?i)(at \d{2}:\d{2}:\d{2}|since \d{2}:\d{2}:\d{2}|timestamp|completed at|started at)`),
		Required:    false,
		Weight:      0.1,
		Description: "Agent includes timing information in updates",
	})
}

// ComplianceResult represents the result of compliance checking
type ComplianceResult struct {
	OverallScore     float64                    `json:"overall_score"`     // 0.0-1.0
	RequiredPassed   bool                       `json:"required_passed"`   // All required patterns found
	PatternResults   map[string]PatternResult   `json:"pattern_results"`   // Results per pattern
	ViolationCount   int                        `json:"violation_count"`   // Number of violations
	ComplianceLevel  ComplianceLevel            `json:"compliance_level"`  // Overall compliance level
	Recommendations  []string                   `json:"recommendations"`   // Suggestions for improvement
	DetectedTasks    []string                   `json:"detected_tasks"`    // Task IDs found in content
	MissingElements  []string                   `json:"missing_elements"`  // Required elements that are missing
}

// PatternResult represents the result for a specific pattern
type PatternResult struct {
	Matched     bool     `json:"matched"`     // Whether pattern was found
	Occurrences int      `json:"occurrences"` // Number of times pattern was found
	Examples    []string `json:"examples"`    // Example matches (up to 3)
	Score       float64  `json:"score"`       // Contribution to overall score
}

// ComplianceLevel represents the overall compliance level
type ComplianceLevel string

const (
	ComplianceLevelExcellent ComplianceLevel = "excellent" // 0.9+
	ComplianceLevelGood      ComplianceLevel = "good"      // 0.7-0.89
	ComplianceLevelFair      ComplianceLevel = "fair"      // 0.5-0.69
	ComplianceLevelPoor      ComplianceLevel = "poor"      // <0.5
)

// CheckCompliance analyzes agent response content for compliance with task tracking requirements
func (cd *ComplianceDetector) CheckCompliance(content string) ComplianceResult {
	result := ComplianceResult{
		PatternResults:  make(map[string]PatternResult),
		DetectedTasks:   cd.extractTaskIDs(content),
		MissingElements: make([]string, 0),
		Recommendations: make([]string, 0),
	}

	totalScore := 0.0
	totalWeight := 0.0
	requiredPassed := true

	// Check each pattern
	for _, pattern := range cd.patterns {
		patternResult := cd.checkPattern(content, pattern)
		result.PatternResults[pattern.Name] = patternResult

		// Calculate weighted score
		score := 0.0
		if patternResult.Matched {
			score = pattern.Weight
		}
		totalScore += score
		totalWeight += pattern.Weight

		// Check required patterns
		if pattern.Required && !patternResult.Matched {
			requiredPassed = false
			result.MissingElements = append(result.MissingElements, pattern.Description)
			result.ViolationCount++
		}
	}

	// Calculate overall score
	if totalWeight > 0 {
		result.OverallScore = totalScore / totalWeight
	}

	result.RequiredPassed = requiredPassed

	// Determine compliance level
	result.ComplianceLevel = cd.determineComplianceLevel(result.OverallScore)

	// Generate recommendations
	result.Recommendations = cd.generateRecommendations(result)

	return result
}

// checkPattern checks a single compliance pattern against content
func (cd *ComplianceDetector) checkPattern(content string, pattern CompliancePattern) PatternResult {
	matches := pattern.Pattern.FindAllStringSubmatch(content, -1)

	result := PatternResult{
		Matched:     len(matches) > 0,
		Occurrences: len(matches),
		Examples:    make([]string, 0),
		Score:       0.0,
	}

	// Collect examples (up to 3)
	for i, match := range matches {
		if i >= 3 {
			break
		}
		if len(match) > 0 {
			example := strings.TrimSpace(match[0])
			if len(example) > 100 {
				example = example[:100] + "..."
			}
			result.Examples = append(result.Examples, example)
		}
	}

	// Calculate score based on occurrences and pattern weight
	if result.Matched {
		result.Score = pattern.Weight
		// Bonus for multiple occurrences
		if result.Occurrences > 1 {
			result.Score += float64(result.Occurrences-1) * pattern.Weight * 0.1
		}
	}

	return result
}

// extractTaskIDs extracts all task and work package IDs from content
func (cd *ComplianceDetector) extractTaskIDs(content string) []string {
	taskPattern := regexp.MustCompile(`\b(T\d{3,4}|WP\d{2})\b`)
	matches := taskPattern.FindAllString(content, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			unique = append(unique, match)
		}
	}

	return unique
}

// determineComplianceLevel determines the compliance level based on score
func (cd *ComplianceDetector) determineComplianceLevel(score float64) ComplianceLevel {
	switch {
	case score >= 0.9:
		return ComplianceLevelExcellent
	case score >= 0.7:
		return ComplianceLevelGood
	case score >= 0.5:
		return ComplianceLevelFair
	default:
		return ComplianceLevelPoor
	}
}

// generateRecommendations generates specific recommendations based on compliance results
func (cd *ComplianceDetector) generateRecommendations(result ComplianceResult) []string {
	var recommendations []string

	// Check specific missing patterns and provide targeted advice
	for patternName, patternResult := range result.PatternResults {
		if !patternResult.Matched {
			switch patternName {
			case "TaskStartAnnouncement":
				recommendations = append(recommendations,
					"Use explicit task start announcements: 'Starting T001: Initialize project structure'")
			case "TaskCompletionAnnouncement":
				recommendations = append(recommendations,
					"Use explicit task completion announcements: 'Completed T001 successfully'")
			case "ProgressFileCreation":
				recommendations = append(recommendations,
					"Announce progress file creation: 'Creating task-progress.md to track implementation progress'")
			case "TaskIDUsage":
				recommendations = append(recommendations,
					"Use explicit task IDs (T001, T002, WP01) in all status updates")
			}
		}
	}

	// General recommendations based on overall score
	if result.OverallScore < 0.7 {
		recommendations = append(recommendations,
			"Increase frequency of explicit status updates throughout implementation")
	}

	if len(result.DetectedTasks) == 0 {
		recommendations = append(recommendations,
			"Include task IDs in your responses to enable progress tracking")
	}

	if result.ViolationCount > 2 {
		recommendations = append(recommendations,
			"Review the task progress tracking requirements in the implementation prompt")
	}

	return recommendations
}

// AddCustomPattern allows adding custom compliance patterns
func (cd *ComplianceDetector) AddCustomPattern(name string, pattern *regexp.Regexp, required bool, weight float64, description string) {
	cd.patterns = append(cd.patterns, CompliancePattern{
		Name:        name,
		Pattern:     pattern,
		Required:    required,
		Weight:      weight,
		Description: description,
	})
}

// GetPatterns returns all registered patterns
func (cd *ComplianceDetector) GetPatterns() []CompliancePattern {
	// Return a copy to prevent external modification
	patterns := make([]CompliancePattern, len(cd.patterns))
	copy(patterns, cd.patterns)
	return patterns
}

// ValidateProgressFileFormat checks if a progress file follows the required format
func (cd *ComplianceDetector) ValidateProgressFileFormat(content string) ComplianceResult {
	result := ComplianceResult{
		PatternResults:  make(map[string]PatternResult),
		DetectedTasks:   cd.extractTaskIDs(content),
		MissingElements: make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Required format elements
	formatPatterns := []CompliancePattern{
		{
			Name:        "ProgressHeader",
			Pattern:     regexp.MustCompile(`(?m)^# Implementation Progress`),
			Required:    true,
			Weight:      0.2,
			Description: "Progress file has correct header",
		},
		{
			Name:        "StatusField",
			Pattern:     regexp.MustCompile(`(?m)^\*\*Status\*\*:\s*(in_progress|completed|blocked|pending)`),
			Required:    true,
			Weight:      0.2,
			Description: "Progress file includes overall status field",
		},
		{
			Name:        "UpdatedField",
			Pattern:     regexp.MustCompile(`(?m)^\*\*Updated\*\*:\s*\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`),
			Required:    true,
			Weight:      0.2,
			Description: "Progress file includes timestamp field",
		},
		{
			Name:        "ProgressField",
			Pattern:     regexp.MustCompile(`(?m)^\*\*Progress\*\*:\s*\d+/\d+\s+tasks?\s+completed`),
			Required:    true,
			Weight:      0.2,
			Description: "Progress file includes progress summary",
		},
		{
			Name:        "WorkPackageHeaders",
			Pattern:     regexp.MustCompile(`(?m)^## Work Package (WP\d+):`),
			Required:    true,
			Weight:      0.1,
			Description: "Progress file includes work package sections",
		},
		{
			Name:        "TaskCheckboxes",
			Pattern:     regexp.MustCompile(`(?m)^- \[[ x~!s]\] (T\d+):`),
			Required:    true,
			Weight:      0.1,
			Description: "Progress file uses proper task checkbox format",
		},
	}

	totalScore := 0.0
	totalWeight := 0.0
	requiredPassed := true

	// Check each format pattern
	for _, pattern := range formatPatterns {
		patternResult := cd.checkPattern(content, pattern)
		result.PatternResults[pattern.Name] = patternResult

		score := 0.0
		if patternResult.Matched {
			score = pattern.Weight
		}
		totalScore += score
		totalWeight += pattern.Weight

		if pattern.Required && !patternResult.Matched {
			requiredPassed = false
			result.MissingElements = append(result.MissingElements, pattern.Description)
			result.ViolationCount++
		}
	}

	// Calculate overall score
	if totalWeight > 0 {
		result.OverallScore = totalScore / totalWeight
	}

	result.RequiredPassed = requiredPassed
	result.ComplianceLevel = cd.determineComplianceLevel(result.OverallScore)

	// Generate format-specific recommendations
	if !requiredPassed {
		result.Recommendations = append(result.Recommendations,
			"Ensure progress file follows the mandatory format specified in the implementation prompt")
	}

	return result
}

// ComplianceMetrics tracks compliance over time
type ComplianceMetrics struct {
	TotalChecks       int                        `json:"total_checks"`
	PassedChecks      int                        `json:"passed_checks"`
	AverageScore      float64                    `json:"average_score"`
	LastCheck         time.Time                  `json:"last_check"`
	RecentScores      []float64                  `json:"recent_scores"`      // Last 10 scores
	PatternSuccess    map[string]int             `json:"pattern_success"`    // Success count per pattern
	ImprovementTrend  float64                    `json:"improvement_trend"`  // Trend over recent checks
}