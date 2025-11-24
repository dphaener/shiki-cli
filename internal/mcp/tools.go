package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/darinhaener/collab/internal/events"
)

// ToolsBuilder creates MCP collaboration tools for agent communication
type ToolsBuilder struct {
	workspaceDir string
	sessionID    string
	eventBus     *events.EventBus
}

// NewToolsBuilder creates a new tools builder
func NewToolsBuilder(workspaceDir, sessionID string, eventBus *events.EventBus) *ToolsBuilder {
	return &ToolsBuilder{
		workspaceDir: workspaceDir,
		sessionID:    sessionID,
		eventBus:     eventBus,
	}
}

// CreateTools returns all collaboration tools for a specific agent
func (tb *ToolsBuilder) CreateTools(agentID string) []claude.McpTool {
	return []claude.McpTool{
		tb.createSendMessageTool(agentID),
		tb.createReadMessagesTool(agentID),
		tb.createWriteSharedContextTool(agentID),
		tb.createReadSharedContextTool(),
		tb.createUpdateMemoryTool(agentID),
		tb.createReadMemoryTool(agentID),
		tb.createSubmitDeliverableTool(agentID),
	}
}

// createSendMessageTool creates the send_message tool
func (tb *ToolsBuilder) createSendMessageTool(agentID string) claude.McpTool {
	return claude.Tool(
		"send_message",
		"Send a message to your collaborator agent",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "The message content to send to your collaborator",
				},
			},
			"required": []string{"message"},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			message, ok := args["message"].(string)
			if !ok {
				return errorResult("message parameter must be a string"), nil
			}

			// Determine recipient (other agent)
			recipientID := "agent_2"
			if strings.Contains(agentID, "agent_2") {
				recipientID = "agent_1"
			}

			// Write message using storage
			messageData := fmt.Sprintf("From: %s\nTo: %s\nTime: %s\n\n%s\n\n---\n\n",
				agentID, recipientID, time.Now().Format(time.RFC3339), message)

			messagesFile := filepath.Join(tb.workspaceDir, "messages", fmt.Sprintf("%s_to_%s.md", agentID, recipientID))

			// Ensure messages directory exists
			if err := os.MkdirAll(filepath.Dir(messagesFile), 0700); err != nil {
				return errorResult(fmt.Sprintf("Failed to create messages directory: %v", err)), nil
			}

			// Append to messages file
			f, err := os.OpenFile(messagesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to open messages file: %v", err)), nil
			}
			defer f.Close()

			if _, err := f.WriteString(messageData); err != nil {
				return errorResult(fmt.Sprintf("Failed to write message: %v", err)), nil
			}

			return successResult(fmt.Sprintf("Message sent to %s", recipientID)), nil
		},
	)
}

// createReadMessagesTool creates the read_messages tool
func (tb *ToolsBuilder) createReadMessagesTool(agentID string) claude.McpTool {
	return claude.Tool(
		"read_messages",
		"Read messages from your collaborator agent",
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			// Determine sender (other agent)
			senderID := "agent_2"
			if strings.Contains(agentID, "agent_2") {
				senderID = "agent_1"
			}

			messagesFile := filepath.Join(tb.workspaceDir, "messages", fmt.Sprintf("%s_to_%s.md", senderID, agentID))

			// Read messages file
			content, err := os.ReadFile(messagesFile)
			if err != nil {
				if os.IsNotExist(err) {
					return successResult("No messages yet"), nil
				}
				return errorResult(fmt.Sprintf("Failed to read messages: %v", err)), nil
			}

			if len(content) == 0 {
				return successResult("No messages yet"), nil
			}

			return successResult(string(content)), nil
		},
	)
}

// createWriteSharedContextTool creates the write_shared_context tool
func (tb *ToolsBuilder) createWriteSharedContextTool(agentID string) claude.McpTool {
	return claude.Tool(
		"write_shared_context",
		"Write to the shared context that both agents can see",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "Content to add to shared context",
				},
			},
			"required": []string{"content"},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			content, ok := args["content"].(string)
			if !ok {
				return errorResult("content parameter must be a string"), nil
			}

			contextFile := filepath.Join(tb.workspaceDir, "shared_context.md")

			// Append to shared context
			entry := fmt.Sprintf("\n---\n**Added by:** %s at %s\n\n%s\n",
				agentID, time.Now().Format(time.RFC3339), content)

			f, err := os.OpenFile(contextFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to open shared context: %v", err)), nil
			}
			defer f.Close()

			if _, err := f.WriteString(entry); err != nil {
				return errorResult(fmt.Sprintf("Failed to write shared context: %v", err)), nil
			}

			return successResult("Shared context updated"), nil
		},
	)
}

