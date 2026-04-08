package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/telemetry"
)

// AssertSignalSequence checks that the signals match the expected workstream+level pairs in order.
func AssertSignalSequence(t *testing.T, signals []telemetry.Signal, expected []struct {
	Workstream string
	Level      telemetry.FlagLevel
}) {
	t.Helper()
	if len(signals) < len(expected) {
		t.Fatalf("got %d signals, want at least %d", len(signals), len(expected))
	}
	for i, exp := range expected {
		if signals[i].Workstream != exp.Workstream {
			t.Fatalf("signal[%d].Workstream = %q, want %q", i, signals[i].Workstream, exp.Workstream)
		}
		if signals[i].Level != exp.Level {
			t.Fatalf("signal[%d].Level = %v, want %v", i, signals[i].Level, exp.Level)
		}
	}
}

// AssertAndonLevel checks that the computed andon level matches.
func AssertAndonLevel(t *testing.T, health map[string]telemetry.WorkstreamHealth, want telemetry.FlagLevel) {
	t.Helper()
	worst := telemetry.Green
	for k := range health {
		if health[k].Level > worst {
			worst = health[k].Level
		}
	}
	if worst != want {
		t.Fatalf("andon level = %v, want %v", worst, want)
	}
}

// AssertWorkstreamLevel checks that a specific workstream has the expected level.
func AssertWorkstreamLevel(t *testing.T, health map[string]telemetry.WorkstreamHealth, workstream string, want telemetry.FlagLevel) {
	t.Helper()
	h, ok := health[workstream]
	if !ok {
		t.Fatalf("workstream %q not found in health map", workstream)
	}
	if h.Level != want {
		t.Fatalf("workstream %q level = %v, want %v", workstream, h.Level, want)
	}
}
