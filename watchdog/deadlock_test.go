package watchdog

import (
	"context"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/signal"
)

func TestDeadlockWatchdog_DetectsTimeout(t *testing.T) {
	bus := signal.NewSignalBus()
	w := NewDeadlockWatchdog(bus, "ws-1", 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for timeout + a check cycle
	time.Sleep(100 * time.Millisecond)

	if !w.Detected() {
		t.Fatal("expected deadlock detection after timeout")
	}

	signals := bus.Signals()
	hasDetection := false
	for _, s := range signals {
		if s.Source == deadlockWatchdogName && s.Level == signal.Red {
			hasDetection = true
		}
	}
	if !hasDetection {
		t.Fatal("expected Red signal from deadlock watchdog")
	}
}

func TestDeadlockWatchdog_ResetBySignal(t *testing.T) {
	bus := signal.NewSignalBus()
	w := NewDeadlockWatchdog(bus, "ws-1", 80*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Keep sending signals to prevent timeout
	for range 3 {
		time.Sleep(30 * time.Millisecond)
		bus.Emit(signal.Signal{Workstream: "ws-1", Level: signal.Green})
	}

	if w.Detected() {
		t.Fatal("signals should prevent deadlock detection")
	}
}

func TestDeadlockWatchdog_IgnoresOtherWorkstreams(t *testing.T) {
	bus := signal.NewSignalBus()
	w := NewDeadlockWatchdog(bus, "ws-1", 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Emit signals for a different workstream — should NOT reset timer
	for range 3 {
		time.Sleep(20 * time.Millisecond)
		bus.Emit(signal.Signal{Workstream: "ws-2", Level: signal.Green})
	}

	time.Sleep(40 * time.Millisecond)

	if !w.Detected() {
		t.Fatal("signals for other workstream should not prevent detection")
	}
}
