package orchestrator

import (
	"time"

	agentpkg "github.com/darinhaener/collab/internal/agent"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// TasksOrchestrator manages the tasks generation workflow with a single agent.
// It embeds BaseOrchestrator for common functionality and adds session-specific behavior.
type TasksOrchestrator struct {
	*BaseOrchestrator
	session *types.WorkflowSession
}

// NewTasksOrchestrator creates a new tasks orchestrator
func NewTasksOrchestrator(session *types.WorkflowSession, apiKey string, eventBus *events.EventBus) *TasksOrchestrator {
	base := NewBaseOrchestrator(
		session.ID,
		"tasks",
		5*time.Minute, // Longer timeout for tasks generation
		apiKey,
		eventBus,
	)

	// Set session-specific event metadata
	base.SetEventMetadata(map[string]interface{}{
		"feature_number": session.FeatureNumber,
		"slug":           session.Slug,
	})

	return &TasksOrchestrator{
		BaseOrchestrator: base,
		session:          session,
	}
}

// Initialize sets up the agent for the tasks phase
func (o *TasksOrchestrator) Initialize() error {
	// Create tasks agent configuration - this generates the instructions
	agentCfg := agentpkg.NewTasksAgent(o.session)

	// Delegate to BaseOrchestrator
	return o.InitializeWithAgent(agentCfg)
}

// GetSession returns the workflow session (for orchestrators that need direct access)
func (o *TasksOrchestrator) GetSession() *types.WorkflowSession {
	return o.session
}
