package widgets

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dpopsuev/djinn/artifact"
)

func testPlanGraph() *artifact.Graph {
	g := artifact.NewGraph("test-plan", artifact.DefaultRegistry())

	g.Add(artifact.Artifact{Kind: artifact.KindPlanSegment, Title: "Hub Mediators", Status: artifact.StatusComplete})                                  //nolint:errcheck,gocritic // test
	g.Add(artifact.Artifact{Kind: artifact.KindPlanSegment, Title: "Planner Port", Status: artifact.StatusComplete, DependsOn: []string{"seg-1"}})     //nolint:errcheck,gocritic // test
	g.Add(artifact.Artifact{Kind: artifact.KindPlanSegment, Title: "Shell Maturity", Status: artifact.StatusInProgress, DependsOn: []string{"seg-2"}}) //nolint:errcheck,gocritic // test

	return g
}

// --- View states ---

func TestPlanPanel_OverviewRenders(t *testing.T) {
	p := NewPlanPanel(testPlanGraph())
	p.SetFocus(true)
	view := p.View(80) //nolint:mnd // standard width

	if !strings.Contains(view, "Hub Mediators") {
		t.Error("overview should show segment titles")
	}
	if !strings.Contains(view, "⬢") {
		t.Error("overview should show complete glyph")
	}
	if !strings.Contains(view, "⬡") {
		t.Error("overview should show in-progress glyph")
	}
	if !strings.Contains(view, "▸") {
		t.Error("overview should show cursor")
	}
}

func TestPlanPanel_OverviewProgress(t *testing.T) {
	p := NewPlanPanel(testPlanGraph())
	view := p.View(80) //nolint:mnd // standard width

	// 2/3 complete = 66%
	if !strings.Contains(view, "2/3") {
		t.Error("overview should show 2/3 progress")
	}
	if !strings.Contains(view, "66%") {
		t.Error("overview should show 66%")
	}
}

// --- Navigation ---

func TestPlanPanel_DiveToGoal(t *testing.T) {
	p := NewPlanPanel(testPlanGraph())
	p.SetFocus(true)

	// Enter → dive to goal detail.
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if p.view != PlanViewGoal {
		t.Errorf("view = %d, want PlanViewGoal", p.view)
	}
	if p.goalID != "seg-1" {
		t.Errorf("goalID = %q, want seg-1", p.goalID)
	}
}

func TestPlanPanel_ClimbToOverview(t *testing.T) {
	p := NewPlanPanel(testPlanGraph())
	p.SetFocus(true)

	// Dive then climb.
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if p.view != PlanViewOverview {
		t.Errorf("view = %d, want PlanViewOverview", p.view)
	}
}

func TestPlanPanel_CursorNavigation(t *testing.T) {
	p := NewPlanPanel(testPlanGraph())
	p.SetFocus(true)

	if p.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", p.cursor)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.cursor != 1 {
		t.Errorf("after j, cursor = %d, want 1", p.cursor)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor != 0 {
		t.Errorf("after k, cursor = %d, want 0", p.cursor)
	}
}

func TestPlanPanel_CursorClamp(t *testing.T) {
	p := NewPlanPanel(testPlanGraph())
	p.SetFocus(true)

	// Move past end.
	for range 10 {
		p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	if p.cursor != 2 { //nolint:mnd // 3 segments, max cursor = 2
		t.Errorf("cursor should clamp at 2, got %d", p.cursor)
	}
}

// --- FocusContext ---

func TestPlanPanel_FocusContext_Overview(t *testing.T) {
	p := NewPlanPanel(testPlanGraph())
	p.SetFocus(true)

	fc := p.FocusContext()
	if fc.PanelID != "plan" {
		t.Errorf("PanelID = %q, want plan", fc.PanelID)
	}
	if fc.ElementID != "seg-1" {
		t.Errorf("ElementID = %q, want seg-1", fc.ElementID)
	}
	if fc.ElementTitle != "Hub Mediators" {
		t.Errorf("ElementTitle = %q, want Hub Mediators", fc.ElementTitle)
	}
}

func TestPlanPanel_FocusContext_GoalView(t *testing.T) {
	p := NewPlanPanel(testPlanGraph())
	p.SetFocus(true)

	// Dive to goal.
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	fc := p.FocusContext()
	if fc.ElementID != "seg-1" {
		t.Errorf("ElementID = %q, want seg-1", fc.ElementID)
	}
	if fc.Metadata["view"] != "goal" {
		t.Errorf("view = %q, want goal", fc.Metadata["view"])
	}
}

// --- Empty graph ---

func TestPlanPanel_EmptyGraph(t *testing.T) {
	g := artifact.NewGraph("empty", artifact.DefaultRegistry())
	p := NewPlanPanel(g)
	view := p.View(80) //nolint:mnd // standard width

	if !strings.Contains(view, "Empty plan") {
		t.Error("empty graph should show 'Empty plan'")
	}
}

// --- Nil graph ---

func TestPlanPanel_NilGraph(t *testing.T) {
	p := NewPlanPanel(nil)
	view := p.View(80) //nolint:mnd // standard width

	if !strings.Contains(view, "No plan loaded") {
		t.Error("nil graph should show 'No plan loaded'")
	}
}
