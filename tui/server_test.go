package tui

import (
	"strings"
	"testing"
)

func TestServerPanel_GreenView(t *testing.T) {
	s := NewServerPanel("s1", "scribe", StatusGreen, "http://localhost:8080/", nil)
	view := s.View(80)
	if !strings.Contains(view, "scribe") {
		t.Fatalf("view missing name: %q", view)
	}
	// Server panel uses Glyph(StateDone) which renders ⬢.
	if !strings.Contains(view, "⬢") {
		t.Fatalf("green should show done glyph (⬢): %q", view)
	}
}

func TestServerPanel_OfflineView(t *testing.T) {
	s := NewServerPanel("s2", "locus", StatusOffline, "", nil)
	view := s.View(80)
	if !strings.Contains(view, "locus") {
		t.Fatalf("view missing name: %q", view)
	}
}

func TestServerPanel_Children(t *testing.T) {
	tools := []Panel{
		NewEnvelopePanel("t1", "artifact", ""),
		NewEnvelopePanel("t2", "graph", ""),
	}
	s := NewServerPanel("s1", "scribe", StatusGreen, "", tools)
	if len(s.Children()) != 2 {
		t.Fatalf("children = %d, want 2", len(s.Children()))
	}
}
