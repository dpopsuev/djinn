package widgets

import (
	"strings"
	"testing"

	tui "github.com/dpopsuev/djinn/tui"
)

// --- Isolated DashboardPanel tests: zero imports from cli/repl, agent, driver, session ---

func TestDashboard_IdentityViaMessage(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(tui.DashboardIdentityMsg{Workspace: "aeon", Driver: "claude", Model: "opus", Mode: "agent"})
	view := d.View(120)
	if !strings.Contains(view, "aeon") {
		t.Fatalf("view missing workspace: %q", view)
	}
	if !strings.Contains(view, "opus") {
		t.Fatalf("view missing model: %q", view)
	}
}

func TestDashboard_MetricsViaMessage(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(tui.DashboardIdentityMsg{Workspace: "ws", Driver: "drv", Model: "mdl", Mode: "mode"})
	d.Update(tui.DashboardMetricsMsg{TokensIn: 100, TokensOut: 50, Turns: 3})
	view := d.View(120)
	if !strings.Contains(view, "100") || !strings.Contains(view, "50") {
		t.Fatalf("view missing metrics: %q", view)
	}
}

func TestDashboard_UIStateViaMessage(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(tui.DashboardUIStateMsg{State: "STREAMING"})
	view := d.View(120)
	if !strings.Contains(view, "STREAMING") {
		t.Fatalf("view missing STREAMING: %q", view)
	}

	d.Update(tui.DashboardUIStateMsg{State: "APPROVAL"})
	view = d.View(120)
	if !strings.Contains(view, "APPROVAL") {
		t.Fatalf("view missing APPROVAL: %q", view)
	}
}

func TestDashboard_HealthViaMessage(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(tui.DashboardHealthMsg{Reports: []tui.HealthReport{
		{Component: "scribe", Status: tui.StatusGreen, Message: "5 tools"},
		{Component: "locus", Status: tui.StatusGreen, Message: "3 tools"},
	}})
	view := d.View(120)
	if !strings.Contains(view, "scribe") || !strings.Contains(view, "locus") {
		t.Fatalf("view should show individual component names: %q", view)
	}
}

func TestDashboard_DefaultInsert(t *testing.T) {
	d := NewDashboardPanel()
	view := d.View(120)
	if !strings.Contains(view, "INSERT") {
		t.Fatalf("default state should be INSERT: %q", view)
	}
}

func TestDashboardPanel_CellSight(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(tui.DashboardMetricsMsg{
		TokensIn:   500,
		TokensOut:  200,
		Turns:      3,
		AgentCount: 2,
		Operation:  "plan",
	})

	cs := d.CellSight()
	if cs.PanelID != "dashboard" {
		t.Fatalf("PanelID = %q, want dashboard", cs.PanelID)
	}
	if cs.Kind != "status" {
		t.Fatalf("Kind = %q, want status", cs.Kind)
	}
	if len(cs.Fields) != 4 {
		t.Fatalf("Fields count = %d, want 4", len(cs.Fields))
	}

	// Verify field keys and sensitivity.
	fieldMap := make(map[string]tui.SightField)
	for _, f := range cs.Fields {
		fieldMap[f.Key] = f
	}

	if f, ok := fieldMap["operation"]; !ok || f.Value != "plan" || f.Sensitive {
		t.Errorf("operation field: got %+v", fieldMap["operation"])
	}
	if f, ok := fieldMap["turns"]; !ok || f.Value != "3" || f.Sensitive {
		t.Errorf("turns field: got %+v", fieldMap["turns"])
	}
	if f, ok := fieldMap["agents"]; !ok || f.Value != "2" || f.Sensitive {
		t.Errorf("agents field: got %+v", fieldMap["agents"])
	}
	if f, ok := fieldMap["tokens"]; !ok || !f.Sensitive {
		t.Errorf("tokens field should be sensitive: got %+v", fieldMap["tokens"])
	}
}

func TestDashboardPanel_SightGate(t *testing.T) {
	d := NewDashboardPanel()
	if !d.SightGate() {
		t.Fatal("SightGate should be true by default")
	}
}

func TestDashboardPanel_AndonUpdate_Green(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(tui.AndonUpdateMsg{State: tui.AndonState{
		Level:   tui.AndonGreen,
		Source:  "budget",
		Message: "all clear",
	}})
	view := d.View(120)
	// Green uses ● glyph (U+25CF).
	if !strings.Contains(view, "\u25cf") {
		t.Fatalf("green andon should show solid circle glyph: %q", view)
	}
}

func TestDashboardPanel_AndonUpdate_Yellow(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(tui.AndonUpdateMsg{State: tui.AndonState{
		Level:   tui.AndonYellow,
		Source:  "gate",
		Message: "quality degraded",
	}})
	view := d.View(120)
	// Yellow uses ◉ glyph (U+25C9).
	if !strings.Contains(view, "\u25c9") {
		t.Fatalf("yellow andon should show fisheye glyph: %q", view)
	}
}

func TestDashboardPanel_AndonUpdate_Red(t *testing.T) {
	d := NewDashboardPanel()
	_, cmd := d.Update(tui.AndonUpdateMsg{State: tui.AndonState{
		Level:   tui.AndonRed,
		Source:  "budget",
		Message: "budget exceeded",
	}})
	view := d.View(120)
	// Red uses ⬤ glyph (U+2B24).
	if !strings.Contains(view, "\u2b24") {
		t.Fatalf("red andon should show large circle glyph: %q", view)
	}
	// Red should emit a CordonMsg command.
	if cmd == nil {
		t.Fatal("red andon should emit a CordonMsg command")
	}
	msg := cmd()
	cordon, ok := msg.(tui.CordonMsg)
	if !ok {
		t.Fatalf("cmd should produce CordonMsg, got %T", msg)
	}
	if cordon.Reason != "budget exceeded" {
		t.Fatalf("CordonMsg.Reason = %q, want 'budget exceeded'", cordon.Reason)
	}
	if cordon.AgentID != "budget" {
		t.Fatalf("CordonMsg.AgentID = %q, want 'budget'", cordon.AgentID)
	}
}
