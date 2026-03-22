package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// MCPTool wraps an MCP server tool as a builtin.Tool.
// The agent loop and Claude API see it as any other tool.
type MCPTool struct {
	client     *Client
	serverName string
	def        ToolDef
}

// MCPTools returns all connected MCP tools as MCPTool adapters.
func (c *Client) MCPTools() []*MCPTool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var tools []*MCPTool
	for serverName, conn := range c.servers {
		for _, def := range conn.tools {
			tools = append(tools, &MCPTool{
				client:     c,
				serverName: serverName,
				def:        def,
			})
		}
	}
	return tools
}

// Name returns the prefixed tool name: mcp__<server>__<tool>
func (t *MCPTool) Name() string {
	return fmt.Sprintf("mcp__%s__%s", t.serverName, t.def.Name)
}

// Description returns the tool's description from the MCP server.
func (t *MCPTool) Description() string {
	return t.def.Description
}

// InputSchema returns the JSON Schema for the tool's input.
func (t *MCPTool) InputSchema() json.RawMessage {
	return t.def.InputSchema
}

// Execute calls the MCP server's tools/call endpoint.
func (t *MCPTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return t.client.Call(ctx, t.serverName, t.def.Name, input)
}
