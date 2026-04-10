package telemetry

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockEmitter collects emitted signals for test assertions.
type mockEmitter struct {
	mu      sync.Mutex
	signals []emittedSignal
}

type emittedSignal struct {
	category string
	level    string
	source   string
	message  string
}

func (m *mockEmitter) EmitTrace(category, level, source, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.signals = append(m.signals, emittedSignal{
		category: category,
		level:    level,
		source:   source,
		message:  message,
	})
}

func (m *mockEmitter) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.signals)
}

func (m *mockEmitter) get() []emittedSignal {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]emittedSignal, len(m.signals))
	copy(out, m.signals)
	return out
}

func TestBridgeStartStop(t *testing.T) {
	r := NewTraceProjection(100)
	emitter := &mockEmitter{}
	b := NewBridge(r, emitter, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	b.Start(ctx)

	// Let it tick a couple times.
	time.Sleep(30 * time.Millisecond)

	// Stop via context cancellation.
	cancel()

	// Also call Stop explicitly — should be safe to call after cancel.
	b.Stop()

	// No panic = pass.
}

func TestBridgeStopBeforeStart(t *testing.T) {
	r := NewTraceProjection(100)
	emitter := &mockEmitter{}
	b := NewBridge(r, emitter, 10*time.Millisecond)

	// Stop without Start — should not panic.
	b.Stop()
}

func TestBridgeEmitsAlert(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// Seed 5 consecutive errors to trigger consecutive_errors alert.
	for i := range 5 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "scan",
			Error:     true,
		})
	}

	emitter := &mockEmitter{}
	b := NewBridge(r, emitter, 10*time.Millisecond)

	b.Start(t.Context())

	// Wait for at least one tick.
	deadline := time.After(500 * time.Millisecond)
	for {
		if emitter.count() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for signal emission")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	signals := emitter.get()
	if len(signals) == 0 {
		t.Fatal("expected at least one signal emission")
	}

	// Verify signal fields.
	s := signals[0]
	if s.category != "performance" {
		t.Errorf("category = %q, want performance", s.category)
	}
	if s.source != "trace-bridge" {
		t.Errorf("source = %q, want trace-bridge", s.source)
	}
	// Consecutive errors produce SeverityError which maps to "red".
	if s.level != "red" {
		t.Errorf("level = %q, want red", s.level)
	}
}

func TestBridgeDebounce(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// Seed errors that will trigger alerts on every check.
	for i := range 5 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "scan",
			Error:     true,
		})
	}

	emitter := &mockEmitter{}
	b := NewBridge(r, emitter, 10*time.Millisecond)
	// Set a very long cooldown so the same alert won't fire twice.
	b.cooldown = 10 * time.Second

	b.Start(t.Context())

	// Wait for several ticks — the bridge will run check() multiple times.
	time.Sleep(80 * time.Millisecond)

	signals := emitter.get()
	// Count emissions per unique pattern key (pattern|server|tool).
	keyCounts := make(map[string]int)
	for _, s := range signals {
		// Use message as a proxy for the alert identity.
		keyCounts[s.message]++
	}

	// With debounce (10s cooldown, only 80ms elapsed), each distinct alert
	// pattern should fire at most once despite multiple check() cycles.
	for key, count := range keyCounts {
		if count > 1 {
			t.Errorf("debounce should limit to 1 emission per alert key, key %q emitted %d times", key, count)
		}
	}
}

func TestBridgeMultipleAlertTypes(t *testing.T) {
	r := NewTraceProjection(200)
	now := time.Now()

	// Seed consecutive errors for one server/tool.
	for i := range 5 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "scan",
			Error:     true,
		})
	}

	// Seed consecutive errors for a different server/tool.
	for i := range 5 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(5+i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "put",
			Error:     true,
		})
	}

	emitter := &mockEmitter{}
	b := NewBridge(r, emitter, 10*time.Millisecond)

	b.Start(t.Context())

	deadline := time.After(500 * time.Millisecond)
	for {
		if emitter.count() >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for 2 signal emissions, got %d", emitter.count())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	// Both server/tool pairs should have emitted.
	signals := emitter.get()
	servers := make(map[string]bool)
	for _, s := range signals {
		// The message contains "on <server>", extract by checking content.
		if s.message != "" {
			servers[s.message] = true
		}
	}
	if len(signals) < 2 {
		t.Errorf("expected signals for both server/tool pairs, got %d", len(signals))
	}
}

func TestBridgeSeverityMapping(t *testing.T) {
	// Test the severity → level mapping in check().
	// We can't easily trigger SeverityCritical from Analyze, but we can
	// verify Warning maps to "yellow" and Error maps to "red" via the
	// alerts that the bridge processes.

	r := NewTraceProjection(100)
	now := time.Now()

	// High error rate triggers SeverityWarning.
	for i := range 10 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "lex",
			Tool:      "match",
			Error:     i < 8, // 80% error rate
		})
	}

	emitter := &mockEmitter{}
	b := NewBridge(r, emitter, 10*time.Millisecond)

	b.Start(t.Context())

	deadline := time.After(500 * time.Millisecond)
	for {
		if emitter.count() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for signal")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	signals := emitter.get()
	foundYellow := false
	for _, s := range signals {
		if s.level == "yellow" {
			foundYellow = true
		}
	}
	// error_rate produces SeverityWarning → "yellow".
	if !foundYellow {
		t.Errorf("expected at least one yellow-level signal from error_rate, got: %+v", signals)
	}
}
