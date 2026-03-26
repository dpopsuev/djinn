package tui

import (
	"strings"
	"testing"
)

func TestOdometerLine_ViewDefault(t *testing.T) {
	o := NewOdometerLine()
	view := o.View(80)
	if !strings.Contains(view, "round-trips:0") {
		t.Fatalf("default should show 0 round-trips: %q", view)
	}
	if !strings.Contains(view, "$0.00 spent") {
		t.Fatalf("default should show $0.00: %q", view)
	}
	if !strings.Contains(view, "0 tasks") {
		t.Fatalf("default should show 0 tasks: %q", view)
	}
	if !strings.Contains(view, "0 relays") {
		t.Fatalf("default should show 0 relays: %q", view)
	}
}

func TestOdometerLine_UpdateFromMsg(t *testing.T) {
	o := NewOdometerLine()
	o.Update(OdometerUpdateMsg{
		RoundTrips: 247,
		Cost:       6.77,
		Tasks:      14,
		Relays:     3,
	})

	view := o.View(80)
	if !strings.Contains(view, "round-trips:247") {
		t.Fatalf("should show 247: %q", view)
	}
	if !strings.Contains(view, "$6.77 spent") {
		t.Fatalf("should show $6.77: %q", view)
	}
	if !strings.Contains(view, "14 tasks") {
		t.Fatalf("should show 14 tasks: %q", view)
	}
	if !strings.Contains(view, "3 relays") {
		t.Fatalf("should show 3 relays: %q", view)
	}
}

func TestOdometerLine_PanelInterface(t *testing.T) {
	o := NewOdometerLine()
	if o.ID() != "odometer" {
		t.Fatalf("ID = %q, want odometer", o.ID())
	}
	if o.Height() != 1 {
		t.Fatalf("Height = %d, want 1", o.Height())
	}
}

func TestOdometerLine_UpdateIgnoresOtherMsgs(t *testing.T) {
	o := NewOdometerLine()
	o.Update(OdometerUpdateMsg{RoundTrips: 10, Cost: 1.50, Tasks: 5, Relays: 1})
	// Send an unrelated message — values should not change.
	o.Update(DashboardUIStateMsg{State: "STREAMING"})

	view := o.View(80)
	if !strings.Contains(view, "round-trips:10") {
		t.Fatalf("should preserve state after unrelated msg: %q", view)
	}
}

func TestOdometerLine_UpdateReplacesValues(t *testing.T) {
	o := NewOdometerLine()
	o.Update(OdometerUpdateMsg{RoundTrips: 10, Cost: 1.0, Tasks: 5, Relays: 1})
	o.Update(OdometerUpdateMsg{RoundTrips: 20, Cost: 2.0, Tasks: 10, Relays: 2})

	view := o.View(80)
	if !strings.Contains(view, "round-trips:20") {
		t.Fatalf("should show updated value: %q", view)
	}
	if !strings.Contains(view, "$2.00 spent") {
		t.Fatalf("should show updated cost: %q", view)
	}
}
