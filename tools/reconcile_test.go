package tools

import (
	"math"
	"testing"

	"github.com/dpopsuev/djinn/artifact"
)

func TestComputeDrift_AllDone(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	id1, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "task one", Status: artifact.StatusPending})
	id2, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "task two", Status: artifact.StatusPending})
	g.UpdateStatus(id1, artifact.StatusDone)
	g.UpdateStatus(id2, artifact.StatusDone)

	arch := &ArchReport{}
	test := &TestResult{Passed: 5, Failed: 0}

	dr := ComputeDrift(g, arch, test)

	if dr.Functionality.Score != 100 {
		t.Errorf("Functionality score = %.1f, want 100", dr.Functionality.Score)
	}
	if dr.Structure.Score != 100 {
		t.Errorf("Structure score = %.1f, want 100", dr.Structure.Score)
	}
	if dr.Performance.Score != 100 {
		t.Errorf("Performance score = %.1f, want 100", dr.Performance.Score)
	}
	if dr.TasksToConvergence != 0 {
		t.Errorf("TasksToConvergence = %d, want 0", dr.TasksToConvergence)
	}
}

func TestComputeDrift_HalfDone(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	id1, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "done task", Status: artifact.StatusPending})
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "pending task", Status: artifact.StatusPending})
	g.UpdateStatus(id1, artifact.StatusDone)

	dr := ComputeDrift(g, &ArchReport{}, &TestResult{Passed: 1})

	if dr.Functionality.Score != 50 {
		t.Errorf("Functionality score = %.1f, want 50", dr.Functionality.Score)
	}
	if dr.TasksToConvergence != 1 {
		t.Errorf("TasksToConvergence = %d, want 1", dr.TasksToConvergence)
	}
}

func TestComputeDrift_CyclesReduceStructure(t *testing.T) {
	arch := &ArchReport{
		Cycles: [][]string{
			{"pkg/a", "pkg/b"},
			{"pkg/c", "pkg/d"},
		},
	}

	dr := ComputeDrift(nil, arch, nil)

	want := 80.0 // 100 - (2 * 10)
	if dr.Structure.Score != want {
		t.Errorf("Structure score = %.1f, want %.1f", dr.Structure.Score, want)
	}
}

func TestComputeDrift_ManyCyclesClampToZero(t *testing.T) {
	cycles := make([][]string, 15)
	for i := range cycles {
		cycles[i] = []string{"a", "b"}
	}
	arch := &ArchReport{Cycles: cycles}

	dr := ComputeDrift(nil, arch, nil)

	if dr.Structure.Score != 0 {
		t.Errorf("Structure score = %.1f, want 0 (clamped)", dr.Structure.Score)
	}
}

func TestComputeDrift_AllTestsPass(t *testing.T) {
	test := &TestResult{Passed: 10, Failed: 0}
	dr := ComputeDrift(nil, nil, test)

	if dr.Performance.Score != 100 {
		t.Errorf("Performance score = %.1f, want 100", dr.Performance.Score)
	}
}

func TestComputeDrift_SomeTestsFail(t *testing.T) {
	test := &TestResult{
		Passed: 3,
		Failed: 2,
		Failures: []TestFailure{
			{Name: "TestA", Package: "pkg/a"},
			{Name: "TestB", Package: "pkg/b"},
		},
	}
	dr := ComputeDrift(nil, nil, test)

	want := 60.0 // 3/(3+2) * 100
	if dr.Performance.Score != want {
		t.Errorf("Performance score = %.1f, want %.1f", dr.Performance.Score, want)
	}
	if dr.Performance.Summary != "2 failing" {
		t.Errorf("Performance summary = %q, want %q", dr.Performance.Summary, "2 failing")
	}
	if len(dr.Performance.Details) != 2 {
		t.Errorf("Performance details len = %d, want 2", len(dr.Performance.Details))
	}
}

func TestComputeDrift_NoTests(t *testing.T) {
	test := &TestResult{Passed: 0, Failed: 0}
	dr := ComputeDrift(nil, nil, test)

	if dr.Performance.Score != 100 {
		t.Errorf("Performance score = %.1f, want 100 (no tests)", dr.Performance.Score)
	}
}

func TestComputeDrift_NilInputs(t *testing.T) {
	dr := ComputeDrift(nil, nil, nil)

	if dr.Functionality.Score != 100 {
		t.Errorf("Functionality score = %.1f, want 100", dr.Functionality.Score)
	}
	if dr.Structure.Score != 100 {
		t.Errorf("Structure score = %.1f, want 100", dr.Structure.Score)
	}
	if dr.Performance.Score != 100 {
		t.Errorf("Performance score = %.1f, want 100", dr.Performance.Score)
	}
	if dr.TasksToConvergence != 0 {
		t.Errorf("TasksToConvergence = %d, want 0", dr.TasksToConvergence)
	}
}

func TestComputeDrift_EmptyGraph(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	dr := ComputeDrift(g, nil, nil)

	if dr.Functionality.Score != 100 {
		t.Errorf("Functionality score = %.1f, want 100 for empty store", dr.Functionality.Score)
	}
}

func TestComputeStructureWithViolations(t *testing.T) {
	arch := &ArchReport{
		Cycles: [][]string{{"a", "b"}},
	}
	violations := []string{"tui imports driver/acp", "staff imports tui"}

	ps := ComputeStructureWithViolations(arch, violations)

	// 100 - 10 (1 cycle) - 10 (2 violations * 5) = 80
	want := 80.0
	if ps.Score != want {
		t.Errorf("Structure score = %.1f, want %.1f", ps.Score, want)
	}
}

func TestClampScore(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{-10, 0},
		{0, 0},
		{50, 50},
		{100, 100},
		{150, 100},
	}
	for _, tc := range tests {
		got := clampScore(tc.in)
		if got != tc.want {
			t.Errorf("clampScore(%.0f) = %.0f, want %.0f", tc.in, got, tc.want)
		}
	}
}

func TestReconcile_FunctionalitySummary(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	for _, title := range []string{"one", "two", "three", "four", "five"} {
		g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: title, Status: artifact.StatusPending})
	}

	all := g.ListSorted()
	for _, task := range all[:4] {
		g.UpdateStatus(task.ID, artifact.StatusDone)
	}

	dr := ComputeDrift(g, nil, nil)

	if dr.Functionality.Summary != "4/5 specs" {
		t.Errorf("Functionality summary = %q, want %q", dr.Functionality.Summary, "4/5 specs")
	}
	if math.Abs(dr.Functionality.Score-80) > 0.1 {
		t.Errorf("Functionality score = %.1f, want 80", dr.Functionality.Score)
	}
}
