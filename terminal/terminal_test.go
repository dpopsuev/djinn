package terminal

import (
	"testing"
	"time"
)

func TestViewEventKinds(t *testing.T) {
	kinds := []ViewEventKind{
		EventOutput, EventThinking, EventToolCall, EventToolResult,
		EventAgentStatus, EventDashboard, EventError, EventDone,
	}
	seen := make(map[ViewEventKind]bool)
	for _, k := range kinds {
		if k == "" {
			t.Error("empty ViewEventKind")
		}
		if seen[k] {
			t.Errorf("duplicate kind: %s", k)
		}
		seen[k] = true
	}
	if len(kinds) != 8 {
		t.Fatalf("expected 8 event kinds, got %d", len(kinds))
	}
}

func TestRunState_ZeroValue(t *testing.T) {
	var rs RunState
	if rs.Operation != "" {
		t.Error("zero RunState should have empty operation")
	}
	if rs.IsStreaming {
		t.Error("zero RunState should not be streaming")
	}
}

func TestIntrospectionReport_Embeds_RunState(t *testing.T) {
	report := IntrospectionReport{
		RunState: RunState{
			Operation:  "agent",
			AgentCount: 2,
			AgentCap:   4,
			TokensIn:   1000,
			TokensOut:  500,
		},
		ToolLevel:    "L2",
		EnabledTools: []string{"Read", "Write", "Edit", "plan", "test"},
		Agents: []AgentInfo{
			{ID: "a1", Role: "executor", State: "streaming", Color: "#6F8FAF"},
			{ID: "a2", Role: "inspector", State: "idle"},
		},
		Uptime: 5 * time.Minute,
	}

	// RunState fields accessible directly.
	if report.Operation != "agent" {
		t.Errorf("Operation = %q, want agent", report.Operation)
	}
	if report.AgentCount != 2 {
		t.Errorf("AgentCount = %d, want 2", report.AgentCount)
	}

	// Introspection-specific fields.
	if report.ToolLevel != "L2" {
		t.Errorf("ToolLevel = %q, want L2", report.ToolLevel)
	}
	if len(report.EnabledTools) != 5 {
		t.Errorf("EnabledTools = %d, want 5", len(report.EnabledTools))
	}
	if len(report.Agents) != 2 {
		t.Errorf("Agents = %d, want 2", len(report.Agents))
	}
	if report.Agents[0].Color != "#6F8FAF" {
		t.Errorf("Agent color = %q, want #6F8FAF", report.Agents[0].Color)
	}
}

func TestViewEvent_Timestamp(t *testing.T) {
	ev := ViewEvent{
		Kind:      EventOutput,
		Text:      "hello",
		Timestamp: time.Now(),
	}
	if ev.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
	if ev.Kind != EventOutput {
		t.Errorf("kind = %s, want output", ev.Kind)
	}
}

// Compile-time interface compliance checks.
// These verify that any concrete type implementing Terminal
// also satisfies Controller and Viewer individually.
func TestInterfaceCompliance(t *testing.T) {
	// A concrete type that implements Terminal must satisfy Controller and Viewer.
	// This is guaranteed by Go's type system (Terminal embeds both),
	// but we document the contract explicitly.
	var _ interface {
		Controller
		Viewer
	} = Terminal(nil)
}
