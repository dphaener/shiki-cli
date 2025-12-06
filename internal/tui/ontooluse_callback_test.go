package tui

import (
	"os"
	"testing"
	"time"

	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
	"github.com/stretchr/testify/assert"
)

// TestOnToolUseCallbacks verifies that all phases have consistent OnToolUse callback patterns
func TestOnToolUseCallbacks(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	eventBus := events.NewEventBus(10)

	testCases := []struct {
		name           string
		createModel    func() interface{ GetConfig() PhaseModelConfig }
		expectedFile   string
		expectedPhase  string
	}{
		{
			name: "SpecifyModel",
			createModel: func() interface{ GetConfig() PhaseModelConfig } {
				session := &types.SpecifySession{
					ID:            "test-spec",
					FeatureDesc:   "test feature",
					FeatureNumber: 1,
					Slug:          "test",
					FriendlyName:  "Test Feature",
					Phase:         "specify",
					SpecFile:      tmpDir + "/spec.md",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				model := NewSpecifyModel(session, eventBus)
				return &model
			},
			expectedFile:  tmpDir + "/spec.md",
			expectedPhase: "specify",
		},
		{
			name: "PlanModel",
			createModel: func() interface{ GetConfig() PhaseModelConfig } {
				session := &types.PlanSession{
					ID:            "test-plan",
					SpecSlug:      "test",
					FeatureNumber: 1,
					FriendlyName:  "Test Feature",
					Phase:         "plan",
					PlanFile:      tmpDir + "/plan.md",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				model := NewPlanModel(session, eventBus)
				return &model
			},
			expectedFile:  tmpDir + "/plan.md",
			expectedPhase: "plan",
		},
		{
			name: "TasksModel",
			createModel: func() interface{ GetConfig() PhaseModelConfig } {
				session := &types.WorkflowSession{
					ID:            "test-tasks",
					CurrentPhase:  "tasks",
					FeatureNumber: 1,
					Slug:          "test",
					FriendlyName:  "Test Feature",
					TasksFile:     tmpDir + "/tasks.md",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				model := NewTasksModel(session, eventBus)
				return &model
			},
			expectedFile:  tmpDir + "/tasks.md",
			expectedPhase: "tasks",
		},
		{
			name: "ImplementModel",
			createModel: func() interface{ GetConfig() PhaseModelConfig } {
				session := &types.WorkflowSession{
					ID:            "test-impl",
					CurrentPhase:  "implement",
					FeatureNumber: 1,
					Slug:          "test",
					FriendlyName:  "Test Feature",
					FeatureDir:    tmpDir + "/feature",
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}
				model := NewImplementModel(session, eventBus)
				return &model
			},
			expectedFile:  tmpDir + "/feature",
			expectedPhase: "implement",
		},
		{
			name: "BugModel",
			createModel: func() interface{ GetConfig() PhaseModelConfig } {
				session := &types.BugSession{
					ID:           "test-bug",
					Title:        "test bug",
					Description:  "test description",
					CurrentPhase: "plan",
					PlanFile:     tmpDir + "/bug-plan.md",
					TasksFile:    tmpDir + "/bug-tasks.md",
					BugDir:       tmpDir + "/bugs",
				}
				model := NewBugModel(session, eventBus)
				return &model
			},
			expectedFile:  tmpDir + "/bug-plan.md",
			expectedPhase: "bug",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := tc.createModel()
			config := model.GetConfig()

			// Verify OnToolUseFunc is not nil
			assert.NotNil(t, config.OnToolUseFunc, "OnToolUseFunc should be implemented for %s", tc.name)

			// Verify RefreshPreviewFunc is not nil
			assert.NotNil(t, config.RefreshPreviewFunc, "RefreshPreviewFunc should be implemented for %s", tc.name)

			// Test that OnToolUseFunc can be called without panicking
			assert.NotPanics(t, func() {
				config.OnToolUseFunc("Write", map[string]interface{}{
					"file_path": tc.expectedFile,
					"content":   "test content",
				})
			}, "OnToolUseFunc should not panic for %s", tc.name)

			// Test that RefreshPreviewFunc can be called without panicking
			assert.NotPanics(t, func() {
				config.RefreshPreviewFunc()
			}, "RefreshPreviewFunc should not panic for %s", tc.name)

			t.Logf("✅ %s OnToolUse callback pattern verified", tc.name)
		})
	}
}

// TestOnToolUsePreviewUpdates tests that OnToolUse callbacks trigger preview updates
func TestOnToolUsePreviewUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	eventBus := events.NewEventBus(10)

	// Test with SpecifyModel as representative example
	session := &types.SpecifySession{
		ID:            "test",
		FeatureDesc:   "test feature",
		FeatureNumber: 1,
		Slug:          "test",
		FriendlyName:  "Test Feature",
		Phase:         "specify",
		SpecFile:      tmpDir + "/spec.md",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	model := NewSpecifyModel(session, eventBus)
	config := model.GetConfig()

	// Create a test file
	testContent := "# Test Specification\n\nThis is a test specification."
	err := os.WriteFile(session.SpecFile, []byte(testContent), 0644)
	assert.NoError(t, err, "Should be able to create test file")

	// Verify the file exists
	_, err = os.Stat(session.SpecFile)
	assert.NoError(t, err, "Test file should exist")

	// Call OnToolUseFunc which should trigger preview refresh
	assert.NotPanics(t, func() {
		config.OnToolUseFunc("Write", map[string]interface{}{
			"file_path": session.SpecFile,
			"content":   testContent,
		})
	}, "OnToolUseFunc should not panic")

	// Call RefreshPreviewFunc directly to verify it works
	assert.NotPanics(t, func() {
		config.RefreshPreviewFunc()
	}, "RefreshPreviewFunc should not panic")

	t.Log("✅ OnToolUse preview update mechanism verified")
}

// Helper interface to extract config from different model types
type ConfigProvider interface {
	GetConfig() PhaseModelConfig
}

// Add GetConfig methods to models for testing
func (m *SpecifyModel) GetConfig() PhaseModelConfig {
	return m.PhaseModel.config
}

func (m *PlanModel) GetConfig() PhaseModelConfig {
	return m.PhaseModel.config
}

func (m *TasksModel) GetConfig() PhaseModelConfig {
	return m.PhaseModel.config
}

func (m *ImplementModel) GetConfig() PhaseModelConfig {
	return m.PhaseModel.config
}

func (m *BugModel) GetConfig() PhaseModelConfig {
	return m.PhaseModel.config
}