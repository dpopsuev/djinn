package tui

import (
	"strings"
	"testing"
)

func TestCoherenceGauge_Zone(t *testing.T) {
	g := NewCoherenceGauge("agent-1")

	tests := []struct {
		usage float64
		zone  string
	}{
		{0.0, "cold"},
		{0.10, "cold"},
		{0.20, "warm"},
		{0.39, "warm"},
		{0.40, "focused"},
		{0.64, "focused"},
		{0.65, "hot"},
		{0.84, "hot"},
		{0.85, "redline"},
		{1.0, "redline"},
	}
	for _, tt := range tests {
		g.SetUsage(tt.usage)
		if zone := g.Zone(); zone != tt.zone {
			t.Errorf("usage=%.2f zone=%q, want %q", tt.usage, zone, tt.zone)
		}
	}
}

func TestCoherenceGauge_ViewContainsPercentage(t *testing.T) {
	g := NewCoherenceGauge("agent-1")
	g.SetUsage(0.55)

	view := g.View(60)
	if !strings.Contains(view, "55%") {
		t.Fatalf("view should contain 55%%: %q", view)
	}
	if !strings.Contains(view, "focused") {
		t.Fatalf("view should contain zone name 'focused': %q", view)
	}
}

func TestCoherenceGauge_ViewContainsBar(t *testing.T) {
	g := NewCoherenceGauge("agent-1")
	g.SetUsage(0.50)

	view := g.View(60)
	// Should contain the bar prefix.
	if !strings.Contains(view, "ctx:") {
		t.Fatalf("view should contain ctx: prefix: %q", view)
	}
	// Should contain filled blocks (U+2588) or empty blocks (U+2591).
	if !strings.Contains(view, "\u2588") && !strings.Contains(view, "\u2591") {
		t.Fatalf("view should contain bar characters: %q", view)
	}
}

func TestCoherenceGauge_SetUsage_Clamped(t *testing.T) {
	g := NewCoherenceGauge("agent-1")

	g.SetUsage(-0.5)
	if g.usage != 0 {
		t.Fatalf("negative usage should clamp to 0, got %f", g.usage)
	}

	g.SetUsage(1.5)
	if g.usage != 1 {
		t.Fatalf("usage > 1 should clamp to 1, got %f", g.usage)
	}
}

func TestCoherenceGauge_PanelInterface(t *testing.T) {
	g := NewCoherenceGauge("agent-1")
	if g.ID() != "coherence" {
		t.Fatalf("ID = %q, want coherence", g.ID())
	}
	if g.Height() != 1 {
		t.Fatalf("Height = %d, want 1", g.Height())
	}
}

func TestCoherenceGauge_ZeroUsage(t *testing.T) {
	g := NewCoherenceGauge("agent-1")
	g.SetUsage(0)
	view := g.View(60)
	if !strings.Contains(view, "0%") {
		t.Fatalf("zero usage should show 0%%: %q", view)
	}
	if !strings.Contains(view, "cold") {
		t.Fatalf("zero usage should be cold zone: %q", view)
	}
}
