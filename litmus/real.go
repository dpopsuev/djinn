// real.go — RealLitmus: production test/build cache.
//
// Wraps exec go test -json. Caches results per package with source hash.
// Cache hit when source hash matches (no file changed). Cache miss
// triggers re-run. ORANGE instrumentation on failures, YELLOW on success.
//
// GOL-175, TSK-1120, TSK-1121
package litmus

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

var _ Litmus = (*RealLitmus)(nil)

// RealLitmus caches test/build results, keyed by source hash.
type RealLitmus struct {
	mu      sync.RWMutex
	workDir string
	tests   map[string]TestResultEntry
	builds  map[string]BuildResultEntry
	log     *slog.Logger
}

// NewRealLitmus creates a production Litmus for the given workspace.
func NewRealLitmus(workDir string, log *slog.Logger) *RealLitmus {
	if log == nil {
		log = telemetry.Nop()
	}
	return &RealLitmus{
		workDir: workDir,
		tests:   make(map[string]TestResultEntry),
		builds:  make(map[string]BuildResultEntry),
		log:     log,
	}
}

func (l *RealLitmus) TestResult(pkg string) (TestResultEntry, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	r, ok := l.tests[pkg]
	return r, ok
}

func (l *RealLitmus) BuildResult(pkg string) (BuildResultEntry, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	r, ok := l.builds[pkg]
	return r, ok
}

func (l *RealLitmus) RecordTest(pkg string, result TestResultEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tests[pkg] = result
	l.log.DebugContext(context.Background(), "litmus: test recorded",
		slog.String(telemetry.KeyComponent, pkg),
		slog.String(telemetry.KeyStatus, fmt.Sprintf("pass=%t total=%d", result.Pass, result.Total)),
	)
}

func (l *RealLitmus) RecordBuild(pkg string, result BuildResultEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.builds[pkg] = result
	l.log.DebugContext(context.Background(), "litmus: build recorded",
		slog.String(telemetry.KeyComponent, pkg),
		slog.String(telemetry.KeyStatus, fmt.Sprintf("pass=%t", result.Pass)),
	)
}

func (l *RealLitmus) Invalidate(pkg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.tests, pkg)
	delete(l.builds, pkg)
}

func (l *RealLitmus) InvalidateAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tests = make(map[string]TestResultEntry)
	l.builds = make(map[string]BuildResultEntry)
}

// RunTests executes `go test -json` on a package and records the result.
func (l *RealLitmus) RunTests(ctx context.Context, pkg, sourceHash string) (TestResultEntry, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "go", "test", "-json", "-count=1", pkg)
	cmd.Dir = l.workDir

	out, err := cmd.Output()
	elapsed := time.Since(start)

	result := l.parseTestJSON(string(out), pkg, sourceHash, elapsed)

	if err != nil && !result.hasResults {
		// Total failure — couldn't even parse.
		l.log.WarnContext(ctx, "litmus: test execution failed",
			slog.String(telemetry.KeyComponent, pkg),
			slog.String(telemetry.KeyError, err.Error()),
		)
		entry := TestResultEntry{
			Pass:       false,
			SourceHash: sourceHash,
			Timestamp:  time.Now(),
			Duration:   elapsed,
			Output:     truncateOutput(string(out)),
		}
		l.RecordTest(pkg, entry)
		return entry, fmt.Errorf("litmus: go test %s: %w", pkg, err)
	}

	l.RecordTest(pkg, result.entry)
	return result.entry, nil
}

// RunBuild executes `go build` on a package and records the result.
func (l *RealLitmus) RunBuild(ctx context.Context, pkg, sourceHash string) (BuildResultEntry, error) {
	cmd := exec.CommandContext(ctx, "go", "build", pkg)
	cmd.Dir = l.workDir

	out, err := cmd.CombinedOutput()
	entry := BuildResultEntry{
		Pass:       err == nil,
		SourceHash: sourceHash,
		Timestamp:  time.Now(),
	}
	if err != nil {
		entry.Errors = parseErrors(string(out))
		l.log.WarnContext(ctx, "litmus: build failed",
			slog.String(telemetry.KeyComponent, pkg),
			slog.Int(telemetry.KeyCount, len(entry.Errors)),
		)
	}

	l.RecordBuild(pkg, entry)
	return entry, err
}

// --- JSON parsing ---

type testJSONEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

type parsedResult struct {
	entry      TestResultEntry
	hasResults bool
}

func (l *RealLitmus) parseTestJSON(output, _ /* pkg */, sourceHash string, elapsed time.Duration) parsedResult {
	scanner := bufio.NewScanner(strings.NewReader(output))

	var total, failed, skipped int
	var failedTests []string
	hasResults := false

	for scanner.Scan() {
		var ev testJSONEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}

		if ev.Test == "" {
			continue // package-level event
		}

		switch ev.Action {
		case "pass":
			total++
			hasResults = true
		case "fail":
			total++
			failed++
			failedTests = append(failedTests, ev.Test)
			hasResults = true
		case "skip":
			total++
			skipped++
			hasResults = true
		}
	}

	return parsedResult{
		entry: TestResultEntry{
			Pass:        failed == 0 && hasResults,
			Total:       total,
			Failed:      failed,
			Skipped:     skipped,
			FailedTests: failedTests,
			Duration:    elapsed,
			SourceHash:  sourceHash,
			Timestamp:   time.Now(),
			Output:      truncateOutput(output),
		},
		hasResults: hasResults,
	}
}

func parseErrors(output string) []string {
	var errors []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			errors = append(errors, line)
		}
	}
	return errors
}

func truncateOutput(s string) string {
	const maxLen = 2000
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
