package tasks

import (
	"context"
	"sync"
	"time"

	"github.com/darinhaener/collab/internal/broker"
	"github.com/darinhaener/collab/internal/conversation"
)

// Task progress event types
const (
	EventTaskProgressUpdated    broker.EventType = "task.progress.updated"
	EventWorkPackageStatusChanged broker.EventType = "task.workpackage.status_changed"
)

// TaskProgressUpdatedEvent is emitted when a task's progress changes
type TaskProgressUpdatedEvent struct {
	broker.BaseEvent
	TaskID    string            `json:"task_id"`
	Status    TaskProgressState `json:"status"`
	Reason    string            `json:"reason,omitempty"`
	PreviousStatus TaskProgressState `json:"previous_status"`
	Confidence float64           `json:"confidence,omitempty"` // For detected changes
}

// NewTaskProgressUpdatedEvent creates a new task progress updated event
func NewTaskProgressUpdatedEvent(sessionID, taskID string, status, previousStatus TaskProgressState, reason string, confidence float64) TaskProgressUpdatedEvent {
	return TaskProgressUpdatedEvent{
		BaseEvent:      broker.BaseEvent{Type: EventTaskProgressUpdated, Session: sessionID, CreatedAt: time.Now()},
		TaskID:         taskID,
		Status:         status,
		PreviousStatus: previousStatus,
		Reason:         reason,
		Confidence:     confidence,
	}
}

func (e TaskProgressUpdatedEvent) EventType() broker.EventType { return e.Type }
func (e TaskProgressUpdatedEvent) SessionID() string          { return e.Session }
func (e TaskProgressUpdatedEvent) Timestamp() time.Time       { return e.CreatedAt }

// WorkPackageStatusChangedEvent is emitted when a work package's status changes
type WorkPackageStatusChangedEvent struct {
	broker.BaseEvent
	WorkPackageID  string            `json:"work_package_id"`
	Status         TaskProgressState `json:"status"`
	PreviousStatus TaskProgressState `json:"previous_status"`
	CompletedTasks int               `json:"completed_tasks"`
	TotalTasks     int               `json:"total_tasks"`
}

// NewWorkPackageStatusChangedEvent creates a new work package status changed event
func NewWorkPackageStatusChangedEvent(sessionID, workPackageID string, status, previousStatus TaskProgressState, completedTasks, totalTasks int) WorkPackageStatusChangedEvent {
	return WorkPackageStatusChangedEvent{
		BaseEvent:      broker.BaseEvent{Type: EventWorkPackageStatusChanged, Session: sessionID, CreatedAt: time.Now()},
		WorkPackageID:  workPackageID,
		Status:         status,
		PreviousStatus: previousStatus,
		CompletedTasks: completedTasks,
		TotalTasks:     totalTasks,
	}
}

func (e WorkPackageStatusChangedEvent) EventType() broker.EventType { return e.Type }
func (e WorkPackageStatusChangedEvent) SessionID() string          { return e.Session }
func (e WorkPackageStatusChangedEvent) Timestamp() time.Time       { return e.CreatedAt }

// TaskProgressEventHandler handles events and updates task progress
type TaskProgressEventHandler struct {
	progressTracker *ProgressTracker
	analyzer        *ResponseAnalyzer
	eventBroker     *broker.Broker
	sessionID       string
	fileManager     *ProgressFileManager

	// State tracking
	lastUpdateTime  time.Time
	pendingUpdates  []TaskStatusUpdate
	updateMutex     sync.Mutex

	// Configuration
	autoSaveInterval time.Duration
	maxPendingUpdates int

	ctx    context.Context
	cancel context.CancelFunc
}

// NewTaskProgressEventHandler creates a new event handler
func NewTaskProgressEventHandler(tracker *ProgressTracker, analyzer *ResponseAnalyzer, broker *broker.Broker, sessionID string, fileManager *ProgressFileManager) *TaskProgressEventHandler {
	ctx, cancel := context.WithCancel(context.Background())

	handler := &TaskProgressEventHandler{
		progressTracker:   tracker,
		analyzer:          analyzer,
		eventBroker:       broker,
		sessionID:         sessionID,
		fileManager:       fileManager,
		autoSaveInterval:  30 * time.Second, // Save every 30 seconds
		maxPendingUpdates: 10,               // Save after 10 updates
		ctx:               ctx,
		cancel:            cancel,
	}

	// Start auto-save routine
	go handler.autoSaveLoop()

	return handler
}

// Start subscribes to relevant events and begins processing
func (tpeh *TaskProgressEventHandler) Start() error {
	// Subscribe to tool and assistant response events
	subscription := tpeh.eventBroker.Subscribe(
		"task-progress-handler",
		broker.EventToolStarted,
		broker.EventToolCompleted,
		broker.EventStreamFinished, // Assistant responses
		broker.EventPartAdded,      // Message parts
	)

	// Process events in background
	go tpeh.processEvents(subscription)

	return nil
}

