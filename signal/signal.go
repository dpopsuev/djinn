package signal

import "time"

// FlagLevel represents the health status of a workstream.
type FlagLevel int

const (
	Green  FlagLevel = iota // Healthy
	Yellow                  // Degraded
	Red                     // Failing
	Black                   // Dead / unrecoverable
)

func (f FlagLevel) String() string {
	switch f {
	case Green:
		return "green"
	case Yellow:
		return "yellow"
	case Red:
		return "red"
	case Black:
		return "black"
	default:
		return "unknown"
	}
}

// Signal categories.
const (
	CategoryTest        = "test"
	CategorySecurity    = "security"
	CategoryPerformance = "performance"
	CategoryDrift       = "drift"
	CategoryBudget      = "budget"
	CategoryLifecycle   = "lifecycle"
)

// Signal represents a health or status event emitted by an agent workstream.
type Signal struct {
	Workstream string
	Level      FlagLevel
	Confidence float64   // 0.0-1.0, agent's self-assessed confidence
	Source     string    // agent ID or watchdog ID
	Scope      []string  // affected file/package paths
	Category   string    // one of Category* constants
	Message    string    // human-readable evidence
	Timestamp  time.Time
}
