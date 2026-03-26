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
	"errors"
	"fmt"
)

// Sentinel errors for render tool validation.
var (
	ErrRenderMissingField = errors.New("render: type and title required")
	ErrRenderUnknownType  = errors.New("render: unknown type")
	ErrRenderInvalidData  = errors.New("render: data is not valid JSON")
)

// Valid render types.
var validRenderTypes = map[string]bool{
	"table": true, "tree": true, "progress": true, "chart": true,
	"diff": true, "diagram": true, "timeline": true, "code": true,
}

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
		return "", ErrRenderMissingField
	}

	if !validRenderTypes[req.Type] {
		return "", fmt.Errorf("%w: %q", ErrRenderUnknownType, req.Type)
	}

	if req.Data != "" && !json.Valid([]byte(req.Data)) {
		return "", ErrRenderInvalidData
	}

	// Return the input as-is — the handler reads type+title+data
	// and emits RenderPanelMsg to the TUI.
	return string(input), nil
}
