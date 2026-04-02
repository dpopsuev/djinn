package widgets

import (
	"testing"

	"github.com/dpopsuev/djinn/tui"
)

func TestSymbolGraphPanel_Sighted(t *testing.T) {
	p := NewSymbolGraphPanel()

	// Compile-time check already verifies interface, but let's test behavior.
	var s tui.Sighted = p

	// SightGate should be true.
	if !s.SightGate() {
		t.Fatal("SightGate should be true")
	}

	// Empty panel — sight has panel ID but no file.
	sight := s.CellSight()
	if sight.PanelID != "symbol-graph" {
		t.Fatalf("PanelID = %q, want symbol-graph", sight.PanelID)
	}
	if sight.CellID != "" {
		t.Fatalf("CellID should be empty, got %q", sight.CellID)
	}

	// After setting content.
	p.SetContent("auth/handler.go", "<symbol-graph>...</symbol-graph>")
	sight = p.CellSight()
	if sight.CellID != "auth/handler.go" {
		t.Fatalf("CellID = %q, want auth/handler.go", sight.CellID)
	}
	if sight.Kind != "symbol-graph" {
		t.Fatalf("Kind = %q, want symbol-graph", sight.Kind)
	}
	if len(sight.Fields) != 1 || sight.Fields[0].Key != "file" {
		t.Fatalf("Fields = %+v, want [{file, auth/handler.go}]", sight.Fields)
	}
}

func TestSymbolGraphPanel_View_Empty(t *testing.T) {
	p := NewSymbolGraphPanel()
	view := p.View(80)
	if view == "" {
		t.Fatal("empty panel should render placeholder text")
	}
}

func TestSymbolGraphPanel_View_Content(t *testing.T) {
	p := NewSymbolGraphPanel()
	content := "  ValidateToken (func, 4 callers) | middleware.go:42, cli/repl/model.go:8, ..."
	p.SetContent("auth/handler.go", content)
	view := p.View(200)
	if view != content {
		t.Fatalf("view = %q, want %q", view, content)
	}
}

func TestSymbolGraphPanel_ID(t *testing.T) {
	p := NewSymbolGraphPanel()
	if p.ID() != "symbol-graph" {
		t.Fatalf("ID = %q, want symbol-graph", p.ID())
	}
}
