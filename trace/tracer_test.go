package trace

import (
	"testing"
	"time"
)

func TestTracerBeginEnd(t *testing.T) {
	r := NewRing(100)
	tr := r.For(ComponentMCP)

	span := tr.Begin("call", "artifact.list")
	time.Sleep(5 * time.Millisecond)
	span.End()

	events := r.Last(10)
	if len(events) != 2 {
		t.Fatalf("expected 2 events (begin + end), got %d", len(events))
	}
	if events[0].Action != "call" {
		t.Errorf("begin action = %q, want call", events[0].Action)
	}
	if events[1].Action != "call_done" {
		t.Errorf("end action = %q, want call_done", events[1].Action)
	}
	if events[1].Latency < 5*time.Millisecond {
		t.Errorf("latency = %v, expected >= 5ms", events[1].Latency)
	}
	if events[1].ParentID != events[0].ID {
		t.Error("end event should reference begin event as parent")
	}
}

func TestTracerAutoComponent(t *testing.T) {
	r := NewRing(100)
	tr := r.For(ComponentSignal)

	tr.Event("emit", "budget yellow")

	events := r.Last(1)
	if events[0].Component != ComponentSignal {
		t.Errorf("component = %q, want signal", events[0].Component)
	}
}

func TestSpanChild(t *testing.T) {
	r := NewRing(100)
	tr := r.For(ComponentAgent)

	parent := tr.Begin("turn", "turn 1/5")
	child := parent.Child("tool_call", "Read main.go")
	child.End()
	parent.End()

	events := r.Last(10)
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}
	// Child's begin should have parent's ID as ParentID.
	if events[1].ParentID != events[0].ID {
		t.Error("child begin should reference parent begin")
	}
	// Child's end should reference child's begin.
	if events[2].ParentID != events[1].ID {
		t.Error("child end should reference child begin")
	}
}

func TestSpanWithServerTool(t *testing.T) {
	r := NewRing(100)
	tr := r.For(ComponentMCP)

	span := tr.Begin("call", "scanning").WithServer("locus").WithTool("codograph.scan")
	span.End()

	events := r.Last(1)
	if events[0].Server != "locus" {
		t.Errorf("server = %q, want locus", events[0].Server)
	}
	if events[0].Tool != "codograph.scan" {
		t.Errorf("tool = %q, want codograph.scan", events[0].Tool)
	}
}

func TestSpanEndWithError(t *testing.T) {
	r := NewRing(100)
	tr := r.For(ComponentMCP)

	span := tr.Begin("call", "failing")
	span.EndWithError()

	events := r.Last(1)
	if !events[0].Error {
		t.Error("error span should have Error=true")
	}
}

func TestNilTracerSafe(t *testing.T) {
	var tr *Tracer

	// All methods should be safe no-ops.
	span := tr.Begin("call", "nothing")
	span.End()
	span.EndWithError()
	child := span.Child("sub", "nothing")
	child.End()
	tr.Event("emit", "nothing")

	// No panic = pass.
}

func TestTracerEvent(t *testing.T) {
	r := NewRing(100)
	tr := r.For(ComponentTUI)

	tr.Event("render", "frame 42")

	events := r.Last(1)
	if events[0].Action != "render" {
		t.Errorf("action = %q, want render", events[0].Action)
	}
	if events[0].Component != ComponentTUI {
		t.Errorf("component = %q, want tui", events[0].Component)
	}
}
