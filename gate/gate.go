package gate

import "context"

// GateConfig holds configuration for creating a gate.
type GateConfig struct {
	Name       string
	Severity   string // "warning" or "blocking"
	Thresholds map[string]float64
}

// Gate validates whether a sandbox meets quality criteria before promotion.
type Gate interface {
	Validate(ctx context.Context, sandboxID string) error
}
