// Package components provides specialized UI components for the TUI.
// ToolChainManager handles the state and lifecycle of related tool executions.
package components

import (
	"strings"
	"sync"
	"time"

	"github.com/dphaener/shiki-cli/internal/broker"
)

// ExecutionState represents the current state of a tool execution.
type ExecutionState string

const (
	ExecutionStatePending   ExecutionState = "pending"
	ExecutionStateRunning   ExecutionState = "running"
	ExecutionStateCompleted ExecutionState = "completed"
	ExecutionStateError     ExecutionState = "error"
	ExecutionStateCancelled ExecutionState = "cancelled"
)

// ToolExecution represents a single tool invocation with its complete lifecycle.
type ToolExecution struct {
	ToolCallID    string                 `json:"tool_call_id"`
	ToolName      string                 `json:"tool_name"`
	State         ExecutionState         `json:"state"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Resource      string                 `json:"resource"`        // File path, URL, or other resource identifier
	Input         map[string]interface{} `json:"input"`
	Output        string                 `json:"output"`
	IsError       bool                   `json:"is_error"`
	DurationMs    int64                  `json:"duration_ms"`
	Turn          int                    `json:"turn"`
	MessageID     string                 `json:"message_id"`
	ParticipantID string                 `json:"participant_id"`
}

// ToolChain represents a sequence of related tools operating on the same resource.
type ToolChain struct {
	ID            string           `json:"id"`
	TurnID        int              `json:"turn_id"`
	Resource      string           `json:"resource"`         // Primary resource identifier
	Tools         []*ToolExecution `json:"tools"`            // Ordered list of all tools in chain
	CurrentTool   *ToolExecution   `json:"current_tool"`     // Currently active/displayed tool
	ReplacedTools []*ToolExecution `json:"replaced_tools"`   // Previous tools hidden by rollup
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// ToolChainManager manages active tool chains and their rollup behavior.
type ToolChainManager struct {
	mu              sync.RWMutex
	activeChains    map[string]*ToolChain  // ChainID -> ToolChain
	toolToChain     map[string]string      // ToolCallID -> ChainID
	turnBoundaries  map[int]time.Time      // Track turn completions
	resourceMatcher *ResourceMatcher       // Path matching logic
	maxChains       int                    // Memory limit (default 50)
}

// NewToolChainManager creates a new tool chain manager.
func NewToolChainManager() *ToolChainManager {
	return &ToolChainManager{
		activeChains:    make(map[string]*ToolChain),
		toolToChain:     make(map[string]string),
		turnBoundaries:  make(map[int]time.Time),
		resourceMatcher: NewResourceMatcher(),
		maxChains:       50, // Memory limit per turn
	}
}

// ProcessToolStarted handles a tool started event and determines chain placement.
func (tcm *ToolChainManager) ProcessToolStarted(event broker.ToolStartedEvent) *ToolChain {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	// Create tool execution from event
	execution := &ToolExecution{
		ToolCallID:    event.ToolCallID,
		ToolName:      event.ToolName,
		State:         ExecutionStatePending,
		StartTime:     event.Timestamp(),
		Resource:      tcm.extractResourceFromInput(event.ToolName, event.Input),
		Input:         event.Input,
		Turn:          event.Turn,
		MessageID:     event.MessageID,
		ParticipantID: event.ParticipantID,
	}

	// Update state to running
	execution.State = ExecutionStateRunning

	// Find existing chain for this resource in the current turn
	existingChain := tcm.findChainForResource(execution.Resource, execution.Turn)

	if existingChain != nil {
		// Add to existing chain and replace current tool
		tcm.addToExistingChain(existingChain, execution)
		return existingChain
	}

	// Create new chain
	chain := tcm.createNewChain(execution)
	tcm.activeChains[chain.ID] = chain
	tcm.toolToChain[execution.ToolCallID] = chain.ID

	// Cleanup if we exceed memory limits
	tcm.enforceMemoryLimits(execution.Turn)

	return chain
}

// ProcessToolCompleted handles a tool completed event and updates chain state.
func (tcm *ToolChainManager) ProcessToolCompleted(event broker.ToolCompletedEvent) *ToolChain {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	chainID, exists := tcm.toolToChain[event.ToolCallID]
	if !exists {
		return nil // Tool not managed by us
	}

	chain := tcm.activeChains[chainID]
	if chain == nil {
		return nil
	}

	// Find and update the tool execution
	for _, tool := range chain.Tools {
		if tool.ToolCallID == event.ToolCallID {
			endTime := event.Timestamp()
			tool.EndTime = &endTime
			tool.Output = event.Output
			tool.IsError = event.IsError
			tool.DurationMs = event.DurationMs

			if event.IsError {
				tool.State = ExecutionStateError
			} else {
				tool.State = ExecutionStateCompleted
			}
			break
		}
	}

	chain.UpdatedAt = time.Now()
	return chain
}

// ProcessTurnCompleted handles turn boundary and cleans up active chains.
func (tcm *ToolChainManager) ProcessTurnCompleted(turn int) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	tcm.turnBoundaries[turn] = time.Now()

	// Clear chains for this turn but preserve history
	for chainID, chain := range tcm.activeChains {
		if chain.TurnID == turn {
			// Remove tool mappings
			for _, tool := range chain.Tools {
				delete(tcm.toolToChain, tool.ToolCallID)
			}
			delete(tcm.activeChains, chainID)
		}
	}

	// Clean up old turn boundaries (keep last 10)
	if len(tcm.turnBoundaries) > 10 {
		for t := range tcm.turnBoundaries {
			if t < turn-10 {
				delete(tcm.turnBoundaries, t)
			}
		}
	}
}

// GetActiveChains returns a copy of all active chains for display.
func (tcm *ToolChainManager) GetActiveChains() map[string]*ToolChain {
	tcm.mu.RLock()
	defer tcm.mu.RUnlock()

	chains := make(map[string]*ToolChain, len(tcm.activeChains))
	for id, chain := range tcm.activeChains {
		chains[id] = chain
	}
	return chains
}

// GetChainByToolID returns the chain containing a specific tool.
func (tcm *ToolChainManager) GetChainByToolID(toolCallID string) *ToolChain {
	tcm.mu.RLock()
	defer tcm.mu.RUnlock()

	chainID, exists := tcm.toolToChain[toolCallID]
	if !exists {
		return nil
	}
	return tcm.activeChains[chainID]
}

// extractResourceFromInput determines the primary resource a tool operates on.
func (tcm *ToolChainManager) extractResourceFromInput(toolName string, input map[string]interface{}) string {
	// Priority order for resource extraction
	resourceKeys := []string{"file_path", "path", "pattern", "command", "url", "resource"}

	for _, key := range resourceKeys {
		if value, exists := input[key]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				// Normalize file paths
				if key == "file_path" || key == "path" {
					return tcm.resourceMatcher.NormalizePath(strValue)
				}
				return strValue
			}
		}
	}

	// Fallback: use tool name if no specific resource identified
	return toolName
}

// findChainForResource finds an existing chain that operates on the same or related resource.
func (tcm *ToolChainManager) findChainForResource(resource string, turn int) *ToolChain {
	for _, chain := range tcm.activeChains {
		if chain.TurnID != turn {
			continue // Only match within same turn
		}

		if tcm.resourceMatcher.AreRelated(resource, chain.Resource) {
			return chain
		}
	}
	return nil
}

// addToExistingChain adds a tool to an existing chain and handles replacement logic.
func (tcm *ToolChainManager) addToExistingChain(chain *ToolChain, execution *ToolExecution) {
	// Move current tool to replaced tools if it exists
	if chain.CurrentTool != nil {
		chain.ReplacedTools = append(chain.ReplacedTools, chain.CurrentTool)
	}

	// Set new tool as current
	chain.CurrentTool = execution
	chain.Tools = append(chain.Tools, execution)
	chain.UpdatedAt = time.Now()

	// Map tool to chain
	tcm.toolToChain[execution.ToolCallID] = chain.ID
}

// createNewChain creates a new tool chain for the given execution.
func (tcm *ToolChainManager) createNewChain(execution *ToolExecution) *ToolChain {
	now := time.Now()
	chain := &ToolChain{
		ID:            generateChainID(),
		TurnID:        execution.Turn,
		Resource:      execution.Resource,
		Tools:         []*ToolExecution{execution},
		CurrentTool:   execution,
		ReplacedTools: make([]*ToolExecution, 0),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return chain
}

// enforceMemoryLimits ensures we don't exceed memory limits per turn.
func (tcm *ToolChainManager) enforceMemoryLimits(currentTurn int) {
	// Count chains for current turn
	turnChainCount := 0
	for _, chain := range tcm.activeChains {
		if chain.TurnID == currentTurn {
			turnChainCount++
		}
	}

	// If we exceed limit, remove oldest chains for this turn
	if turnChainCount > tcm.maxChains {
		var oldestChain *ToolChain
		for _, chain := range tcm.activeChains {
			if chain.TurnID == currentTurn {
				if oldestChain == nil || chain.CreatedAt.Before(oldestChain.CreatedAt) {
					oldestChain = chain
				}
			}
		}

		if oldestChain != nil {
			// Clean up mappings
			for _, tool := range oldestChain.Tools {
				delete(tcm.toolToChain, tool.ToolCallID)
			}
			delete(tcm.activeChains, oldestChain.ID)
		}
	}
}

// generateChainID creates a unique chain identifier.
func generateChainID() string {
	return "chain_" + strings.ReplaceAll(time.Now().Format("20060102_150405.000"), ".", "_")
}

// CanCancelTool checks if a tool can be cancelled.
func (tcm *ToolChainManager) CanCancelTool(toolCallID string) bool {
	tcm.mu.RLock()
	defer tcm.mu.RUnlock()

	chainID, exists := tcm.toolToChain[toolCallID]
	if !exists {
		return false
	}

	chain := tcm.activeChains[chainID]
	if chain == nil {
		return false
	}

	// Tool can be cancelled if it's currently running
	for _, tool := range chain.Tools {
		if tool.ToolCallID == toolCallID {
			return tool.State == ExecutionStateRunning
		}
	}

	return false
}

// CancelTool marks a tool as cancelled.
func (tcm *ToolChainManager) CancelTool(toolCallID string) *ToolChain {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	chainID, exists := tcm.toolToChain[toolCallID]
	if !exists {
		return nil
	}

	chain := tcm.activeChains[chainID]
	if chain == nil {
		return nil
	}

	// Find and cancel the tool
	for _, tool := range chain.Tools {
		if tool.ToolCallID == toolCallID && tool.State == ExecutionStateRunning {
			tool.State = ExecutionStateCancelled
			endTime := time.Now()
			tool.EndTime = &endTime
			break
		}
	}

	chain.UpdatedAt = time.Now()
	return chain
}