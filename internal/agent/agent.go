package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

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

	cmd       *exec.Cmd
	pid       int
	isHealthy bool
	mu        sync.RWMutex

	// Communication channels
	stdin  chan string
	stdout chan string
	stderr chan string
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
		stdin:        make(chan string, 10),
		stdout:       make(chan string, 100),
		stderr:       make(chan string, 100),
		isHealthy:    false,
	}
}

// Start initializes the agent subprocess with MCP connection
func (a *Agent) Start(ctx context.Context, mcpEndpoint string, apiKey string) error {
	// Prepare environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", apiKey))
	env = append(env, fmt.Sprintf("MCP_SERVER_ENDPOINT=%s", mcpEndpoint))
	env = append(env, fmt.Sprintf("WORKSPACE_DIR=%s", a.WorkspaceDir))
	env = append(env, fmt.Sprintf("AGENT_ID=%s", a.ID))
	env = append(env, fmt.Sprintf("AGENT_NAME=%s", a.Name))
	env = append(env, fmt.Sprintf("AGENT_ROLE=%s", a.Role))

	// Create command to start Claude CLI
	// NOTE: This assumes 'claude' CLI is available in PATH
	// In production, this would use the claude-agent-sdk-go
	a.cmd = exec.CommandContext(ctx, "claude", "--model", a.Model)
	a.cmd.Env = env
	a.cmd.Dir = a.WorkspaceDir

	// Set up process group for clean shutdown
	a.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process
	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start agent process: %w", err)
	}

	a.mu.Lock()
	a.pid = a.cmd.Process.Pid
	a.isHealthy = true
	a.mu.Unlock()

	return nil
}

// Stop gracefully shuts down the agent subprocess
func (a *Agent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cmd == nil || a.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown
	if err := a.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait up to 5 seconds for graceful shutdown
	done := make(chan error, 1)
	go func() {
		done <- a.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Graceful shutdown timed out, force kill
		if err := a.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		<-done // Wait for process to actually die
	case err := <-done:
		// Process exited gracefully
		if err != nil && err.Error() != "signal: terminated" {
			return fmt.Errorf("process exit error: %w", err)
		}
	}

	a.isHealthy = false
	close(a.stdin)
	close(a.stdout)
	close(a.stderr)

	return nil
}

// IsHealthy checks if the agent process is still running
func (a *Agent) IsHealthy() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isHealthy
}

// GetPID returns the process ID of the agent
func (a *Agent) GetPID() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.pid
}

// SetHealth updates the health status (used by health monitor)
func (a *Agent) SetHealth(healthy bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isHealthy = healthy
}
