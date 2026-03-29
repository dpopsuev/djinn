package review

import (
	"context"
	"testing"
)

func TestFilesHeuristic_Exceeded(t *testing.T) {
	h := &FilesHeuristic{MaxFiles: 5, MaxLOC: 300}
	diff := &DiffSnapshot{
		ChangedFiles: []string{"a.go", "b.go", "c.go"},
		AddedFiles:   []string{"d.go", "e.go"},
		LOCDelta:     350,
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(signals))
	}

	// Files: 5 total = 5 >= 5 threshold → exceeded.
	if !signals[0].Exceeded {
		t.Error("files_touched should be exceeded (5 >= 5)")
	}
	// LOC: 350 >= 300 → exceeded.
	if !signals[1].Exceeded {
		t.Error("loc_delta should be exceeded (350 >= 300)")
	}
}

func TestFilesHeuristic_NotExceeded(t *testing.T) {
	h := &FilesHeuristic{MaxFiles: 10, MaxLOC: 500}
	diff := &DiffSnapshot{
		ChangedFiles: []string{"a.go"},
		LOCDelta:     50,
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range signals {
		if s.Exceeded {
			t.Errorf("%s should not be exceeded", s.Metric)
		}
	}
}

func TestFilesHeuristic_NegativeLOC(t *testing.T) {
	h := &FilesHeuristic{MaxLOC: 100}
	diff := &DiffSnapshot{LOCDelta: -150}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	// Absolute value: 150 >= 100 → exceeded.
	if !signals[0].Exceeded {
		t.Error("negative LOC delta should use absolute value")
	}
}

func TestFilesHeuristic_Disabled(t *testing.T) {
	h := &FilesHeuristic{MaxFiles: 0, MaxLOC: 0}
	diff := &DiffSnapshot{ChangedFiles: []string{"a.go"}, LOCDelta: 1000}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 0 {
		t.Errorf("disabled heuristic should produce 0 signals, got %d", len(signals))
	}
}

func TestPackagesHeuristic_Exceeded(t *testing.T) {
	h := &PackagesHeuristic{MaxPackages: 3, MaxNewFiles: 2, MaxDeletedFiles: 2}
	diff := &DiffSnapshot{
		PackagesHit:  []string{"auth", "config", "middleware", "tui"},
		AddedFiles:   []string{"new1.go", "new2.go", "new3.go"},
		DeletedFiles: []string{"old1.go", "old2.go"},
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 3 {
		t.Fatalf("expected 3 signals, got %d", len(signals))
	}

	exceeded := Exceeded(signals)
	if len(exceeded) != 3 {
		t.Errorf("expected 3 exceeded, got %d", len(exceeded))
	}
}

func TestPackagesHeuristic_NotExceeded(t *testing.T) {
	h := &PackagesHeuristic{MaxPackages: 10, MaxNewFiles: 10, MaxDeletedFiles: 10}
	diff := &DiffSnapshot{
		PackagesHit: []string{"auth"},
		AddedFiles:  []string{"new.go"},
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range signals {
		if s.Exceeded {
			t.Errorf("%s should not be exceeded", s.Metric)
		}
	}
}

func TestDepsHeuristic_DependencyFile(t *testing.T) {
	h := &DepsHeuristic{OnNewDependency: true}
	diff := &DiffSnapshot{
		ChangedFiles: []string{"go.mod", "auth/handler.go"},
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}

	exceeded := Exceeded(signals)
	if len(exceeded) != 1 {
		t.Fatalf("expected 1 exceeded (go.mod), got %d", len(exceeded))
	}
	if exceeded[0].Metric != "dependency_changed" {
		t.Errorf("metric = %q, want dependency_changed", exceeded[0].Metric)
	}
}

func TestDepsHeuristic_ConfigFile(t *testing.T) {
	h := &DepsHeuristic{OnNewDependency: true}
	diff := &DiffSnapshot{
		ChangedFiles: []string{"config/app.yaml", "auth/handler.go"},
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}

	exceeded := Exceeded(signals)
	if len(exceeded) != 1 {
		t.Fatalf("expected 1 exceeded (config file), got %d", len(exceeded))
	}
	if exceeded[0].Metric != "config_changed" {
		t.Errorf("metric = %q, want config_changed", exceeded[0].Metric)
	}
}

func TestDepsHeuristic_Disabled(t *testing.T) {
	h := &DepsHeuristic{OnNewDependency: false}
	diff := &DiffSnapshot{
		ChangedFiles: []string{"go.mod"},
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 0 {
		t.Error("disabled heuristic should produce 0 signals")
	}
}

func TestDepsHeuristic_MultipleDepFiles(t *testing.T) {
	h := &DepsHeuristic{OnNewDependency: true}
	diff := &DiffSnapshot{
		ChangedFiles: []string{"go.mod", "go.sum"},
		AddedFiles:   []string{"package.json"},
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}

	exceeded := Exceeded(signals)
	if len(exceeded) != 1 {
		t.Fatalf("expected 1 dependency_changed signal, got %d exceeded", len(exceeded))
	}
	// All 3 dep files should be in one signal.
	if exceeded[0].Value != 3 {
		t.Errorf("value = %.0f, want 3", exceeded[0].Value)
	}
}

func TestDepsHeuristic_NormalFiles(t *testing.T) {
	h := &DepsHeuristic{OnNewDependency: true}
	diff := &DiffSnapshot{
		ChangedFiles: []string{"auth/handler.go", "auth/store.go"},
	}

	signals, err := h.Evaluate(context.Background(), diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(Exceeded(signals)) != 0 {
		t.Error("normal Go files should not trigger deps heuristic")
	}
}

func TestTier1Integration_BudgetMonitor(t *testing.T) {
	cfg := DefaultConfig()
	bm := NewBudgetMonitor()
	bm.Register(NewFilesHeuristic(cfg))
	bm.Register(NewPackagesHeuristic(cfg))
	bm.Register(NewDepsHeuristic(cfg))

	diff := &DiffSnapshot{
		ChangedFiles: make([]string, 12), // 12 > MaxFiles(10)
		PackagesHit:  []string{"a", "b"},
		LOCDelta:     100,
	}

	signals := bm.Check(context.Background(), diff)
	exceeded := Exceeded(signals)

	// Only files_touched should exceed (12 >= 10).
	found := false
	for _, s := range exceeded {
		if s.Metric == "files_touched" {
			found = true
		}
	}
	if !found {
		t.Error("files_touched should be in exceeded signals")
	}
}
