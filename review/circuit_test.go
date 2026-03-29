package review

import (
	"testing"

	"github.com/dpopsuev/djinn/render"
)

func TestDetectEntryPoints(t *testing.T) {
	deps := []Dependency{
		{From: "handler.CreateOrder", To: "validator.Validate", Package: "handler"},
		{From: "validator.Validate", To: "domain.New", Package: "validator"},
		{From: "cmd.Run", To: "app.Start", Package: "cmd"},
		{From: "worker.Process", To: "queue.Dequeue", Package: "worker"},
		{From: "domain.New", To: "repo.Save", Package: "domain"},
	}

	entries := DetectEntryPoints(deps)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entry points, got %d", len(entries))
	}

	kinds := make(map[string]bool)
	for _, e := range entries {
		kinds[e.Kind] = true
	}
	if !kinds["handler"] {
		t.Error("missing handler entry point")
	}
	if !kinds["cmd"] {
		t.Error("missing cmd entry point")
	}
	if !kinds["worker"] {
		t.Error("missing worker entry point")
	}
}

func TestWalkCircuit_Linear(t *testing.T) {
	deps := []Dependency{
		{From: "handler.Create", To: "validator.Check"},
		{From: "validator.Check", To: "domain.New"},
		{From: "domain.New", To: "repo.Save"},
	}
	changed := map[string]bool{
		"handler.Create": true,
		"repo.Save":      true,
	}

	entry := EntryPoint{Name: "handler.Create", Package: "handler", Kind: "handler"}
	stops := WalkCircuit(entry, deps, changed)

	if len(stops) != 4 {
		t.Fatalf("expected 4 stops, got %d", len(stops))
	}
	if !stops[0].Changed {
		t.Error("first stop (handler.Create) should be changed")
	}
	if !stops[1].PassThrough {
		t.Error("second stop (validator.Check) should be pass-through")
	}
	if !stops[3].Changed {
		t.Error("last stop (repo.Save) should be changed")
	}
}

func TestWalkCircuit_CycleAvoidance(t *testing.T) {
	deps := []Dependency{
		{From: "a.Foo", To: "b.Bar"},
		{From: "b.Bar", To: "a.Foo"}, // cycle
	}
	changed := map[string]bool{"a.Foo": true}

	entry := EntryPoint{Name: "a.Foo"}
	stops := WalkCircuit(entry, deps, changed)

	if len(stops) != 2 {
		t.Fatalf("expected 2 stops (cycle broken), got %d", len(stops))
	}
}

func TestExtractCircuits_OnlyAffected(t *testing.T) {
	deps := []Dependency{
		{From: "handler.Create", To: "domain.New", Package: "handler"},
		{From: "handler.Get", To: "repo.Find", Package: "handler"},
		{From: "domain.New", To: "repo.Save", Package: "domain"},
	}

	// Only handler.Create path has changes.
	circuits := ExtractCircuits([]string{"domain.New"}, deps)

	if len(circuits) != 1 {
		t.Fatalf("expected 1 circuit (only Create path has changes), got %d", len(circuits))
	}
	if circuits[0].EntryPoint != "handler.Create" {
		t.Errorf("circuit entry = %q, want handler.Create", circuits[0].EntryPoint)
	}
}

func TestExtractCircuits_SharedStops(t *testing.T) {
	deps := []Dependency{
		{From: "handler.Create", To: "domain.Validate", Package: "handler"},
		{From: "handler.Update", To: "domain.Validate", Package: "handler"},
		{From: "domain.Validate", To: "repo.Save", Package: "domain"},
	}

	circuits := ExtractCircuits([]string{"domain.Validate"}, deps)
	if len(circuits) != 2 {
		for i, c := range circuits {
			t.Logf("circuit %d: entry=%q stops=%d", i, c.EntryPoint, len(c.Stops))
		}
		t.Fatalf("expected 2 circuits, got %d", len(circuits))
	}

	// domain.Validate should be shared across both circuits.
	totalShared := 0
	for _, c := range circuits {
		totalShared += len(c.Shared)
	}
	if totalShared == 0 {
		t.Error("domain.Validate should be marked as shared")
	}
}

func TestFormatTour_Summary(t *testing.T) {
	circuits := []Circuit{
		{
			Title:      "handler: Create",
			EntryPoint: "handler.Create",
			Stops: []render.CircuitStop{
				{ID: "1", Name: "Create", Package: "handler", Changed: true},
				{ID: "2", Name: "Validate", Package: "domain", PassThrough: true},
			},
		},
	}

	view := FormatTour(circuits, ModeSummary)
	if len(view.Circuits) != 1 {
		t.Fatalf("expected 1 circuit view, got %d", len(view.Circuits))
	}
	if len(view.Circuits[0].Stops) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(view.Circuits[0].Stops))
	}
	// Summary mode: no detail.
	if view.Circuits[0].Stops[0].Detail != "" {
		t.Errorf("summary mode should have empty detail, got %q", view.Circuits[0].Stops[0].Detail)
	}
}

func TestFormatTour_Signatures(t *testing.T) {
	circuits := []Circuit{
		{
			Title: "handler: Create",
			Stops: []render.CircuitStop{
				{ID: "1", Name: "Create", Package: "handler",
					SignatureBefore: "(int) error",
					SignatureAfter:  "(string) error",
					Changed:         true},
			},
		},
	}

	view := FormatTour(circuits, ModeSignatures)
	detail := view.Circuits[0].Stops[0].Detail
	if detail == "" {
		t.Error("signatures mode should show before/after")
	}
}

func TestParseTourMode(t *testing.T) {
	tests := []struct {
		input string
		want  TourMode
	}{
		{"", ModeSummary},
		{"sigs", ModeSignatures},
		{"signatures", ModeSignatures},
		{"io", ModeIO},
		{"impact", ModeImpact},
		{"diagram", ModeDiagram},
		{"d", ModeDiagram},
		{"unknown", ModeSummary},
	}
	for _, tt := range tests {
		got := ParseTourMode(tt.input)
		if got != tt.want {
			t.Errorf("ParseTourMode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDepsFromLocusEdges(t *testing.T) {
	edges := []LocusEdge{
		{From: "handler", To: "domain", Weight: 10},
		{From: "domain", To: "repo", Weight: 5},
	}
	deps := DepsFromLocusEdges(edges)
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}
	if deps[0].From != "handler" || deps[0].To != "domain" {
		t.Errorf("dep[0] = %v, want handler→domain", deps[0])
	}
}