// createReadSharedContextTool creates the read_shared_context tool
func (tb *ToolsBuilder) createReadSharedContextTool() claude.McpTool {
	return claude.Tool(
		"read_shared_context",
		"Read the shared context that both agents can see",
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			contextFile := filepath.Join(tb.workspaceDir, "shared_context.md")

			content, err := os.ReadFile(contextFile)
			if err != nil {
				if os.IsNotExist(err) {
					return successResult("# Shared Context\n\nNo shared context yet"), nil
				}
				return errorResult(fmt.Sprintf("Failed to read shared context: %v", err)), nil
			}

			return successResult(string(content)), nil
		},
	)
}

// createUpdateMemoryTool creates the update_memory tool
func (tb *ToolsBuilder) createUpdateMemoryTool(agentID string) claude.McpTool {
	return claude.Tool(
		"update_memory",
		"Update your private memory (only you can see this)",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "Content to add to your memory",
				},
			},
			"required": []string{"content"},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			content, ok := args["content"].(string)
			if !ok {
				return errorResult("content parameter must be a string"), nil
			}
			memoryFile := filepath.Join(tb.workspaceDir, "memory", fmt.Sprintf("%s_memory.md", agentID))

			// Ensure memory directory exists
			if err := os.MkdirAll(filepath.Dir(memoryFile), 0700); err != nil {
				return errorResult(fmt.Sprintf("Failed to create memory directory: %v", err)), nil
			}

			// Append to memory
			entry := fmt.Sprintf("\n---\n**%s**\n\n%s\n",
				time.Now().Format(time.RFC3339), content)

			f, err := os.OpenFile(memoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to open memory: %v", err)), nil
			}
			defer f.Close()

			if _, err := f.WriteString(entry); err != nil {
				return errorResult(fmt.Sprintf("Failed to write memory: %v", err)), nil
			}

			return successResult("Memory updated"), nil
		},
	)
}

// createReadMemoryTool creates the read_memory tool
func (tb *ToolsBuilder) createReadMemoryTool(agentID string) claude.McpTool {
	return claude.Tool(
		"read_memory",
		"Read your private memory",
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			memoryFile := filepath.Join(tb.workspaceDir, "memory", fmt.Sprintf("%s_memory.md", agentID))

			content, err := os.ReadFile(memoryFile)
			if err != nil {
				if os.IsNotExist(err) {
					return successResult("# Memory\n\nNo memory yet"), nil
				}
				return errorResult(fmt.Sprintf("Failed to read memory: %v", err)), nil
			}

			return successResult(string(content)), nil
		},
	)
}

// createSubmitDeliverableTool creates the submit_deliverable tool
func (tb *ToolsBuilder) createSubmitDeliverableTool(agentID string) claude.McpTool {
	return claude.Tool(
		"submit_deliverable",
		"Submit the final deliverable when task is complete (both agents must submit matching deliverables)",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "The deliverable content",
				},
				"notes": map[string]any{
					"type":        "string",
					"description": "Notes about the deliverable",
				},
			},
			"required": []string{"content"},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			content, ok := args["content"].(string)
			if !ok {
				return errorResult("content parameter must be a string"), nil
			}

			notes, _ := args["notes"].(string)

			// Determine agent-specific file name
			deliverableFileName := "agent_1_deliverable.md"
			if strings.Contains(agentID, "agent_2") {
				deliverableFileName = "agent_2_deliverable.md"
			}

			deliverableFile := filepath.Join(tb.workspaceDir, deliverableFileName)

			// Write just the content to enable byte-for-byte comparison
			// Metadata would differ between agents and prevent matching
			if err := os.WriteFile(deliverableFile, []byte(content), 0600); err != nil {
				return errorResult(fmt.Sprintf("Failed to write deliverable: %v", err)), nil
			}

			// Also write metadata to a separate file for reference
			metadataFile := filepath.Join(tb.workspaceDir, fmt.Sprintf("%s_metadata.json", strings.TrimSuffix(deliverableFileName, ".md")))
			metadata := fmt.Sprintf(`{"agent": "%s", "timestamp": "%s", "notes": "%s"}`,
				agentID, time.Now().Format(time.RFC3339), notes)
			_ = os.WriteFile(metadataFile, []byte(metadata), 0600) // Best effort, ignore errors

			return successResult("Deliverable submitted successfully. When both agents submit matching deliverables, the session will complete."), nil
		},
	)
}

// Helper functions

func successResult(message string) *claude.McpToolResult {
	return &claude.McpToolResult{
		Content: []claude.ContentBlock{
			claude.TextContentBlock{
				Type: "text",
				Text: message,
			},
		},
		IsError: false,
	}
}

func errorResult(message string) *claude.McpToolResult {
	return &claude.McpToolResult{
		Content: []claude.ContentBlock{
			claude.TextContentBlock{
				Type: "text",
				Text: message,
			},
		},
		IsError: true,
	}
}

// getAgentIDFromContext extracts agent ID from context
// The agent manager will set this when calling tools
func getAgentIDFromContext(ctx context.Context) string {
	if agentID, ok := ctx.Value("agent_id").(string); ok {
		return agentID
	}
	return "unknown"
}
