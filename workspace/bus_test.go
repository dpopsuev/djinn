package workspace

import (
	"sync/atomic"
	"testing"

	"github.com/dpopsuev/djinn/djinnlog"
)

func TestBus_EmitCallsHandlers(t *testing.T) {
	bus := NewBus(djinnlog.Nop())
	var called int
	bus.On("test1", func(Event) { called++ })
	bus.On("test2", func(Event) { called++ })

	bus.Emit(Event{Type: EventSwitch, New: &Workspace{Name: "ws"}})

	if called != 2 {
		t.Fatalf("called = %d, want 2", called)
	}
}

func TestBus_EmitPassesEvent(t *testing.T) {
	bus := NewBus(djinnlog.Nop())
	var received Event
	bus.On("capture", func(evt Event) { received = evt })

	ws := &Workspace{Name: "test"}
	bus.Emit(Event{Type: EventSwitch, New: ws})

	if received.New.Name != "test" {
		t.Fatalf("received workspace = %q", received.New.Name)
	}
	if received.Type != EventSwitch {
		t.Fatalf("type = %v", received.Type)
	}
}

func TestBus_PanicRecovery(t *testing.T) {
	bus := NewBus(djinnlog.Nop())
	var secondCalled bool

	bus.On("panicker", func(Event) { panic("boom") })
	bus.On("survivor", func(Event) { secondCalled = true })

	// Should not panic
	bus.Emit(Event{Type: EventSwitch, New: &Workspace{Name: "ws"}})

	if !secondCalled {
		t.Fatal("second handler should run despite first panicking")
	}
}

func TestBus_NoHandlers(t *testing.T) {
	bus := NewBus(djinnlog.Nop())
	// Should not panic
	bus.Emit(Event{Type: EventSwitch, New: &Workspace{Name: "ws"}})
}

func TestBus_HandlerCount(t *testing.T) {
	bus := NewBus(djinnlog.Nop())
	if bus.HandlerCount() != 0 {
		t.Fatal("should start empty")
	}
	bus.On("a", func(Event) {})
	bus.On("b", func(Event) {})
	if bus.HandlerCount() != 2 {
		t.Fatalf("count = %d", bus.HandlerCount())
	}
}

func TestBus_ConcurrentEmit(t *testing.T) {
	bus := NewBus(djinnlog.Nop())
	var count atomic.Int64
	bus.On("counter", func(Event) { count.Add(1) })

	done := make(chan struct{})
	for range 10 {
		go func() {
			bus.Emit(Event{Type: EventSwitch, New: &Workspace{Name: "ws"}})
			done <- struct{}{}
		}()
	}
	for range 10 {
		<-done
	}

	if count.Load() != 10 {
		t.Fatalf("count = %d, want 10", count.Load())
	}
}

func TestEventType_String(t *testing.T) {
	if EventSwitch.String() != "switch" {
		t.Fatalf("EventSwitch = %q", EventSwitch.String())
	}
	if EventRepoAdd.String() != "repo.add" {
		t.Fatalf("EventRepoAdd = %q", EventRepoAdd.String())
	}
}
