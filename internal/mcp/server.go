package mcp

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/darinhaener/collab/internal/events"
)

// Server manages the MCP server instance and handles tool invocations
// from both agents via STDIO transport
type Server struct {
	workspaceDir string
	eventBus     *events.EventBus
	sessionID    string
	ctx          context.Context
	cancel       context.CancelFunc

	// Deliverable tracking for completion detection
	deliverables   map[string]string // agentID -> content
	deliverablesMu sync.RWMutex
}

// NewServer creates a new MCP server instance
func NewServer(workspaceDir, sessionID string, eventBus *events.EventBus) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		workspaceDir: workspaceDir,
		eventBus:     eventBus,
		sessionID:    sessionID,
		ctx:          ctx,
		cancel:       cancel,
		deliverables: make(map[string]string),
	}
}

// Start initializes and starts the MCP server with STDIO transport
// This will integrate with the actual MCP library when available
// For now, it provides the tool handler infrastructure
func (s *Server) Start() error {
	fmt.Fprintf(os.Stderr, "MCP server started for session %s\n", s.sessionID)
	return nil
}

// Shutdown gracefully stops the MCP server
// Target: <100ms shutdown time per requirement
func (s *Server) Shutdown() error {
	s.cancel()
	fmt.Fprintf(os.Stderr, "MCP server shutdown\n")
	return nil
}

// Context returns the server's context for cancellation
func (s *Server) Context() context.Context {
	return s.ctx
}
