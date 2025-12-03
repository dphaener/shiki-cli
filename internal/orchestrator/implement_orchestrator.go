package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	agentpkg "github.com/dphaener/shiki-cli/internal/agent"
	"github.com/dphaener/shiki-cli/internal/broker"
	"github.com/dphaener/shiki-cli/internal/conversation"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/internal/tasks"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// ImplementOrchestrator manages the implementation workflow with a single agent.
// It embeds BaseOrchestrator for common functionality and adds session-specific behavior.
type ImplementOrchestrator struct {
	*BaseOrchestrator
	session *types.WorkflowSession

	// Task progress tracking components
	progressTracker   *tasks.ProgressTracker
	progressHandler   *tasks.TaskProgressEventHandler
	responseAnalyzer  *tasks.ResponseAnalyzer
	fileManager       *tasks.ProgressFileManager

	// Task progress tracking state
	taskProgressEnabled  bool
	taskProgressBroker   *broker.Broker // Dedicated broker for task progress events
	legacyEventSubscriber *events.Subscriber // Subscription to legacy event bus
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
	err := o.InitializeWithAgent(agentCfg)
	if err != nil {
		return err
	}

	// Setup task progress tracking after agent initialization
	if err := o.setupTaskProgressTracking(); err != nil {
		// Log error but don't fail initialization - task progress is optional
		fmt.Printf("Warning: Failed to setup task progress tracking for session %s: %v\n", o.sessionID, err)
		fmt.Printf("Implementation will continue without task progress tracking.\n")
		o.taskProgressEnabled = false
	} else if o.taskProgressEnabled {
		fmt.Printf("Task progress tracking enabled for session %s\n", o.sessionID)
	}

	return nil
}

// setupTaskProgressTracking initializes the task progress tracking components
func (o *ImplementOrchestrator) setupTaskProgressTracking() error {
	// Get the feature directory path from session
	featurePath := o.session.FeatureDir
	tasksFilePath := filepath.Join(featurePath, "tasks.md")
	progressFilePath := filepath.Join(featurePath, "task-progress.md")

	// Initialize ProgressFileManager
	o.fileManager = tasks.NewProgressFileManager(progressFilePath)

	// Check if tasks.md exists, if not, skip task progress tracking
	if _, err := os.Stat(tasksFilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("tasks.md file not found at %s - task progress tracking requires a tasks.md file", tasksFilePath)
		}
		return fmt.Errorf("error checking tasks file: %w", err)
	}

	// Parse tasks.md to create task structure
	parser := tasks.NewTasksParser()
	taskStructure, err := parser.ParseTasksFile(tasksFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse tasks file %s: %w", tasksFilePath, err)
	}

	// Validate that we have at least some tasks to track
	if len(taskStructure.TaskMap) == 0 {
		return fmt.Errorf("no tasks found in %s - task progress tracking requires at least one task", tasksFilePath)
	}

	// Create ResponseAnalyzer
	o.responseAnalyzer = tasks.NewResponseAnalyzer(taskStructure)

	// Create ProgressTracker
	o.progressTracker = tasks.NewProgressTracker(taskStructure, progressFilePath)

	// Create a dedicated broker for task progress events
	// This allows us to have typed events while the main codebase still uses the legacy event bus
	o.taskProgressBroker = broker.New()

	// Create TaskProgressEventHandler
	o.progressHandler = tasks.NewTaskProgressEventHandler(
		o.progressTracker,
		o.responseAnalyzer,
		o.taskProgressBroker,
		o.sessionID,
		o.fileManager,
	)

	// Create initial progress file if it doesn't exist
	if !o.fileManager.Exists() {
		progressFilePath := o.fileManager.GetFilePath()
		fmt.Printf("Creating initial task progress file: %s\n", progressFilePath)

		// Ensure the directory exists
		if err := os.MkdirAll(filepath.Dir(progressFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create progress file directory: %w", err)
		}

		// Create the file with retry mechanism for concurrent access
		var createErr error
		for retries := 3; retries > 0; retries-- {
			createErr = o.fileManager.CreateInitialProgressFile(tasksFilePath)
			if createErr == nil {
				break
			}

			// Check if it's a concurrent access error
			if os.IsExist(createErr) {
				// File was created by another process, that's fine
				break
			}

			// For other errors, retry after a short delay
			if retries > 1 {
				time.Sleep(50 * time.Millisecond)
			}
		}

		if createErr != nil {
			return fmt.Errorf("failed to create initial progress file %s from tasks file %s after retries: %w", progressFilePath, tasksFilePath, createErr)
		}

		// Validate that the file was created successfully
		if err := o.fileManager.Validate(); err != nil {
			return fmt.Errorf("created progress file failed validation: %w", err)
		}

		fmt.Printf("Task progress file created successfully with %d tasks\n", len(taskStructure.TaskMap))
	} else {
		fmt.Printf("Task progress file already exists: %s\n", o.fileManager.GetFilePath())

		// Validate existing file to ensure it's not corrupted
		if err := o.fileManager.Validate(); err != nil {
			fmt.Printf("Warning: Existing progress file validation failed: %v\n", err)
			fmt.Printf("Task progress tracking will attempt to recover.\n")
			// Don't fail initialization - let the system attempt recovery
		}
	}

	o.taskProgressEnabled = true

	// Set up event bridge to connect legacy EventBus to task progress broker
	if err := o.setupEventBridge(); err != nil {
		return fmt.Errorf("failed to setup event bridge: %w", err)
	}

	return nil
}