// Stop stops the event handler
func (tpeh *TaskProgressEventHandler) Stop() error {
	tpeh.cancel()

	// Save any pending updates
	tpeh.updateMutex.Lock()
	defer tpeh.updateMutex.Unlock()

	if len(tpeh.pendingUpdates) > 0 {
		tpeh.applyPendingUpdates()
	}

	// Save progress file
	return tpeh.fileManager.WriteProgressFile(tpeh.progressTracker)
}

// processEvents processes incoming events
func (tpeh *TaskProgressEventHandler) processEvents(subscription *broker.Subscription) {
	for {
		select {
		case <-tpeh.ctx.Done():
			return
		case event := <-subscription.Events():
			if event == nil {
				continue
			}

			switch e := event.(type) {
			case broker.ToolStartedEvent:
				tpeh.handleToolStartedEvent(e)
			case broker.ToolCompletedEvent:
				tpeh.handleToolCompletedEvent(e)
			case broker.StreamFinishedEvent:
				tpeh.handleStreamFinishedEvent(e)
			case broker.PartAddedEvent:
				tpeh.handlePartAddedEvent(e)
			}
		}
	}
}

// handleToolStartedEvent processes tool started events
func (tpeh *TaskProgressEventHandler) handleToolStartedEvent(event broker.ToolStartedEvent) {
	// Analyze tool name and input for task references
	content := event.ToolName
	if event.Input != nil {
		// Convert input to string for analysis
		for key, value := range event.Input {
			if str, ok := value.(string); ok {
				content += " " + key + " " + str
			}
		}
	}

	updates := tpeh.analyzer.AnalyzeResponse(content)
	tpeh.queueUpdates(updates)
}

// handleToolCompletedEvent processes tool completed events
func (tpeh *TaskProgressEventHandler) handleToolCompletedEvent(event broker.ToolCompletedEvent) {
	// Analyze tool output for task completion indicators
	updates := tpeh.analyzer.AnalyzeResponse(event.Output)

	// If tool completed successfully, bias towards completion
	if !event.IsError && len(updates) > 0 {
		for i := range updates {
			if updates[i].Status == TaskStateInProgress {
				updates[i].Status = TaskStateCompleted
				updates[i].Confidence += 0.1 // Small boost for successful tool completion
			}
		}
	} else if event.IsError {
		// If tool failed, check for blocked status
		for i := range updates {
			if updates[i].Status == TaskStateInProgress {
				updates[i].Status = TaskStateBlocked
				updates[i].Reason = "Tool execution failed"
				updates[i].Confidence += 0.1
			}
		}
	}

	tpeh.queueUpdates(updates)
}

// handleStreamFinishedEvent processes assistant response completion
func (tpeh *TaskProgressEventHandler) handleStreamFinishedEvent(event broker.StreamFinishedEvent) {
	// For stream finished events, we need to look at the message content
	// This would require access to the message, which might not be directly available
	// For now, we'll rely on PartAddedEvent to get the actual content
}

// handlePartAddedEvent processes message part additions (including assistant responses)
func (tpeh *TaskProgressEventHandler) handlePartAddedEvent(event broker.PartAddedEvent) {
	// Extract text content from the part
	content := tpeh.extractTextFromPart(event.Part)
	if content == "" {
		return
	}

	// Analyze the content for task status updates
	updates := tpeh.analyzer.AnalyzeResponse(content)
	tpeh.queueUpdates(updates)
}

// extractTextFromPart extracts text content from a conversation part
func (tpeh *TaskProgressEventHandler) extractTextFromPart(part conversation.Part) string {
	switch p := part.(type) {
	case *conversation.TextPart:
		return p.Content
	case *conversation.ToolResultPart:
		return p.Content
	case *conversation.ReasoningPart:
		return p.Content
	case conversation.TextPart:
		return p.Content
	case conversation.ToolResultPart:
		return p.Content
	case conversation.ReasoningPart:
		return p.Content
	default:
		// For other part types, we don't extract text
		return ""
	}
}

// queueUpdates adds updates to the pending queue
func (tpeh *TaskProgressEventHandler) queueUpdates(updates []TaskStatusUpdate) {
	if len(updates) == 0 {
		return
	}

	tpeh.updateMutex.Lock()
	defer tpeh.updateMutex.Unlock()

	tpeh.pendingUpdates = append(tpeh.pendingUpdates, updates...)
	tpeh.lastUpdateTime = time.Now()

	// Apply updates if we have too many pending
	if len(tpeh.pendingUpdates) >= tpeh.maxPendingUpdates {
		tpeh.applyPendingUpdates()
	}
}

