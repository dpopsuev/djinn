package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	globToolName = "Glob"
	globToolDesc = "Find files matching a glob pattern."
)

type globInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"` // base directory, defaults to "."
}

// GlobTool finds files by pattern.
type GlobTool struct{}

func (t *GlobTool) Name() string        { return globToolName }
func (t *GlobTool) Description() string { return globToolDesc }

func (t *GlobTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Glob pattern (e.g. **/*.go)"},
			"path": {"type": "string", "description": "Base directory (default: current dir)"}
		},
		"required": ["pattern"]
	}`)
}

func (t *GlobTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in globInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("glob: %w", err)
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("glob: %w", ErrEmptyInput)
	}

	pattern := in.Pattern
	if in.Path != "" {
		pattern = filepath.Join(in.Path, in.Pattern)
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		return "no files found", nil
	}

	return strings.Join(matches, "\n"), nil
}
