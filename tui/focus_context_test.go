package tui

import (
	"strings"
	"testing"
)

func TestFocusContext_IsEmpty(t *testing.T) {
	if !(FocusContext{}).IsEmpty() {
		t.Error("zero-value FocusContext should be empty")
	}
	if (FocusContext{PanelID: "debug"}).IsEmpty() {
		t.Error("FocusContext with PanelID should not be empty")
	}
	if (FocusContext{ElementID: "trace-1"}).IsEmpty() {
		t.Error("FocusContext with ElementID should not be empty")
	}
}

func TestFocusContext_FormatPrompt_Empty(t *testing.T) {
	fc := FocusContext{}
	if fc.FormatPrompt() != "" {
		t.Error("empty context should produce empty prompt")
	}
}

func TestFocusContext_FormatPrompt_Full(t *testing.T) {
	fc := FocusContext{
		PanelID:      "plan",
		ElementID:    "GOL-59",
		ElementTitle: "Artifact Primitive",
		Kind:         "goal",
		Metadata:     map[string]string{"status": "draft", "priority": "critical"},
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
		t.Error("should contain element ID")
	}
	if !strings.Contains(result, "Artifact Primitive") {
		t.Error("should contain element title")
	}
	if !strings.Contains(result, "Kind: goal") {
		t.Error("should contain kind")
	}
}

func TestFocusContext_FormatPrompt_TokenBudget(t *testing.T) {
	fc := FocusContext{
		PanelID:      "debug",
		ElementID:    "trace-42",
		ElementTitle: "call mcp_server tool_name with detailed parameters",
		Kind:         "trace-event",
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

func TestFocusContext_FormatPrompt_PanelOnly(t *testing.T) {
	fc := FocusContext{PanelID: "output"}
	result := fc.FormatPrompt()
	if !strings.Contains(result, "Panel: output") {
		t.Error("should contain panel ID even without element")
	}
	if strings.Contains(result, "Selected:") {
		t.Error("should not contain Selected when no element")
	}
}
