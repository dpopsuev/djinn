//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/substrate"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/uniform/quality"
	"github.com/dpopsuev/troupe/signal"
)

// TestTraceChain_ByTraceID_ReturnsFullStory proves the GOL-162 capstone:
// one ByTraceID call returns every event from intent to result.
//
// Components wired (real, not stubs):
//   - Troupe MemLog (EventLog)
//   - Djinn TraceProjection (ring → EventLog bridge)
//   - MetricsHandler (cognitive events: Think, Decide, GiveUp)
//   - ToolEventRecorder (tool.executed events)
//   - LocalDirector (troupe/director.Director interface)
//
// LLM is not involved — this tests the plumbing, not the model.
func TestTraceChain_ByTraceID_ReturnsFullStory(t *testing.T) {
	// --- Compose ---
	eventLog := signal.NewMemLog()
	ring := telemetry.NewTraceProjection(100).WithEventLog(eventLog)

	// Simulate intent: operator says "fix the auth bug".
	traceID := "intent-fix-auth-42"
	ring.SetTraceID(traceID)

	// MetricsHandler with cognitive events.
	metrics := quality.NewAgentMetrics("gensec", "director")
	police := quality.NewAgentPolice(quality.DefaultCordonConfig())
	bus := telemetry.NewSignalBus()
	handler := substrate.NewMetricsHandler(metrics, police, bus, nil, quality.DefaultCordonConfig())
	handler.WithCognitiveEvents(eventLog, ring.TraceID)

	// ToolEventRecorder bridges tool calls to EventLog.
	recorder := substrate.NewToolEventRecorder(eventLog, ring.TraceID)
	middleware.SetDefaultRecorder(recorder)

	// LocalDirector — Troupe Director interface.
	director := substrate.NewLocalDirector(substrate.DefaultScheduler())

	// --- Act: simulate the agent loop ---

	// 1. Director starts orchestration.
	ch, err := director.Direct(context.Background(), nil)
	if err != nil {
		t.Fatalf("Director.Direct: %v", err)
	}
	// Drain director events (LocalDirector emits Started then closes).
	for e := range ch {
		eventLog.Emit(signal.Event{
			TraceID: traceID,
			Source:  "director",
			Kind:    string(e.Kind),
			Data:    e.Step,
		})
	}

	// 2. Agent thinks about the problem.
	handler.StartTurn()
	handler.OnThinking("analyzing auth module for timeout bug")

	// 3. Agent decides to read a file.
	handler.OnToolCall(driver.ToolCall{Name: "Read", ID: "tc-1"})

	// 4. Tool executes — recorder fires.
	recorder.Record(context.Background(), "Read", []byte(`{"path":"auth/handler.go"}`), "func HandleAuth()...", nil, 50*time.Millisecond)

	// 5. Agent thinks about the fix.
	handler.OnThinking("found the timeout — need to increase from 5s to 30s")

	// 6. Agent decides to write the fix.
	handler.OnToolCall(driver.ToolCall{Name: "Edit", ID: "tc-2"})

	// 7. Tool executes.
	recorder.Record(context.Background(), "Edit", []byte(`{"path":"auth/handler.go"}`), "edited", nil, 30*time.Millisecond)

	// 8. Turn completes.
	handler.OnDone(&driver.Usage{InputTokens: 500, OutputTokens: 200})

	// --- Assert: one ByTraceID call returns the full story ---
	events := eventLog.ByTraceID(traceID)

	if len(events) == 0 {
		t.Fatal("ByTraceID returned no events")
	}

	// Verify event kinds appear in the trace.
	kindSet := make(map[string]int)
	for _, e := range events {
		kindSet[e.Kind]++
		if e.TraceID != traceID {
			t.Errorf("event %q has TraceID %q, want %q", e.Kind, e.TraceID, traceID)
		}
	}

	// Must have: director started, think (x2), decide (x2), tool.executed (x2).
	assertKind(t, kindSet, "started", 1)
	assertKind(t, kindSet, signal.KindThink, 2)
	assertKind(t, kindSet, signal.KindDecide, 2)
	assertKind(t, kindSet, "tool.executed", 2)

	// Verify chronological order.
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Before(events[i-1].Timestamp) {
			t.Errorf("events[%d] timestamp %v before events[%d] %v — not chronological",
				i, events[i].Timestamp, i-1, events[i-1].Timestamp)
		}
	}

	t.Logf("ByTraceID returned %d events for %q — full story verified", len(events), traceID)
}

func assertKind(t *testing.T, kindSet map[string]int, kind string, wantAtLeast int) {
	t.Helper()
	if kindSet[kind] < wantAtLeast {
		t.Errorf("kind %q: got %d, want >= %d", kind, kindSet[kind], wantAtLeast)
	}
}
