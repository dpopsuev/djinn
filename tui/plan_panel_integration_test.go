package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/render"
)

// integrationGraph builds a realistic plan with 4 segments in different states,
// parent-child relationships, components, and dependencies — exercising every
// view path in PlanPanel against a real artifact.Graph.
func integrationGraph() *artifact.Graph {
	g := artifact.NewGraph("integration-plan", artifact.DefaultRegistry())

	// seg-1: complete, root segment with components.
	g.Add(artifact.Artifact{ //nolint:errcheck,gocritic // test
		Kind:   artifact.KindPlanSegment,
		Title:  "Database Schema",
		Status: artifact.StatusComplete,
		Components: artifact.ComponentMap{
			Files:       []string{"db/schema.sql", "db/migrations/001.sql"},
			Directories: []string{"db/"},
		},
		Content: "Define the core database schema for user management",
	})

	// seg-2: in_progress, depends on seg-1, has children.
	g.Add(artifact.Artifact{ //nolint:errcheck,gocritic // test
		Kind:      artifact.KindPlanSegment,
		Title:     "API Endpoints",
		Status:    artifact.StatusInProgress,
		DependsOn: []string{"seg-1"},
		Components: artifact.ComponentMap{
			Files:   []string{"api/handler.go", "api/routes.go"},
			Symbols: []string{"HandleCreate", "HandleList"},
		},
	})

	// seg-3: draft, depends on seg-2, child of seg-2.
	g.Add(artifact.Artifact{ //nolint:errcheck,gocritic // test
		Kind:      artifact.KindPlanSegment,
		Title:     "Auth Middleware",
		Status:    artifact.StatusDraft,
		DependsOn: []string{"seg-2"},
		Parent:    "seg-2",
		Components: artifact.ComponentMap{
			Files: []string{"api/auth.go"},
		},
	})

	// seg-4: complete, depends on seg-1.
	g.Add(artifact.Artifact{ //nolint:errcheck,gocritic // test
		Kind:      artifact.KindPlanSegment,
		Title:     "Config Loader",
		Status:    artifact.StatusComplete,
		DependsOn: []string{"seg-1"},
	})

	return g
}

// TestPlanPanel_Integration_ThreeViews verifies that PlanPanel renders all three
// views correctly from a real artifact.Graph with segments in mixed states.
func TestPlanPanel_Integration_ThreeViews(t *testing.T) {
	g := integrationGraph()
	p := NewPlanPanel(g)
	p.SetFocus(true)

	// --- Overview ---
	overview := p.View(80) //nolint:mnd // standard width

	if overview == "" {
		t.Fatal("overview should not be empty")
	}

	// All 4 segment titles visible.
	for _, title := range []string{"Database Schema", "API Endpoints", "Auth Middleware", "Config Loader"} {
		if !strings.Contains(overview, title) {
			t.Errorf("overview missing segment title %q", title)
		}
	}

	// Progress: 2 complete out of 4 = 50%.
	if !strings.Contains(overview, "2/4") {
		t.Error("overview should show 2/4 progress")
	}
	if !strings.Contains(overview, "50%") {
		t.Error("overview should show 50%")
	}

	// Status glyphs: complete (⬢), in-progress (⬡), draft (○).
	if !strings.Contains(overview, "⬢") {
		t.Error("overview should contain complete glyph ⬢")
	}
	if !strings.Contains(overview, "⬡") {
		t.Error("overview should contain in-progress glyph ⬡")
	}
	if !strings.Contains(overview, "○") {
		t.Error("overview should contain draft glyph ○")
	}

	// Dependency chain rendered (seg-1 → seg-2 edge exists).
	if !strings.Contains(overview, "──→") {
		t.Error("overview should contain dependency arrow")
	}

	// --- Navigate to Goal Detail (seg-1) ---
	p.Update(tea.KeyMsg{Type: tea.KeyEnter}) // dive into first segment

	if p.view != PlanViewGoal {
		t.Fatalf("expected PlanViewGoal, got %d", p.view)
	}
	if p.goalID != "seg-1" {
		t.Fatalf("expected goalID=seg-1, got %q", p.goalID)
	}

	goalView := p.View(80) //nolint:mnd // standard width
	if goalView == "" {
		t.Fatal("goal view should not be empty")
	}
	if !strings.Contains(goalView, "Database Schema") {
		t.Error("goal view should show segment title")
	}
	if !strings.Contains(goalView, "complete") {
		t.Error("goal view should show status")
	}

	// --- Navigate to seg-2 (API Endpoints) which has a child ---
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}) // back to overview
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // cursor to seg-2
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})                     // dive into seg-2

	if p.goalID != "seg-2" {
		t.Fatalf("expected goalID=seg-2, got %q", p.goalID)
	}

	goalView2 := p.View(80) //nolint:mnd // standard width
	if !strings.Contains(goalView2, "API Endpoints") {
		t.Error("goal view should show API Endpoints title")
	}
	// seg-3 is a child of seg-2.
	if !strings.Contains(goalView2, "Auth Middleware") {
		t.Error("goal view for seg-2 should list child segment Auth Middleware")
	}

	// --- Dive to Segment Detail (seg-3 child) ---
	p.Update(tea.KeyMsg{Type: tea.KeyEnter}) // dive into child seg-3

	if p.view != PlanViewSegment {
		t.Fatalf("expected PlanViewSegment, got %d", p.view)
	}
	if p.segID != "seg-3" {
		t.Fatalf("expected segID=seg-3, got %q", p.segID)
	}

	segView := p.View(80) //nolint:mnd // standard width
	if segView == "" {
		t.Fatal("segment view should not be empty")
	}
	if !strings.Contains(segView, "Auth Middleware") {
		t.Error("segment view should show title")
	}
	if !strings.Contains(segView, "seg-3") {
		t.Error("segment view should show segment ID")
	}
	if !strings.Contains(segView, "draft") {
		t.Error("segment view should show draft status")
	}
	// seg-3 depends on seg-2.
	if !strings.Contains(segView, "seg-2") {
		t.Error("segment view should show dependency on seg-2")
	}

	// --- Climb back to overview ---
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}) // back to goal
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}) // back to overview

	if p.view != PlanViewOverview {
		t.Errorf("expected PlanViewOverview after climbing, got %d", p.view)
	}

	// --- FocusContext reflects current state ---
	fc := p.FocusContext()
	if fc.PanelID != "plan" {
		t.Errorf("FocusContext PanelID = %q, want plan", fc.PanelID)
	}
	if fc.Metadata["view"] != "overview" {
		t.Errorf("FocusContext view = %q, want overview", fc.Metadata["view"])
	}
}

