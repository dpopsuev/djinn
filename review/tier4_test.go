package review

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestComplexityHeuristic_Exceeded(t *testing.T) {
	dir := t.TempDir()

	// Create a file with high complexity.
	src := `package main

func complex() {
	if x > 0 {
		for i := 0; i < 10; i++ {
			if y > 0 && z > 0 {
				switch v {
				case 1:
					if a || b {
					}
				case 2:
				case 3:
				}
			}
		}
	}
}
`
	writeTestFile(t, dir, "complex.go", src)

	h := NewComplexityHeuristic(5)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		AddedFiles: []string{"complex.go"},
		WorkDir:    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if !signals[0].Exceeded {
		t.Errorf("complexity should exceed threshold 5, value=%.0f", signals[0].Value)
	}
}

func TestComplexityHeuristic_Simple(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "simple.go", "package main\n\nfunc simple() { return }\n")

	h := NewComplexityHeuristic(10)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		AddedFiles: []string{"simple.go"},
		WorkDir:    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Exceeded {
		t.Error("simple function should not exceed complexity threshold")
	}
}

func TestComplexityHeuristic_SkipsNonSource(t *testing.T) {
	h := NewComplexityHeuristic(1)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		AddedFiles: []string{"README.md", "config.yaml"},
		WorkDir:    t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Exceeded {
		t.Error("non-source files should not trigger complexity")
	}
}

func TestTestRatioHeuristic_LowRatio(t *testing.T) {
	h := NewTestRatioHeuristic(0.5)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		AddedFiles: []string{"handler.go", "store.go", "service.go", "handler_test.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	// 1 test / 3 source = 0.33 < 0.5
	if !signals[0].Exceeded {
		t.Errorf("ratio %.2f should be below 0.5", signals[0].Value)
	}
}

func TestTestRatioHeuristic_GoodRatio(t *testing.T) {
	h := NewTestRatioHeuristic(0.5)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		AddedFiles: []string{"handler.go", "handler_test.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Exceeded {
		t.Error("1:1 ratio should not be exceeded")
	}
}

func TestTestRatioHeuristic_NoSourceFiles(t *testing.T) {
	h := NewTestRatioHeuristic(0.5)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		AddedFiles: []string{"handler_test.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 0 {
		t.Error("no source files should produce no signal")
	}
}

func TestTestRatio_FileClassification(t *testing.T) {
	tests := []struct {
		file string
		test bool
	}{
		{"handler_test.go", true},
		{"handler.go", false},
		{"test_utils.py", true},
		{"utils.py", false},
		{"handler.test.ts", true},
		{"handler.spec.ts", true},
		{"handler.ts", false},
		{"tests/integration.rs", true},
		{"src/main.rs", false},
	}
	for _, tt := range tests {
		if got := isTestFile(tt.file); got != tt.test {
			t.Errorf("isTestFile(%q) = %v, want %v", tt.file, got, tt.test)
		}
	}
}

func TestMarkersHeuristic_TODOs(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "handler.go", `package main
// TODO: fix this later
// FIXME: broken on edge case
func handler() {
	// HACK: workaround for issue #123
	// XXX: remove before merge
}
`)

	h := NewMarkersHeuristic(3, 0)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"handler.go"},
		WorkDir:      dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	// 4 markers >= 3 threshold
	if !signals[0].Exceeded {
		t.Errorf("4 TODOs should exceed threshold 3, value=%.0f", signals[0].Value)
	}
}

func TestMarkersHeuristic_NoMarkers(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "clean.go", "package main\n\nfunc clean() {}\n")

	h := NewMarkersHeuristic(1, 0)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		ChangedFiles: []string{"clean.go"},
		WorkDir:      dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if signals[0].Exceeded {
		t.Error("clean file should have no TODOs")
	}
}

func TestDeadCodeHeuristic_UnreferencedFunc(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "utils.go", `package main

func usedFunc() {}

func unusedHelper() {}
`)
	writeTestFile(t, dir, "main.go", `package main

func main() {
	usedFunc()
}
`)

	h := NewDeadCodeHeuristic(1, 0)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		AddedFiles:   []string{"utils.go"},
		ChangedFiles: []string{"main.go"},
		WorkDir:      dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	// unusedHelper is unexported and not referenced in main.go
	if !signals[0].Exceeded {
		t.Errorf("unreferenced unexported func should trigger, value=%.0f", signals[0].Value)
	}
}

func TestDeadCodeHeuristic_ErrorRatio(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "service.go", `package main

func DoWork() error { return nil }
func DoMore() error { return nil }
func Helper() {}
func Util() {}
`)

	h := NewDeadCodeHeuristic(0, 0.5)
	signals, err := h.Evaluate(context.Background(), &DiffSnapshot{
		AddedFiles: []string{"service.go"},
		WorkDir:    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	// 2/4 = 0.5, not exceeded (>= threshold)
	if signals[0].Exceeded {
		t.Errorf("50%% error ratio should not be below 0.5 threshold, value=%.2f", signals[0].Value)
	}
}

func TestTier4Integration_BudgetMonitor(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "complex.go", `package main
func f() {
	if x { for y { if z && w { switch { case 1: case 2: case 3: } } } }
}
`)

	bm := NewBudgetMonitor()
	bm.Register(NewComplexityHeuristic(3))
	bm.Register(NewTestRatioHeuristic(0.5))

	signals := bm.Check(context.Background(), &DiffSnapshot{
		AddedFiles: []string{"complex.go"},
		WorkDir:    dir,
	})

	exceeded := Exceeded(signals)
	if len(exceeded) == 0 {
		t.Error("expected at least one exceeded signal from Tier 4")
	}
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
