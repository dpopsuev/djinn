package broker

import "github.com/dpopsuev/djinn/telemetry"

// AndonBoard represents the aggregate health of all workstreams.
type AndonBoard struct {
	Level       telemetry.FlagLevel
	Workstreams map[string]telemetry.WorkstreamHealth
	Cordons     []Cordon
}

// ComputeAndon computes the Andon board from workstream health and cordons.
// Worst-flag-wins across all workstreams. Active cordons escalate to at least Red.
func ComputeAndon(health map[string]telemetry.WorkstreamHealth, cordons []Cordon) AndonBoard {
	board := AndonBoard{
		Level:       telemetry.Green,
		Workstreams: health,
		Cordons:     cordons,
	}

	for i := range health {
		if health[i].Level > board.Level {
			board.Level = health[i].Level
		}
	}

	if len(cordons) > 0 && board.Level < telemetry.Red {
		board.Level = telemetry.Red
	}

	return board
}
