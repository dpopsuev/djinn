package review

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/signal"
)

// mockHeuristic implements BudgetHeuristic for testing.
type mockHeuristic struct {
	name    string
	signals []Signal
}

func (m *mockHeuristic) Name() string { return m.name }

func (m *mockHeuristic) Evaluate(_ context.Context, _ *DiffSnapshot) ([]Signal, error) {
	return m.signals, nil
}

func TestBudgetMonitorCheck(t *testing.T) {
	bm := NewBudgetMonitor()
	bm.Register(&mockHeuristic{
		name: "files",
		signals: []Signal{
			{Metric: "files_touched", Value: 12, Threshold: 10, Exceeded: true, Detail: "12 files"},
		},
	})
	bm.Register(&mockHeuristic{
		name: "loc",
		signals: []Signal{
			{Metric: "loc_delta", Value: 200, Threshold: 500, Exceeded: false, Detail: "200 LOC"},
		},
	})

	signals := bm.Check(context.Background(), &DiffSnapshot{})
	if len(signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(signals))
	}

	exceeded := Exceeded(signals)
	if len(exceeded) != 1 {
		t.Fatalf("expected 1 exceeded, got %d", len(exceeded))
	}
	if exceeded[0].Metric != "files_touched" {
		t.Errorf("exceeded metric = %q, want %q", exceeded[0].Metric, "files_touched")
	}
}

func TestReviewWindowLifecycle(t *testing.T) {
	bm := NewBudgetMonitor()
	rw := NewReviewWindow("Fix auth/ login bug", bm)

	if rw.State != WindowWorking {
		t.Fatalf("initial state = %s, want working", rw.State)
	}

	rw.RecordChange("auth/handler.go")
	rw.RecordChange("auth/handler.go") // duplicate
	rw.RecordChange("auth/store.go")
	if len(rw.ChangedFiles) != 2 {
		t.Errorf("changed files = %d, want 2", len(rw.ChangedFiles))
	}

	// Working → Reviewing
	if err := rw.EnterReview(); err != nil {
		t.Fatal(err)
	}
	if rw.State != WindowReviewing {
		t.Fatalf("state = %s, want reviewing", rw.State)
	}

	// Can't enter review again
	if err := rw.EnterReview(); err == nil {
		t.Error("should fail: already reviewing")
	}

	// Reviewing → Approved
	if err := rw.Approve(); err != nil {
		t.Fatal(err)
	}
	if rw.State != WindowApproved {
		t.Fatalf("state = %s, want approved", rw.State)
	}
}

func TestReviewWindowReject(t *testing.T) {
	bm := NewBudgetMonitor()
	rw := NewReviewWindow("Fix bug", bm)
	rw.RecordChange("file.go")

	if err := rw.EnterReview(); err != nil {
		t.Fatal(err)
	}
	if err := rw.Reject(); err != nil {
		t.Fatal(err)
	}
	if rw.State != WindowRejected {
		t.Fatalf("state = %s, want rejected", rw.State)
	}
}

func TestReviewWindowSplit(t *testing.T) {
	bm := NewBudgetMonitor()
	rw := NewReviewWindow("Fix bug", bm)
	rw.RecordChange("good.go")
	rw.RecordChange("bad.go")

	if err := rw.EnterReview(); err != nil {
		t.Fatal(err)
	}
	if err := rw.Split([]string{"good.go"}, []string{"bad.go"}); err != nil {
		t.Fatal(err)
	}
	if rw.State != WindowSplit {
		t.Fatalf("state = %s, want split", rw.State)
	}
}

func TestScopeAnchorOnScope(t *testing.T) {
	sa := NewScopeAnchor("Fix login timeout in auth/handler.go")
	verdict, drifted := sa.CheckDrift([]string{"auth/handler.go", "auth/store.go"})
	if verdict != VerdictOnScope {
		t.Errorf("verdict = %s, want ON_SCOPE", verdict)
	}
	if len(drifted) != 0 {
		t.Errorf("drifted = %v, want empty", drifted)
	}
}

