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

// Signal represents a health or status event emitted by an agent workstream.
type Signal struct {
	Workstream string
	Level      FlagLevel
	Message    string
	Timestamp  time.Time
}
