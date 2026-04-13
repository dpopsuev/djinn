package substrate

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/hook"
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

func TestSubstrate_WarnsOnMissingIntegration(t *testing.T) {
	// Capture log output to verify ORANGE warnings.
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := slog.New(handler)

	_ = New(t.TempDir(), WithSubstrateLogger(logger))

	output := buf.String()
	if !strings.Contains(output, "no Director configured") {
		t.Error("expected Director warning in log output")
	}
	if !strings.Contains(output, "no ToolRecorder configured") {
		t.Error("expected ToolRecorder warning in log output")
	}
	if !strings.Contains(output, "no TraceProjection configured") {
		t.Error("expected TraceProjection warning in log output")
	}
}

func TestSubstrate_NoWarnings_WhenFullyWired(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := slog.New(handler)

	eventLog := signal.NewMemLog()
	ring := telemetry.NewTraceProjection(100).WithEventLog(eventLog)
	rec := NewToolEventRecorder(eventLog, nil)
	dir := NewLocalDirector(DefaultScheduler())

	_ = New(t.TempDir(),
		WithSubstrateLogger(logger),
		WithEventLog(eventLog),
		WithDirector(dir),
		WithTraceProjection(ring),
		WithToolRecorder(rec),
	)

	output := buf.String()
	if strings.Contains(output, "no Director") || strings.Contains(output, "no ToolRecorder") || strings.Contains(output, "no TraceProjection") {
		t.Errorf("expected no warnings when fully wired, got: %s", output)
	}
}

func TestSubstrate_WithHookDispatcher(t *testing.T) {
	eventLog := signal.NewMemLog()
	dispatcher := hook.New([]hook.Hook{
		{Name: "test", On: hook.PhasePreToolUse, Match: hook.Matcher{Tool: "Bash"}, Action: hook.Action{Deny: "no"}},
	}, eventLog)

	sub := New(t.TempDir(), WithHookDispatcher(dispatcher))

	if sub.HookDispatcher() == nil {
		t.Fatal("HookDispatcher should not be nil")
	}
	if len(sub.HookDispatcher().Hooks()) != 1 {
		t.Fatalf("hooks = %d, want 1", len(sub.HookDispatcher().Hooks()))
	}
}

func TestSubstrate_IntegrationServices_Durable(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "events.jsonl")
	sub := New(t.TempDir(), IntegrationServices(nil, logPath)...)

	// Emit an event — should persist.
	sub.EventLog().Emit(signal.Event{Kind: "test.durable", Source: "test"})

	if sub.EventLog().Len() != 1 {
		t.Fatalf("EventLog.Len = %d, want 1", sub.EventLog().Len())
	}

	// Verify file exists.
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("event log file should exist: %v", err)
	}
}

func TestSubstrate_IntegrationServices(t *testing.T) {
	sub := New(t.TempDir(), IntegrationServices(nil, "")...)

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

	// TraceID propagation works.
	sub.TraceProjection().SetTraceID("int-svc-test")
	if sub.TraceProjection().TraceID() != "int-svc-test" {
		t.Errorf("TraceID = %q, want int-svc-test", sub.TraceProjection().TraceID())
	}
	sub.TraceProjection().SetTraceID("")
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
