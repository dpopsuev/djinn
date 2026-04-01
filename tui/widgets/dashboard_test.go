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
