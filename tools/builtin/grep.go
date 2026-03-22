package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	grepToolName = "Grep"
	grepToolDesc = "Search file contents for a regex pattern."
	maxGrepResults = 100
)

type grepInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Limit   int    `json:"limit,omitempty"` // max results, default 100
}

// GrepTool searches files by regex pattern.
type GrepTool struct{}

func (t *GrepTool) Name() string        { return grepToolName }
func (t *GrepTool) Description() string { return grepToolDesc }

func (t *GrepTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Regex pattern to search for"},
			"path": {"type": "string", "description": "File path to search"},
			"limit": {"type": "integer", "description": "Max results (default 100)"}
		},
		"required": ["pattern", "path"]
	}`)
}

func (t *GrepTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in grepInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("grep: %w", err)
	}
	if in.Pattern == "" || in.Path == "" {
		return "", fmt.Errorf("grep: %w", ErrEmptyInput)
	}

	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		return "", fmt.Errorf("grep invalid regex %q: %w", in.Pattern, err)
	}

	limit := maxGrepResults
	if in.Limit > 0 {
		limit = in.Limit
	}

	f, err := os.Open(in.Path)
	if err != nil {
		return "", fmt.Errorf("grep open %s: %w", in.Path, err)
	}
	defer f.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(f)
	lineNum := 0
	matchCount := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			fmt.Fprintf(&sb, "%s:%d: %s\n", in.Path, lineNum, line)
			matchCount++
			if matchCount >= limit {
				fmt.Fprintf(&sb, "... (truncated at %d matches)\n", limit)
				break
			}
		}
	}

	if matchCount == 0 {
		return "no matches found", nil
	}
	return sb.String(), nil
}
