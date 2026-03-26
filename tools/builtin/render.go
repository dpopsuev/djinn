// render.go — RenderTool: agent-callable visual expression.
//
// The agent calls render() with structured data (type + title + data payload).
// The tool validates the input and returns it as JSON. The TUI handler
// intercepts render results and emits RenderPanelMsg to draw the panel.
//
// This is Aeon Shell tool #8: plan, test, git, arch, discourse, reconcile,
// latency, render.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
)

// RenderTool lets agents create structured visual panels in the TUI.
type RenderTool struct{}

func (t *RenderTool) Name() string        { return "render" }
func (t *RenderTool) Description() string { return "Create visual panels: table, tree, progress, chart, diff, diagram, timeline, code" }
func (t *RenderTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"type":  {"type": "string", "enum": ["table", "tree", "progress", "chart", "diff", "diagram", "timeline", "code"]},
			"title": {"type": "string"},
			"data":  {"type": "string", "description": "JSON payload, type-specific"}
		},
		"required": ["type", "title", "data"]
	}`)
}

// Execute validates the render input and returns it as JSON.
// The TUI handler intercepts this result and emits a RenderPanelMsg.
func (t *RenderTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Type  string `json:"type"`
		Title string `json:"title"`
		Data  string `json:"data"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("render: %w", err)
	}

	if req.Type == "" || req.Title == "" {
		return "", fmt.Errorf("render: type and title required")
	}

	switch req.Type {
	case "table", "tree", "progress", "chart", "diff", "diagram", "timeline", "code":
		// Valid type — pass through
	default:
		return "", fmt.Errorf("render: unknown type %q (supported: table, tree, progress, chart, diff, diagram, timeline, code)", req.Type)
	}

	// Validate data is valid JSON.
	if req.Data != "" {
		if !json.Valid([]byte(req.Data)) {
			return "", fmt.Errorf("render: data is not valid JSON")
		}
	}

	// Return the input as-is — the handler reads type+title+data
	// and emits RenderPanelMsg to the TUI.
	return string(input), nil
}
