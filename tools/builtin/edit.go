package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	editToolName = "Edit"
	editToolDesc = "Replace a specific string in a file. The old_string must match exactly once."
)

var ErrNoMatch = errors.New("old_string not found in file")
var ErrMultipleMatches = errors.New("old_string matches multiple locations")

type editInput struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// EditTool performs search-and-replace edits on files.
type EditTool struct{}

func (t *EditTool) Name() string        { return editToolName }
func (t *EditTool) Description() string { return editToolDesc }

func (t *EditTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path to edit"},
			"old_string": {"type": "string", "description": "Exact string to find (must match once)"},
			"new_string": {"type": "string", "description": "Replacement string"}
		},
		"required": ["path", "old_string", "new_string"]
	}`)
}

func (t *EditTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in editInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}
	if in.Path == "" || in.OldString == "" {
		return "", fmt.Errorf("edit: %w", ErrEmptyInput)
	}

	data, err := os.ReadFile(in.Path)
	if err != nil {
		return "", fmt.Errorf("edit read %s: %w", in.Path, err)
	}

	content := string(data)
	count := strings.Count(content, in.OldString)
	switch count {
	case 0:
		return "", fmt.Errorf("edit %s: %w", in.Path, ErrNoMatch)
	case 1:
		// exactly one match — proceed
	default:
		return "", fmt.Errorf("edit %s: %w (%d occurrences)", in.Path, ErrMultipleMatches, count)
	}

	newContent := strings.Replace(content, in.OldString, in.NewString, 1)
	if err := os.WriteFile(in.Path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("edit write %s: %w", in.Path, err)
	}

	return fmt.Sprintf("edited %s: replaced 1 occurrence", in.Path), nil
}