func TestScopeAnchorDrift(t *testing.T) {
	sa := NewScopeAnchor("Fix login timeout in auth/handler.go")
	verdict, drifted := sa.CheckDrift([]string{"auth/handler.go", "config/app.go", "middleware/cors.go"})
	if verdict != VerdictDrift {
		t.Errorf("verdict = %s, want DRIFT_DETECTED", verdict)
	}
	if len(drifted) != 2 {
		t.Errorf("drifted count = %d, want 2", len(drifted))
	}
}

func TestScopeAnchorNoPackages(t *testing.T) {
	sa := NewScopeAnchor("Fix the login bug")
	verdict, _ := sa.CheckDrift([]string{"anything.go"})
	if verdict != VerdictUnknown {
		t.Errorf("verdict = %s, want SCOPE_UNKNOWN", verdict)
	}
}

func TestReviewContextFormat(t *testing.T) {
	rc := &ReviewContext{
		ScopeAnchor:   "Fix auth/ login bug",
		DriftVerdict:  VerdictOnScope,
		TriggerReason: "budget: files_touched exceeded",
		ChangedFiles:  []string{"auth/handler.go", "auth/store.go"},
		FocusedFile:   "auth/handler.go",
		Annotations: []Annotation{
			{File: "auth/handler.go", Kind: "+", Comment: "clean fix"},
		},
	}
	prompt := rc.FormatPrompt()
	for _, want := range []string{"REVIEW MODE", "auth/", "ON_SCOPE", "2 changed", "handler.go", "clean fix"} {
		if !containsStr(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxFiles != 10 {
		t.Errorf("MaxFiles = %d, want 10", cfg.MaxFiles)
	}
	if cfg.MaxLOCDelta != 500 {
		t.Errorf("MaxLOCDelta = %d, want 500", cfg.MaxLOCDelta)
	}
	if !cfg.OnNewDependency {
		t.Error("OnNewDependency should default to true")
	}
	if cfg.AgentNarration {
		t.Error("AgentNarration should default to false")
	}
}

func TestEmitBudgetSignals(t *testing.T) {
	bus := signal.NewSignalBus()
	signals := []Signal{
		{Metric: "files_touched", Value: 12, Threshold: 10, Exceeded: true},
		{Metric: "loc_delta", Value: 200, Threshold: 500, Exceeded: false},
	}

	EmitBudgetSignals(bus, "test", signals)

	all := bus.Signals()
	if len(all) != 1 {
		t.Fatalf("expected 1 bus signal, got %d", len(all))
	}
	if all[0].Level != signal.Yellow {
		t.Errorf("level = %s, want yellow", all[0].Level)
	}
}

func TestEmitBudgetSignalsNone(t *testing.T) {
	bus := signal.NewSignalBus()
	signals := []Signal{
		{Metric: "files_touched", Value: 3, Threshold: 10, Exceeded: false},
	}

	EmitBudgetSignals(bus, "test", signals)

	if len(bus.Signals()) != 0 {
		t.Error("no exceeded signals should produce no bus emission")
	}
}

func TestCursorMoveTo(t *testing.T) {
	c := NewCursor()
	if c.CircuitIndex != -1 {
		t.Error("initial circuit should be -1")
	}
	c.MoveTo(2, 5)
	if c.CircuitIndex != 2 || c.StopIndex != 5 {
		t.Errorf("cursor = (%d, %d), want (2, 5)", c.CircuitIndex, c.StopIndex)
	}
	c.Reset()
	if c.CircuitIndex != -1 || c.File != "" {
		t.Error("reset should clear all fields")
	}
}

func TestCheckBudgetExceeded(t *testing.T) {
	bm := NewBudgetMonitor()
	bm.Register(&mockHeuristic{
		name: "files",
		signals: []Signal{
			{Metric: "files_touched", Value: 15, Threshold: 10, Exceeded: true},
		},
	})

	rw := NewReviewWindow("Fix bug", bm)
	for i := range 15 {
		rw.RecordChange("file" + string(rune('a'+i)) + ".go")
	}

	exceeded, signals := rw.CheckBudget(context.Background())
	if !exceeded {
		t.Error("should be exceeded")
	}
	if len(signals) != 1 {
		t.Errorf("exceeded signals = %d, want 1", len(signals))
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
