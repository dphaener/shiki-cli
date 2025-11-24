package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
	"github.com/darinhaener/collab/internal/events"
)

// ToolsBuilder creates MCP collaboration tools for agent communication
type ToolsBuilder struct {
	workspaceDir string
	sessionID    string
	eventBus     *events.EventBus
	server       *Server
}

// NewToolsBuilder creates a new tools builder
func NewToolsBuilder(server *Server) *ToolsBuilder {
	return &ToolsBuilder{
		workspaceDir: server.workspaceDir,
		sessionID:    server.sessionID,
		eventBus:     server.eventBus,
		server:       server,
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
				"content": map[string]any{
					"type":        "string",
					"description": "The message content to send to your collaborator",
				},
			},
			"required": []string{"content"},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			// Extract turn number from context
			turn := 0
			if turnVal := ctx.Value("turn"); turnVal != nil {
				if t, ok := turnVal.(int); ok {
					turn = t
				}
			}

			// Call server handler which handles storage and events
			result, err := tb.server.HandleSendMessage(agentID, turn, args)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to send message: %v", err)), nil
			}

			// Extract message path from result
			messagePath, _ := result["message_path"].(string)
			return successResult(fmt.Sprintf("Message sent successfully (saved to %s)", messagePath)), nil
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
			// Extract turn number from context
			turn := 0
			if turnVal := ctx.Value("turn"); turnVal != nil {
				if t, ok := turnVal.(int); ok {
					turn = t
				}
			}

			// Call server handler
			result, err := tb.server.HandleReadMessages(agentID, turn, args)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to read messages: %v", err)), nil
			}

			// Format messages for display
			messages, ok := result["messages"].([]map[string]interface{})
			if !ok || len(messages) == 0 {
				return successResult("No messages yet"), nil
			}

			// Build formatted message list
			var output strings.Builder
			output.WriteString(fmt.Sprintf("Found %d message(s):\n\n", len(messages)))
			for i, msg := range messages {
				from, _ := msg["from"].(string)
				timestamp, _ := msg["timestamp"].(string)
				content, _ := msg["content"].(string)
				output.WriteString(fmt.Sprintf("Message %d:\nFrom: %s\nTime: %s\n\n%s\n\n---\n\n", i+1, from, timestamp, content))
			}

			return successResult(output.String()), nil
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
			// Extract turn number from context
			turn := 0
			if turnVal := ctx.Value("turn"); turnVal != nil {
				if t, ok := turnVal.(int); ok {
					turn = t
				}
			}

			// Call server handler
			result, err := tb.server.HandleWriteSharedContext(agentID, turn, args)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to write shared context: %v", err)), nil
			}

			path, _ := result["path"].(string)
			return successResult(fmt.Sprintf("Shared context updated (saved to %s)", path)), nil
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
			// Extract turn number from context
			turn := 0
			if turnVal := ctx.Value("turn"); turnVal != nil {
				if t, ok := turnVal.(int); ok {
					turn = t
				}
			}

			// Need agentID from context since this tool doesn't take it as parameter
			agentID := "unknown"
			if agentVal := ctx.Value("agent_id"); agentVal != nil {
				if id, ok := agentVal.(string); ok {
					agentID = id
				}
			}

			// Call server handler
			result, err := tb.server.HandleReadSharedContext(agentID, turn, args)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to read shared context: %v", err)), nil
			}

			content, _ := result["content"].(string)
			return successResult(content), nil
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
			// Extract turn number from context
			turn := 0
			if turnVal := ctx.Value("turn"); turnVal != nil {
				if t, ok := turnVal.(int); ok {
					turn = t
				}
			}

			// Call server handler
			result, err := tb.server.HandleUpdateMemory(agentID, turn, args)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to update memory: %v", err)), nil
			}

			path, _ := result["path"].(string)
			return successResult(fmt.Sprintf("Memory updated (saved to %s)", path)), nil
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
			// Extract turn number from context
			turn := 0
			if turnVal := ctx.Value("turn"); turnVal != nil {
				if t, ok := turnVal.(int); ok {
					turn = t
				}
			}

			// Call server handler
			result, err := tb.server.HandleReadMemory(agentID, turn, args)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to read memory: %v", err)), nil
			}

			content, _ := result["content"].(string)
			return successResult(content), nil
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
				"approved": map[string]any{
					"type":        "boolean",
					"description": "Confirm that you approve this deliverable (must be true)",
				},
			},
			"required": []string{"content", "approved"},
		},
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
			// Extract turn number from context
			turn := 0
			if turnVal := ctx.Value("turn"); turnVal != nil {
				if t, ok := turnVal.(int); ok {
					turn = t
				}
			}

			// Call server handler
			result, err := tb.server.HandleSubmitDeliverable(agentID, turn, args)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to submit deliverable: %v", err)), nil
			}

			status, _ := result["status"].(string)
			message, _ := result["message"].(string)

			return successResult(fmt.Sprintf("[%s] %s", status, message)), nil
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
