package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	readToolName = "Read"
	readToolDesc = "Read a file and return its contents with line numbers."
)

type readInput struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"` // 1-based line to start from
	Limit  int    `json:"limit,omitempty"`  // max lines to return
}

// ReadTool reads files with optional line offset and limit.
type ReadTool struct{}

func (t *ReadTool) Name() string        { return readToolName }
func (t *ReadTool) Description() string { return readToolDesc }

func (t *ReadTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Absolute or relative file path"},
			"offset": {"type": "integer", "description": "Line number to start from (1-based)"},
			"limit": {"type": "integer", "description": "Max lines to return"}
		},
		"required": ["path"]
	}`)
}

func (t *ReadTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in readInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	if in.Path == "" {
		return "", fmt.Errorf("read: %w", ErrEmptyInput)
	}

	data, err := os.ReadFile(in.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", in.Path, err)
	}

	lines := strings.Split(string(data), "\n")

	start := 0
	if in.Offset > 0 {
		start = in.Offset - 1
	}
	if start > len(lines) {
		start = len(lines)
	}

	end := len(lines)
	if in.Limit > 0 && start+in.Limit < end {
		end = start + in.Limit
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%6d\t%s\n", i+1, lines[i])
	}
	return sb.String(), nil
}
