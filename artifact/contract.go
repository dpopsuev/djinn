// contract.go — ContractGate validates quality criteria before promotion.
//
// Moved from gate/ package to artifact/ because gates are quality contracts
// on artifacts, not standalone infrastructure.
package artifact

import "context"

// ContractGate severity levels.
const (
	SeverityWarning  = "warning"
	SeverityBlocking = "blocking"
)

// ContractGateConfig holds configuration for creating a gate.
type ContractGateConfig struct {
	Name       string
	Severity   string // one of Severity* constants
	Thresholds map[string]float64
}

// ContractGate validates whether a sandbox meets quality criteria before promotion.
type ContractGate interface {
	Validate(ctx context.Context, sandboxID string) error
}
