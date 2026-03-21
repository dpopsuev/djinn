package gate

import "context"

// Gate severity levels.
const (
	SeverityWarning  = "warning"
	SeverityBlocking = "blocking"
)

// GateConfig holds configuration for creating a gate.
type GateConfig struct {
	Name       string
	Severity   string // one of Severity* constants
	Thresholds map[string]float64
}

// Gate validates whether a sandbox meets quality criteria before promotion.
type Gate interface {
	Validate(ctx context.Context, sandboxID string) error
}
