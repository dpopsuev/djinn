package watchdog

import (
	"context"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/signal"
)

const deadlockWatchdogName = "deadlock-watchdog"

// DeadlockWatchdog detects workstream stalls by monitoring signal activity.
// If no signal arrives within the timeout, emits a Red signal.
type DeadlockWatchdog struct {
	timeout    time.Duration
	bus        *signal.SignalBus
	workstream string

	mu         sync.Mutex
	lastSignal time.Time
	detected   bool
	cancelFunc context.CancelFunc
}

// NewDeadlockWatchdog creates a deadlock watchdog for the given workstream.
func NewDeadlockWatchdog(bus *signal.SignalBus, workstream string, timeout time.Duration) *DeadlockWatchdog {
	return &DeadlockWatchdog{
		timeout:    timeout,
		bus:        bus,
		workstream: workstream,
		lastSignal: time.Now(),
	}
}

func (w *DeadlockWatchdog) Name() string     { return deadlockWatchdogName }
func (w *DeadlockWatchdog) Category() string { return CategoryDeadlock }

func (w *DeadlockWatchdog) Start(ctx context.Context) error {
	watchCtx, cancel := context.WithCancel(ctx)
	w.cancelFunc = cancel

	w.bus.OnSignal(func(s signal.Signal) {
		if s.Workstream == w.workstream {
			w.mu.Lock()
			w.lastSignal = time.Now()
			w.mu.Unlock()
		}
	})

	go w.monitor(watchCtx)
	return nil
}

func (w *DeadlockWatchdog) monitor(ctx context.Context) {
	ticker := time.NewTicker(w.timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			elapsed := time.Since(w.lastSignal)
			alreadyDetected := w.detected
			w.mu.Unlock()

			if elapsed >= w.timeout && !alreadyDetected {
				w.mu.Lock()
				w.detected = true
				w.mu.Unlock()

				w.bus.Emit(signal.Signal{
					Workstream: w.workstream,
					Level:      signal.Red,
					Source:     deadlockWatchdogName,
					Category:   signal.CategoryLifecycle,
					Message:    "deadlock detected: no signals within timeout",
				})
			}
		}
	}
}

func (w *DeadlockWatchdog) Stop(ctx context.Context) error {
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	return nil
}

// Detected reports whether a deadlock has been detected.
func (w *DeadlockWatchdog) Detected() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.detected
}
