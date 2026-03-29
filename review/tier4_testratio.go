// tier4_testratio.go — Tier 4 heuristic: test coverage ratio (TSK-463).
//
// Classifies new files as source or test by naming convention.
// Signals when the ratio of test files to source files is too low.
package review

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// TestRatioHeuristic checks that new code includes sufficient tests.
type TestRatioHeuristic struct {
	MinTestRatio float64 // minimum test files / source files ratio (default: 0.5)
}

// NewTestRatioHeuristic creates a heuristic with the given minimum ratio.
func NewTestRatioHeuristic(minRatio float64) *TestRatioHeuristic {
	return &TestRatioHeuristic{MinTestRatio: minRatio}
}

func (h *TestRatioHeuristic) Name() string { return "tier4_test_ratio" }

func (h *TestRatioHeuristic) Evaluate(_ context.Context, diff *DiffSnapshot) ([]Signal, error) {
	if h.MinTestRatio <= 0 {
		return nil, nil
	}

	sourceFiles := 0
	testFiles := 0

	for _, f := range diff.AddedFiles {
		if !isSourceFile(f) {
			continue
		}
		if isTestFile(f) {
			testFiles++
		} else {
			sourceFiles++
		}
	}

	if sourceFiles == 0 {
		return nil, nil // no new source files → nothing to check
	}

	ratio := float64(testFiles) / float64(sourceFiles)
	exceeded := ratio < h.MinTestRatio

	return []Signal{{
		Metric:    "test_coverage_ratio",
		Value:     ratio,
		Threshold: h.MinTestRatio,
		Exceeded:  exceeded,
		Detail:    fmt.Sprintf("%d new source files, %d test files (ratio: %.1f)", sourceFiles, testFiles, ratio),
	}}, nil
}

// isTestFile checks if a filename follows test naming conventions.
func isTestFile(path string) bool {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Go: _test.go
	if strings.HasSuffix(base, "_test.go") {
		return true
	}
	// JS/TS: .test.ts, .spec.ts
	if strings.HasSuffix(name, ".test") || strings.HasSuffix(name, ".spec") {
		return true
	}
	// Python: test_*.py
	if strings.HasPrefix(name, "test_") {
		return true
	}
	// Tests directory convention across languages.
	if strings.Contains(path, "tests/") || strings.Contains(path, "test/") {
		return true
	}
	return false
}
