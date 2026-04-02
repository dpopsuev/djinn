package tui

import (
	"strings"
	"testing"
)

func TestSightManager_DefaultGateOpen(t *testing.T) {
	m := NewSightManager()

	// Panels not explicitly configured should default to open.
	if !m.IsGateOpen("debug") {
		t.Error("default gate should be open for 'debug'")
	}
	if !m.IsGateOpen("plan") {
		t.Error("default gate should be open for 'plan'")
	}
	if !m.IsGateOpen("nonexistent") {
		t.Error("default gate should be open for unknown panels")
	}
}

func TestSightManager_SetGateOff_BlocksInjection(t *testing.T) {
	m := NewSightManager()

	// Close the gate.
	m.SetGate("debug", false)
	if m.IsGateOpen("debug") {
		t.Error("gate should be closed after SetGate(false)")
	}

	// Other panels unaffected.
	if !m.IsGateOpen("plan") {
		t.Error("closing 'debug' gate should not affect 'plan'")
	}

	// Re-open the gate.
	m.SetGate("debug", true)
	if !m.IsGateOpen("debug") {
		t.Error("gate should be open after SetGate(true)")
	}
}

func TestSightManager_Reveal_OverridesSensitive(t *testing.T) {
	m := NewSightManager()

	// Before reveal, IsRevealed should be false.
	if m.IsRevealed("debug", "latency") {
		t.Error("field should not be revealed by default")
	}

	// Reveal the field.
	m.Reveal("debug", "latency")
	if !m.IsRevealed("debug", "latency") {
		t.Error("field should be revealed after Reveal()")
	}

	// Verify via ApplyCellSight.
	cs := CellSight{
		PanelID: "debug",
		CellID:  "trace-1",
		Fields: []SightField{
			{Key: "component", Value: "mcp"},
			{Key: "latency", Value: "42ms", Sensitive: true},
		},
	}
	applied := m.ApplyCellSight(cs)

	// Latency should no longer be sensitive.
	for _, f := range applied.Fields {
		if f.Key == "latency" && f.Sensitive {
			t.Error("revealed field 'latency' should not be Sensitive")
		}
	}
	// Component should remain non-sensitive (unchanged).
	for _, f := range applied.Fields {
		if f.Key == "component" && f.Sensitive {
			t.Error("'component' should remain non-Sensitive")
		}
	}

	// FormatPrompt should now include the revealed field.
	prompt := applied.FormatPrompt()
	if !strings.Contains(prompt, "latency: 42ms") {
		t.Errorf("revealed field should appear in prompt, got:\n%s", prompt)
	}
}

func TestSightManager_Hide_RestoresSensitive(t *testing.T) {
	m := NewSightManager()

	// Reveal, then hide.
	m.Reveal("debug", "latency")
	if !m.IsRevealed("debug", "latency") {
		t.Fatal("precondition: field should be revealed")
	}

	m.Hide("debug", "latency")
	if m.IsRevealed("debug", "latency") {
		t.Error("field should not be revealed after Hide()")
	}

	// Verify via ApplyCellSight — originally non-sensitive field now hidden.
	cs := CellSight{
		PanelID: "debug",
		CellID:  "trace-1",
		Fields: []SightField{
			{Key: "latency", Value: "42ms", Sensitive: false}, // originally visible
		},
	}
	applied := m.ApplyCellSight(cs)

	for _, f := range applied.Fields {
		if f.Key == "latency" && !f.Sensitive {
			t.Error("hidden field 'latency' should be Sensitive after Hide()")
		}
	}

	// FormatPrompt should NOT include the hidden field.
	prompt := applied.FormatPrompt()
	if strings.Contains(prompt, "latency: 42ms") {
		t.Errorf("hidden field should not appear in prompt, got:\n%s", prompt)
	}
}

func TestSightManager_Status_ShowsState(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		m := NewSightManager()
		status := m.Status()
		if !strings.Contains(status, "all panels open") {
			t.Errorf("default status should mention 'all panels open', got:\n%s", status)
		}
		if !strings.Contains(status, "no field overrides") {
			t.Errorf("default status should mention 'no field overrides', got:\n%s", status)
		}
	})

	t.Run("with_gates_and_reveals", func(t *testing.T) {
		m := NewSightManager()
		m.SetGate("debug", false)
		m.SetGate("plan", true)
		m.Reveal("debug", "latency")
		m.Hide("plan", "secret")

		status := m.Status()

		// Gates section.
		if !strings.Contains(status, "debug: off") {
			t.Errorf("status should show debug gate off, got:\n%s", status)
		}
		if !strings.Contains(status, "plan: on") {
			t.Errorf("status should show plan gate on, got:\n%s", status)
		}

		// Fields section.
		if !strings.Contains(status, "debug.latency: revealed") {
			t.Errorf("status should show debug.latency revealed, got:\n%s", status)
		}
		if !strings.Contains(status, "plan.secret: hidden") {
			t.Errorf("status should show plan.secret hidden, got:\n%s", status)
		}
	})
}

func TestSightManager_ApplyCellSight_NoFields(t *testing.T) {
	m := NewSightManager()
	cs := CellSight{PanelID: "debug"}
	applied := m.ApplyCellSight(cs)
	if applied.PanelID != "debug" {
		t.Error("PanelID should be preserved")
	}
	if len(applied.Fields) != 0 {
		t.Error("no fields should remain empty")
	}
}

func TestSightManager_ApplyCellSight_DoesNotMutateOriginal(t *testing.T) {
	m := NewSightManager()
	m.Reveal("debug", "latency")

	cs := CellSight{
		PanelID: "debug",
		Fields: []SightField{
			{Key: "latency", Value: "42ms", Sensitive: true},
		},
	}

	_ = m.ApplyCellSight(cs)

	// Original should still be sensitive.
	if !cs.Fields[0].Sensitive {
		t.Error("ApplyCellSight should not mutate the original CellSight")
	}
}
