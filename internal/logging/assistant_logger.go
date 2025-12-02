package logging

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/dphaener/shiki-cli/internal/config"
	"github.com/dphaener/shiki-cli/internal/events"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// AssistantEventLogger subscribes to EventBus and writes only assistant events to assistant.log
type AssistantEventLogger struct {
	logger     *Logger
	logFile    *os.File
	fileClosed bool
	mu         sync.Mutex
}

// NewAssistantEventLogger creates a new event logger that writes assistant events to XDG data directory
func NewAssistantEventLogger() (*AssistantEventLogger, error) {
	// Get XDG-compliant log path
	logPath, err := config.GetAssistantLogPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant log path: %w", err)
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open assistant log file: %w", err)
	}

	logger := NewLogger(LevelInfo, file)

	return &AssistantEventLogger{
		logger:  logger,
		logFile: file,
	}, nil
}

// Start subscribes to the event bus and begins logging assistant events
func (ael *AssistantEventLogger) Start(ctx context.Context, bus *events.EventBus) {
	sub := bus.Subscribe("assistant-event-logger")

	go func() {
		defer func() {
			ael.mu.Lock()
			ael.fileClosed = true
			ael.mu.Unlock()
			_ = ael.logFile.Close() // Ignore error on deferred close
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-sub.Events():
				if !ok {
					// Channel closed, shutdown
					return
				}
				if ael.isAssistantEvent(event.Type) {
					ael.logEvent(event)
				}
			}
		}
	}()
}

// isAssistantEvent checks if the event type is one of the assistant logging events
func (ael *AssistantEventLogger) isAssistantEvent(eventType types.EventType) bool {
	switch eventType {
	case types.EventAssistantRequest,
		types.EventAssistantResponse,
		types.EventAssistantError,
		types.EventAssistantMetadata:
		return true
	default:
		return false
	}
}

// logEvent writes a single assistant event to the log file
func (ael *AssistantEventLogger) logEvent(event events.Event) {
	// For assistant events, we want the payload to be the main content
	// This makes it easier to query with jq
	fields := map[string]interface{}{
		"event_type": string(event.Type),
		"session_id": event.SessionID,
		"timestamp":  event.Timestamp,
	}

	// Add the payload fields directly to make jq queries easier
	switch payload := event.Payload.(type) {
	case events.AssistantRequestPayload:
		fields["message_type"] = "request"
		fields["orchestrator"] = payload.OrchestratorID
		fields["agent_id"] = payload.AgentID
		fields["conversation_id"] = payload.ConversationID
		fields["content"] = payload.Content
		if payload.Metadata != nil {
			fields["metadata"] = payload.Metadata
		}

	case events.AssistantResponsePayload:
		fields["message_type"] = "response"
		fields["orchestrator"] = payload.OrchestratorID
		fields["agent_id"] = payload.AgentID
		fields["conversation_id"] = payload.ConversationID
		fields["content"] = payload.Content
		if payload.TokensUsed > 0 {
			fields["tokens_used"] = payload.TokensUsed
		}
		if payload.TimingMs > 0 {
			fields["timing_ms"] = payload.TimingMs
		}
		if payload.Model != "" {
			fields["model"] = payload.Model
		}
		if payload.Metadata != nil {
			fields["metadata"] = payload.Metadata
		}

	case events.AssistantErrorPayload:
		fields["message_type"] = "error"
		fields["orchestrator"] = payload.OrchestratorID
		fields["agent_id"] = payload.AgentID
		fields["error_type"] = payload.ErrorType
		fields["error_message"] = payload.ErrorMessage
		if payload.Context != "" {
			fields["context"] = payload.Context
		}
		if payload.RecoveryAction != "" {
			fields["recovery_action"] = payload.RecoveryAction
		}
		if payload.Metadata != nil {
			fields["metadata"] = payload.Metadata
		}

	case events.AssistantMetadataPayload:
		fields["message_type"] = "metadata"
		fields["orchestrator"] = payload.OrchestratorID
		fields["agent_id"] = payload.AgentID
		if payload.TokensUsed > 0 {
			fields["tokens_used"] = payload.TokensUsed
		}
		if payload.Cost > 0 {
			fields["cost"] = payload.Cost
		}
		if payload.DurationMs > 0 {
			fields["duration_ms"] = payload.DurationMs
		}
		if payload.Model != "" {
			fields["model"] = payload.Model
		}
		if payload.PerformanceData != nil {
			fields["performance_data"] = payload.PerformanceData
		}
		if payload.Metadata != nil {
			fields["metadata"] = payload.Metadata
		}

	default:
		// Fallback for unknown assistant event types
		fields["payload"] = event.Payload
	}

	ael.logger.Info(string(event.Type), fields)
}

// Close closes the log file (idempotent)
func (ael *AssistantEventLogger) Close() error {
	ael.mu.Lock()
	defer ael.mu.Unlock()

	if ael.logFile != nil && !ael.fileClosed {
		ael.fileClosed = true
		return ael.logFile.Close()
	}
	return nil
}