package substrate

import (
	"context"
	"testing"

	"github.com/dpopsuev/troupe/signal"
)

func TestToolEventRecorder_EmitsToEventLog(t *testing.T) {
	log := signal.NewMemLog()
	recorder := NewToolEventRecorder(log, func() string { return "tr-42" })

	recorder.Record(context.Background(), "Read", []byte(`{"path":"main.go"}`), "file contents", nil, 100)

	events := log.Since(-1)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Kind != "tool.executed" {
		t.Errorf("Kind = %q, want tool.executed", e.Kind)
	}
	if e.TraceID != "tr-42" {
		t.Errorf("TraceID = %q, want tr-42", e.TraceID)
	}
	if e.Source != "tool" {
		t.Errorf("Source = %q, want tool", e.Source)
	}

	data, ok := e.Data.(toolExecution)
	if !ok {
		t.Fatalf("Data is %T, want toolExecution", e.Data)
	}
	if data.Tool != "Read" {
		t.Errorf("Tool = %q, want Read", data.Tool)
	}
	if data.IsError {
		t.Error("IsError should be false")
	}
}

func TestToolEventRecorder_NilTraceFunc(t *testing.T) {
	log := signal.NewMemLog()
	recorder := NewToolEventRecorder(log, nil)

	recorder.Record(context.Background(), "Write", nil, "", nil, 50)

	events := log.Since(-1)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].TraceID != "" {
		t.Errorf("TraceID = %q, want empty", events[0].TraceID)
	}
}

func TestToolEventRecorder_ErrorFlag(t *testing.T) {
	log := signal.NewMemLog()
	recorder := NewToolEventRecorder(log, nil)

	recorder.Record(context.Background(), "Bash", nil, "", context.Canceled, 10)

	data := log.Since(-1)[0].Data.(toolExecution)
	if !data.IsError {
		t.Error("IsError should be true for errored tool calls")
	}
}
