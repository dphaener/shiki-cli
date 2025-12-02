package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/dphaener/shiki-cli/pkg/types"
)

// Agent represents an AI agent subprocess
type Agent struct {
	ID           string
	Name         string
	Role         string
	SystemPrompt string
	Model        string
	WorkspaceDir string

	client    *claude.ClaudeSDKClient
	isHealthy bool
	mu        sync.RWMutex

	// Store creation parameters for restart capability
	creationAPIKey string

	// Session ID for resume capability
	sessionID string
}

// NewAgent creates a new agent instance
func NewAgent(cfg *types.Agent) *Agent {
	return &Agent{
		ID:           cfg.ID,
		Name:         cfg.Name,
		Role:         cfg.Role,
		SystemPrompt: cfg.SystemPrompt,
		Model:        cfg.Model,
		WorkspaceDir: cfg.WorkspaceDir,
		isHealthy:    false,
	}
}

// Start initializes the agent subprocess with built-in tools only
func (a *Agent) Start(ctx context.Context, apiKey string, agentID string) error {
	// Store creation parameters for restart capability
	a.creationAPIKey = apiKey

	return a.startInternal(ctx, apiKey)
}

// startInternal is the common client creation logic used by Start and Restart
func (a *Agent) startInternal(ctx context.Context, apiKey string) error {
	// Only allow built-in tools - no MCP tools needed
	// Agents communicate via file-based protocol instead
	allowedTools := []string{
		"Read", "Write", "Edit", "Glob", "Grep", "Bash",
	}

	// Configure SDK options
	env := map[string]string{
		"AGENT_ID":   a.ID,
		"AGENT_NAME": a.Name,
		"AGENT_ROLE": a.Role,
	}
	// Only set API key if provided (Claude Code SDK doesn't require it)
	if apiKey != "" {
		env["ANTHROPIC_API_KEY"] = apiKey
	}

	opts := &claude.Options{
		Context:                         ctx,
		Model:                           a.Model,
		Cwd:                             a.WorkspaceDir,
		SystemPrompt:                    claude.SystemPromptLiteral(a.SystemPrompt),
		MaxTurns:                        100, // Default max turns per query
		Env:                             env,
		AllowedTools:                    allowedTools,
		AllowDangerouslySkipPermissions: true, // Allow tools to run without prompts in headless mode
		Continue:                        a.sessionID != "", // Resume if we have a session ID
		Resume:                          a.sessionID,       // Session ID to resume (empty string = new session)
	}

	// Create SDK client
	client, err := claude.NewClient(opts)
	if err != nil {
		return fmt.Errorf("failed to create SDK client: %w", err)
	}

	a.mu.Lock()
	a.client = client
	a.isHealthy = true
	a.mu.Unlock()

	return nil
}

// Restart recreates the SDK client with the same configuration.
// Use this to recover from broken pipes or stale connections.
func (a *Agent) Restart(ctx context.Context) error {
	a.mu.Lock()
	// Close existing client if present
	if a.client != nil {
		_ = a.client.Close() // Ignore errors on dead client
		a.client = nil
		a.isHealthy = false
	}
	apiKey := a.creationAPIKey // May be empty - Claude Code SDK doesn't require it
	a.mu.Unlock()

	err := a.startInternal(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("restart failed: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the agent subprocess
func (a *Agent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client == nil {
		return nil
	}

	// Close SDK client (this will gracefully shutdown the Claude CLI process)
	if err := a.client.Close(); err != nil {
		return fmt.Errorf("failed to close SDK client: %w", err)
	}

	a.isHealthy = false
	a.client = nil

	return nil
}

// IsHealthy checks if the agent SDK client is active
func (a *Agent) IsHealthy() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isHealthy && a.client != nil
}

// SetHealth updates the health status (used by health monitor)
func (a *Agent) SetHealth(healthy bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isHealthy = healthy
}

// GetClient returns the underlying SDK client (for testing/internal use)
func (a *Agent) GetClient() *claude.ClaudeSDKClient {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

// SetSessionID stores the session ID for resume capability
func (a *Agent) SetSessionID(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sessionID == "" && sessionID != "" {
		a.sessionID = sessionID
	}
}

// GetSessionID returns the current session ID
func (a *Agent) GetSessionID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.sessionID
}
