package orchestrator

// PhaseOrchestrator defines the common interface for all phase orchestrators.
// This enables workflows to work with any orchestrator implementation
// (Plan, Tasks, Implement, Specify, Bug) through a common interface.
//
// Note: This is different from the legacy Orchestrator type which handles
// multi-agent collaboration. PhaseOrchestrator is for single-agent phases.
type PhaseOrchestrator interface {
	// Initialize sets up the agent and prepares the orchestrator for use.
	// Must be called before SendMessage.
	Initialize() error

	// SendMessage sends a user message to the agent and returns streaming updates.
	// Returns two channels:
	//   - updateChan: receives MessageUpdate for streaming text, tool use, errors, and completion
	//   - errorChan: receives fatal errors that terminate the operation
	// Both channels are closed when the operation completes.
	SendMessage(userMessage string) (<-chan MessageUpdate, <-chan error)

	// Interrupt gracefully interrupts the current agent operation without stopping the orchestrator.
	// Can be called to cancel a long-running operation and allow new operations.
	Interrupt() error

	// Stop gracefully shuts down the orchestrator and releases all resources.
	// After calling Stop, the orchestrator should not be used.
	Stop() error
}

// OrchestratorConfig provides configuration for creating an orchestrator.
// This allows different workflows to customize orchestrator behavior
// without modifying the orchestrator implementation.
type OrchestratorConfig struct {
	// SessionID is the unique identifier for the session
	SessionID string

	// PhaseType identifies the type of phase (e.g., "plan", "tasks", "implement", "specify", "bug")
	// Used for logging and event publishing
	PhaseType string

	// Timeout is the maximum duration to wait for a response
	Timeout int // in minutes

	// Instructions are the system prompt/instructions sent with the first message
	Instructions string

	// EventMetadata contains additional metadata to include in published events
	EventMetadata map[string]interface{}
}
