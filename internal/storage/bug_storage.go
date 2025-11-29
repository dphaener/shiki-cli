package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/darinhaener/collab/internal/config"
	"github.com/darinhaener/collab/pkg/types"
)

// BugSetupResult contains the result of bug workflow setup
type BugSetupResult struct {
	BugID    string
	BugDir   string
	PlanFile string
	TasksFile string
}

// BugMeta represents bug workflow state stored in bug-workflow.json
type BugMeta struct {
	SessionID    string                                    `json:"session_id"`
	CurrentPhase types.BugPhase                            `json:"current_phase"`
	Checkpoints  map[types.BugPhase]*types.PhaseCheckpoint `json:"checkpoints"`
	CreatedAt    time.Time                                 `json:"created_at"`
	UpdatedAt    time.Time                                 `json:"updated_at"`
	Status       string                                    `json:"status"`
	Title        string                                    `json:"title"`
	Description  string                                    `json:"description"`
}

// GetBugsDir returns the base directory for bug workflows
func GetBugsDir() string {
	return filepath.Join(config.GetDataDir(), "bugs")
}

// GetBugDir returns the directory for a specific bug
func GetBugDir(bugID string) string {
	return filepath.Join(GetBugsDir(), bugID)
}

// GetBugMetaPath returns the path to bug-workflow.json for a bug
func GetBugMetaPath(bugID string) string {
	return filepath.Join(GetBugDir(bugID), "bug-workflow.json")
}

// GetBugPlanPath returns the path to bug-plan.md for a bug
func GetBugPlanPath(bugID string) string {
	return filepath.Join(GetBugDir(bugID), "bug-plan.md")
}

// GetBugTasksPath returns the path to bug-tasks.md for a bug
func GetBugTasksPath(bugID string) string {
	return filepath.Join(GetBugDir(bugID), "bug-tasks.md")
}

// BugSetupConfig contains configuration for setting up a new bug workflow
type BugSetupConfig struct {
	Title       string
	Description string
}

// SetupBugSession sets up a new bug workflow session
func SetupBugSession(cfg BugSetupConfig) (*BugSetupResult, error) {
	// Generate bug ID from title (similar to feature slug generation)
	bugID := generateBugID(cfg.Title)
	bugDir := GetBugDir(bugID)

	// Create bug directory structure
	if err := os.MkdirAll(bugDir, 0o755); err != nil {
		return nil, fmt.Errorf("create bug directory: %w", err)
	}

	// Define file paths
	planFile := GetBugPlanPath(bugID)
	tasksFile := GetBugTasksPath(bugID)

	// Create initial bug-plan.md with frontmatter
	planContent := fmt.Sprintf(`---
bug_id: %s
title: "%s"
created: %s
phase: plan
---

# Bug Fix Plan: %s

## Problem Description

%s

## Analysis

_This section will be populated during the plan phase_

## Solution Approach

_This section will be populated during the plan phase_

## Impact Assessment

_This section will be populated during the plan phase_

## Testing Strategy

_This section will be populated during the plan phase_
`, bugID, cfg.Title, time.Now().Format(time.RFC3339), cfg.Title, cfg.Description)

	if err := AtomicWrite(planFile, []byte(planContent), 0o644); err != nil {
		return nil, fmt.Errorf("create bug-plan.md: %w", err)
	}

	return &BugSetupResult{
		BugID:     bugID,
		BugDir:    bugDir,
		PlanFile:  planFile,
		TasksFile: tasksFile,
	}, nil
}

// SaveBugSession persists the bug session state
func SaveBugSession(session *types.BugSession) error {
	metaPath := GetBugMetaPath(session.ID)

	meta := BugMeta{
		SessionID:    session.ID,
		CurrentPhase: session.CurrentPhase,
		Checkpoints:  session.Checkpoints,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    time.Now(),
		Status:       string(session.Status),
		Title:        session.Title,
		Description:  session.Description,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bug-workflow.json: %w", err)
	}

	if err := AtomicWrite(metaPath, data, 0o644); err != nil {
		return fmt.Errorf("write bug-workflow.json: %w", err)
	}

	return nil
}

// LoadBugSession loads an existing bug session
func LoadBugSession(bugID string) (*types.BugSession, error) {
	// Load bug metadata
	metaPath := GetBugMetaPath(bugID)
	var meta BugMeta

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read bug-workflow.json: %w", err)
	}

	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse bug-workflow.json: %w", err)
	}

	// Build paths
	bugDir := GetBugDir(bugID)
	planFile := GetBugPlanPath(bugID)
	tasksFile := GetBugTasksPath(bugID)

	// Load current content if files exist
	var currentPlan, currentTasks string
	if content, err := os.ReadFile(planFile); err == nil {
		currentPlan = stripFrontmatter(string(content))
	}
	if content, err := os.ReadFile(tasksFile); err == nil {
		currentTasks = string(content)
	}

	session := &types.BugSession{
		ID:           meta.SessionID,
		Title:        meta.Title,
		Description:  meta.Description,
		CurrentPhase: meta.CurrentPhase,
		PlanFile:     planFile,
		TasksFile:    tasksFile,
		BugDir:       bugDir,
		Checkpoints:  meta.Checkpoints,
		CurrentPlan:  currentPlan,
		CurrentTasks: currentTasks,
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
		Status:       types.SessionStatus(meta.Status),
	}

	return session, nil
}