// setupEventBridge creates a bridge between legacy EventBus and task progress broker
func (o *ImplementOrchestrator) setupEventBridge() error {
	if o.eventBus == nil {
		return fmt.Errorf("legacy event bus not available")
	}

	// Subscribe to events we're interested in for task progress tracking
	o.legacyEventSubscriber = o.eventBus.Subscribe(
		"implement-task-progress",
		types.EventToolInvoked,
		types.EventAssistantResponse,
		types.EventAssistantMessage,
	)

	// Start a goroutine to bridge events from legacy bus to task progress broker
	go o.eventBridgeLoop()

	return nil
}

// eventBridgeLoop bridges events from legacy event bus to task progress broker
func (o *ImplementOrchestrator) eventBridgeLoop() {
	if o.legacyEventSubscriber == nil {
		return
	}

	for event := range o.legacyEventSubscriber.Events() {
		// Process events safely with panic recovery and error handling
		o.processEventSafely(event)
	}
}

// processEventSafely handles individual events with comprehensive error recovery
func (o *ImplementOrchestrator) processEventSafely(event events.Event) {
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't crash the application
			fmt.Printf("Warning: Panic in event processing for session %s: %v\n", o.sessionID, r)
		}
	}()

	// Validate event structure
	if event.SessionID == "" {
		return // Skip malformed events
	}

	// Only process events for this session
	if event.SessionID != o.sessionID {
		return
	}

	// Validate broker availability before processing
	if o.taskProgressBroker == nil {
		return
	}

	// Convert legacy events to broker events and publish to task progress broker
	var brokerEvent broker.Event

	switch event.Type {
	case types.EventToolInvoked:
		if payload, ok := event.Payload.(events.ToolInvokedPayload); ok {
			brokerEvent = o.createToolStartedEvent(payload)
		}

	case types.EventAssistantResponse:
		if payload, ok := event.Payload.(events.AssistantResponsePayload); ok {
			brokerEvent = o.createStreamFinishedEvent(payload)
		}

	case types.EventAssistantMessage:
		if payload, ok := event.Payload.(events.AssistantMessagePayload); ok {
			brokerEvent = o.createPartAddedEvent(payload)
		}
	}

	// Publish event if conversion was successful
	if brokerEvent != nil {
		success := o.taskProgressBroker.PublishAsync(brokerEvent)
		if !success {
			// Publishing failed (likely due to buffer full or shutdown)
			fmt.Printf("Warning: Failed to publish event to task progress broker for session %s\n", o.sessionID)
		}
	}
}

// createToolStartedEvent converts legacy tool invoked payload to broker ToolStartedEvent
func (o *ImplementOrchestrator) createToolStartedEvent(payload events.ToolInvokedPayload) broker.Event {
	return broker.NewToolStartedEvent(
		o.sessionID,
		"", // messageID - not available in legacy payload
		payload.AgentID,
		"", // toolCallID - not available in legacy payload
		payload.ToolName,
		payload.Args,
		payload.Turn,
	)
}

// createStreamFinishedEvent converts legacy assistant response payload to broker StreamFinishedEvent
func (o *ImplementOrchestrator) createStreamFinishedEvent(payload events.AssistantResponsePayload) broker.Event {
	return broker.NewStreamFinishedEvent(
		o.sessionID,
		"", // messageID - not available in legacy payload
		payload.AgentID,
		conversation.FinishReasonStop, // Default finish reason
		0, // inputTokens - not available in legacy payload
		payload.TokensUsed,
	)
}

// createPartAddedEvent converts legacy assistant message payload to broker PartAddedEvent
func (o *ImplementOrchestrator) createPartAddedEvent(payload events.AssistantMessagePayload) broker.Event {
	// Create a text part from the content
	textPart := &conversation.TextPart{
		Content: payload.Content,
	}

	return broker.NewPartAddedEvent(
		o.sessionID,
		"", // messageID - not available in legacy payload
		payload.AgentID,
		textPart,
	)
}

// GetSession returns the workflow session (for orchestrators that need direct access)
func (o *ImplementOrchestrator) GetSession() *types.WorkflowSession {
	return o.session
}

// StartTaskProgress starts the task progress tracking if enabled
func (o *ImplementOrchestrator) StartTaskProgress() error {
	if !o.taskProgressEnabled || o.progressHandler == nil {
		return nil
	}

	// Use a timeout to prevent hanging
	done := make(chan error, 1)
	go func() {
		done <- o.progressHandler.Start()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout starting task progress tracking for session %s", o.sessionID)
	}
}

// StopTaskProgress stops the task progress tracking
func (o *ImplementOrchestrator) StopTaskProgress() error {
	if !o.taskProgressEnabled || o.progressHandler == nil {
		return nil
	}

	// Use a timeout to prevent hanging during shutdown
	done := make(chan error, 1)
	go func() {
		err := o.progressHandler.Stop()

		// Unsubscribe from the legacy event bus
		if o.legacyEventSubscriber != nil && o.eventBus != nil {
			o.eventBus.Unsubscribe("implement-task-progress")
			o.legacyEventSubscriber = nil
		}

		// Shutdown the dedicated broker
		if o.taskProgressBroker != nil {
			o.taskProgressBroker.Shutdown()
			o.taskProgressBroker = nil
		}

		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		// Force cleanup after timeout
		o.legacyEventSubscriber = nil
		if o.taskProgressBroker != nil {
			o.taskProgressBroker.Shutdown()
			o.taskProgressBroker = nil
		}
		return fmt.Errorf("timeout stopping task progress tracking for session %s (forced cleanup completed)", o.sessionID)
	}
}

// GetTaskProgress returns current task progress if tracking is enabled
func (o *ImplementOrchestrator) GetTaskProgress() *tasks.ProgressTracker {
	if !o.taskProgressEnabled {
		return nil
	}
	return o.progressTracker
}

// IsTaskProgressEnabled returns whether task progress tracking is enabled
func (o *ImplementOrchestrator) IsTaskProgressEnabled() bool {
	return o.taskProgressEnabled
}
