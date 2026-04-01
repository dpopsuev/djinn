package tui

import (
	"strings"
	"testing"
)

func TestCellSight_IsEmpty(t *testing.T) {
	if !(CellSight{}).IsEmpty() {
		t.Error("zero-value CellSight should be empty")
	}
	if (CellSight{PanelID: "debug"}).IsEmpty() {
		t.Error("CellSight with PanelID should not be empty")
	}
	if (CellSight{CellID: "trace-1"}).IsEmpty() {
		t.Error("CellSight with CellID should not be empty")
	}
}

func TestCellSight_FormatPrompt_Empty(t *testing.T) {
	fc := CellSight{}
	if fc.FormatPrompt() != "" {
		t.Error("empty context should produce empty prompt")
	}
}

func TestCellSight_FormatPrompt_Full(t *testing.T) {
	fc := CellSight{
		PanelID:   "plan",
		CellID:    "GOL-59",
		CellTitle: "Artifact Primitive",
		Kind:      "goal",
		Metadata:  map[string]string{"status": "draft", "priority": "critical"},
	}

	result := fc.FormatPrompt()

	if !strings.Contains(result, "<focus-context>") {
		t.Error("should contain opening tag")
	}
	if !strings.Contains(result, "</focus-context>") {
		t.Error("should contain closing tag")
	}
	if !strings.Contains(result, "Panel: plan") {
		t.Error("should contain panel ID")
	}
	if !strings.Contains(result, "GOL-59") {
		t.Error("should contain cell ID")
	}
	if !strings.Contains(result, "Artifact Primitive") {
		t.Error("should contain cell title")
	}
	if !strings.Contains(result, "Kind: goal") {
		t.Error("should contain kind")
	}
}

func TestCellSight_FormatPrompt_TokenBudget(t *testing.T) {
	fc := CellSight{
		PanelID:   "debug",
		CellID:    "trace-42",
		CellTitle: "call mcp_server tool_name with detailed parameters",
		Kind:      "trace-event",
		Metadata: map[string]string{
			"component": "mcp",
			"action":    "call",
			"server":    "scribe",
			"tool":      "artifact",
			"latency":   "150ms",
		},
	}

	result := fc.FormatPrompt()
	// Rough token estimate: ~4 chars per token. 200 tokens = 800 chars.
	if len(result) > 800 { //nolint:mnd // 200 token budget ≈ 800 chars
		t.Errorf("FormatPrompt too long: %d chars (budget: ~800)", len(result))
	}
}

func TestCellSight_FormatPrompt_PanelOnly(t *testing.T) {
	fc := CellSight{PanelID: "output"}
	result := fc.FormatPrompt()
	if !strings.Contains(result, "Panel: output") {
		t.Error("should contain panel ID even without cell")
	}
	if strings.Contains(result, "Selected:") {
		t.Error("should not contain Selected when no cell")
	}
}