// BugExists checks if a bug session exists for the given bug ID
func BugExists(bugID string) bool {
	metaPath := GetBugMetaPath(bugID)
	_, err := os.Stat(metaPath)
	return err == nil
}

// ListActiveBugs returns all bug workflows that are not complete
func ListActiveBugs() ([]*types.BugSession, error) {
	bugsDir := GetBugsDir()

	// Check if bugs directory exists
	if _, err := os.Stat(bugsDir); os.IsNotExist(err) {
		return []*types.BugSession{}, nil
	}

	entries, err := os.ReadDir(bugsDir)
	if err != nil {
		return nil, fmt.Errorf("read bugs directory: %w", err)
	}

	var bugs []*types.BugSession
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		bugID := entry.Name()
		if BugExists(bugID) {
			session, err := LoadBugSession(bugID)
			if err != nil {
				continue // Skip invalid bug sessions
			}
			if session.CurrentPhase != types.BugPhaseComplete {
				bugs = append(bugs, session)
			}
		}
	}

	return bugs, nil
}

// SaveBugCheckpoint saves a checkpoint for a completed phase
func SaveBugCheckpoint(session *types.BugSession, phase types.BugPhase, chatHistory []types.ChatMessage, artifactPath string) error {
	if session.Checkpoints == nil {
		session.Checkpoints = make(map[types.BugPhase]*types.PhaseCheckpoint)
	}

	session.Checkpoints[phase] = &types.PhaseCheckpoint{
		Phase:       types.WorkflowPhase(phase),
		CompletedAt: time.Now(),
		ChatHistory: chatHistory,
		ArtifactRef: artifactPath,
	}

	return SaveBugSession(session)
}

// GetBugCheckpoint retrieves a checkpoint for a specific phase
func GetBugCheckpoint(session *types.BugSession, phase types.BugPhase) *types.PhaseCheckpoint {
	if session.Checkpoints == nil {
		return nil
	}
	return session.Checkpoints[phase]
}

// BugTasksFileExists checks if bug-tasks.md exists for a bug
func BugTasksFileExists(bugID string) bool {
	tasksPath := GetBugTasksPath(bugID)
	_, err := os.Stat(tasksPath)
	return err == nil
}

// LoadBugTasksContent loads the content of bug-tasks.md
func LoadBugTasksContent(bugID string) (string, error) {
	tasksPath := GetBugTasksPath(bugID)
	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return "", fmt.Errorf("read bug-tasks.md: %w", err)
	}
	return string(content), nil
}

// generateBugID creates a bug ID from the title, similar to feature slug generation
func generateBugID(title string) string {
	// Get the next bug number
	bugNumber := getNextBugNumber()

	// Create slug from title using the same pattern as features
	return CreateBugSlugFromTitle(title, bugNumber)
}

// CreateBugSlugFromTitle converts a bug title to a slug format following feature patterns
func CreateBugSlugFromTitle(title string, bugNumber int) string {
	// Trim whitespace first
	name := strings.TrimSpace(title)

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
		slug = "bug"
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

	// Add bug number prefix (similar to features)
	return fmt.Sprintf("%03d-%s", bugNumber, slug)
}

// getNextBugNumber gets the next bug number similar to how features work
func getNextBugNumber() int {
	// Get bugs directory
	bugsDir := GetBugsDir()

	// Check if bugs directory exists
	if _, err := os.Stat(bugsDir); os.IsNotExist(err) {
		return 1 // First bug
	}

	entries, err := os.ReadDir(bugsDir)
	if err != nil {
		return 1 // Default to 1 if can't read directory
	}

	maxNumber := 0
	bugPattern := regexp.MustCompile(`^(\d{3})-`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Extract number from directory name like "001-bug-title"
		matches := bugPattern.FindStringSubmatch(entry.Name())
		if len(matches) > 1 {
			var number int
			if _, err := fmt.Sscanf(matches[1], "%d", &number); err == nil {
				if number > maxNumber {
					maxNumber = number
				}
			}
		}
	}

	return maxNumber + 1
}

// HasActiveBug checks if there are any active bug workflows
func HasActiveBug() (bool, error) {
	bugs, err := ListActiveBugs()
	if err != nil {
		return false, err
	}
	return len(bugs) > 0, nil
}

// GetActiveBug returns the first active bug workflow (since we enforce only one active bug)
func GetActiveBug() (*types.BugSession, error) {
	bugs, err := ListActiveBugs()
	if err != nil {
		return nil, err
	}
	if len(bugs) == 0 {
		return nil, fmt.Errorf("no active bug workflows")
	}
	return bugs[0], nil
}