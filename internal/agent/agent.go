package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/darinhaener/collab/pkg/types"
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

// Start initializes the agent subprocess with MCP tools
func (a *Agent) Start(ctx context.Context, mcpTools []claude.McpTool, apiKey string, agentID string) error {
	// Create SDK MCP server with the collaboration tools
	mcpServer := claude.CreateSdkMcpServer(
		"collaboration",
		"1.0.0",
		mcpTools,
	)

	// Build list of allowed tools (include both built-in and MCP tools)
	allowedTools := []string{
		// Common built-in tools
		"Read", "Write", "Edit", "Glob", "Grep", "Bash",
	}
	// Add MCP collaboration tools with full prefixed names
	for _, tool := range mcpTools {
		fullToolName := fmt.Sprintf("mcp__collaboration__%s", tool.Name())
		allowedTools = append(allowedTools, fullToolName)
	}

	// Configure SDK options
	opts := &claude.Options{
		Context:      ctx,
		Model:        a.Model,
		Cwd:          a.WorkspaceDir,
		SystemPrompt: claude.SystemPromptLiteral(a.SystemPrompt),
		MaxTurns:     100, // Default max turns per query
		Env: map[string]string{
			"ANTHROPIC_API_KEY": apiKey,
			"AGENT_ID":          agentID,
			"AGENT_NAME":        a.Name,
			"AGENT_ROLE":        a.Role,
		},
		McpServers: map[string]claude.McpServerConfig{
			"collaboration": mcpServer,
		},
		AllowedTools:                    allowedTools,
		AllowDangerouslySkipPermissions: true, // Allow tools to run without prompts in headless mode
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
