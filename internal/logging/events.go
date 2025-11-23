package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/darinhaener/collab/internal/events"
)

// EventLogger subscribes to EventBus and writes all events to orchestrator.log
type EventLogger struct {
	logger     *Logger
	logFile    *os.File
	fileClosed bool
	mu         sync.Mutex
}

// NewEventLogger creates a new event logger that writes to orchestrator.log
func NewEventLogger(workspaceDir string) (*EventLogger, error) {
	logPath := filepath.Join(workspaceDir, "orchestrator.log")

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	logger := NewLogger(LevelInfo, file)

	return &EventLogger{
		logger:  logger,
		logFile: file,
	}, nil
}

// Start subscribes to the event bus and begins logging events
func (el *EventLogger) Start(ctx context.Context, bus *events.EventBus) {
	sub := bus.Subscribe("event-logger") // Subscribe to all events

	go func() {
		defer func() {
			el.mu.Lock()
			el.fileClosed = true
			el.mu.Unlock()
			_ = el.logFile.Close() // Ignore error on deferred close
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
				el.logEvent(event)
			}
		}
	}()
}

// logEvent writes a single event to the log file
func (el *EventLogger) logEvent(event events.Event) {
	fields := map[string]interface{}{
		"event_type": string(event.Type),
		"session_id": event.SessionID,
		"payload":    event.Payload,
	}

	el.logger.Info(fmt.Sprintf("Event: %s", event.Type), fields)
}

// Close closes the log file (idempotent)
func (el *EventLogger) Close() error {
	el.mu.Lock()
	defer el.mu.Unlock()

	if el.logFile != nil && !el.fileClosed {
		el.fileClosed = true
		return el.logFile.Close()
	}
	return nil
}