// applyPendingUpdates applies all pending updates to the progress tracker
func (tpeh *TaskProgressEventHandler) applyPendingUpdates() {
	if len(tpeh.pendingUpdates) == 0 {
		return
	}

	for _, update := range tpeh.pendingUpdates {
		// Get current status for comparison
		currentTask, exists := tpeh.progressTracker.GetTask(update.TaskID)
		if !exists {
			continue
		}

		previousStatus := currentTask.Status

		// Only update if status actually changed and confidence is high enough
		if update.Status != previousStatus && update.Confidence >= 0.7 {
			err := tpeh.progressTracker.UpdateTaskStatus(update.TaskID, update.Status, update.Reason)
			if err == nil {
				// Emit progress updated event
				progressEvent := NewTaskProgressUpdatedEvent(
					tpeh.sessionID,
					update.TaskID,
					update.Status,
					previousStatus,
					update.Reason,
					update.Confidence,
				)
				tpeh.eventBroker.PublishAsync(progressEvent)

				// Check if work package status changed
				tpeh.checkWorkPackageStatusChange(update.TaskID, previousStatus, update.Status)
			}
		}
	}

	// Clear pending updates
	tpeh.pendingUpdates = nil

	// Save to file
	go func() {
		if err := tpeh.fileManager.WriteProgressFile(tpeh.progressTracker); err != nil {
			// Log error (would use proper logging in production)
		}
	}()
}

// checkWorkPackageStatusChange checks if a task update caused work package status to change
func (tpeh *TaskProgressEventHandler) checkWorkPackageStatusChange(taskID string, previousTaskStatus, newTaskStatus TaskProgressState) {
	// Find which work package contains this task
	workPackages := tpeh.progressTracker.GetWorkPackages()

	for _, wp := range workPackages {
		for _, task := range wp.Tasks {
			if task.ID == taskID {
				// Get fresh work package status
				freshWP, exists := tpeh.progressTracker.GetWorkPackage(wp.ID)
				if exists && freshWP.Status != wp.Status {
					// Work package status changed
					wpEvent := NewWorkPackageStatusChangedEvent(
						tpeh.sessionID,
						wp.ID,
						freshWP.Status,
						wp.Status,
						tpeh.countCompletedTasks(freshWP.Tasks),
						len(freshWP.Tasks),
					)
					tpeh.eventBroker.PublishAsync(wpEvent)
				}
				return
			}
		}
	}
}

// countCompletedTasks counts completed and skipped tasks
func (tpeh *TaskProgressEventHandler) countCompletedTasks(tasks []TaskProgress) int {
	count := 0
	for _, task := range tasks {
		if task.Status == TaskStateCompleted || task.Status == TaskStateSkipped {
			count++
		}
	}
	return count
}

// autoSaveLoop runs in background to periodically save progress
func (tpeh *TaskProgressEventHandler) autoSaveLoop() {
	ticker := time.NewTicker(tpeh.autoSaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tpeh.ctx.Done():
			return
		case <-ticker.C:
			tpeh.updateMutex.Lock()
			if len(tpeh.pendingUpdates) > 0 {
				tpeh.applyPendingUpdates()
			}
			tpeh.updateMutex.Unlock()
		}
	}
}

// EmitProgressUpdate manually emits a progress update event
func (tpeh *TaskProgressEventHandler) EmitProgressUpdate(taskID string, status TaskProgressState, reason string) {
	// Get current status
	currentTask, exists := tpeh.progressTracker.GetTask(taskID)
	if !exists {
		return
	}

	previousStatus := currentTask.Status

	// Update progress tracker
	err := tpeh.progressTracker.UpdateTaskStatus(taskID, status, reason)
	if err != nil {
		return
	}

	// Emit event
	event := NewTaskProgressUpdatedEvent(
		tpeh.sessionID,
		taskID,
		status,
		previousStatus,
		reason,
		1.0, // Manual updates have full confidence
	)
	tpeh.eventBroker.Publish(event)

	// Check work package status change
	tpeh.checkWorkPackageStatusChange(taskID, previousStatus, status)

	// Save to file
	go func() {
		tpeh.fileManager.WriteProgressFile(tpeh.progressTracker)
	}()
}

// GetPendingUpdateCount returns the number of pending updates
func (tpeh *TaskProgressEventHandler) GetPendingUpdateCount() int {
	tpeh.updateMutex.Lock()
	defer tpeh.updateMutex.Unlock()
	return len(tpeh.pendingUpdates)
}

// ForceSave forces immediate application of pending updates and save
func (tpeh *TaskProgressEventHandler) ForceSave() error {
	tpeh.updateMutex.Lock()
	defer tpeh.updateMutex.Unlock()

	if len(tpeh.pendingUpdates) > 0 {
		tpeh.applyPendingUpdates()
	}

	return tpeh.fileManager.WriteProgressFile(tpeh.progressTracker)
}