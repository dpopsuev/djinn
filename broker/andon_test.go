package broker

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/signal"
)

func TestComputeAndon_AllGreen(t *testing.T) {
	health := map[string]signal.WorkstreamHealth{
		"w1": {Workstream: "w1", Level: signal.Green},
		"w2": {Workstream: "w2", Level: signal.Green},
	}
	board := ComputeAndon(health, nil)
	if board.Level != signal.Green {
		t.Fatalf("Level = %v, want Green", board.Level)
	}
}

func TestComputeAndon_MixedLevels(t *testing.T) {
	health := map[string]signal.WorkstreamHealth{
		"w1": {Workstream: "w1", Level: signal.Green},
		"w2": {Workstream: "w2", Level: signal.Yellow},
		"w3": {Workstream: "w3", Level: signal.Red},
	}
	board := ComputeAndon(health, nil)
	if board.Level != signal.Red {
		t.Fatalf("Level = %v, want Red (worst-flag-wins)", board.Level)
	}
}

func TestComputeAndon_CordonEscalates(t *testing.T) {
	health := map[string]signal.WorkstreamHealth{
		"w1": {Workstream: "w1", Level: signal.Green},
	}
	cordons := []Cordon{{Scope: []string{"auth"}, Reason: "broken"}}
	board := ComputeAndon(health, cordons)
	if board.Level != signal.Red {
		t.Fatalf("Level = %v, want Red (cordon escalation)", board.Level)
	}
}

func TestComputeAndon_CordonNoDowngrade(t *testing.T) {
	health := map[string]signal.WorkstreamHealth{
		"w1": {Workstream: "w1", Level: signal.Black},
	}
	cordons := []Cordon{{Scope: []string{"auth"}, Reason: "broken"}}
	board := ComputeAndon(health, cordons)
	if board.Level != signal.Black {
		t.Fatalf("Level = %v, want Black (cordon should not downgrade)", board.Level)
	}
}

func TestComputeAndon_Empty(t *testing.T) {
	board := ComputeAndon(nil, nil)
	if board.Level != signal.Green {
		t.Fatalf("Level = %v, want Green (empty = healthy)", board.Level)
	}
}

func TestComputeAndon_Workstreams(t *testing.T) {
	now := time.Now()
	health := map[string]signal.WorkstreamHealth{
		"w1": {
			Workstream: "w1",
			Level:      signal.Yellow,
			Latest: signal.Signal{
				Workstream: "w1",
				Level:      signal.Yellow,
				Timestamp:  now,
			},
		},
	}
	board := ComputeAndon(health, nil)
	if len(board.Workstreams) != 1 {
		t.Fatalf("Workstreams = %d, want 1", len(board.Workstreams))
	}
	if board.Workstreams["w1"].Level != signal.Yellow {
		t.Fatalf("w1 level = %v, want Yellow", board.Workstreams["w1"].Level)
	}
}
