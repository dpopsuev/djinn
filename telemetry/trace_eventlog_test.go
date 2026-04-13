package telemetry

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/dpopsuev/troupe/testkit"
)

func TestRing_BridgesToEventLog(t *testing.T) {
	log := testkit.NewStubEventLog()
	ring := NewTraceProjection(100).WithEventLog(log)

	ring.Append(TraceEvent{
		Component: ComponentTool,
		Action:    "tool_call",
		Detail:    "Write main.go",
		Tool:      "Write",
	})

	events := log.Since(0)
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}

	e := events[0]
	if e.Source != "tool" {
		t.Fatalf("Source = %q, want tool", e.Source)
	}
	if e.Kind != "tool_call" {
		t.Fatalf("Kind = %q, want tool_call", e.Kind)
	}

	// Data is the full TraceEvent — type-assert to access fields
	te, ok := e.Data.(TraceEvent)
	if !ok {
		t.Fatalf("Data is %T, want TraceEvent", e.Data)
	}
	if te.Tool != "Write" {
		t.Fatalf("Tool = %q", te.Tool)
	}
	if e.ID == "" {
		t.Fatal("ID should be set")
	}
	if e.Timestamp.IsZero() {
		t.Fatal("Timestamp should be set")
	}
}

func TestRing_NilEventLog_NoPanic(t *testing.T) {
	ring := NewTraceProjection(100)
	ring.Append(TraceEvent{Component: ComponentAgent, Action: "turn", Detail: "turn 1"})
	if ring.Stats().Count != 1 {
		t.Fatal("Ring should still work without EventLog")
	}
}

func TestRing_EventLogMetadata(t *testing.T) {
	log := testkit.NewStubEventLog()
	ring := NewTraceProjection(100).WithEventLog(log)

	ring.Append(TraceEvent{
		Component: ComponentMCP,
		Action:    "call",
		Server:    "locus",
		Tool:      "scan",
		Error:     true,
		Metadata:  map[string]string{"path": "/tmp"},
	})

	e := log.Since(0)[0]
	te, ok := e.Data.(TraceEvent)
	if !ok {
		t.Fatalf("Data is %T, want TraceEvent", e.Data)
	}
	if te.Server != "locus" {
		t.Fatalf("server = %q", te.Server)
	}
	if te.Tool != "scan" {
		t.Fatalf("tool = %q", te.Tool)
	}
	if !te.Error {
		t.Fatal("error should be true")
	}
	if te.Metadata["path"] != "/tmp" {
		t.Fatalf("path = %q", te.Metadata["path"])
	}
}

func TestRing_EventLogConcurrent(t *testing.T) {
	log := testkit.NewStubEventLog()
	ring := NewTraceProjection(100).WithEventLog(log)

	done := make(chan struct{})
	for range 20 {
		go func() {
			ring.Append(TraceEvent{Component: ComponentAgent, Action: "turn"})
			done <- struct{}{}
		}()
	}
	for range 20 {
		<-done
	}

	if log.Len() != 20 {
		t.Fatalf("events = %d, want 20", log.Len())
	}
}

func TestRing_TraceID_Propagation(t *testing.T) {
	log := testkit.NewStubEventLog()
	ring := NewTraceProjection(100).WithEventLog(log)

	// Set intent-level trace ID.
	ring.SetTraceID("intent-42")

	ring.Append(TraceEvent{Component: ComponentAgent, Action: "turn", Detail: "first"})
	ring.Append(TraceEvent{Component: ComponentTool, Action: "call", Detail: "second"})

	// Explicit TraceID overrides inherited.
	ring.Append(TraceEvent{TraceID: "other-trace", Component: ComponentMCP, Action: "call", Detail: "third"})

	events := log.Since(0)
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3", len(events))
	}

	// First two inherit from ring.
	if events[0].TraceID != "intent-42" {
		t.Errorf("events[0].TraceID = %q, want intent-42", events[0].TraceID)
	}
	if events[1].TraceID != "intent-42" {
		t.Errorf("events[1].TraceID = %q, want intent-42", events[1].TraceID)
	}
	// Third has explicit override.
	if events[2].TraceID != "other-trace" {
		t.Errorf("events[2].TraceID = %q, want other-trace", events[2].TraceID)
	}

	// TraceEvent Data also carries it.
	te, ok := events[0].Data.(TraceEvent)
	if !ok {
		t.Fatalf("Data is %T, want TraceEvent", events[0].Data)
	}
	if te.TraceID != "intent-42" {
		t.Errorf("TraceEvent.TraceID = %q, want intent-42", te.TraceID)
	}

	// Verify TraceID accessor.
	if ring.TraceID() != "intent-42" {
		t.Errorf("TraceID() = %q, want intent-42", ring.TraceID())
	}

	// Clear trace ID.
	ring.SetTraceID("")
	ring.Append(TraceEvent{Component: ComponentAgent, Action: "idle", Detail: "fourth"})
	events = log.Since(3)
	if len(events) != 1 {
		t.Fatalf("events after clear = %d, want 1", len(events))
	}
	if events[0].TraceID != "" {
		t.Errorf("events[3].TraceID = %q, want empty after clear", events[0].TraceID)
	}
}

func TestRing_TraceSummary_OnClear(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	ring := NewTraceProjection(100).WithLogger(logger)

	ring.SetTraceID("intent-99")
	ring.Append(TraceEvent{Action: "agent.cognitive.think", Detail: "analyzing"})
	ring.Append(TraceEvent{Action: "agent.cognitive.decide", Detail: "Read"})
	ring.Append(TraceEvent{Action: "tool.executed", Detail: "Read main.go"})
	ring.Append(TraceEvent{Action: "agent.cognitive.think", Detail: "reasoning"})
	ring.Append(TraceEvent{Action: "tool.executed", Detail: "Edit main.go"})

	// Clear trace — should log summary.
	ring.SetTraceID("")

	output := buf.String()
	if !strings.Contains(output, "intent completed") {
		t.Errorf("expected 'intent completed' in log, got: %s", output)
	}
	if !strings.Contains(output, "intent-99") {
		t.Errorf("expected intent ID in log, got: %s", output)
	}
	if !strings.Contains(output, "entries=3") {
		t.Errorf("expected entries=3 (cognitive) in log, got: %s", output)
	}
	if !strings.Contains(output, "tool=2") {
		t.Errorf("expected tool=2 in log, got: %s", output)
	}
}

func TestRing_TraceSummary_NoLogWithoutLogger(t *testing.T) {
	ring := NewTraceProjection(100)
	ring.SetTraceID("intent-1")
	ring.Append(TraceEvent{Action: "agent.cognitive.think"})
	ring.SetTraceID("") // no panic without logger
}

// Ring is NOT an EventLog — it's a CQRS projection (bounded read model).
// EventLog is the write side. Ring bridges TO it via WithEventLog.
