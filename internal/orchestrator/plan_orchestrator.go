package orchestrator

import (
	"time"

	agentpkg "github.com/darinhaener/collab/internal/agent"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// PlanOrchestrator manages the planning workflow with a single agent.
// It embeds BaseOrchestrator for common functionality and adds session-specific behavior.
type PlanOrchestrator struct {
	*BaseOrchestrator
	session *types.PlanSession
}

// NewPlanOrchestrator creates a new plan orchestrator
func NewPlanOrchestrator(session *types.PlanSession, apiKey string, eventBus *events.EventBus) *PlanOrchestrator {
	base := NewBaseOrchestrator(
		session.ID,
		"plan",
		3*time.Minute,
		apiKey,
		eventBus,
	)

	// Set session-specific event metadata
	base.SetEventMetadata(map[string]interface{}{
		"feature_number": session.FeatureNumber,
		"spec_slug":      session.SpecSlug,
	})

	return &PlanOrchestrator{
		BaseOrchestrator: base,
		session:          session,
	}
}

// Initialize sets up the agent for the plan phase
func (o *PlanOrchestrator) Initialize() error {
	// Create plan agent configuration - this generates the instructions
	agentCfg := agentpkg.NewPlanAgent(o.session)

	// Delegate to BaseOrchestrator
	return o.InitializeWithAgent(agentCfg)
}

// GetSession returns the plan session (for orchestrators that need direct access)
func (o *PlanOrchestrator) GetSession() *types.PlanSession {
	return o.session
}
