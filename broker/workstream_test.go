package broker

import (
	"testing"

	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tier"
)

func TestWorkstreamRegistry_RegisterAndGet(t *testing.T) {
	r := NewWorkstreamRegistry()

	r.Register(&WorkstreamInfo{
		ID:       "ws-1",
		IntentID: "int-1",
		Action:   "fix",
		Status:   WorkstreamRunning,
		Scopes:   []tier.Scope{{Level: tier.Mod, Name: "auth"}},
		Health:   signal.Green,
	})

	ws, ok := r.Get("ws-1")
	if !ok {
		t.Fatal("expected ws-1 to exist")
	}
	if ws.Action != "fix" {
		t.Fatalf("Action = %q, want %q", ws.Action, "fix")
	}
	if ws.Status != WorkstreamRunning {
		t.Fatalf("Status = %q, want %q", ws.Status, WorkstreamRunning)
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Fatal("expected nonexistent to not exist")
	}
}

func TestWorkstreamRegistry_Complete(t *testing.T) {
	r := NewWorkstreamRegistry()
	r.Register(&WorkstreamInfo{ID: "ws-1", Status: WorkstreamRunning})

	r.Complete("ws-1", WorkstreamCompleted)

	ws, _ := r.Get("ws-1")
	if ws.Status != WorkstreamCompleted {
		t.Fatalf("Status = %q, want %q", ws.Status, WorkstreamCompleted)
	}
	if ws.EndedAt.IsZero() {
		t.Fatal("EndedAt should be set after Complete")
	}
}

func TestWorkstreamRegistry_Active(t *testing.T) {
	r := NewWorkstreamRegistry()
	r.Register(&WorkstreamInfo{ID: "ws-1", Status: WorkstreamRunning})
	r.Register(&WorkstreamInfo{ID: "ws-2", Status: WorkstreamRunning})
	r.Register(&WorkstreamInfo{ID: "ws-3", Status: WorkstreamRunning})

	r.Complete("ws-2", WorkstreamCompleted)

	active := r.Active()
	if len(active) != 2 {
		t.Fatalf("Active() = %d, want 2", len(active))
	}
}

func TestWorkstreamRegistry_All(t *testing.T) {
	r := NewWorkstreamRegistry()
	r.Register(&WorkstreamInfo{ID: "ws-1", Status: WorkstreamRunning})
	r.Register(&WorkstreamInfo{ID: "ws-2", Status: WorkstreamRunning})
	r.Complete("ws-2", WorkstreamFailed)

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() = %d, want 2", len(all))
	}
}
