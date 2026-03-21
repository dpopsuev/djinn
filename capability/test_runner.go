package capability

import "context"

// TestResult holds the outcome of a test execution.
type TestResult struct {
	Passed   int
	Failed   int
	Skipped  int
	Coverage float64
	Output   string
}

// TestRunnerPort runs test suites and reports results.
type TestRunnerPort interface {
	RunTests(ctx context.Context, target string) (TestResult, error)
}
