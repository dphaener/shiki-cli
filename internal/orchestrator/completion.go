package orchestrator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/darinhaener/collab/pkg/types"
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
// Returns true if both agents have submitted matching deliverables
func (o *Orchestrator) CheckCompletion() (bool, string, error) {
	status, err := o.getDeliverableStatus()
	if err != nil {
		return false, "", fmt.Errorf("get deliverable status: %w", err)
	}

	// Both agents must submit
	if !status.Agent1Submitted || !status.Agent2Submitted {
		return false, "", nil
	}

	// Compare content byte-for-byte
	if !bytes.Equal(status.Agent1Content, status.Agent2Content) {
		return false, "", nil
	}

	// Both submitted and content matches
	return true, status.Agent1Path, nil
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
			return "Both agents submitted matching deliverables - COMPLETED"
		}
		return "Both agents submitted deliverables but content does not match - CONTINUE"
	}

	if status.Agent1Submitted {
		return "Only Agent 1 submitted deliverable - WAITING FOR AGENT 2"
	}

	if status.Agent2Submitted {
		return "Only Agent 2 submitted deliverable - WAITING FOR AGENT 1"
	}

	return "No deliverables submitted yet"
}
