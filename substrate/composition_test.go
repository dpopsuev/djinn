package substrate

import (
	"testing"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/troupe/director"
	"github.com/dpopsuev/troupe/signal"
)

// Contract: Substrate.New composes Director + Recorder + TraceProjection
// through a single functional options call. GOL-162, TSK-1078.

func TestSubstrate_WithDirector(t *testing.T) {
	dir := NewLocalDirector(DefaultScheduler())
	sub := New(t.TempDir(), WithDirector(dir))

	if sub.Director() == nil {
		t.Fatal("Director() should not be nil after WithDirector")
	}

	// Director satisfies troupe interface.
	var _ director.Director = sub.Director()
}

func TestSubstrate_WithToolRecorder(t *testing.T) {
	log := signal.NewMemLog()
	rec := NewToolEventRecorder(log, nil)
	sub := New(t.TempDir(), WithToolRecorder(rec))

	if sub.ToolRecorder() == nil {
		t.Fatal("ToolRecorder() should not be nil after WithToolRecorder")
	}
}

func TestSubstrate_WithTraceProjection(t *testing.T) {
	log := signal.NewMemLog()
	ring := telemetry.NewTraceProjection(100).WithEventLog(log)
	ring.SetTraceID("test-trace")

	sub := New(t.TempDir(), WithTraceProjection(ring))

	if sub.TraceProjection() == nil {
		t.Fatal("TraceProjection() should not be nil after WithTraceProjection")
	}
	if sub.TraceProjection().TraceID() != "test-trace" {
		t.Errorf("TraceID = %q, want test-trace", sub.TraceProjection().TraceID())
	}
}

func TestSubstrate_CompositionRoot_AllWired(t *testing.T) {
	// Single New() call wires everything.
	eventLog := signal.NewMemLog()
	ring := telemetry.NewTraceProjection(100).WithEventLog(eventLog)
	ring.SetTraceID("intent-123")
	dir := NewLocalDirector(DefaultScheduler())
	rec := NewToolEventRecorder(eventLog, ring.TraceID)

	sub := New(t.TempDir(),
		WithEventLog(eventLog),
		WithDirector(dir),
		WithTraceProjection(ring),
		WithToolRecorder(rec),
	)

	// All components accessible.
	if sub.EventLog() == nil {
		t.Error("EventLog nil")
	}
	if sub.Director() == nil {
		t.Error("Director nil")
	}
	if sub.TraceProjection() == nil {
		t.Error("TraceProjection nil")
	}
	if sub.ToolRecorder() == nil {
		t.Error("ToolRecorder nil")
	}

	// TraceID propagates through the ring.
	if sub.TraceProjection().TraceID() != "intent-123" {
		t.Errorf("TraceID = %q, want intent-123", sub.TraceProjection().TraceID())
	}

	// EventLog is the shared bus.
	if sub.EventLog() != eventLog {
		t.Error("EventLog should be the same instance passed to New")
	}
}

func TestSubstrate_Defaults_NilDirector(t *testing.T) {
	// Without WithDirector, Director() returns nil. Not an error — REPL
	// creates the director in its own composition root.
	sub := New(t.TempDir(), DefaultServices()...)

	if sub.Director() != nil {
		t.Error("Director should be nil without WithDirector")
	}
	if sub.TraceProjection() != nil {
		t.Error("TraceProjection should be nil without WithTraceProjection")
	}
	if sub.ToolRecorder() != nil {
		t.Error("ToolRecorder should be nil without WithToolRecorder")
	}
}
