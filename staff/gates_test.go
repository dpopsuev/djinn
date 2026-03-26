package staff

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// stubGate is a QualityGate that returns a canned result.
type stubGate struct {
	result GateResult
	err    error
	delay  time.Duration
}

func (g *stubGate) Check(ctx context.Context, _ string) (GateResult, error) {
	if g.delay > 0 {
		select {
		case <-time.After(g.delay):
		case <-ctx.Done():
			return GateResult{}, ctx.Err()
		}
	}
	return g.result, g.err
}

func TestTripleGate_AllPass(t *testing.T) {
	build := &stubGate{result: GateResult{Passed: true}}
	arch := func(_ context.Context, _ string) error { return nil }
	spec := func(_ context.Context, _ string) error { return nil }

	gate := NewTripleGate(build, arch, spec)
	result, err := gate.Check(context.Background(), "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected gate to pass")
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %d", len(result.Diagnostics))
	}
}

func TestTripleGate_BuildFails(t *testing.T) {
	build := &stubGate{result: GateResult{
		Passed: false,
		Diagnostics: []Diagnostic{
			{Source: "make", Level: "error", Message: "compilation failed"},
		},
	}}
	arch := func(_ context.Context, _ string) error { return nil }
	spec := func(_ context.Context, _ string) error { return nil }

	gate := NewTripleGate(build, arch, spec)
	result, err := gate.Check(context.Background(), "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatal("expected gate to fail")
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected diagnostics from build failure")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Source == "make" && d.Message == "compilation failed" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected build diagnostic with source=make")
	}
}

func TestTripleGate_ArchFails(t *testing.T) {
	build := &stubGate{result: GateResult{Passed: true}}
	arch := func(_ context.Context, _ string) error {
		return fmt.Errorf("circular dependency detected")
	}
	spec := func(_ context.Context, _ string) error { return nil }

	gate := NewTripleGate(build, arch, spec)
	result, err := gate.Check(context.Background(), "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatal("expected gate to fail when arch fails")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Source == "locus" && d.Level == "error" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected arch diagnostic with source=locus")
	}
}

func TestTripleGate_NilChecksSkipped(t *testing.T) {
	build := &stubGate{result: GateResult{Passed: true}}
	gate := NewTripleGate(build, nil, nil)

	result, err := gate.Check(context.Background(), "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected gate to pass with nil checks skipped")
	}
}

func TestTripleGate_AllFail(t *testing.T) {
	build := &stubGate{result: GateResult{
		Passed: false,
		Diagnostics: []Diagnostic{
			{Source: "make", Level: "error", Message: "build broken"},
		},
	}}
	arch := func(_ context.Context, _ string) error {
		return fmt.Errorf("arch violation")
	}
	spec := func(_ context.Context, _ string) error {
		return fmt.Errorf("spec mismatch")
	}

	gate := NewTripleGate(build, arch, spec)
	result, err := gate.Check(context.Background(), "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatal("expected gate to fail when all checks fail")
	}

	// Should have diagnostics from all three sources.
	sources := map[string]bool{}
	for _, d := range result.Diagnostics {
		sources[d.Source] = true
	}
	for _, want := range []string{"make", "locus", "scribe"} {
		if !sources[want] {
			t.Errorf("missing diagnostic from source %q", want)
		}
	}
}

func TestTripleGate_Concurrent(t *testing.T) {
	var running atomic.Int32
	var maxConcurrent atomic.Int32

	trackConcurrency := func(delay time.Duration) func(context.Context, string) error {
		return func(_ context.Context, _ string) error {
			cur := running.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(delay)
			running.Add(-1)
			return nil
		}
	}

	build := &stubGate{result: GateResult{Passed: true}, delay: 50 * time.Millisecond}
	arch := trackConcurrency(50 * time.Millisecond)
	spec := trackConcurrency(50 * time.Millisecond)

	gate := NewTripleGate(build, arch, spec)

	start := time.Now()
	result, err := gate.Check(context.Background(), "/tmp")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected gate to pass")
	}

	// If sequential, would take ~150ms. Parallel should be ~50ms.
	if elapsed > 120*time.Millisecond {
		t.Fatalf("checks appear sequential: elapsed %v (expected <120ms for parallel)", elapsed)
	}

	// At least 2 concurrent (build runs in its own goroutine + one of arch/spec).
	if maxConcurrent.Load() < 2 {
		t.Fatalf("expected at least 2 concurrent checks, got %d", maxConcurrent.Load())
	}
}
