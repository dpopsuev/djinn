// bridge.go — TraceSignalBridge auto-emits signals from trace health alerts (TSK-487).
//
// Polls the ring on a configurable interval, runs TraceHealthAnalyzer,
// and emits signals on the SignalBus when alerts are detected. Debounces
// to avoid re-alerting for the same pattern within a cooldown window.
package trace

import (
	"context"
	"sync"
	"time"
)

// SignalEmitter is the interface for emitting signals from trace alerts.
// Satisfied by signal.SignalBus — avoids circular import.
type SignalEmitter interface {
	EmitTrace(category, level, source, message string)
}

// Bridge watches the trace ring and auto-emits signals on health alerts.
type Bridge struct {
	ring     *Ring
	emitter  SignalEmitter
	cfg      HealthConfig
	interval time.Duration
	cooldown time.Duration

	mu       sync.Mutex
	lastSeen map[string]time.Time // pattern key → last alert time
	cancel   context.CancelFunc
}

// NewBridge creates a trace-to-signal bridge.
func NewBridge(ring *Ring, emitter SignalEmitter, interval time.Duration) *Bridge {
	return &Bridge{
		ring:     ring,
		emitter:  emitter,
		cfg:      DefaultHealthConfig(),
		interval: interval,
		cooldown: 30 * time.Second, //nolint:mnd // 30s cooldown between same alerts
		lastSeen: make(map[string]time.Time),
	}
}

// Start begins periodic health analysis. Call Stop() to halt.
func (b *Bridge) Start(ctx context.Context) {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.loop(ctx)
}

// Stop halts the bridge.
func (b *Bridge) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
}

func (b *Bridge) loop(ctx context.Context) {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.check()
		}
	}
}

func (b *Bridge) check() {
	alerts := Analyze(b.ring, b.cfg)

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	for i := range alerts {
		a := &alerts[i]
		key := a.Pattern + "|" + a.Server + "|" + a.Tool
		if last, ok := b.lastSeen[key]; ok && now.Sub(last) < b.cooldown {
			continue // debounce
		}
		b.lastSeen[key] = now

		var level string
		switch a.Severity {
		case SeverityError:
			level = "red"
		case SeverityCritical:
			level = "black"
		default:
			level = "yellow"
		}

		b.emitter.EmitTrace("performance", level, "trace-bridge", a.Detail)
	}
}
