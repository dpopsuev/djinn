package troupe

import (
	"context"
	"encoding/json"
	"testing"

	batterytool "github.com/dpopsuev/battery/tool"
)

// stubTool implements battery tool.Tool for testing.
type stubTool struct {
	name   string
	desc   string
	schema json.RawMessage
}

func (s *stubTool) Name() string                                                 { return s.name }
func (s *stubTool) Description() string                                          { return s.desc }
func (s *stubTool) InputSchema() json.RawMessage                                 { return s.schema }
func (s *stubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return "", nil }

func TestWithBatteryTools_ConvertsCorrectly(t *testing.T) {
	tools := []batterytool.Tool{
		&stubTool{
			name:   "Write",
			desc:   "Write a file",
			schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
		},
		&stubTool{
			name:   "Read",
			desc:   "Read a file",
			schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		},
	}

	converted := convertBatteryTools(tools)

	if len(converted) != 2 {
		t.Fatalf("converted = %d, want 2", len(converted))
	}

	if converted[0].Function.Name != "Write" {
		t.Fatalf("name = %q, want Write", converted[0].Function.Name)
	}
	if converted[0].Function.Description != "Write a file" {
		t.Fatalf("desc = %q", converted[0].Function.Description)
	}
	if converted[0].Type != "function" {
		t.Fatalf("type = %q, want function", converted[0].Type)
	}

	// Parameters should be parsed from JSON to map.
	params := converted[0].Function.Parameters
	if params == nil {
		t.Fatal("parameters should not be nil")
	}
	if params["type"] != "object" {
		t.Fatalf("params type = %v", params["type"])
	}

	if converted[1].Function.Name != "Read" {
		t.Fatalf("second tool name = %q, want Read", converted[1].Function.Name)
	}
}

func TestWithBatteryTools_EmptyList(t *testing.T) {
	converted := convertBatteryTools(nil)
	if len(converted) != 0 {
		t.Fatalf("converted = %d, want 0", len(converted))
	}
}

func TestWithBatteryTools_BadSchema(t *testing.T) {
	tools := []batterytool.Tool{
		&stubTool{
			name:   "Broken",
			desc:   "Has invalid JSON schema",
			schema: json.RawMessage(`{{{invalid`),
		},
	}

	converted := convertBatteryTools(tools)

	// Should still convert — Parameters will be nil but no crash.
	if len(converted) != 1 {
		t.Fatalf("converted = %d, want 1", len(converted))
	}
	if converted[0].Function.Name != "Broken" {
		t.Fatalf("name = %q", converted[0].Function.Name)
	}
}
