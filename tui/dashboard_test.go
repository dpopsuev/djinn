package tui

import (
	"strings"
	"testing"
)

// --- Isolated DashboardPanel tests: zero imports from cli/repl, agent, driver, session ---

func TestDashboard_IdentityViaMessage(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(DashboardIdentityMsg{"aeon", "claude", "opus", "agent"})
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
	d.Update(DashboardIdentityMsg{"ws", "drv", "mdl", "mode"})
	d.Update(DashboardMetricsMsg{100, 50, 3})
	view := d.View(120)
	if !strings.Contains(view, "100") || !strings.Contains(view, "50") {
		t.Fatalf("view missing metrics: %q", view)
	}
}

func TestDashboard_UIStateViaMessage(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(DashboardUIStateMsg{"STREAMING"})
	view := d.View(120)
	if !strings.Contains(view, "STREAMING") {
		t.Fatalf("view missing STREAMING: %q", view)
	}

	d.Update(DashboardUIStateMsg{"APPROVAL"})
	view = d.View(120)
	if !strings.Contains(view, "APPROVAL") {
		t.Fatalf("view missing APPROVAL: %q", view)
	}
}

func TestDashboard_HealthViaMessage(t *testing.T) {
	d := NewDashboardPanel()
	d.Update(DashboardHealthMsg{[]HealthReport{
		{Component: "scribe", Status: StatusGreen, Message: "5 tools"},
		{Component: "locus", Status: StatusGreen, Message: "3 tools"},
	}})
	view := d.View(120)
	// Status line shows aggregated count, e.g. "mcp:✓2"
	if !strings.Contains(view, "2") {
		t.Fatalf("view should show health count: %q", view)
	}
}

func TestDashboard_DefaultInsert(t *testing.T) {
	d := NewDashboardPanel()
	view := d.View(120)
	if !strings.Contains(view, "INSERT") {
		t.Fatalf("default state should be INSERT: %q", view)
	}
}
