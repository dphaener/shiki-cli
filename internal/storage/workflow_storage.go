package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dphaener/shiki-cli/internal/config"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// WorkflowSetupResult contains the result of workflow setup
type WorkflowSetupResult struct {
	FeatureNumber int
	Slug          string
	FeatureDir    string
	SpecFile      string
	PlanFile      string
	TasksFile     string
	ChecklistDir  string
	ContractsDir  string
}

// WorkflowMeta represents workflow state stored in workflow.json
type WorkflowMeta struct {
	SessionID    string                                       `json:"session_id"`
	CurrentPhase types.WorkflowPhase                          `json:"current_phase"`
	Checkpoints  map[types.WorkflowPhase]*types.PhaseCheckpoint `json:"checkpoints"`
	CreatedAt    time.Time                                    `json:"created_at"`
	UpdatedAt    time.Time                                    `json:"updated_at"`
	Status       string                                       `json:"status"`
}

// GetWorkflowsDir returns the base directory for workflow state
func GetWorkflowsDir() string {
	return filepath.Join(config.GetDataDir(), "workflows")
}

// GetWorkflowDir returns the directory for a specific workflow
func GetWorkflowDir(slug string) string {
	return filepath.Join(GetWorkflowsDir(), slug)
}

// GetWorkflowMetaPath returns the path to workflow.json for a feature
func GetWorkflowMetaPath(slug string) string {
	// Store workflow state in the feature's spec directory for cohesion
	return filepath.Join(GetSpecDir(slug), "workflow.json")
}

// SetupWorkflowSession sets up a new unified workflow session
func SetupWorkflowSession(cfg FeatureSetupConfig) (*WorkflowSetupResult, error) {
	// First, set up the feature directory structure (same as specify)
	featureResult, err := SetupFeature(cfg)
	if err != nil {
		return nil, fmt.Errorf("setup feature: %w", err)
	}

	// Define additional paths for workflow
	specDir := featureResult.FeatureDir
	planFile := filepath.Join(specDir, "plan.md")
	tasksFile := filepath.Join(specDir, "tasks.md")
	contractsDir := filepath.Join(specDir, "contracts")

	// Create contracts directory
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create contracts directory: %w", err)
	}

	return &WorkflowSetupResult{
		FeatureNumber: featureResult.FeatureNumber,
		Slug:          featureResult.Slug,
		FeatureDir:    featureResult.FeatureDir,
		SpecFile:      featureResult.SpecFile,
		PlanFile:      planFile,
		TasksFile:     tasksFile,
		ChecklistDir:  featureResult.ChecklistDir,
		ContractsDir:  contractsDir,
	}, nil
}

// SaveWorkflowSession persists the workflow session state
func SaveWorkflowSession(session *types.WorkflowSession) error {
	metaPath := GetWorkflowMetaPath(session.Slug)

	meta := WorkflowMeta{
		SessionID:    session.ID,
		CurrentPhase: session.CurrentPhase,
		Checkpoints:  session.Checkpoints,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    time.Now(),
		Status:       string(session.Status),
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workflow.json: %w", err)
	}

	if err := AtomicWrite(metaPath, data, 0o644); err != nil {
		return fmt.Errorf("write workflow.json: %w", err)
	}

	return nil
}

