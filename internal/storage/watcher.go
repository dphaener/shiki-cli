package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/darinhaener/collab/pkg/types"
	"github.com/fsnotify/fsnotify"
)

// Watcher monitors filesystem changes and emits events
type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	workspaceDir string
	eventChan    chan<- types.Event
}

// NewWatcher creates a new file watcher for the workspace
func NewWatcher(workspaceDir string, eventChan chan<- types.Event) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	// Watch workspace directory recursively
	if err := fsWatcher.Add(workspaceDir); err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("watch workspace: %w", err)
	}

	// Watch subdirectories
	subdirs := []string{"messages", "memory"}
	for _, dir := range subdirs {
		path := filepath.Join(workspaceDir, dir)
		if err := fsWatcher.Add(path); err != nil {
			// Continue if dir doesn't exist yet
			continue
		}
	}

	return &Watcher{
		fsWatcher:    fsWatcher,
		workspaceDir: workspaceDir,
		eventChan:    eventChan,
	}, nil
}

// Start begins watching for file changes
func (w *Watcher) Start(ctx context.Context, sessionID string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-w.fsWatcher.Events:
				if !ok {
					return
				}

				// Ignore temp files (from atomic writes)
				baseName := filepath.Base(event.Name)
				if len(baseName) >= 4 && baseName[:4] == ".tmp" {
					continue
				}

				operation := "modified"
				if event.Op&fsnotify.Create != 0 {
					operation = "created"
				} else if event.Op&fsnotify.Remove != 0 {
					operation = "deleted"
				}

				// Emit FileUpdated event
				w.eventChan <- types.Event{
					Type:      types.EventFileUpdated,
					SessionID: sessionID,
					Timestamp: time.Now(),
					Payload: map[string]interface{}{
						"path":      event.Name,
						"operation": operation,
					},
				}

			case err, ok := <-w.fsWatcher.Errors:
				if !ok {
					return
				}
				// Log error but continue (graceful degradation)
				// In production, this should use the logger
				_ = err
			}
		}
	}()
}

// Close stops the watcher
func (w *Watcher) Close() error {
	return w.fsWatcher.Close()
}

// AddDirectory adds a new directory to watch
func (w *Watcher) AddDirectory(dir string) error {
	path := filepath.Join(w.workspaceDir, dir)
	return w.fsWatcher.Add(path)
}

// shouldIgnoreFile determines if a file should be ignored
func shouldIgnoreFile(path string) bool {
	baseName := filepath.Base(path)

	// Ignore temp files
	if strings.HasPrefix(baseName, ".tmp") {
		return true
	}

	// Ignore hidden files (except .gitkeep)
	if strings.HasPrefix(baseName, ".") && baseName != ".gitkeep" {
		return true
	}

	return false
}