// TestPlanPanel_Integration_EmptyGraph verifies that an empty artifact.Graph
// renders without panic and shows the expected placeholder text.
func TestPlanPanel_Integration_EmptyGraph(t *testing.T) {
	g := artifact.NewGraph("empty-plan", artifact.DefaultRegistry())
	p := NewPlanPanel(g)
	p.SetFocus(true)

	view := p.View(80) //nolint:mnd // standard width

	if view == "" {
		t.Fatal("empty graph view should not be empty string")
	}
	if !strings.Contains(view, "Empty plan") {
		t.Errorf("empty graph should render 'Empty plan', got %q", view)
	}

	// Navigation on empty graph should not panic.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Still at overview after all navigation attempts.
	if p.view != PlanViewOverview {
		t.Errorf("view should remain overview on empty graph, got %d", p.view)
	}

	// FocusContext should return sensible defaults (no panic).
	fc := p.FocusContext()
	if fc.PanelID != "plan" {
		t.Errorf("FocusContext PanelID = %q, want plan", fc.PanelID)
	}
	if fc.ElementID != "" {
		t.Errorf("FocusContext ElementID should be empty on empty graph, got %q", fc.ElementID)
	}
}

// TestPlanPanel_Integration_DataFlowAdapter verifies that the render adapter
// correctly converts the same artifact.Graph used by PlanPanel into a DataFlowGraph
// with the expected nodes and edges.
func TestPlanPanel_Integration_DataFlowAdapter(t *testing.T) {
	g := integrationGraph()

	dfg := render.ArtifactGraphToDataFlow(g, "integration")

	// 4 segments → 4 nodes.
	if len(dfg.Nodes) != 4 { //nolint:mnd // 4 segments
		t.Errorf("DataFlowGraph nodes = %d, want 4", len(dfg.Nodes))
	}

	// 3 dependency edges: seg-2→seg-1, seg-3→seg-2, seg-4→seg-1.
	if len(dfg.Edges) != 3 { //nolint:mnd // 3 deps
		t.Errorf("DataFlowGraph edges = %d, want 3", len(dfg.Edges))
	}

	// Verify node properties reflect artifact status.
	nodeByID := make(map[string]render.Node, len(dfg.Nodes))
	for _, n := range dfg.Nodes {
		nodeByID[n.ID] = n
	}

	// seg-1: complete → PassThrough=true, Changed=false.
	if n, ok := nodeByID["seg-1"]; ok {
		if !n.PassThrough {
			t.Error("seg-1 (complete) should be PassThrough")
		}
		if n.Changed {
			t.Error("seg-1 (complete) should not be Changed")
		}
	} else {
		t.Error("seg-1 node missing from DataFlowGraph")
	}

	// seg-2: in_progress → Changed=true, PassThrough=false.
	if n, ok := nodeByID["seg-2"]; ok {
		if !n.Changed {
			t.Error("seg-2 (in_progress) should be Changed")
		}
		if n.PassThrough {
			t.Error("seg-2 (in_progress) should not be PassThrough")
		}
	} else {
		t.Error("seg-2 node missing from DataFlowGraph")
	}

	// seg-3: draft → neither Changed nor PassThrough.
	if n, ok := nodeByID["seg-3"]; ok {
		if n.Changed {
			t.Error("seg-3 (draft) should not be Changed")
		}
		if n.PassThrough {
			t.Error("seg-3 (draft) should not be PassThrough")
		}
	} else {
		t.Error("seg-3 node missing from DataFlowGraph")
	}
}

// TestPlanPanel_Integration_SegmentWithComponents verifies that segment detail
// renders ComponentMap entries (files, directories, symbols) from the graph.
func TestPlanPanel_Integration_SegmentWithComponents(t *testing.T) {
	g := integrationGraph()
	p := NewPlanPanel(g)
	p.SetFocus(true)

	// Navigate to seg-1 (Database Schema) which has files and directories.
	p.Update(tea.KeyMsg{Type: tea.KeyEnter}) // dive to goal detail

	// From goal detail, seg-1 has no children, so we need to check
	// if we can get to segment detail. Since seg-1 has no children,
	// let's directly set the segment view to test component rendering.
	p.view = PlanViewSegment
	p.segID = "seg-1"

	segView := p.View(80) //nolint:mnd // standard width

	if !strings.Contains(segView, "Database Schema") {
		t.Error("segment view should show title")
	}
	if !strings.Contains(segView, "schema.sql") {
		t.Error("segment view should show component file schema.sql")
	}
	if !strings.Contains(segView, "db/") {
		t.Error("segment view should show component directory db/")
	}
	if !strings.Contains(segView, "Define the core database schema") {
		t.Error("segment view should show content")
	}
}
