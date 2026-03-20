package signal

import (
	"sync"
	"testing"
	"time"
)

func TestSignalBus_EmitAndSubscribe(t *testing.T) {
	bus := NewSignalBus()

	var received []Signal
	bus.OnSignal(func(s Signal) {
		received = append(received, s)
	})

	bus.Emit(Signal{Workstream: "w1", Level: Green, Message: "ok"})
	bus.Emit(Signal{Workstream: "w2", Level: Red, Message: "fail"})

	if len(received) != 2 {
		t.Fatalf("received %d signals, want 2", len(received))
	}
	if received[0].Workstream != "w1" {
		t.Fatalf("first signal workstream = %q, want %q", received[0].Workstream, "w1")
	}
	if received[1].Level != Red {
		t.Fatalf("second signal level = %v, want Red", received[1].Level)
	}
}

func TestSignalBus_Signals(t *testing.T) {
	bus := NewSignalBus()
	bus.Emit(Signal{Workstream: "w1", Level: Green})
	bus.Emit(Signal{Workstream: "w2", Level: Yellow})

	all := bus.Signals()
	if len(all) != 2 {
		t.Fatalf("Signals() returned %d, want 2", len(all))
	}
}

func TestSignalBus_Since(t *testing.T) {
	bus := NewSignalBus()
	t1 := time.Now()
	bus.Emit(Signal{Workstream: "w1", Level: Green, Timestamp: t1})
	time.Sleep(time.Millisecond)
	t2 := time.Now()
	bus.Emit(Signal{Workstream: "w2", Level: Red, Timestamp: t2})

	after := bus.Since(t1)
	if len(after) != 1 {
		t.Fatalf("Since() returned %d, want 1", len(after))
	}
	if after[0].Workstream != "w2" {
		t.Fatalf("Since() workstream = %q, want %q", after[0].Workstream, "w2")
	}
}

func TestSignalBus_TimestampAutoFill(t *testing.T) {
	bus := NewSignalBus()
	before := time.Now()
	bus.Emit(Signal{Workstream: "w1", Level: Green})
	after := time.Now()

	all := bus.Signals()
	if all[0].Timestamp.Before(before) || all[0].Timestamp.After(after) {
		t.Fatalf("auto-filled timestamp %v not between %v and %v", all[0].Timestamp, before, after)
	}
}

func TestSignalBus_ConcurrentSafety(t *testing.T) {
	bus := NewSignalBus()
	var wg sync.WaitGroup

	bus.OnSignal(func(s Signal) {})

	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bus.Emit(Signal{Workstream: "w", Level: Green, Message: "concurrent"})
		}(i)
	}
	wg.Wait()

	if got := len(bus.Signals()); got != 100 {
		t.Fatalf("got %d signals, want 100", got)
	}
}
