// litmus.go — Litmus interface and core types.
//
// GOL-175, TSK-1118
package litmus

import "time"

// Litmus caches build/test results per package. Thread-safe.
type Litmus interface {
	// TestResult returns the cached test result for a package.
	// Returns false if not cached or stale.
	TestResult(pkg string) (TestResultEntry, bool)

	// BuildResult returns the cached build result for a package.
	BuildResult(pkg string) (BuildResultEntry, bool)

	// RecordTest stores a test result for a package with its source hash.
	RecordTest(pkg string, result TestResultEntry)

	// RecordBuild stores a build result for a package with its source hash.
	RecordBuild(pkg string, result BuildResultEntry)

	// Invalidate evicts cached results for a specific package.
	Invalidate(pkg string)

	// InvalidateAll evicts all cached results.
	InvalidateAll()
}

// TestResultEntry is a cached test run outcome for one package.
type TestResultEntry struct {
	Pass        bool          `json:"pass"`
	Total       int           `json:"total"`                  // total tests
	Failed      int           `json:"failed"`                 // failed count
	Skipped     int           `json:"skipped"`                // skipped count
	FailedTests []string      `json:"failed_tests,omitempty"` // names of failed tests
	Duration    time.Duration `json:"duration"`
	Output      string        `json:"output,omitempty"` // truncated output
	SourceHash  string        `json:"source_hash"`      // combined hash of package .go files
	Timestamp   time.Time     `json:"timestamp"`
}

// BuildResultEntry is a cached build outcome for one package.
type BuildResultEntry struct {
	Pass       bool      `json:"pass"`
	Errors     []string  `json:"errors,omitempty"`
	SourceHash string    `json:"source_hash"`
	Timestamp  time.Time `json:"timestamp"`
}
