// testparse.go — parser for `go test -json` output.
//
// ParseGoTestJSON reads JSON-encoded test events line by line and
// produces a structured TestResult with pass/fail/skip counts,
// coverage percentage, and failure details.
package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// TestResult summarizes a go test run.
type TestResult struct {
	Suite    string
	Passed   int
	Failed   int
	Skipped  int
	Coverage float64
	Duration time.Duration
	Failures []TestFailure
}

// TestFailure records a single failing test.
type TestFailure struct {
	Name    string
	Package string
	Output  string
}

// Total returns the total number of tests.
func (r *TestResult) Total() int {
	return r.Passed + r.Failed + r.Skipped
}

// testEvent is the JSON shape emitted by `go test -json`.
type testEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

// ParseGoTestJSON parses go test -json output from r and returns
// aggregated results.
func ParseGoTestJSON(r io.Reader) (*TestResult, error) {
	result := &TestResult{}
	scanner := bufio.NewScanner(r)

	// Track output per test for failure messages.
	type testKey struct {
		pkg  string
		name string
	}
	outputs := make(map[testKey][]string)
	var maxElapsed float64

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var evt testEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			// Skip non-JSON lines (e.g. compilation messages).
			continue
		}

		// Package-level result (no Test field).
		if evt.Test == "" && evt.Package != "" { //nolint:nestif // JSON event processing with multiple fields
			if result.Suite == "" {
				result.Suite = evt.Package
			}
			if evt.Elapsed > maxElapsed {
				maxElapsed = evt.Elapsed
			}

			// Check output for coverage.
			if evt.Action == "output" {
				cov := parseCoverage(evt.Output)
				if cov > 0 {
					result.Coverage = cov
				}
			}
			continue
		}

		key := testKey{pkg: evt.Package, name: evt.Test}

		switch evt.Action {
		case "output":
			outputs[key] = append(outputs[key], evt.Output)
		case "pass":
			result.Passed++
		case "fail":
			result.Failed++
			result.Failures = append(result.Failures, TestFailure{
				Name:    evt.Test,
				Package: evt.Package,
				Output:  strings.Join(outputs[key], ""),
			})
		case "skip":
			result.Skipped++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan test output: %w", err)
	}

	result.Duration = time.Duration(maxElapsed * float64(time.Second))
	return result, nil
}

// parseCoverage extracts coverage percentage from output like:
// "coverage: 75.3% of statements"
func parseCoverage(output string) float64 {
	const prefix = "coverage: "
	idx := strings.Index(output, prefix)
	if idx < 0 {
		return 0
	}
	s := output[idx+len(prefix):]
	pct := strings.Index(s, "%")
	if pct < 0 {
		return 0
	}
	v, err := strconv.ParseFloat(s[:pct], 64)
	if err != nil {
		return 0
	}
	return v
}
