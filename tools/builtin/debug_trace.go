// debug_trace.go — Builtin tool wrapping the debug.Server for self-debugging.
//
// Exposes djinn_trace as a builtin tool so the agent can query its own
// trace ring. Actions: list, get, tree, health, stats.
package builtin

import (
	"context"
	"encoding/json"

	"github.com/dpopsuev/djinn/debug"
)

// DebugTraceTool wraps debug.Server as a builtin tool.
type DebugTraceTool struct {
	Server *debug.Server
}

// Name returns the tool name.
func (t *DebugTraceTool) Name() string { return "djinn_trace" }

// Description returns the tool description.
func (t *DebugTraceTool) Description() string {
	return "Query Djinn's trace ring for MCP call debugging: list, get, tree, health, stats"
}

// InputSchema returns the JSON schema for the tool input.
func (t *DebugTraceTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["list", "get", "tree", "health", "stats"]},
			"id": {"type": "string"},
			"parent_id": {"type": "string"},
			"component": {"type": "string"},
			"limit": {"type": "integer"}
		},
		"required": ["action"]
	}`)
}

// Execute runs the debug server dispatch.
func (t *DebugTraceTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	if t.Server == nil {
		return "trace not enabled", nil
	}

	var params debug.TraceInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "invalid input: " + err.Error(), nil //nolint:nilerr // user-facing error, not Go error
	}

	return t.Server.Handle(params)
}
