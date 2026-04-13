// symbol.go — Symbol tool: agent-facing search over Lector's indexes.
//
// Replaces grep/sed bashism with structured symbol queries.
// Actions: search (fuzzy), symbols (per file), callers (future).
//
// GOL-152, TSK-1117
package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dpopsuev/djinn/lector"
)

// SymbolTool provides structured symbol queries backed by Lector.
type SymbolTool struct {
	Lector lector.Lector
}

func (t *SymbolTool) Name() string        { return "Symbol" }
func (t *SymbolTool) Description() string { return "Search symbols and files: search, symbols, files" }
func (t *SymbolTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["search", "symbols", "files"]},
			"query":  {"type": "string"},
			"file":   {"type": "string"},
			"scope":  {"type": "string", "enum": ["symbols", "files", "both"]}
		},
		"required": ["action"]
	}`)
}

func (t *SymbolTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var req struct {
		Action string `json:"action"`
		Query  string `json:"query"`
		File   string `json:"file"`
		Scope  string `json:"scope"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("symbol: %w", err)
	}

	switch req.Action {
	case "search":
		return t.search(req.Query, req.Scope)
	case "symbols":
		return t.symbolsForFile(req.File)
	case "files":
		return t.fuzzyFiles(req.Query)
	default:
		return "", fmt.Errorf("symbol: unknown action %q", req.Action)
	}
}

func (t *SymbolTool) search(query, scope string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("symbol search: query is required")
	}
	if scope == "" {
		scope = "both"
	}

	type searchResult struct {
		Files   []lector.FileEntry `json:"files,omitempty"`
		Symbols []lector.Symbol    `json:"symbols,omitempty"`
	}

	var result searchResult
	if scope == "files" || scope == "both" {
		result.Files = t.Lector.FuzzyFiles(query)
	}
	if scope == "symbols" || scope == "both" {
		result.Symbols = t.Lector.FuzzySymbols(query)
	}

	j, _ := json.Marshal(result)
	return string(j), nil
}

func (t *SymbolTool) symbolsForFile(file string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("symbol symbols: file is required")
	}

	// Ensure file is indexed.
	t.Lector.OnFileRead(file)

	syms := t.Lector.SymbolsForFile(file)
	j, _ := json.Marshal(syms)
	return string(j), nil
}

func (t *SymbolTool) fuzzyFiles(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("symbol files: query is required")
	}
	files := t.Lector.FuzzyFiles(query)
	j, _ := json.Marshal(files)
	return string(j), nil
}
