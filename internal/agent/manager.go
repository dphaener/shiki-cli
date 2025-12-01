package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/darinhaener/collab/internal/events"
	"github.com/darinhaener/collab/pkg/types"
)

// Note: claude import kept for SDK message types in StartTurn

// ToolExecutionContext captures file state before tool execution for diff generation
type ToolExecutionContext struct {
	ToolCallID   string            `json:"tool_call_id"`
	ToolName     string            `json:"tool_name"`
	FilePaths    []string          `json:"file_paths"`    // Extracted from tool arguments
	FileContents map[string]string `json:"file_contents"` // path -> content before modification
	Timestamp    time.Time         `json:"timestamp"`     // When context was captured
}

// NewToolExecutionContext creates a new tool execution context
func NewToolExecutionContext(toolCallID, toolName string) *ToolExecutionContext {
	return &ToolExecutionContext{
		ToolCallID:   toolCallID,
		ToolName:     toolName,
		FilePaths:    make([]string, 0),
		FileContents: make(map[string]string),
		Timestamp:    time.Now(),
	}
}

// AddFilePath adds a file path to track and captures its current content
func (ctx *ToolExecutionContext) AddFilePath(filePath string) error {
	// Check if file already tracked
	for _, path := range ctx.FilePaths {
		if path == filePath {
			return nil // Already tracked
		}
	}

	// Add to tracked paths
	ctx.FilePaths = append(ctx.FilePaths, filePath)

	// Capture current file content if it exists
	if content, err := os.ReadFile(filePath); err == nil {
		ctx.FileContents[filePath] = string(content)
	} else {
		// File doesn't exist or can't be read - store empty content
		ctx.FileContents[filePath] = ""
	}

	return nil
}

// GetOriginalContent returns the original content of a file before tool execution
func (ctx *ToolExecutionContext) GetOriginalContent(filePath string) (string, bool) {
	content, exists := ctx.FileContents[filePath]
	return content, exists
}

// Manager handles agent lifecycle, health monitoring, and turn execution
type Manager struct {
	agents    map[string]*Agent
	eventBus  *events.EventBus
	sessionID string
	apiKey    string
	mu        sync.RWMutex

	// Health monitoring
	healthChecks map[string]context.CancelFunc
	healthMu     sync.Mutex

	// Tool execution context cache for diff generation
	executionContexts map[string]*ToolExecutionContext
	contextMu         sync.RWMutex
}

// NewManager creates a new agent manager
func NewManager(eventBus *events.EventBus, sessionID, apiKey string) *Manager {
	return &Manager{
		agents:            make(map[string]*Agent),
		eventBus:          eventBus,
		sessionID:         sessionID,
		apiKey:            apiKey,
		healthChecks:      make(map[string]context.CancelFunc),
		executionContexts: make(map[string]*ToolExecutionContext),
	}
}

// storeExecutionContext stores a tool execution context for later retrieval
func (m *Manager) storeExecutionContext(ctx *ToolExecutionContext) {
	m.contextMu.Lock()
	defer m.contextMu.Unlock()
	m.executionContexts[ctx.ToolCallID] = ctx
}

// getExecutionContext retrieves a tool execution context by tool call ID
func (m *Manager) getExecutionContext(toolCallID string) (*ToolExecutionContext, bool) {
	m.contextMu.RLock()
	defer m.contextMu.RUnlock()
	ctx, exists := m.executionContexts[toolCallID]
	return ctx, exists
}

// cleanupExecutionContext removes a tool execution context from cache
func (m *Manager) cleanupExecutionContext(toolCallID string) {
	m.contextMu.Lock()
	defer m.contextMu.Unlock()
	delete(m.executionContexts, toolCallID)
}

// extractFilePathsFromArgs extracts file paths from Write tool arguments
func (m *Manager) extractFilePathsFromArgs(toolName string, args map[string]interface{}) []string {
	var filePaths []string

	// Currently only handle Write tool
	if toolName != "Write" {
		return filePaths
	}

	// Extract file_path parameter
	if filePath, ok := args["file_path"].(string); ok && filePath != "" {
		filePaths = append(filePaths, filePath)
	}

	return filePaths
}

// SpawnAgent creates and starts a new agent subprocess
// Note: MCP tools removed - agents use built-in tools only and communicate via files
func (m *Manager) SpawnAgent(ctx context.Context, cfg *types.Agent) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if agent already exists
	if _, exists := m.agents[cfg.ID]; exists {
		return nil, fmt.Errorf("agent %s already spawned", cfg.ID)
	}

	// Create agent instance
	agent := NewAgent(cfg)

	// Start agent process with SDK (built-in tools only)
	if err := agent.Start(ctx, m.apiKey, cfg.ID); err != nil {
		return nil, fmt.Errorf("failed to start agent %s: %w", cfg.ID, err)
	}

	// Register agent
	m.agents[cfg.ID] = agent

	// Start health monitoring
	m.startHealthMonitoring(ctx, agent)

	return agent, nil
}

