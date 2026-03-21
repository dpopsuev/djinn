package capability

import "context"

// Finding represents a single static analysis issue.
type Finding struct {
	File     string
	Line     int
	Severity string
	Message  string
}

// StaticAnalysisPort runs linters and type checkers.
type StaticAnalysisPort interface {
	Lint(ctx context.Context, target string) ([]Finding, error)
	TypeCheck(ctx context.Context, target string) error
}
