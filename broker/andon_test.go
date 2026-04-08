package broker

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

func TestComputeAndon_AllGreen(t *testing.T) {
	health := map[string]telemetry.WorkstreamHealth{
		"w1": {Workstream: "w1", Level: telemetry.Green},
		"w2": {Workstream: "w2", Level: telemetry.Green},
	}
	board := ComputeAndon(health, nil)
	if board.Level != telemetry.Green {
		t.Fatalf("Level = %v, want Green", board.Level)
	}
}

func TestComputeAndon_MixedLevels(t *testing.T) {
	health := map[string]telemetry.WorkstreamHealth{
		"w1": {Workstream: "w1", Level: telemetry.Green},
		"w2": {Workstream: "w2", Level: telemetry.Yellow},
		"w3": {Workstream: "w3", Level: telemetry.Red},
	}
	board := ComputeAndon(health, nil)
	if board.Level != telemetry.Red {
		t.Fatalf("Level = %v, want Red (worst-flag-wins)", board.Level)
	}
}

func TestComputeAndon_CordonEscalates(t *testing.T) {
	health := map[string]telemetry.WorkstreamHealth{
		"w1": {Workstream: "w1", Level: telemetry.Green},
	}
	cordons := []Cordon{{Scope: []string{"auth"}, Reason: "broken"}}
	board := ComputeAndon(health, cordons)
	if board.Level != telemetry.Red {
		t.Fatalf("Level = %v, want Red (cordon escalation)", board.Level)
	}
}

func TestComputeAndon_CordonNoDowngrade(t *testing.T) {
	health := map[string]telemetry.WorkstreamHealth{
		"w1": {Workstream: "w1", Level: telemetry.Black},
	}
	cordons := []Cordon{{Scope: []string{"auth"}, Reason: "broken"}}
	board := ComputeAndon(health, cordons)
	if board.Level != telemetry.Black {
		t.Fatalf("Level = %v, want Black (cordon should not downgrade)", board.Level)
	}
}

func TestComputeAndon_Empty(t *testing.T) {
	board := ComputeAndon(nil, nil)
	if board.Level != telemetry.Green {
		t.Fatalf("Level = %v, want Green (empty = healthy)", board.Level)
	}
}

func TestComputeAndon_Workstreams(t *testing.T) {
	now := time.Now()
	health := map[string]telemetry.WorkstreamHealth{
		"w1": {
			Workstream: "w1",
			Level:      telemetry.Yellow,
			Latest: telemetry.Signal{
				Workstream: "w1",
				Level:      telemetry.Yellow,
				Timestamp:  now,
			},
		},
	}
	board := ComputeAndon(health, nil)
	if len(board.Workstreams) != 1 {
		t.Fatalf("Workstreams = %d, want 1", len(board.Workstreams))
	}
	if board.Workstreams["w1"].Level != telemetry.Yellow {
		t.Fatalf("w1 level = %v, want Yellow", board.Workstreams["w1"].Level)
	}
}