// LoadWorkflowSession loads an existing workflow session
func LoadWorkflowSession(slug string) (*types.WorkflowSession, error) {
	// Load feature metadata first
	spec, err := LoadSpec(slug)
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}

	// Load workflow metadata
	metaPath := GetWorkflowMetaPath(slug)
	var meta WorkflowMeta

	if data, err := os.ReadFile(metaPath); err == nil {
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, fmt.Errorf("parse workflow.json: %w", err)
		}
	} else if os.IsNotExist(err) {
		// No workflow state yet - start fresh
		meta = WorkflowMeta{
			SessionID:    fmt.Sprintf("workflow-%d-%s", time.Now().Unix(), slug),
			CurrentPhase: types.WorkflowPhaseSpecify,
			Checkpoints:  make(map[types.WorkflowPhase]*types.PhaseCheckpoint),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Status:       "running",
		}
	} else {
		return nil, fmt.Errorf("read workflow.json: %w", err)
	}

	// Build paths
	specDir := GetSpecDir(slug)
	specFile := filepath.Join(specDir, "spec.md")
	planFile := filepath.Join(specDir, "plan.md")
	tasksFile := filepath.Join(specDir, "tasks.md")
	checklistDir := filepath.Join(specDir, "checklists")
	contractsDir := filepath.Join(specDir, "contracts")

	// Load current content if files exist
	var currentSpec, currentPlan, currentTasks string
	if content, err := os.ReadFile(specFile); err == nil {
		currentSpec = stripFrontmatter(string(content))
	}
	if content, err := os.ReadFile(planFile); err == nil {
		currentPlan = stripFrontmatter(string(content))
	}
	if content, err := os.ReadFile(tasksFile); err == nil {
		currentTasks = string(content)
	}

	session := &types.WorkflowSession{
		ID:            meta.SessionID,
		CurrentPhase:  meta.CurrentPhase,
		FeatureNumber: spec.Number,
		Slug:          spec.Slug,
		FriendlyName:  spec.FeatureName,
		FeatureDesc:   spec.FeatureName,
		SpecFile:      specFile,
		PlanFile:      planFile,
		TasksFile:     tasksFile,
		FeatureDir:    specDir,
		ChecklistDir:  checklistDir,
		ContractsDir:  contractsDir,
		Checkpoints:   meta.Checkpoints,
		CurrentSpec:   currentSpec,
		CurrentPlan:   currentPlan,
		CurrentTasks:  currentTasks,
		CreatedAt:     meta.CreatedAt,
		UpdatedAt:     meta.UpdatedAt,
		Status:        types.SessionStatus(meta.Status),
	}

	return session, nil
}

// stripFrontmatter removes YAML frontmatter from markdown content
func stripFrontmatter(content string) string {
	if len(content) < 3 || content[:3] != "---" {
		return content
	}

	// Find the second "---"
	end := 3
	for i := 3; i < len(content)-2; i++ {
		if content[i:i+3] == "---" {
			// Skip the "---" and any following newline
			end = i + 3
			if end < len(content) && content[end] == '\n' {
				end++
			}
			break
		}
	}

	if end >= len(content) {
		return content
	}

	return content[end:]
}

// WorkflowExists checks if a workflow session exists for the given slug
func WorkflowExists(slug string) bool {
	metaPath := GetWorkflowMetaPath(slug)
	_, err := os.Stat(metaPath)
	return err == nil
}

// ListActiveWorkflows returns all workflows that are not complete
func ListActiveWorkflows() ([]*types.WorkflowSession, error) {
	specs, err := ListSpecs()
	if err != nil {
		return nil, err
	}

	var workflows []*types.WorkflowSession
	for _, spec := range specs {
		if WorkflowExists(spec.Slug) {
			session, err := LoadWorkflowSession(spec.Slug)
			if err != nil {
				continue // Skip invalid workflows
			}
			if session.CurrentPhase != types.WorkflowPhaseComplete {
				workflows = append(workflows, session)
			}
		}
	}

	return workflows, nil
}

// SaveCheckpoint saves a checkpoint for a completed phase
func SaveCheckpoint(session *types.WorkflowSession, phase types.WorkflowPhase, chatHistory []types.ChatMessage, artifactPath string) error {
	if session.Checkpoints == nil {
		session.Checkpoints = make(map[types.WorkflowPhase]*types.PhaseCheckpoint)
	}

	session.Checkpoints[phase] = &types.PhaseCheckpoint{
		Phase:       phase,
		CompletedAt: time.Now(),
		ChatHistory: chatHistory,
		ArtifactRef: artifactPath,
	}

	return SaveWorkflowSession(session)
}

// GetCheckpoint retrieves a checkpoint for a specific phase
func GetCheckpoint(session *types.WorkflowSession, phase types.WorkflowPhase) *types.PhaseCheckpoint {
	if session.Checkpoints == nil {
		return nil
	}
	return session.Checkpoints[phase]
}

// TasksFileExists checks if tasks.md exists for a feature
func TasksFileExists(slug string) bool {
	tasksPath := filepath.Join(GetSpecDir(slug), "tasks.md")
	_, err := os.Stat(tasksPath)
	return err == nil
}

// LoadTasksContent loads the content of tasks.md
func LoadTasksContent(slug string) (string, error) {
	tasksPath := filepath.Join(GetSpecDir(slug), "tasks.md")
	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return "", fmt.Errorf("read tasks.md: %w", err)
	}
	return string(content), nil
}
