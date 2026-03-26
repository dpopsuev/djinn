package tui

import (
	"strings"
	"testing"
)

func TestDriftPanel_NewDefaults(t *testing.T) {
	dp := NewDriftPanel()
	if dp.funcScore != 0 || dp.archScore != 0 || dp.perfScore != 0 {
		t.Errorf("new panel scores should be 0, got func=%.0f arch=%.0f perf=%.0f",
			dp.funcScore, dp.archScore, dp.perfScore)
	}
	if dp.tasksLeft != 0 {
		t.Errorf("new panel tasksLeft = %d, want 0", dp.tasksLeft)
	}
	if dp.ID() != panelIDDrift {
		t.Errorf("ID = %q, want %q", dp.ID(), panelIDDrift)
	}
}

func TestDriftPanel_SetDrift(t *testing.T) {
	dp := NewDriftPanel()
	dp.SetDrift(80, 90, 60, "4/5 specs", "0 cycles", "3 failing", 3)

	if dp.funcScore != 80 {
		t.Errorf("funcScore = %.0f, want 80", dp.funcScore)
	}
	if dp.archScore != 90 {
		t.Errorf("archScore = %.0f, want 90", dp.archScore)
	}
	if dp.perfScore != 60 {
		t.Errorf("perfScore = %.0f, want 60", dp.perfScore)
	}
	if dp.funcLabel != "4/5 specs" {
		t.Errorf("funcLabel = %q, want %q", dp.funcLabel, "4/5 specs")
	}
	if dp.tasksLeft != 3 {
		t.Errorf("tasksLeft = %d, want 3", dp.tasksLeft)
	}
}

func TestDriftPanel_SetDriftClamp(t *testing.T) {
	dp := NewDriftPanel()
	dp.SetDrift(150, -10, 50, "", "", "", 0)

	if dp.funcScore != 100 {
		t.Errorf("funcScore = %.0f, want 100 (clamped)", dp.funcScore)
	}
	if dp.archScore != 0 {
		t.Errorf("archScore = %.0f, want 0 (clamped)", dp.archScore)
	}
}

func TestDriftPanel_ViewContainsPillars(t *testing.T) {
	dp := NewDriftPanel()
	dp.SetDrift(80, 90, 60, "4/5 specs", "0 cycles", "3 failing", 3)

	view := dp.View(60)

	if !strings.Contains(view, "Func") {
		t.Errorf("View missing 'Func'")
	}
	if !strings.Contains(view, "Arch") {
		t.Errorf("View missing 'Arch'")
	}
	if !strings.Contains(view, "Perf") {
		t.Errorf("View missing 'Perf'")
	}
	if !strings.Contains(view, "3 tasks to convergence") {
		t.Errorf("View missing convergence line, got:\n%s", view)
	}
}

func TestDriftPanel_ViewZeroScores(t *testing.T) {
	dp := NewDriftPanel()
	dp.SetDrift(0, 0, 0, "0/5 specs", "3 cycles", "5 failing", 5)

	view := dp.View(60)

	if !strings.Contains(view, "0%") {
		t.Errorf("View should show 0%% for zero scores, got:\n%s", view)
	}
	if !strings.Contains(view, "5 tasks to convergence") {
		t.Errorf("View missing convergence count")
	}
}

func TestDriftPanel_ViewFullScores(t *testing.T) {
	dp := NewDriftPanel()
	dp.SetDrift(100, 100, 100, "all done", "clean", "all pass", 0)

	view := dp.View(60)

	if !strings.Contains(view, "100%") {
		t.Errorf("View should show 100%% for full scores, got:\n%s", view)
	}
	if !strings.Contains(view, "0 tasks to convergence") {
		t.Errorf("View missing zero convergence")
	}
}

func TestDriftPanel_UpdateDriftMsg(t *testing.T) {
	dp := NewDriftPanel()

	msg := DriftUpdateMsg{
		FuncScore: 75,
		ArchScore: 85,
		PerfScore: 95,
		FuncLabel: "3/4 specs",
		ArchLabel: "1 cycle",
		PerfLabel: "0 failing",
		TasksLeft: 1,
	}

	dp.Update(msg)

	if dp.funcScore != 75 {
		t.Errorf("after Update, funcScore = %.0f, want 75", dp.funcScore)
	}
	if dp.archScore != 85 {
		t.Errorf("after Update, archScore = %.0f, want 85", dp.archScore)
	}
	if dp.perfScore != 95 {
		t.Errorf("after Update, perfScore = %.0f, want 95", dp.perfScore)
	}
	if dp.tasksLeft != 1 {
		t.Errorf("after Update, tasksLeft = %d, want 1", dp.tasksLeft)
	}
}

func TestDriftPanel_UpdateIgnoresOtherMsg(t *testing.T) {
	dp := NewDriftPanel()
	dp.SetDrift(50, 50, 50, "x", "y", "z", 2)

	dp.Update(OutputClearMsg{})

	if dp.funcScore != 50 {
		t.Errorf("funcScore changed after unrelated msg: %.0f", dp.funcScore)
	}
}

func TestDriftPanel_NarrowWidth(t *testing.T) {
	dp := NewDriftPanel()
	dp.SetDrift(50, 50, 50, "", "", "", 0)

	// Should not panic even with very narrow width.
	view := dp.View(10)
	if view == "" {
		t.Error("View should produce output even at narrow width")
	}
}

func TestDriftPanel_RenderPillarBar(t *testing.T) {
	bar := renderPillarBar("Test", 50, "half", 40)
	if !strings.Contains(bar, "Test") {
		t.Errorf("bar missing name 'Test': %q", bar)
	}
	if !strings.Contains(bar, "50%") {
		t.Errorf("bar missing '50%%': %q", bar)
	}
}

func TestDriftPanel_PillarStyle(t *testing.T) {
	// Just ensure no panics and correct thresholds.
	_ = pillarStyle(90)  // good
	_ = pillarStyle(65)  // mid
	_ = pillarStyle(30)  // bad
	_ = pillarStyle(0)   // bad
	_ = pillarStyle(100) // good
	_ = pillarStyle(80)  // good (boundary)
	_ = pillarStyle(50)  // mid (boundary)
}
