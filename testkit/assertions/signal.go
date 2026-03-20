package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/signal"
)

// AssertSignalSequence checks that the signals match the expected workstream+level pairs in order.
func AssertSignalSequence(t *testing.T, signals []signal.Signal, expected []struct {
	Workstream string
	Level      signal.FlagLevel
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
func AssertAndonLevel(t *testing.T, health map[string]signal.WorkstreamHealth, want signal.FlagLevel) {
	t.Helper()
	worst := signal.Green
	for _, h := range health {
		if h.Level > worst {
			worst = h.Level
		}
	}
	if worst != want {
		t.Fatalf("andon level = %v, want %v", worst, want)
	}
}

// AssertWorkstreamLevel checks that a specific workstream has the expected level.
func AssertWorkstreamLevel(t *testing.T, health map[string]signal.WorkstreamHealth, workstream string, want signal.FlagLevel) {
	t.Helper()
	h, ok := health[workstream]
	if !ok {
		t.Fatalf("workstream %q not found in health map", workstream)
	}
	if h.Level != want {
		t.Fatalf("workstream %q level = %v, want %v", workstream, h.Level, want)
	}
}
