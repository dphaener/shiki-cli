package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PhaseContext contains the unified data structure for template variable substitution
// across all workflow phases. This replaces custom data structures like SpecifyPromptData,
// PlanPromptData, TasksPromptData, etc.
type PhaseContext struct {
	// Core identification
	FriendlyName  string
	FeatureNumber int
	Slug          string
	Phase         string

	// File paths
	SpecFile     string
	PlanFile     string
	TasksFile    string
	SpecDir      string
	ContractsDir string
	ChecklistDir string
	FeatureDir   string
	BugDir       string

	// Phase-specific data for Specify phase
	HasDescription bool
	FeatureDesc    string

	// Phase-specific data for Bug phase
	BugID       string
	Title       string
	Description string

	// Collaboration-specific data
	AgentName           string
	AgentRole           string
	AgentID             string
	PartnerName         string
	TurnNumber          int
	TurnNumberPadded    string
	MaxTurns           int
	Task               string
	Messages           string
	SharedContext      string
	Memory             string

	// Computed/dynamic properties (populated by helper methods)
	Timestamp      string
	CurrentDate    string
	CurrentTime    string
	WorkingDir     string
	ProjectName    string
	IsFirstPhase   bool
	IsLastPhase    bool
	PhaseProgress  float64
	FileExists     map[string]bool
	FileContents   map[string]string
	DirectoryList  []string

	// Template metadata
	TemplateName   string
	TemplateVars   map[string]interface{}
	CustomData     map[string]interface{}
}

// NewPhaseContext creates a PhaseContext with common fields populated
func NewPhaseContext() *PhaseContext {
	ctx := &PhaseContext{
		FileExists:   make(map[string]bool),
		FileContents: make(map[string]string),
		TemplateVars: make(map[string]interface{}),
		CustomData:   make(map[string]interface{}),
	}
	ctx.WithTimestamps()
	return ctx
}

// WithCore populates the core identification fields
func (pc *PhaseContext) WithCore(friendlyName string, featureNumber int, slug, phase string) *PhaseContext {
	pc.FriendlyName = friendlyName
	pc.FeatureNumber = featureNumber
	pc.Slug = slug
	pc.Phase = phase
	return pc
}

// WithPaths populates the file path fields
func (pc *PhaseContext) WithPaths(specFile, planFile, tasksFile, specDir, contractsDir, checklistDir, featureDir, bugDir string) *PhaseContext {
	pc.SpecFile = specFile
	pc.PlanFile = planFile
	pc.TasksFile = tasksFile
	pc.SpecDir = specDir
	pc.ContractsDir = contractsDir
	pc.ChecklistDir = checklistDir
	pc.FeatureDir = featureDir
	pc.BugDir = bugDir
	return pc
}

// WithSpecifyData populates specify phase specific fields
func (pc *PhaseContext) WithSpecifyData(hasDescription bool, featureDesc string) *PhaseContext {
	pc.HasDescription = hasDescription
	pc.FeatureDesc = featureDesc
	return pc
}

// WithBugData populates bug phase specific fields
func (pc *PhaseContext) WithBugData(bugID, title, description string) *PhaseContext {
	pc.BugID = bugID
	pc.Title = title
	pc.Description = description
	return pc
}

// WithCollaborationData populates collaboration specific fields
func (pc *PhaseContext) WithCollaborationData(agentName, agentRole, agentID, partnerName string, turnNumber, maxTurns int, task, messages, sharedContext, memory string) *PhaseContext {
	pc.AgentName = agentName
	pc.AgentRole = agentRole
	pc.AgentID = agentID
	pc.PartnerName = partnerName
	pc.TurnNumber = turnNumber
	pc.TurnNumberPadded = fmt.Sprintf("%02d", turnNumber)
	pc.MaxTurns = maxTurns
	pc.Task = task
	pc.Messages = messages
	pc.SharedContext = sharedContext
	pc.Memory = memory
	return pc
}

// WithTimestamps populates timestamp-related fields
func (pc *PhaseContext) WithTimestamps() *PhaseContext {
	now := time.Now()
	pc.Timestamp = now.Format(time.RFC3339)
	pc.CurrentDate = now.Format("2006-01-02")
	pc.CurrentTime = now.Format("15:04:05")
	return pc
}

// WithEnvironment populates environment-related fields
func (pc *PhaseContext) WithEnvironment() *PhaseContext {
	if wd, err := os.Getwd(); err == nil {
		pc.WorkingDir = wd
		pc.ProjectName = filepath.Base(wd)
	}
	return pc
}

// WithPhaseInfo populates phase progression information
func (pc *PhaseContext) WithPhaseInfo(isFirst, isLast bool, progress float64) *PhaseContext {
	pc.IsFirstPhase = isFirst
	pc.IsLastPhase = isLast
	pc.PhaseProgress = progress
	return pc
}

// WithFileCheck checks if specified files exist and populates FileExists map
func (pc *PhaseContext) WithFileCheck(filePaths ...string) *PhaseContext {
	for _, path := range filePaths {
		if _, err := os.Stat(path); err == nil {
			pc.FileExists[path] = true
		} else {
			pc.FileExists[path] = false
		}
	}
	return pc
}

// WithFileContent loads content from specified files
func (pc *PhaseContext) WithFileContent(filePaths ...string) *PhaseContext {
	for _, path := range filePaths {
		if content, err := os.ReadFile(path); err == nil {
			pc.FileContents[path] = string(content)
		} else {
			pc.FileContents[path] = ""
		}
	}
	return pc
}

// WithDirectoryList populates DirectoryList with files in the specified directory
func (pc *PhaseContext) WithDirectoryList(dirPath string) *PhaseContext {
	if entries, err := os.ReadDir(dirPath); err == nil {
		pc.DirectoryList = make([]string, 0, len(entries))
		for _, entry := range entries {
			pc.DirectoryList = append(pc.DirectoryList, entry.Name())
		}
	}
	return pc
}

// WithTemplateMetadata sets template-specific metadata
func (pc *PhaseContext) WithTemplateMetadata(templateName string, vars map[string]interface{}) *PhaseContext {
	pc.TemplateName = templateName
	if vars != nil {
		for k, v := range vars {
			pc.TemplateVars[k] = v
		}
	}
	return pc
}

// WithCustomData adds custom key-value data to the context
func (pc *PhaseContext) WithCustomData(key string, value interface{}) *PhaseContext {
	pc.CustomData[key] = value
	return pc
}

// Get retrieves a custom data value by key
func (pc *PhaseContext) Get(key string) interface{} {
	return pc.CustomData[key]
}

// Set stores a custom data value by key
func (pc *PhaseContext) Set(key string, value interface{}) {
	pc.CustomData[key] = value
}

// Has checks if a custom data key exists
func (pc *PhaseContext) Has(key string) bool {
	_, exists := pc.CustomData[key]
	return exists
}

// FileExistsCheck returns true if the specified file exists
func (pc *PhaseContext) FileExistsCheck(path string) bool {
	if exists, found := pc.FileExists[path]; found {
		return exists
	}
	// Dynamic check if not in cache
	_, err := os.Stat(path)
	return err == nil
}

// GetFileContent returns the content of the specified file
func (pc *PhaseContext) GetFileContent(path string) string {
	if content, found := pc.FileContents[path]; found {
		return content
	}
	// Dynamic load if not in cache
	if content, err := os.ReadFile(path); err == nil {
		return string(content)
	}
	return ""
}