package broker

import "github.com/dpopsuev/djinn/signal"

// AndonBoard represents the aggregate health of all workstreams.
type AndonBoard struct {
	Level       signal.FlagLevel
	Workstreams map[string]signal.WorkstreamHealth
	Cordons     []Cordon
}

// ComputeAndon computes the Andon board from workstream health and cordons.
// Worst-flag-wins across all workstreams. Active cordons escalate to at least Red.
func ComputeAndon(health map[string]signal.WorkstreamHealth, cordons []Cordon) AndonBoard {
	board := AndonBoard{
		Level:       signal.Green,
		Workstreams: health,
		Cordons:     cordons,
	}

	for _, h := range health {
		if h.Level > board.Level {
			board.Level = h.Level
		}
	}

	if len(cordons) > 0 && board.Level < signal.Red {
		board.Level = signal.Red
	}

	return board
}
