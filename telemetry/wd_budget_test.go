package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestBudgetWatchdog_Warning(t *testing.T) {
	bus := NewSignalBus()
	w := NewBudgetWatchdog(bus, 10) // 10 token limit

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Emit 8 budget signals (80% threshold)
	for i := range 8 {
		bus.Emit(Signal{
			Workstream: "ws-1",
			Category:   CategoryBudget,
			Level:      Green,
			Message:    "token",
			Timestamp:  time.Now().Add(time.Duration(i) * time.Millisecond),
		})
	}

	// Check for warning signal
	signals := bus.Signals()
	hasWarning := false
	for _, s := range signals {
		if s.Source == budgetWatchdogName && s.Level == Yellow {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Fatal("expected Yellow warning at 80% budget")
	}
}

func TestBudgetWatchdog_Exceeded(t *testing.T) {
	bus := NewSignalBus()
	w := NewBudgetWatchdog(bus, 5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	for i := range 5 {
		bus.Emit(Signal{
			Workstream: "ws-1",
			Category:   CategoryBudget,
			Level:      Green,
			Timestamp:  time.Now().Add(time.Duration(i) * time.Millisecond),
		})
	}

	if !w.Exceeded() {
		t.Fatal("expected Exceeded() = true")
	}

	signals := bus.Signals()
	hasRed := false
	for _, s := range signals {
		if s.Source == budgetWatchdogName && s.Level == Red {
			hasRed = true
		}
	}
	if !hasRed {
		t.Fatal("expected Red signal at 100% budget")
	}
}

func TestBudgetWatchdog_IgnoresNonBudgetSignals(t *testing.T) {
	bus := NewSignalBus()
	w := NewBudgetWatchdog(bus, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)

	// Emit non-budget signals
	for range 10 {
		bus.Emit(Signal{
			Workstream: "ws-1",
			Category:   CategoryLifecycle,
			Level:      Green,
		})
	}

	if w.Exceeded() {
		t.Fatal("non-budget signals should not count")
	}
}
