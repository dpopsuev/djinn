package broker

import (
	"testing"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/signal"
)

func TestRuntimeAdapter_InterfaceSatisfaction(t *testing.T) {
	var _ ari.Runtime = (*RuntimeAdapter)(nil)
}

func TestRuntimeAdapter_Andon(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	b := NewBroker(BrokerConfig{Bus: bus, Cordons: cordons})

	adapter := NewRuntimeAdapter(b)

	snap := adapter.Andon()
	if snap.Level != "green" {
		t.Fatalf("Level = %q, want %q", snap.Level, "green")
	}
	if snap.Cordons != 0 {
		t.Fatalf("Cordons = %d, want 0", snap.Cordons)
	}

	bus.Emit(signal.Signal{Workstream: "w1", Level: signal.Red})
	snap = adapter.Andon()
	if snap.Level != "red" {
		t.Fatalf("Level = %q, want %q", snap.Level, "red")
	}
}

func TestRuntimeAdapter_ListWorkstreams(t *testing.T) {
	bus := signal.NewSignalBus()
	b := NewBroker(BrokerConfig{Bus: bus, Cordons: NewCordonRegistry()})

	b.Workstreams().Register(&WorkstreamInfo{
		ID:       "ws-1",
		IntentID: "int-1",
		Action:   "fix",
		Status:   WorkstreamRunning,
		Health:   signal.Green,
	})

	adapter := NewRuntimeAdapter(b)
	ws := adapter.ListWorkstreams()
	if len(ws) != 1 {
		t.Fatalf("workstreams = %d, want 1", len(ws))
	}
	if ws[0].ID != "ws-1" {
		t.Fatalf("ID = %q, want %q", ws[0].ID, "ws-1")
	}
	if ws[0].Status != string(WorkstreamRunning) {
		t.Fatalf("Status = %q, want %q", ws[0].Status, WorkstreamRunning)
	}
}

func TestRuntimeAdapter_ClearCordon(t *testing.T) {
	cordons := NewCordonRegistry()
	cordons.Set([]string{"auth"}, "broken", "agent-1")

	b := NewBroker(BrokerConfig{Bus: signal.NewSignalBus(), Cordons: cordons})
	adapter := NewRuntimeAdapter(b)

	adapter.ClearCordon([]string{"auth"})

	if len(cordons.Active()) != 0 {
		t.Fatal("expected 0 active cordons after clear")
	}
}
