package mcp

// ToolDefinition represents an MCP tool with its schema
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetTools returns all 7 collaboration tools exposed by the MCP server
// Schemas match contracts/mcp-tools.json specification
func (s *Server) GetTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "send_message",
			Description: "Send a markdown message to the other agent in the collaboration",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Markdown content of the message to send",
						"minLength":   1,
					},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "read_messages",
			Description: "Read all messages from the other agent, optionally filtered by turn number",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"since_turn": map[string]interface{}{
						"type":        "integer",
						"description": "Only return messages from turns >= this value. Omit to get all messages.",
						"minimum":     0,
					},
				},
			},
		},
		{
			Name:        "write_shared_context",
			Description: "Overwrite the shared collaboration context file with new content. Both agents can read and write this file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "New markdown content for shared_context.md",
						"minLength":   0,
					},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "read_shared_context",
			Description: "Read the current content of the shared collaboration context file",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "update_memory",
			Description: "Overwrite this agent's private memory file with new content. Only this agent can read/write this file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "New markdown content for this agent's memory.md",
						"minLength":   0,
					},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "read_memory",
			Description: "Read the current content of this agent's private memory file",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "submit_deliverable",
			Description: "Submit the final collaboration deliverable with approval. Session completes when both agents submit identical content with approval=true.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Final deliverable markdown content",
						"minLength":   1,
					},
					"approved": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether this agent approves this deliverable. Must be true.",
						"const":       true,
					},
				},
				"required": []string{"content", "approved"},
			},
		},
	}
}
