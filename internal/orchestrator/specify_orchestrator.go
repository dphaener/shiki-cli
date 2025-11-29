package orchestrator

import (
	"time"

	agentpkg "github.com/darinhaener/collab/internal/agent"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// SpecifyOrchestrator manages the specification workflow with a single agent.
// It embeds BaseOrchestrator for common functionality and adds session-specific behavior.
type SpecifyOrchestrator struct {
	*BaseOrchestrator
	session *types.SpecifySession
}

// NewSpecifyOrchestrator creates a new specify orchestrator
func NewSpecifyOrchestrator(session *types.SpecifySession, apiKey string, eventBus *events.EventBus) *SpecifyOrchestrator {
	base := NewBaseOrchestrator(
		session.ID,
		"specify",
		3*time.Minute,
		apiKey,
		eventBus,
	)

	// Set session-specific event metadata
	base.SetEventMetadata(map[string]interface{}{
		"feature_number": session.FeatureNumber,
		"slug":           session.Slug,
	})

	return &SpecifyOrchestrator{
		BaseOrchestrator: base,
		session:          session,
	}
}

// Initialize sets up the agent for the specify phase
func (o *SpecifyOrchestrator) Initialize() error {
	// Create specify agent configuration - this generates the instructions
	agentCfg := agentpkg.NewSpecifyAgent(o.session)

	// Delegate to BaseOrchestrator
	return o.InitializeWithAgent(agentCfg)
}

// GetSession returns the specify session (for orchestrators that need direct access)
func (o *SpecifyOrchestrator) GetSession() *types.SpecifySession {
	return o.session
}
