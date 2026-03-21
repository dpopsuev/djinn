package capability

import "context"

// Vulnerability represents a detected security issue.
type Vulnerability struct {
	ID          string
	Severity    string
	Package     string
	Description string
}

// SecurityScanningPort scans for vulnerabilities in code and dependencies.
type SecurityScanningPort interface {
	Scan(ctx context.Context, target string) ([]Vulnerability, error)
}
