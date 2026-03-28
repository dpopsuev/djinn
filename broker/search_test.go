package broker

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tier"
)

func TestBroker_Search_Signals(t *testing.T) {
	bus := signal.NewSignalBus()
	b := NewBroker(&BrokerConfig{Bus: bus, Cordons: NewCordonRegistry()})

	bus.Emit(signal.Signal{Workstream: "ws-1", Level: signal.Red, Message: "auth test failing"})
	bus.Emit(signal.Signal{Workstream: "ws-2", Level: signal.Green, Message: "billing ok"})

	results := b.Search("auth")
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Kind != ResultKindSignal {
		t.Fatalf("kind = %q, want %q", results[0].Kind, ResultKindSignal)
	}
	if results[0].ID != "ws-1" {
		t.Fatalf("ID = %q, want %q", results[0].ID, "ws-1")
	}
}

func TestBroker_Search_Workstreams(t *testing.T) {
	bus := signal.NewSignalBus()
	b := NewBroker(&BrokerConfig{Bus: bus, Cordons: NewCordonRegistry()})

	b.workstreams.Register(&WorkstreamInfo{
		ID:        "ws-fix",
		Action:    "fix auth regression",
		Status:    WorkstreamRunning,
		StartedAt: time.Now(),
	})

	results := b.Search("regression")
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Kind != ResultKindWorkstream {
		t.Fatalf("kind = %q", results[0].Kind)
	}
}

func TestBroker_Search_Cordons(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	b := NewBroker(&BrokerConfig{Bus: bus, Cordons: cordons})

	cordons.Set([]string{"auth/middleware.go"}, "security vulnerability", "agent-1")

	results := b.Search("security")
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Kind != ResultKindCordon {
		t.Fatalf("kind = %q", results[0].Kind)
	}
}

func TestBroker_Search_CrossSubsystem(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	b := NewBroker(&BrokerConfig{Bus: bus, Cordons: cordons})

	bus.Emit(signal.Signal{Workstream: "ws-auth", Level: signal.Red, Message: "auth broken"})
	b.workstreams.Register(&WorkstreamInfo{
		ID:        "ws-auth",
		Action:    "fix auth",
		Status:    WorkstreamRunning,
		Scopes:    []tier.Scope{{Level: tier.Mod, Name: "auth"}},
		StartedAt: time.Now(),
	})
	cordons.Set([]string{"auth/"}, "auth cordoned", "watchdog")

	results := b.Search("auth")
	if len(results) != 3 {
		t.Fatalf("cross-subsystem results = %d, want 3 (signal + workstream + cordon)", len(results))
	}
}

func TestBroker_Search_NoMatch(t *testing.T) {
	bus := signal.NewSignalBus()
	b := NewBroker(&BrokerConfig{Bus: bus, Cordons: NewCordonRegistry()})

	bus.Emit(signal.Signal{Workstream: "ws-1", Message: "all good"})

	results := b.Search("nonexistent")
	if len(results) != 0 {
		t.Fatalf("results = %d, want 0", len(results))
	}
}

func TestBroker_Search_CaseInsensitive(t *testing.T) {
	bus := signal.NewSignalBus()
	b := NewBroker(&BrokerConfig{Bus: bus, Cordons: NewCordonRegistry()})

	bus.Emit(signal.Signal{Workstream: "ws-1", Message: "Auth Module Error"})

	results := b.Search("auth module")
	if len(results) != 1 {
		t.Fatalf("case-insensitive search results = %d, want 1", len(results))
	}
}

func TestBroker_Search_SortedByRecency(t *testing.T) {
	bus := signal.NewSignalBus()
	b := NewBroker(&BrokerConfig{Bus: bus, Cordons: NewCordonRegistry()})

	t1 := time.Now().Add(-2 * time.Hour)
	t2 := time.Now().Add(-1 * time.Hour)
	t3 := time.Now()

	bus.Emit(signal.Signal{Workstream: "ws-old", Message: "error old", Timestamp: t1})
	bus.Emit(signal.Signal{Workstream: "ws-mid", Message: "error mid", Timestamp: t2})
	bus.Emit(signal.Signal{Workstream: "ws-new", Message: "error new", Timestamp: t3})

	results := b.Search("error")
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	if results[0].ID != "ws-new" {
		t.Fatalf("first result = %q, want newest (ws-new)", results[0].ID)
	}
	if results[2].ID != "ws-old" {
		t.Fatalf("last result = %q, want oldest (ws-old)", results[2].ID)
	}
}
