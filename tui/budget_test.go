package tui

import (
	"strings"
	"testing"
)

func TestBudgetGauge_ViewContainsCosts(t *testing.T) {
	g := NewBudgetGauge("agent-1", 10.0)
	g.SetSpent(2.40)

	view := g.View(60)
	if !strings.Contains(view, "$2.40") {
		t.Fatalf("view should contain spent: %q", view)
	}
	if !strings.Contains(view, "$10.00") {
		t.Fatalf("view should contain ceiling: %q", view)
	}
	if !strings.Contains(view, "24%") {
		t.Fatalf("view should contain percentage: %q", view)
	}
}

func TestBudgetGauge_ViewZeroSpent(t *testing.T) {
	g := NewBudgetGauge("agent-1", 5.0)
	g.SetSpent(0)

	view := g.View(60)
	if !strings.Contains(view, "$0.00") {
		t.Fatalf("view should show $0.00: %q", view)
	}
	if !strings.Contains(view, "0%") {
		t.Fatalf("view should show 0%%: %q", view)
	}
}

func TestBudgetGauge_ViewFullSpent(t *testing.T) {
	g := NewBudgetGauge("agent-1", 10.0)
	g.SetSpent(10.0)

	view := g.View(60)
	if !strings.Contains(view, "100%") {
		t.Fatalf("view should show 100%%: %q", view)
	}
}

func TestBudgetGauge_SetSpent_Negative(t *testing.T) {
	g := NewBudgetGauge("agent-1", 10.0)
	g.SetSpent(-5.0)
	if g.spent != 0 {
		t.Fatalf("negative spent should clamp to 0, got %f", g.spent)
	}
}

func TestBudgetGauge_PanelInterface(t *testing.T) {
	g := NewBudgetGauge("agent-1", 10.0)
	if g.ID() != "budget" {
		t.Fatalf("ID = %q, want budget", g.ID())
	}
	if g.Height() != 1 {
		t.Fatalf("Height = %d, want 1", g.Height())
	}
}

func TestBudgetGauge_ViewContainsBar(t *testing.T) {
	g := NewBudgetGauge("agent-1", 10.0)
	g.SetSpent(5.0)
	view := g.View(60)
	if !strings.Contains(view, "\u2588") || !strings.Contains(view, "\u2591") {
		t.Fatalf("view should contain bar characters: %q", view)
	}
}

func TestBudgetGauge_ZeroCeiling(t *testing.T) {
	g := NewBudgetGauge("agent-1", 0)
	g.SetSpent(5.0)
	view := g.View(60)
	// Should not panic and should show 0%.
	if !strings.Contains(view, "0%") {
		t.Fatalf("zero ceiling should show 0%%: %q", view)
	}
}
