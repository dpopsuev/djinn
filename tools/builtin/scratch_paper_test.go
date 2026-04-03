package builtin

import (
	"context"
	"encoding/json"
	"testing"
)

func TestScratchPaperTool_Name(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	if tool.Name() != "ScratchPaper" {
		t.Errorf("Name = %q, want ScratchPaper", tool.Name())
	}
}

func TestScratchPaperTool_WriteUnderstanding(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	ctx := context.Background()

	input, _ := json.Marshal(map[string]string{
		"action":     "write_understanding",
		"scratch_id": "SP-001",
		"content":    "The system needs auth",
	})

	result, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == "" {
		t.Error("result should not be empty")
	}

	sections := tool.ReadSections("SP-001")
	if sections["understanding"] != "The system needs auth" {
		t.Errorf("understanding = %q", sections["understanding"])
	}
}

func TestScratchPaperTool_WriteUnderstanding_Replaces(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	ctx := context.Background()

	input1, _ := json.Marshal(map[string]string{
		"action": "write_understanding", "scratch_id": "SP-001", "content": "first",
	})
	input2, _ := json.Marshal(map[string]string{
		"action": "write_understanding", "scratch_id": "SP-001", "content": "second",
	})

	tool.Execute(ctx, input1) //nolint:errcheck // test
	tool.Execute(ctx, input2) //nolint:errcheck // test

	sections := tool.ReadSections("SP-001")
	if sections["understanding"] != "second" {
		t.Errorf("understanding should be replaced, got %q", sections["understanding"])
	}
}

func TestScratchPaperTool_AddStep(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	ctx := context.Background()

	for _, step := range []string{"step 1", "step 2"} {
		input, _ := json.Marshal(map[string]string{
			"action": "add_step", "scratch_id": "SP-001", "content": step,
		})
		if _, err := tool.Execute(ctx, input); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	}

	sections := tool.ReadSections("SP-001")
	if sections["plan"] != "step 1\nstep 2" {
		t.Errorf("plan = %q, want 'step 1\\nstep 2'", sections["plan"])
	}
}

func TestScratchPaperTool_AddRisk(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	ctx := context.Background()

	input, _ := json.Marshal(map[string]string{
		"action": "add_risk", "scratch_id": "SP-001", "content": "data loss",
	})
	if _, err := tool.Execute(ctx, input); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	sections := tool.ReadSections("SP-001")
	if sections["risks"] != "data loss" {
		t.Errorf("risks = %q", sections["risks"])
	}
}

func TestScratchPaperTool_AppendNotes(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	ctx := context.Background()

	for _, note := range []string{"note 1", "note 2"} {
		input, _ := json.Marshal(map[string]string{
			"action": "append_notes", "scratch_id": "SP-001", "content": note,
		})
		tool.Execute(ctx, input) //nolint:errcheck // test
	}

	sections := tool.ReadSections("SP-001")
	if sections["notes"] != "note 1\nnote 2" {
		t.Errorf("notes = %q, want 'note 1\\nnote 2'", sections["notes"])
	}
}

func TestScratchPaperTool_UnknownAction(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	ctx := context.Background()

	input, _ := json.Marshal(map[string]string{
		"action": "unknown", "scratch_id": "SP-001", "content": "test",
	})
	_, err := tool.Execute(ctx, input)
	if err == nil {
		t.Error("unknown action should fail")
	}
}

func TestScratchPaperTool_MissingFields(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	ctx := context.Background()

	// Missing action.
	input, _ := json.Marshal(map[string]string{
		"scratch_id": "SP-001", "content": "test",
	})
	_, err := tool.Execute(ctx, input)
	if err == nil {
		t.Error("missing action should fail")
	}

	// Missing scratch_id.
	input, _ = json.Marshal(map[string]string{
		"action": "write_understanding", "content": "test",
	})
	_, err = tool.Execute(ctx, input)
	if err == nil {
		t.Error("missing scratch_id should fail")
	}

	// Missing content.
	input, _ = json.Marshal(map[string]string{
		"action": "write_understanding", "scratch_id": "SP-001",
	})
	_, err = tool.Execute(ctx, input)
	if err == nil {
		t.Error("missing content should fail")
	}
}

func TestScratchPaperTool_Sections(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	ctx := context.Background()

	// Empty.
	if s := tool.Sections("SP-001"); s != "empty scratch paper" {
		t.Errorf("empty sections = %q", s)
	}

	// Write some content.
	input, _ := json.Marshal(map[string]string{
		"action": "write_understanding", "scratch_id": "SP-001", "content": "auth needed",
	})
	tool.Execute(ctx, input) //nolint:errcheck // test

	summary := tool.Sections("SP-001")
	if summary == "empty scratch paper" {
		t.Error("Sections should not be empty after writing")
	}
}

func TestScratchPaperTool_InputSchema(t *testing.T) {
	tool := NewScratchPaperTool(nil)
	schema := tool.InputSchema()
	if len(schema) == 0 {
		t.Error("InputSchema should not be empty")
	}

	// Verify it's valid JSON.
	var v any
	if err := json.Unmarshal(schema, &v); err != nil {
		t.Errorf("InputSchema is not valid JSON: %v", err)
	}
}
