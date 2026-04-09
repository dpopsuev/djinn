package telemetry

import (
	"testing"

	"github.com/dpopsuev/battery/testkit"
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
	if e.Meta["tool"] != "Write" {
		t.Fatalf("Meta[tool] = %q", e.Meta["tool"])
	}
	if e.ID == "" {
		t.Fatal("ID should be set by Ring.Append")
	}
	if e.Timestamp.IsZero() {
		t.Fatal("Timestamp should be set")
	}
}

func TestRing_NilEventLog_NoPanic(t *testing.T) {
	ring := NewRing(100) // no WithEventLog

	ring.Append(TraceEvent{
		Component: ComponentAgent,
		Action:    "turn",
		Detail:    "turn 1",
	})

	// No panic, Ring works as before
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
	if e.Meta["server"] != "locus" {
		t.Fatalf("server = %q", e.Meta["server"])
	}
	if e.Meta["tool"] != "scan" {
		t.Fatalf("tool = %q", e.Meta["tool"])
	}
	if e.Meta["error"] != "true" {
		t.Fatalf("error = %q", e.Meta["error"])
	}
	if e.Meta["path"] != "/tmp" {
		t.Fatalf("path = %q", e.Meta["path"])
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

// Note: Ring is NOT an EventLog — it's a bounded circular buffer that
// BRIDGES to an EventLog. The EventLog contract (RunEventLogContract)
// applies to the EventLog implementation, not to Ring itself.
// Ring's semantics differ: circular eviction, TraceEvent types, no OnEmit.
