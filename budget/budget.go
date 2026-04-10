// Package budget provides the Budget domain service.
// Observer watches token spend. Controller decides throttle/pause/kill.
// Battery Service pattern: Observer + Controller + Data per domain.
package budget

// Budget tracks resource consumption for an agent session.
type Budget struct {
	TokensUsed int     `json:"tokens_used"`
	TokenLimit int     `json:"token_limit"`
	CostUsed   float64 `json:"cost_used"`
	CostLimit  float64 `json:"cost_limit"`
}

// Observer watches budget consumption and emits signals.
type Observer interface {
	// Check returns true if any limit is exceeded.
	Exceeded() bool

	// Usage returns current consumption as a percentage (0.0-1.0).
	Usage() float64
}

// Controller decides what to do when budget limits are approached.
type Controller interface {
	// ShouldThrottle returns true if the agent should slow down.
	ShouldThrottle() bool

	// ShouldKill returns true if the agent should be terminated.
	ShouldKill() bool
}
