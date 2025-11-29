package orchestrator

import (
	"time"

	agentpkg "github.com/darinhaener/collab/internal/agent"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// BugOrchestrator manages bug fix workflow orchestration.
// It embeds BaseOrchestrator for common functionality and adds session-specific behavior.
type BugOrchestrator struct {
	*BaseOrchestrator
	session *types.BugSession
}

// NewBugOrchestrator creates a new bug orchestrator
func NewBugOrchestrator(session *types.BugSession, apiKey string, eventBus *events.EventBus) *BugOrchestrator {
	base := NewBaseOrchestrator(
		session.ID,
		"bug",
		3*time.Minute,
		apiKey,
		eventBus,
	)

	// Set session-specific event metadata
	base.SetEventMetadata(map[string]interface{}{
		"bug_id": session.ID,
		"phase":  string(session.CurrentPhase),
	})

	return &BugOrchestrator{
		BaseOrchestrator: base,
		session:          session,
	}
}

// Initialize sets up the agent for the bug orchestrator
func (o *BugOrchestrator) Initialize() error {
	// Create bug agent configuration
	agentCfg := agentpkg.NewBugAgent(o.session)

	// Delegate to BaseOrchestrator
	return o.InitializeWithAgent(agentCfg)
}

// SetPlanInstructions sets the instructions to be injected in the first message
func (o *BugOrchestrator) SetPlanInstructions(instructions string) {
	o.SetInstructions(instructions)
}

// GetSession returns the bug session
func (o *BugOrchestrator) GetSession() *types.BugSession {
	return o.session
}

// UpdateSession updates the bug session
func (o *BugOrchestrator) UpdateSession(session *types.BugSession) {
	o.session = session
	// Update event metadata with new phase
	o.SetEventMetadata(map[string]interface{}{
		"bug_id": session.ID,
		"phase":  string(session.CurrentPhase),
	})
}

// Close cleans up the orchestrator resources
func (o *BugOrchestrator) Close() error {
	return o.Stop()
}

// GetEventBus returns the event bus
func (o *BugOrchestrator) GetEventBus() *events.EventBus {
	return o.eventBus
}

// SetEventBus sets the event bus
func (o *BugOrchestrator) SetEventBus(eventBus *events.EventBus) {
	o.eventBus = eventBus
}
