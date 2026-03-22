package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	writeToolName = "Write"
	writeToolDesc = "Create or overwrite a file with the given content."
)

type writeInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WriteTool creates or overwrites files.
type WriteTool struct{}

func (t *WriteTool) Name() string        { return writeToolName }
func (t *WriteTool) Description() string { return writeToolDesc }

func (t *WriteTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path to write"},
			"content": {"type": "string", "description": "File content"}
		},
		"required": ["path", "content"]
	}`)
}

func (t *WriteTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in writeInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	if in.Path == "" {
		return "", fmt.Errorf("write: %w", ErrEmptyInput)
	}

	dir := filepath.Dir(in.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("write mkdir %s: %w", dir, err)
	}

	if err := os.WriteFile(in.Path, []byte(in.Content), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", in.Path, err)
	}

	return fmt.Sprintf("wrote %d bytes to %s", len(in.Content), in.Path), nil
}
