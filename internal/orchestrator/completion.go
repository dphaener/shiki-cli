package orchestrator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dphaener/shiki-cli/pkg/types"
)

// DeliverableStatus tracks deliverable submissions from both agents
type DeliverableStatus struct {
	Agent1Submitted bool
	Agent2Submitted bool
	Agent1Path      string
	Agent2Path      string
	Agent1Content   []byte
	Agent2Content   []byte
}

// CheckCompletion determines if the session should be marked as completed
// Returns true if both agents have submitted deliverables
func (o *Orchestrator) CheckCompletion() (bool, string, error) {
	status, err := o.getDeliverableStatus()
	if err != nil {
		return false, "", fmt.Errorf("get deliverable status: %w", err)
	}

	// Both agents must submit
	if !status.Agent1Submitted || !status.Agent2Submitted {
		return false, "", nil
	}

	// Both submitted - create merged deliverable
	deliverablePath := filepath.Join(o.session.WorkspaceDir, "deliverable.md")

	var mergedContent []byte
	if bytes.Equal(status.Agent1Content, status.Agent2Content) {
		// Identical content - use as-is
		mergedContent = status.Agent1Content
	} else {
		// Different content - combine both with clear separation
		// Use agent2's version as primary (they saw agent1's work)
		// but include agent1's for completeness
		mergedContent = []byte(fmt.Sprintf("# Final Deliverable\n\n## Agent 2 (Approver) Version\n\n%s\n\n---\n\n## Agent 1 (Proposer) Version\n\n%s",
			string(status.Agent2Content),
			string(status.Agent1Content),
		))
	}

	if err := os.WriteFile(deliverablePath, mergedContent, 0o600); err != nil {
		return false, "", fmt.Errorf("write merged deliverable: %w", err)
	}

	return true, deliverablePath, nil
}

// getDeliverableStatus checks if agents have submitted deliverables
func (o *Orchestrator) getDeliverableStatus() (*DeliverableStatus, error) {
	status := &DeliverableStatus{}

	// Check for agent1 deliverable
	agent1Path := filepath.Join(o.session.WorkspaceDir, "agent_1_deliverable.md")
	if content, err := os.ReadFile(agent1Path); err == nil {
		status.Agent1Submitted = true
		status.Agent1Path = agent1Path
		status.Agent1Content = content
	}

	// Check for agent2 deliverable
	agent2Path := filepath.Join(o.session.WorkspaceDir, "agent_2_deliverable.md")
	if content, err := os.ReadFile(agent2Path); err == nil {
		status.Agent2Submitted = true
		status.Agent2Path = agent2Path
		status.Agent2Content = content
	}

	return status, nil
}

// HasReachedMaxTurns checks if the session has reached max turns without completion
func (o *Orchestrator) HasReachedMaxTurns() bool {
	return o.session.CurrentTurn >= o.session.MaxTurns
}

// ShouldContinue determines if the orchestrator should continue executing turns
func (o *Orchestrator) ShouldContinue() (bool, string, error) {
	// Check for completion first
	completed, deliverablePath, err := o.CheckCompletion()
	if err != nil {
		return false, "", fmt.Errorf("check completion: %w", err)
	}
	if completed {
		return false, deliverablePath, nil
	}

	// Check if max turns reached
	if o.HasReachedMaxTurns() {
		return false, "", nil
	}

	// Check session status
	if o.session.Status != types.SessionRunning {
		return false, "", nil
	}

	// Continue execution
	return true, "", nil
}

// GetCompletionSummary returns a human-readable summary of completion status
func (o *Orchestrator) GetCompletionSummary() string {
	status, err := o.getDeliverableStatus()
	if err != nil {
		return fmt.Sprintf("Error checking deliverables: %v", err)
	}

	if status.Agent1Submitted && status.Agent2Submitted {
		if bytes.Equal(status.Agent1Content, status.Agent2Content) {
			return "Both agents submitted identical deliverables - COMPLETED"
		}
		return "Both agents submitted deliverables (merged) - COMPLETED"
	}

	if status.Agent1Submitted {
		return "Only Agent 1 submitted deliverable - WAITING FOR AGENT 2"
	}

	if status.Agent2Submitted {
		return "Only Agent 2 submitted deliverable - WAITING FOR AGENT 1"
	}

	return "No deliverables submitted yet"
}
