package orchestrator

import (
	"time"

	agentpkg "github.com/dphaener/shiki-cli/internal/agent"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// ImplementOrchestrator manages the implementation workflow with a single agent.
// It embeds BaseOrchestrator for common functionality and adds session-specific behavior.
type ImplementOrchestrator struct {
	*BaseOrchestrator
	session *types.WorkflowSession
}

// NewImplementOrchestrator creates a new implement orchestrator
func NewImplementOrchestrator(session *types.WorkflowSession, apiKey string, eventBus *events.EventBus) *ImplementOrchestrator {
	base := NewBaseOrchestrator(
		session.ID,
		"implement",
		10*time.Minute, // Longer timeout for implementation work
		apiKey,
		eventBus,
	)

	// Set session-specific event metadata
	base.SetEventMetadata(map[string]interface{}{
		"feature_number": session.FeatureNumber,
		"slug":           session.Slug,
	})

	return &ImplementOrchestrator{
		BaseOrchestrator: base,
		session:          session,
	}
}

// Initialize sets up the agent for the implement phase
func (o *ImplementOrchestrator) Initialize() error {
	// Create implement agent configuration - this generates the instructions
	agentCfg := agentpkg.NewImplementAgent(o.session)

	// Delegate to BaseOrchestrator
	return o.InitializeWithAgent(agentCfg)
}

// GetSession returns the workflow session (for orchestrators that need direct access)
func (o *ImplementOrchestrator) GetSession() *types.WorkflowSession {
	return o.session
}
