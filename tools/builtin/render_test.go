package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// RED: Validation errors
// ═══════════════════════════════════════════════════════════════════════

func TestRenderTool_MissingType(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{"title": "T", "data": "{}"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "type and title required") {
		t.Fatalf("err = %v, want type/title required", err)
	}
}

func TestRenderTool_MissingTitle(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{"type": "table", "data": "{}"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "type and title required") {
		t.Fatalf("err = %v, want type/title required", err)
	}
}

func TestRenderTool_UnknownType(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{"type": "mermaid", "title": "T", "data": "{}"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "unknown type") {
		t.Fatalf("err = %v, want unknown type", err)
	}
}

func TestRenderTool_InvalidDataJSON(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{"type": "table", "title": "T", "data": "not json"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("err = %v, want not valid JSON", err)
	}
}

func TestRenderTool_InvalidInput(t *testing.T) {
	tool := &RenderTool{}
	_, err := tool.Execute(context.Background(), []byte("garbage"))
	if err == nil {
		t.Fatal("expected error on invalid JSON input")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GREEN: Valid passthrough
// ═══════════════════════════════════════════════════════════════════════

func TestRenderTool_Table(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{
		"type": "table", "title": "Results",
		"data": `{"columns":["A"],"rows":[["1"]]}`,
	})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out, "table") {
		t.Fatalf("output should contain type: %q", out)
	}
}

func TestRenderTool_Tree(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{
		"type": "tree", "title": "Hierarchy",
		"data": `{"root":{"label":"root"}}`,
	})
	out, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out, "tree") {
		t.Fatalf("output = %q", out)
	}
}

func TestRenderTool_Progress(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{
		"type": "progress", "title": "Sprint",
		"data": `{"done":3,"total":5}`,
	})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestRenderTool_Chart(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{
		"type": "chart", "title": "Latency",
		"data": `{"kind":"sparkline","values":[1,2,3]}`,
	})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BLUE: Edge cases
// ═══════════════════════════════════════════════════════════════════════

func TestRenderTool_EmptyData(t *testing.T) {
	tool := &RenderTool{}
	input, _ := json.Marshal(map[string]string{
		"type": "table", "title": "Empty",
		"data": "",
	})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("empty data should pass: %v", err)
	}
}

func TestRenderTool_NameAndDescription(t *testing.T) {
	tool := &RenderTool{}
	if tool.Name() != "render" {
		t.Fatalf("Name = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Fatal("Description should not be empty")
	}
}

func TestRenderTool_InputSchema(t *testing.T) {
	tool := &RenderTool{}
	schema := tool.InputSchema()
	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}
	if parsed["type"] != "object" {
		t.Fatalf("schema type = %v", parsed["type"])
	}
}
