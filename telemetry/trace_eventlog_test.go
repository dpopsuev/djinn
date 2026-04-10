package telemetry

import (
	"testing"

	"github.com/dpopsuev/troupe/testkit"
)

func TestRing_BridgesToEventLog(t *testing.T) {
	log := testkit.NewStubEventLog()
	ring := NewRing(100).WithEventLog(log)

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
	ring := NewRing(100)
	ring.Append(TraceEvent{Component: ComponentAgent, Action: "turn", Detail: "turn 1"})
	if ring.Stats().Count != 1 {
		t.Fatal("Ring should still work without EventLog")
	}
}

func TestRing_EventLogMetadata(t *testing.T) {
	log := testkit.NewStubEventLog()
	ring := NewRing(100).WithEventLog(log)

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
	ring := NewRing(100).WithEventLog(log)

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

// Ring is NOT an EventLog — it's a CQRS projection (bounded read model).
// EventLog is the write side. Ring bridges TO it via WithEventLog.