// maxTurnRetries is the maximum number of times to retry a turn on timeout
const maxTurnRetries = 3

// turnRetryDelay is the delay between retry attempts
const turnRetryDelay = 2 * time.Second

// interruptTimeout is the maximum time to wait for an interrupt call to complete.
// If the SDK's Interrupt method hangs, we don't want to block forever.
const interruptTimeout = 10 * time.Second

// sdkInterruptable interface for SDK clients that support interrupt
type sdkInterruptable interface {
	Interrupt(ctx context.Context) error
}

// interruptClientWithTimeout attempts to interrupt the SDK client with a timeout.
// This prevents hanging if the SDK's Interrupt method blocks.
func interruptClientWithTimeout(client sdkInterruptable) {
	// Create a timeout context for the interrupt
	ctx, cancel := context.WithTimeout(context.Background(), interruptTimeout)
	defer cancel()

	// Channel to signal interrupt completion
	done := make(chan error, 1)
	go func() {
		done <- client.Interrupt(ctx)
	}()

	select {
	case <-done:
		// Interrupt completed (success or failure)
		return
	case <-ctx.Done():
		// Interrupt timed out - force continue
		return
	}
}

// StartTurn executes a single turn for the given agent
func (m *Manager) StartTurn(ctx context.Context, agentID, query string, turnNumber int) (*TurnResult, error) {
	m.mu.RLock()
	agent, exists := m.agents[agentID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	if !agent.IsHealthy() {
		return nil, fmt.Errorf("agent %s is not healthy", agentID)
	}

	start := time.Now()

	// Track last error for recovery decisions
	var lastQueryError error

	// Retry loop for handling timeouts and broken pipes
	for attempt := 0; attempt <= maxTurnRetries; attempt++ {
		if attempt > 0 {
			// Restart client on recoverable errors (broken pipes, etc.) or if client is nil
			if IsRecoverableError(lastQueryError) || agent.GetClient() == nil {
				_ = agent.Restart(context.Background()) // Use fresh context
			}

			// Brief delay before retry
			time.Sleep(turnRetryDelay)
		}

		// Get client inside the loop (may have been recreated)
		client := agent.GetClient()
		if client == nil {
			lastQueryError = fmt.Errorf("agent client is nil")
			if attempt < maxTurnRetries {
				continue
			}
			return nil, fmt.Errorf("agent %s SDK client not initialized after %d retries", agentID, maxTurnRetries)
		}

		// Create timeout context (5 minute default)
		turnCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

		// Inject turn number and agent ID into context for tool handlers
		turnCtx = context.WithValue(turnCtx, "turn", turnNumber)
		turnCtx = context.WithValue(turnCtx, "agent_id", agentID)

		// Send query to agent via SDK
		if err := client.Query(turnCtx, query); err != nil {
			cancel()
			lastQueryError = err

			// Check if this is a recoverable error (broken pipe, etc.)
			if IsRecoverableError(err) && attempt < maxTurnRetries {
				continue
			}

			return nil, fmt.Errorf("failed to send query to agent: %w", err)
		}

		// Receive and collect response messages
		msgChan, errChan := client.ReceiveMessages(turnCtx)

		var totalInputTokens, totalOutputTokens, toolCalls int
		var responseText string
		var lastError error
		completed := false
		shouldRetry := false

	messageLoop:
		for {
			select {
			case <-turnCtx.Done():
				// Send interrupt to stop the current operation and clean up (with timeout)
				interruptClientWithTimeout(client)
				// Check if we should retry
				if attempt < maxTurnRetries {
					shouldRetry = true
					break messageLoop
				}
				// No more retries
				cancel()
				if lastError != nil {
					return nil, fmt.Errorf("turn timeout after %d retries: %w", maxTurnRetries, lastError)
				}
				return nil, fmt.Errorf("turn timeout after %d retries", maxTurnRetries)

			case err := <-errChan:
				if err != nil {
					lastError = err
					// Continue to drain messages
				}

			case msg, ok := <-msgChan:
				if !ok {
					// Channel truly closed - query completed
					completed = true
					break messageLoop
				}

				// Skip nil messages - SDK may send these during normal operation
				if msg == nil {
					continue
				}

				// Capture session ID for resume capability
				if sessionIDProvider, ok := msg.(interface{ SessionID() string }); ok {
					if sid := sessionIDProvider.SessionID(); sid != "" {
						agent.SetSessionID(sid)
					}
				}

				msgType := msg.Type()

				// Handle different message types
				switch msgType {
				case "assistant":
					// Type assert to SDKAssistantMessage to access content
					if assistantMsg, ok := msg.(*claude.SDKAssistantMessage); ok {
						// Process each content block in the message
						for _, block := range assistantMsg.Message.Content {
							switch content := block.(type) {
							case claude.TextContentBlock:
								// Extract and emit text content for TUI display
								if content.Text != "" {
									m.eventBus.Publish(events.NewAssistantMessage(
										agentID,
										m.sessionID,
										turnNumber,
										content.Text,
									))
									responseText += content.Text
								}

							case claude.ToolUseContentBlock:
								toolCalls++
								// Strip mcp__ prefix for cleaner display
								displayName := strings.TrimPrefix(content.Name, "mcp__collaboration__")

								// Parse tool arguments from JSON input
								var toolArgs map[string]interface{}
								if len(content.Input) > 0 {
									if err := json.Unmarshal(content.Input, &toolArgs); err != nil {
										// If unmarshal fails, store raw as string
										toolArgs = map[string]interface{}{"_raw": string(content.Input)}
									}
								}

								// Emit ToolInvoked event for TUI with parsed args
								m.eventBus.Publish(events.NewToolInvoked(
									displayName,
									agentID,
									m.sessionID,
									turnNumber,
									toolArgs,
								))
							}
						}
					} else {
						// Fallback: if type assertion fails, use string representation
						msgStr := fmt.Sprintf("%v", msg)
						m.eventBus.Publish(events.NewAssistantMessage(
							agentID,
							m.sessionID,
							turnNumber,
							msgStr,
						))
						responseText += msgStr
					}

				case "result":
					// Result message signals completion
					completed = true
					break messageLoop
				}
			}
		}

		cancel()

		if completed {
			duration := time.Since(start)

			// Estimate token usage (this should come from SDK result messages)
			totalTokens := totalInputTokens + totalOutputTokens
			if totalTokens == 0 {
				// Fallback estimate if SDK doesn't provide usage
				totalTokens = len(query)/4 + len(responseText)/4
				totalInputTokens = len(query) / 4
				totalOutputTokens = len(responseText) / 4
			}

			// Calculate cost using detailed breakdown
			cost := CalculateCostDetailed(totalInputTokens, totalOutputTokens, agent.Model)

			result := &TurnResult{
				AgentID:      agentID,
				TurnNumber:   turnNumber,
				Duration:     duration,
				TokensUsed:   totalTokens,
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
				Cost:         cost,
				Response:     responseText,
				ToolCalls:    toolCalls,
				Success:      lastError == nil,
				Error:        "",
			}

			if lastError != nil {
				result.Error = lastError.Error()
			}

			return result, lastError
		}

		if !shouldRetry {
			return nil, fmt.Errorf("turn failed unexpectedly")
		}
	}

	return nil, fmt.Errorf("turn failed after %d retries", maxTurnRetries)
}

// startHealthMonitoring begins periodic health checks for an agent
func (m *Manager) startHealthMonitoring(ctx context.Context, agent *Agent) {
	healthCtx, cancel := context.WithCancel(ctx)

	m.healthMu.Lock()
	m.healthChecks[agent.ID] = cancel
	m.healthMu.Unlock()

	go m.monitorHealth(healthCtx, agent)
}

// monitorHealth runs periodic health checks on an agent
func (m *Manager) monitorHealth(ctx context.Context, agent *Agent) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastHealthy := agent.IsHealthy()
	consecutiveFailures := 0
	const maxConsecutiveFailures = 3

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if SDK client is still functional
			client := agent.GetClient()
			isAlive := client != nil

			if !isAlive {
				consecutiveFailures++
			} else {
				consecutiveFailures = 0
			}

			// Mark unhealthy after consecutive failures
			if consecutiveFailures >= maxConsecutiveFailures {
				agent.SetHealth(false)
			} else {
				agent.SetHealth(isAlive)
			}

			currentHealth := agent.IsHealthy()

			// Emit event if health changed (degraded)
			if lastHealthy && !currentHealth {
				m.eventBus.Publish(events.NewSessionError(
					&types.Session{ID: m.sessionID},
					fmt.Sprintf("Agent %s connection degraded", agent.ID),
				))
			}

			lastHealthy = currentHealth
		}
	}
}


// GetAgent returns an agent by ID
func (m *Manager) GetAgent(agentID string) (*Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// Shutdown gracefully stops all agents
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop all health monitors
	m.healthMu.Lock()
	for _, cancel := range m.healthChecks {
		cancel()
	}
	m.healthMu.Unlock()

	// Stop all agents
	var errs []error
	for id, agent := range m.agents {
		if err := agent.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop agent %s: %w", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}
