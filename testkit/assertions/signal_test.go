package assertions

import (
	"testing"

	"github.com/dpopsuev/djinn/telemetry"
)

func TestAssertSignalSequence_Pass(t *testing.T) {
	signals := []telemetry.Signal{
		{Workstream: "w1", Level: telemetry.Green},
		{Workstream: "w2", Level: telemetry.Red},
	}
	expected := []struct {
		Workstream string
		Level      telemetry.FlagLevel
	}{
		{"w1", telemetry.Green},
		{"w2", telemetry.Red},
	}
	AssertSignalSequence(t, signals, expected)
}

func TestAssertAndonLevel_Green(t *testing.T) {
	health := map[string]telemetry.WorkstreamHealth{
		"w1": {Level: telemetry.Green},
		"w2": {Level: telemetry.Green},
	}
	AssertAndonLevel(t, health, telemetry.Green)
}

func TestAssertAndonLevel_WorstWins(t *testing.T) {
	health := map[string]telemetry.WorkstreamHealth{
		"w1": {Level: telemetry.Green},
		"w2": {Level: telemetry.Red},
	}
	AssertAndonLevel(t, health, telemetry.Red)
}

func TestAssertWorkstreamLevel(t *testing.T) {
	health := map[string]telemetry.WorkstreamHealth{
		"auth": {Level: telemetry.Yellow},
	}
	AssertWorkstreamLevel(t, health, "auth", telemetry.Yellow)
}
