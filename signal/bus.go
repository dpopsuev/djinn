package signal

import (
	"sync"
	"time"
)

// Handler is a callback for signal events.
type Handler func(Signal)

// SignalBus collects signals and notifies subscribers.
type SignalBus struct {
	mu       sync.RWMutex
	signals  []Signal
	handlers []Handler
}

// NewSignalBus creates a new signal bus.
func NewSignalBus() *SignalBus {
	return &SignalBus{}
}

// Emit records a signal and notifies all handlers.
func (b *SignalBus) Emit(s Signal) {
	if s.Timestamp.IsZero() {
		s.Timestamp = time.Now()
	}
	b.mu.Lock()
	b.signals = append(b.signals, s)
	handlers := make([]Handler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.Unlock()

	for _, h := range handlers {
		h(s)
	}
}

// OnSignal registers a handler to be called on every emitted signal.
func (b *SignalBus) OnSignal(h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

// Signals returns a copy of all recorded signals.
func (b *SignalBus) Signals() []Signal {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Signal, len(b.signals))
	copy(out, b.signals)
	return out
}

// Health computes the current workstream health from all recorded signals.
func (b *SignalBus) Health() map[string]WorkstreamHealth {
	return ComputeHealth(b.Signals())
}

// Since returns all signals recorded after the given time.
func (b *SignalBus) Since(t time.Time) []Signal {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var out []Signal
	for _, s := range b.signals {
		if s.Timestamp.After(t) {
			out = append(out, s)
		}
	}
	return out
}
