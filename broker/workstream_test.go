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

func TestWorkstreamRegistry_ConcurrencyLimit(t *testing.T) {
	r := NewWorkstreamRegistry(WithMaxConcurrent(2))

	ws1 := &WorkstreamInfo{ID: "ws-1", IntentID: "int-1", Action: "fix"}
	ws2 := &WorkstreamInfo{ID: "ws-2", IntentID: "int-2", Action: "refactor"}
	ws3 := &WorkstreamInfo{ID: "ws-3", IntentID: "int-3", Action: "add"}

	ok1, _ := r.TryRegister(ws1)
	ok2, _ := r.TryRegister(ws2)
	ok3, pos3 := r.TryRegister(ws3)

	if !ok1 || !ok2 {
		t.Fatal("first two should register")
	}
	if ok3 {
		t.Fatal("third should be queued")
	}
	if pos3 != 1 {
		t.Fatalf("queue position = %d, want 1", pos3)
	}
	if ws3.Status != WorkstreamPending {
		t.Fatalf("queued status = %q, want %q", ws3.Status, WorkstreamPending)
	}
	if r.PendingCount() != 1 {
		t.Fatalf("PendingCount = %d, want 1", r.PendingCount())
	}
}

func TestWorkstreamRegistry_Dequeue(t *testing.T) {
	r := NewWorkstreamRegistry(WithMaxConcurrent(1))

	ws1 := &WorkstreamInfo{ID: "ws-1", IntentID: "int-1", Action: "fix"}
	ws2 := &WorkstreamInfo{ID: "ws-2", IntentID: "int-2", Action: "add"}

	r.TryRegister(ws1)
	r.TryRegister(ws2) // queued

	r.Complete("ws-1", WorkstreamCompleted)

	next := r.Dequeue()
	if next == nil {
		t.Fatal("expected dequeued workstream")
	}
	if next.ID != "ws-2" {
		t.Fatalf("dequeued ID = %q, want %q", next.ID, "ws-2")
	}
	if next.Status != WorkstreamRunning {
		t.Fatalf("dequeued status = %q, want %q", next.Status, WorkstreamRunning)
	}
	if r.PendingCount() != 0 {
		t.Fatalf("PendingCount after dequeue = %d, want 0", r.PendingCount())
	}
}

func TestWorkstreamRegistry_DequeueEmpty(t *testing.T) {
	r := NewWorkstreamRegistry()
	if r.Dequeue() != nil {
		t.Fatal("dequeue from empty should return nil")
	}
}

func TestWorkstreamRegistry_FindByScope(t *testing.T) {
	r := NewWorkstreamRegistry()
	r.Register(&WorkstreamInfo{
		ID:     "ws-1",
		Status: WorkstreamRunning,
		Scopes: []tier.Scope{{Level: tier.Mod, Name: "auth"}},
	})

	ws, found := r.FindByScope([]tier.Scope{{Level: tier.Mod, Name: "auth"}})
	if !found {
		t.Fatal("should find by matching scope")
	}
	if ws.ID != "ws-1" {
		t.Fatalf("found ID = %q, want %q", ws.ID, "ws-1")
	}

	_, found = r.FindByScope([]tier.Scope{{Level: tier.Mod, Name: "billing"}})
	if found {
		t.Fatal("should not find non-matching scope")
	}
}

func TestWorkstreamRegistry_FindByScope_IgnoresCompleted(t *testing.T) {
	r := NewWorkstreamRegistry()
	r.Register(&WorkstreamInfo{
		ID:     "ws-1",
		Status: WorkstreamRunning,
		Scopes: []tier.Scope{{Level: tier.Mod, Name: "auth"}},
	})
	r.Complete("ws-1", WorkstreamCompleted)

	_, found := r.FindByScope([]tier.Scope{{Level: tier.Mod, Name: "auth"}})
	if found {
		t.Fatal("should not find completed workstream")
	}
}

func TestWorkstreamRegistry_ActiveCount(t *testing.T) {
	r := NewWorkstreamRegistry()
	r.Register(&WorkstreamInfo{ID: "ws-1", Status: WorkstreamRunning})
	r.Register(&WorkstreamInfo{ID: "ws-2", Status: WorkstreamRunning})

	if r.ActiveCount() != 2 {
		t.Fatalf("ActiveCount = %d, want 2", r.ActiveCount())
	}

	r.Complete("ws-1", WorkstreamCompleted)
	if r.ActiveCount() != 1 {
		t.Fatalf("ActiveCount after complete = %d, want 1", r.ActiveCount())
	}
}

func TestWorkstreamRegistry_PerWorkstreamBus(t *testing.T) {
	r := NewWorkstreamRegistry()
	bus := signal.NewSignalBus()
	r.Register(&WorkstreamInfo{ID: "ws-1", Status: WorkstreamRunning, Bus: bus})

	ws, _ := r.Get("ws-1")
	if ws.Bus == nil {
		t.Fatal("Bus should be set")
	}

	ws.Bus.Emit(signal.Signal{Workstream: "ws-1", Level: signal.Green})
	if len(ws.Bus.Signals()) != 1 {
		t.Fatalf("per-workstream bus signals = %d, want 1", len(ws.Bus.Signals()))
	}
}
