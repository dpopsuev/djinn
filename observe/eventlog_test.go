package observe

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/troupe/signal"
)

func seedEventLog() signal.EventLog {
	log := signal.NewMemLog()
	log.Emit(signal.Event{
		Source: "tool",
		Kind:   "trace.call",
		Data: telemetry.TraceEvent{
			Action: "call",
			Detail: "Read main.go",
		},
	})
	log.Emit(signal.Event{
		Source: "tool",
		Kind:   "trace.call_done",
		Data: telemetry.TraceEvent{
			Action:  "call_done",
			Detail:  "Read main.go done",
			Latency: 42 * time.Millisecond,
		},
	})
	log.Emit(signal.Event{
		Source: "agent",
		Kind:   "trace.turn",
		Data: telemetry.TraceEvent{
			Action: "turn",
			Detail: "turn 1/5",
		},
	})
	log.Emit(signal.Event{
		Source: "mcp",
		Kind:   "trace.call_done",
		Data: telemetry.TraceEvent{
			Action: "call_done",
			Detail: "scribe artifact.list",
			Error:  true,
		},
	})
	return log
}

func TestEventLogObserver_Trace_All(t *testing.T) {
	obs := NewEventLogObserver(seedEventLog())
	lines, err := obs.Trace(TraceOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
}

func TestEventLogObserver_Trace_FilterByKind(t *testing.T) {
	obs := NewEventLogObserver(seedEventLog())
	lines, err := obs.Trace(TraceOpts{Kind: "trace.call_done"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 call_done lines, got %d", len(lines))
	}
}

func TestEventLogObserver_Trace_FilterBySource(t *testing.T) {
	obs := NewEventLogObserver(seedEventLog())
	lines, err := obs.Trace(TraceOpts{Source: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 agent line, got %d", len(lines))
	}
}

func TestEventLogObserver_Trace_Limit(t *testing.T) {
	obs := NewEventLogObserver(seedEventLog())
	lines, err := obs.Trace(TraceOpts{Last: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (last 2), got %d", len(lines))
	}
}

func TestEventLogObserver_Trace_Duration(t *testing.T) {
	obs := NewEventLogObserver(seedEventLog())
	lines, err := obs.Trace(TraceOpts{Kind: "trace.call_done", Source: "tool"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Duration != 42 {
		t.Fatalf("duration = %d, want 42", lines[0].Duration)
	}
}

func TestEventLogObserver_Health(t *testing.T) {
	obs := NewEventLogObserver(seedEventLog())
	report, err := obs.Health()
	if err != nil {
		t.Fatal(err)
	}
	if report.AgentsAlive != 3 {
		t.Fatalf("agents alive = %d, want 3 (tool, agent, mcp)", report.AgentsAlive)
	}
	if report.Errors != 1 {
		t.Fatalf("errors = %d, want 1", report.Errors)
	}
}

func TestEventLogObserver_Health_StuckAgent(t *testing.T) {
	log := signal.NewMemLog()
	log.Emit(signal.Event{
		Source:    "coder-1",
		Kind:      "trace.turn",
		Timestamp: time.Now().Add(-5 * time.Minute), // 5 min ago
	})
	log.Emit(signal.Event{
		Source:    "gensec",
		Kind:      "trace.turn",
		Timestamp: time.Now(), // just now
	})

	obs := NewEventLogObserver(log).WithStuckTimeout(60 * time.Second)
	report, err := obs.Health()
	if err != nil {
		t.Fatal(err)
	}
	if len(report.StuckAgents) != 1 {
		t.Fatalf("stuck agents = %d, want 1", len(report.StuckAgents))
	}
	if report.StuckAgents[0] != "coder-1" {
		t.Fatalf("stuck agent = %q, want coder-1", report.StuckAgents[0])
	}
}

func TestEventLogObserver_Trace_FilterByTraceID(t *testing.T) {
	log := signal.NewMemLog()
	log.Emit(signal.Event{TraceID: "tr-1", Source: "agent", Kind: "turn", Data: "a"})
	log.Emit(signal.Event{TraceID: "tr-2", Source: "tool", Kind: "call", Data: "b"})
	log.Emit(signal.Event{TraceID: "tr-1", Source: "tool", Kind: "call", Data: "c"})
	log.Emit(signal.Event{Source: "mcp", Kind: "call", Data: "no trace"})

	obs := NewEventLogObserver(log)

	// Filter by trace ID.
	lines, err := obs.Trace(TraceOpts{TraceID: "tr-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines for tr-1, got %d", len(lines))
	}
	for _, l := range lines {
		if l.TraceID != "tr-1" {
			t.Errorf("line.TraceID = %q, want tr-1", l.TraceID)
		}
	}

	// No filter returns all.
	all, _ := obs.Trace(TraceOpts{})
	if len(all) != 4 {
		t.Fatalf("expected 4 total lines, got %d", len(all))
	}
}

func TestEventLogObserver_Health_Empty(t *testing.T) {
	obs := NewEventLogObserver(signal.NewMemLog())
	report, err := obs.Health()
	if err != nil {
		t.Fatal(err)
	}
	if report.AgentsAlive != 0 {
		t.Fatalf("agents alive = %d, want 0", report.AgentsAlive)
	}
}
