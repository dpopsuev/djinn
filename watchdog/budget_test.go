package watchdog

import (
	"context"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/signal"
)

func TestBudgetWatchdog_Warning(t *testing.T) {
	bus := signal.NewSignalBus()
	w := NewBudgetWatchdog(bus, 10) // 10 token limit

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Emit 8 budget signals (80% threshold)
	for i := range 8 {
		bus.Emit(signal.Signal{
			Workstream: "ws-1",
			Category:   signal.CategoryBudget,
			Level:      signal.Green,
			Message:    "token",
			Timestamp:  time.Now().Add(time.Duration(i) * time.Millisecond),
		})
	}

	// Check for warning signal
	signals := bus.Signals()
	hasWarning := false
	for _, s := range signals {
		if s.Source == budgetWatchdogName && s.Level == signal.Yellow {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Fatal("expected Yellow warning at 80% budget")
	}
}

func TestBudgetWatchdog_Exceeded(t *testing.T) {
	bus := signal.NewSignalBus()
	w := NewBudgetWatchdog(bus, 5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	for i := range 5 {
		bus.Emit(signal.Signal{
			Workstream: "ws-1",
			Category:   signal.CategoryBudget,
			Level:      signal.Green,
			Timestamp:  time.Now().Add(time.Duration(i) * time.Millisecond),
		})
	}

	if !w.Exceeded() {
		t.Fatal("expected Exceeded() = true")
	}

	signals := bus.Signals()
	hasRed := false
	for _, s := range signals {
		if s.Source == budgetWatchdogName && s.Level == signal.Red {
			hasRed = true
		}
	}
	if !hasRed {
		t.Fatal("expected Red signal at 100% budget")
	}
}

func TestBudgetWatchdog_IgnoresNonBudgetSignals(t *testing.T) {
	bus := signal.NewSignalBus()
	w := NewBudgetWatchdog(bus, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// Emit non-budget signals
	for range 10 {
		bus.Emit(signal.Signal{
			Workstream: "ws-1",
			Category:   signal.CategoryLifecycle,
			Level:      signal.Green,
		})
	}

	if w.Exceeded() {
		t.Fatal("non-budget signals should not count")
	}
}
