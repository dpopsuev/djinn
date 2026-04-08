package telemetry

import (
	"context"
	"sync/atomic"
)

const (
	budgetWatchdogName = "budget-watchdog"
	budgetWarningRatio = 0.8
)

// BudgetWatchdog monitors token usage by observing budget-category signals.
// Emits Yellow at 80% of maxTokens, Red at 100%.
type BudgetWatchdog struct {
	maxTokens  int
	bus        *SignalBus
	tokens     atomic.Int64
	warned     atomic.Bool
	exceeded   atomic.Bool
	cancelFunc context.CancelFunc
}

// NewBudgetWatchdog creates a budget watchdog with the given token limit.
func NewBudgetWatchdog(bus *SignalBus, maxTokens int) *BudgetWatchdog {
	return &BudgetWatchdog{
		maxTokens: maxTokens,
		bus:       bus,
	}
}

func (w *BudgetWatchdog) Name() string     { return budgetWatchdogName }
func (w *BudgetWatchdog) Category() string { return CategoryBudget }

func (w *BudgetWatchdog) Start(ctx context.Context) error {
	watchCtx, cancel := context.WithCancel(ctx)
	w.cancelFunc = cancel

	w.bus.OnSignal(func(s Signal) {
		select {
		case <-watchCtx.Done():
			return
		default:
		}

		if s.Category != CategoryBudget {
			return
		}

		current := w.tokens.Add(1)
		threshold := int64(float64(w.maxTokens) * budgetWarningRatio)

		if current >= threshold && !w.warned.Load() {
			w.warned.Store(true)
			w.bus.Emit(Signal{
				Workstream: s.Workstream,
				Level:      Yellow,
				Source:     budgetWatchdogName,
				Category:   CategoryBudget,
				Message:    "budget at 80%",
			})
		}

		if current >= int64(w.maxTokens) && !w.exceeded.Load() {
			w.exceeded.Store(true)
			w.bus.Emit(Signal{
				Workstream: s.Workstream,
				Level:      Red,
				Source:     budgetWatchdogName,
				Category:   CategoryBudget,
				Message:    "budget exceeded",
			})
		}
	})

	return nil
}

func (w *BudgetWatchdog) Stop(ctx context.Context) error {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	return nil
}

// Exceeded reports whether the budget has been exceeded.
func (w *BudgetWatchdog) Exceeded() bool {
	return w.exceeded.Load()
}
