package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/signal"
)

func TestAssertSignalSequence_Pass(t *testing.T) {
	signals := []signal.Signal{
		{Workstream: "w1", Level: signal.Green},
		{Workstream: "w2", Level: signal.Red},
	}
	expected := []struct {
		Workstream string
		Level      signal.FlagLevel
	}{
		{"w1", signal.Green},
		{"w2", signal.Red},
	}
	AssertSignalSequence(t, signals, expected)
}

func TestAssertAndonLevel_Green(t *testing.T) {
	health := map[string]signal.WorkstreamHealth{
		"w1": {Level: signal.Green},
		"w2": {Level: signal.Green},
	}
	AssertAndonLevel(t, health, signal.Green)
}

func TestAssertAndonLevel_WorstWins(t *testing.T) {
	health := map[string]signal.WorkstreamHealth{
		"w1": {Level: signal.Green},
		"w2": {Level: signal.Red},
	}
	AssertAndonLevel(t, health, signal.Red)
}

func TestAssertWorkstreamLevel(t *testing.T) {
	health := map[string]signal.WorkstreamHealth{
		"auth": {Level: signal.Yellow},
	}
	AssertWorkstreamLevel(t, health, "auth", signal.Yellow)
}
