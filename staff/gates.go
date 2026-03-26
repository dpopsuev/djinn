// gates.go — composite quality gates for the staffed runtime.
//
// TripleGate aggregates three quality checks (build, architecture, spec acceptance)
// and runs them in parallel. All must pass for the gate to pass.
package staff

import (
	"context"
	"fmt"
	"sync"
)

// TripleGate aggregates three quality checks: build, architecture, spec acceptance.
// Runs all three in parallel. All must pass for the gate to pass.
type TripleGate struct {
	build QualityGate                                 // MakeCircuitGate or equivalent
	arch  func(ctx context.Context, dir string) error // architecture check (nil = skip)
	spec  func(ctx context.Context, dir string) error // spec acceptance check (nil = skip)
}

// NewTripleGate creates a composite gate from a build gate and optional arch/spec checks.
// Nil arch or spec checks are skipped gracefully.
func NewTripleGate(build QualityGate, arch, spec func(context.Context, string) error) *TripleGate {
	return &TripleGate{build: build, arch: arch, spec: spec}
}

// Check runs all three quality checks in parallel and aggregates results.
// All checks must pass for the gate to pass. Diagnostics from all failing
// checks are collected.
func (g *TripleGate) Check(ctx context.Context, workDir string) (GateResult, error) {
	type checkResult struct {
		name        string
		passed      bool
		diagnostics []Diagnostic
		err         error
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []checkResult
	)

	collect := func(cr checkResult) {
		mu.Lock()
		results = append(results, cr)
		mu.Unlock()
	}

	// Build check (uses QualityGate interface).
	wg.Add(1)
	go func() {
		defer wg.Done()
		gr, err := g.build.Check(ctx, workDir)
		if err != nil {
			collect(checkResult{name: "build", err: err})
			return
		}
		collect(checkResult{name: "build", passed: gr.Passed, diagnostics: gr.Diagnostics})
	}()

	// Architecture check (nil = skip).
	if g.arch != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := g.arch(ctx, workDir); err != nil {
				collect(checkResult{
					name:   "arch",
					passed: false,
					diagnostics: []Diagnostic{
						{Source: "locus", Level: "error", Message: err.Error()},
					},
				})
				return
			}
			collect(checkResult{name: "arch", passed: true})
		}()
	}

	// Spec acceptance check (nil = skip).
	if g.spec != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := g.spec(ctx, workDir); err != nil {
				collect(checkResult{
					name:   "spec",
					passed: false,
					diagnostics: []Diagnostic{
						{Source: "scribe", Level: "error", Message: err.Error()},
					},
				})
				return
			}
			collect(checkResult{name: "spec", passed: true})
		}()
	}

	wg.Wait()

	// Aggregate: all must pass, collect all diagnostics.
	allPassed := true
	var allDiags []Diagnostic
	for _, r := range results {
		if r.err != nil {
			return GateResult{}, fmt.Errorf("%s check error: %w", r.name, r.err)
		}
		if !r.passed {
			allPassed = false
		}
		allDiags = append(allDiags, r.diagnostics...)
	}

	return GateResult{Passed: allPassed, Diagnostics: allDiags}, nil
}

// Interface compliance.
var _ QualityGate = (*TripleGate)(nil)
