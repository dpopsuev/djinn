package widgets

import (
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/tui"
)

func TestScratchPaperPanel_Empty(t *testing.T) {
	p := NewScratchPaperPanel()
	view := p.View(80) //nolint:mnd // standard width

	if !strings.Contains(view, "No scratch paper") {
		t.Error("empty panel should show 'No scratch paper'")
	}
}

func TestScratchPaperPanel_View(t *testing.T) {
	p := NewScratchPaperPanel()
	p.SetUnderstanding("The system needs a caching layer")
	p.SetPlan([]string{"Add Redis dependency", "Implement cache interface", "Wire to service"})
	p.SetRisks([]string{"Cache invalidation complexity", "Redis availability"})
	p.SetNotes("Consider TTL-based expiry")

	view := p.View(80) //nolint:mnd // standard width

	if !strings.Contains(view, "Understanding") {
		t.Error("view should contain Understanding header")
	}
	if !strings.Contains(view, "caching layer") {
		t.Error("view should contain understanding content")
	}
	if !strings.Contains(view, "Plan") {
		t.Error("view should contain Plan header")
	}
	if !strings.Contains(view, "1. Add Redis") {
		t.Error("view should contain numbered plan steps")
	}
	if !strings.Contains(view, "3. Wire to service") {
		t.Error("view should contain step 3")
	}
	if !strings.Contains(view, "Risks") {
		t.Error("view should contain Risks header")
	}
	if !strings.Contains(view, "Cache invalidation") {
		t.Error("view should contain risk content")
	}
	if !strings.Contains(view, "Notes") {
		t.Error("view should contain Notes header")
	}
	if !strings.Contains(view, "TTL-based") {
		t.Error("view should contain notes content")
	}
}

func TestScratchPaperPanel_Sighted(t *testing.T) {
	p := NewScratchPaperPanel()

	// Compile-time check already verifies interface, test behavior.
	var s tui.Sighted = p

	// SightGate should be true.
	if !s.SightGate() {
		t.Fatal("SightGate should be true")
	}

	// Empty panel — sight has panel ID but no content.
	sight := s.CellSight()
	if sight.PanelID != "scratch-paper" {
		t.Fatalf("PanelID = %q, want scratch-paper", sight.PanelID)
	}
	if sight.Kind != "" {
		t.Fatalf("Kind should be empty on empty panel, got %q", sight.Kind)
	}

	// Set content and verify fields.
	p.SetUnderstanding("test understanding")
	p.SetPlan([]string{"step 1", "step 2"})
	p.SetRisks([]string{"risk 1"})
	p.SetNotes("some notes")

	sight = s.CellSight()
	if sight.Kind != "scratch-paper" {
		t.Fatalf("Kind = %q, want scratch-paper", sight.Kind)
	}
	if sight.CellTitle != "Scratch Paper" {
		t.Fatalf("CellTitle = %q, want 'Scratch Paper'", sight.CellTitle)
	}

	// Check fields.
	understandingField := fieldByKey(sight.Fields, "understanding")
	if understandingField != "test understanding" {
		t.Errorf("understanding field = %q, want 'test understanding'", understandingField)
	}

	planField := fieldByKey(sight.Fields, "plan_steps")
	if planField != "2" {
		t.Errorf("plan_steps field = %q, want '2'", planField)
	}

	risksField := fieldByKey(sight.Fields, "risks")
	if risksField != "1" {
		t.Errorf("risks field = %q, want '1'", risksField)
	}

	notesField := fieldByKey(sight.Fields, "notes")
	if notesField != "some notes" {
		t.Errorf("notes field = %q, want 'some notes'", notesField)
	}
}

func TestScratchPaperPanel_PartialContent(t *testing.T) {
	p := NewScratchPaperPanel()
	p.SetUnderstanding("just understanding")

	view := p.View(80) //nolint:mnd // standard width

	if !strings.Contains(view, "Understanding") {
		t.Error("view should contain Understanding header")
	}
	if strings.Contains(view, "Plan") {
		t.Error("view should not contain Plan when empty")
	}
	if strings.Contains(view, "Risks") {
		t.Error("view should not contain Risks when empty")
	}
	if strings.Contains(view, "Notes") {
		t.Error("view should not contain Notes when empty")
	}
}
