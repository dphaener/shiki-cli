package components

import (
	"testing"

	"github.com/darinhaener/collab/internal/broker"
)

func TestToolChainManager_Basic(t *testing.T) {
	tcm := NewToolChainManager()

	// Test basic initialization
	if tcm == nil {
		t.Fatal("NewToolChainManager returned nil")
	}

	// Test that we start with no active chains
	chains := tcm.GetActiveChains()
	if len(chains) != 0 {
		t.Errorf("Expected 0 active chains, got %d", len(chains))
	}
}

func TestToolChainManager_ProcessToolStarted(t *testing.T) {
	tcm := NewToolChainManager()

	// Create a tool started event
	event := broker.NewToolStartedEvent(
		"session1",
		"message1",
		"agent1",
		"tool_call_1",
		"Read",
		map[string]interface{}{
			"file_path": "/Users/test/file.txt",
		},
		1,
	)

	// Process the event
	chain := tcm.ProcessToolStarted(event)
	if chain == nil {
		t.Fatal("ProcessToolStarted returned nil chain")
	}

	// Verify chain properties
	if chain.ID == "" {
		t.Error("Chain ID should not be empty")
	}

	if chain.TurnID != 1 {
		t.Errorf("Expected TurnID 1, got %d", chain.TurnID)
	}

	if chain.CurrentTool == nil {
		t.Fatal("CurrentTool should not be nil")
	}

	if chain.CurrentTool.ToolName != "Read" {
		t.Errorf("Expected ToolName 'Read', got %s", chain.CurrentTool.ToolName)
	}

	if chain.CurrentTool.State != ExecutionStateRunning {
		t.Errorf("Expected state running, got %s", chain.CurrentTool.State)
	}

	// Verify we now have one active chain
	chains := tcm.GetActiveChains()
	if len(chains) != 1 {
		t.Errorf("Expected 1 active chain, got %d", len(chains))
	}
}

func TestToolChainManager_ProcessToolCompleted(t *testing.T) {
	tcm := NewToolChainManager()

	// Start a tool first
	startEvent := broker.NewToolStartedEvent(
		"session1",
		"message1",
		"agent1",
		"tool_call_1",
		"Read",
		map[string]interface{}{
			"file_path": "/Users/test/file.txt",
		},
		1,
	)

	chain := tcm.ProcessToolStarted(startEvent)
	if chain == nil {
		t.Fatal("Failed to start tool")
	}

	// Complete the tool
	completeEvent := broker.NewToolCompletedEvent(
		"session1",
		"message1",
		"agent1",
		"tool_call_1",
		"File content here",
		false,
		100,
	)

	updatedChain := tcm.ProcessToolCompleted(completeEvent)
	if updatedChain == nil {
		t.Fatal("ProcessToolCompleted returned nil")
	}

	// Verify tool is marked as completed
	if updatedChain.CurrentTool.State != ExecutionStateCompleted {
		t.Errorf("Expected state completed, got %s", updatedChain.CurrentTool.State)
	}

	if updatedChain.CurrentTool.Output != "File content here" {
		t.Errorf("Expected output 'File content here', got %s", updatedChain.CurrentTool.Output)
	}

	if updatedChain.CurrentTool.DurationMs != 100 {
		t.Errorf("Expected duration 100ms, got %d", updatedChain.CurrentTool.DurationMs)
	}
}

func TestToolChainManager_ResourceMatching(t *testing.T) {
	tcm := NewToolChainManager()

	// Start a tool that writes to a file
	writeEvent := broker.NewToolStartedEvent(
		"session1",
		"message1",
		"agent1",
		"tool_call_1",
		"Write",
		map[string]interface{}{
			"file_path": "/Users/test/file.txt",
		},
		1,
	)

	chain1 := tcm.ProcessToolStarted(writeEvent)
	if chain1 == nil {
		t.Fatal("Failed to start write tool")
	}

	// Start another tool that reads the same file
	readEvent := broker.NewToolStartedEvent(
		"session1",
		"message2",
		"agent1",
		"tool_call_2",
		"Read",
		map[string]interface{}{
			"file_path": "/Users/test/file.txt",
		},
		1,
	)

	chain2 := tcm.ProcessToolStarted(readEvent)
	if chain2 == nil {
		t.Fatal("Failed to start read tool")
	}

	// Should be the same chain due to resource matching
	if chain1.ID != chain2.ID {
		t.Errorf("Expected same chain ID for tools operating on same file, got %s and %s", chain1.ID, chain2.ID)
	}

	// The write tool should now be in replaced tools
	if len(chain2.ReplacedTools) != 1 {
		t.Errorf("Expected 1 replaced tool, got %d", len(chain2.ReplacedTools))
	}

	if chain2.ReplacedTools[0].ToolName != "Write" {
		t.Errorf("Expected replaced tool to be 'Write', got %s", chain2.ReplacedTools[0].ToolName)
	}

	// Current tool should be the read tool
	if chain2.CurrentTool.ToolName != "Read" {
		t.Errorf("Expected current tool to be 'Read', got %s", chain2.CurrentTool.ToolName)
	}
}

func TestToolChainManager_TurnBoundaries(t *testing.T) {
	tcm := NewToolChainManager()

	// Start a tool in turn 1
	event1 := broker.NewToolStartedEvent(
		"session1",
		"message1",
		"agent1",
		"tool_call_1",
		"Read",
		map[string]interface{}{
			"file_path": "/Users/test/file.txt",
		},
		1,
	)

	tcm.ProcessToolStarted(event1)

	// Verify we have one chain
	chains := tcm.GetActiveChains()
	if len(chains) != 1 {
		t.Errorf("Expected 1 active chain, got %d", len(chains))
	}

	// Complete turn 1
	tcm.ProcessTurnCompleted(1)

	// Verify chains are cleaned up
	chains = tcm.GetActiveChains()
	if len(chains) != 0 {
		t.Errorf("Expected 0 active chains after turn completion, got %d", len(chains))
	}

	// Start a tool in turn 2
	event2 := broker.NewToolStartedEvent(
		"session1",
		"message2",
		"agent1",
		"tool_call_2",
		"Read",
		map[string]interface{}{
			"file_path": "/Users/test/file.txt",
		},
		2,
	)

	chain2 := tcm.ProcessToolStarted(event2)

	// Should be a new chain (not the same as the previous one)
	if chain2 == nil {
		t.Fatal("Failed to create new chain for turn 2")
	}

	// Verify it's turn 2
	if chain2.TurnID != 2 {
		t.Errorf("Expected TurnID 2, got %d", chain2.TurnID)
	}
}

func TestToolChainManager_GetChainByToolID(t *testing.T) {
	tcm := NewToolChainManager()

	// Start a tool
	event := broker.NewToolStartedEvent(
		"session1",
		"message1",
		"agent1",
		"tool_call_1",
		"Read",
		map[string]interface{}{
			"file_path": "/Users/test/file.txt",
		},
		1,
	)

	chain := tcm.ProcessToolStarted(event)
	if chain == nil {
		t.Fatal("Failed to start tool")
	}

	// Find chain by tool ID
	foundChain := tcm.GetChainByToolID("tool_call_1")
	if foundChain == nil {
		t.Fatal("Failed to find chain by tool ID")
	}

	if foundChain.ID != chain.ID {
		t.Errorf("Found chain ID %s does not match expected %s", foundChain.ID, chain.ID)
	}

	// Test with non-existent tool ID
	nonExistentChain := tcm.GetChainByToolID("non_existent")
	if nonExistentChain != nil {
		t.Error("Expected nil for non-existent tool ID")
	}
}