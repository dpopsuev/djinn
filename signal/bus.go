package signal

import (
	"sync"
	"time"

	"github.com/dpopsuev/djinn/trace"
)

// Handler is a callback for signal events.
type Handler func(Signal)

// SignalBus collects signals and notifies subscribers.
type SignalBus struct {
	mu       sync.RWMutex
	signals  []Signal
	handlers []Handler
	Tracer   *trace.Tracer // optional: set to enable signal tracing
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

	b.Tracer.Event("emit", s.Category+" "+s.Level.String()+" from "+s.Source)

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

// ForWorkstream returns all signals for a specific workstream.
func (b *SignalBus) ForWorkstream(workstream string) []Signal {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var out []Signal
	for i := range b.signals {
		if b.signals[i].Workstream == workstream {
			out = append(out, b.signals[i])
		}
	}
	return out
}

// Since returns all signals recorded after the given time.
func (b *SignalBus) Since(t time.Time) []Signal {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var out []Signal
	for i := range b.signals {
		if b.signals[i].Timestamp.After(t) {
			out = append(out, b.signals[i])
		}
	}
	return out
}
