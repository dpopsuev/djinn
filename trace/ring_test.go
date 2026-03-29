package trace

import (
	"testing"
	"time"
)

func TestRingAppendAndLast(t *testing.T) {
	r := NewRing(5)

	for i := range 3 {
		r.Append(TraceEvent{
			Component: ComponentMCP,
			Action:    "call",
			Detail:    string(rune('A' + i)),
		})
	}

	events := r.Last(3)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Detail != "A" {
		t.Errorf("oldest = %q, want A", events[0].Detail)
	}
	if events[2].Detail != "C" {
		t.Errorf("newest = %q, want C", events[2].Detail)
	}
}

func TestRingWrapping(t *testing.T) {
	r := NewRing(3)

	// Add 5 events to a ring of capacity 3.
	for i := range 5 {
		r.Append(TraceEvent{Detail: string(rune('A' + i))})
	}

	events := r.Last(3)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	// Should have C, D, E (oldest 2 evicted).
	if events[0].Detail != "C" {
		t.Errorf("oldest after wrap = %q, want C", events[0].Detail)
	}
	if events[2].Detail != "E" {
		t.Errorf("newest after wrap = %q, want E", events[2].Detail)
	}
}

func TestRingLastMoreThanCount(t *testing.T) {
	r := NewRing(10)
	r.Append(TraceEvent{Detail: "only"})

	events := r.Last(100)
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestRingByParent(t *testing.T) {
	r := NewRing(10)
	parentID := r.Append(TraceEvent{Component: ComponentAgent, Action: "prompt", Detail: "root"})

	r.Append(TraceEvent{ParentID: parentID, Component: ComponentMCP, Action: "call", Detail: "child1"})
	r.Append(TraceEvent{ParentID: parentID, Component: ComponentMCP, Action: "call", Detail: "child2"})
	r.Append(TraceEvent{Component: ComponentSignal, Action: "emit", Detail: "unrelated"})

	children := r.ByParent(parentID)
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
}

func TestRingByComponent(t *testing.T) {
	r := NewRing(10)
	r.Append(TraceEvent{Component: ComponentMCP, Detail: "mcp1"})
	r.Append(TraceEvent{Component: ComponentAgent, Detail: "agent1"})
	r.Append(TraceEvent{Component: ComponentMCP, Detail: "mcp2"})

	mcp := r.ByComponent(ComponentMCP)
	if len(mcp) != 2 {
		t.Fatalf("expected 2 MCP events, got %d", len(mcp))
	}
}

func TestRingGet(t *testing.T) {
	r := NewRing(10)
	id := r.Append(TraceEvent{Detail: "findme"})

	e, ok := r.Get(id)
	if !ok {
		t.Fatal("event not found")
	}
	if e.Detail != "findme" {
		t.Errorf("detail = %q, want findme", e.Detail)
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent event")
	}
}

func TestRingSince(t *testing.T) {
	r := NewRing(10)
	before := time.Now()
	time.Sleep(time.Millisecond)

	r.Append(TraceEvent{Detail: "after"})

	events := r.Since(before)
	if len(events) != 1 {
		t.Fatalf("expected 1 event since, got %d", len(events))
	}
}

func TestRingStats(t *testing.T) {
	r := NewRing(100)
	stats := r.Stats()
	if stats.Count != 0 || stats.Capacity != 100 {
		t.Errorf("empty ring: count=%d cap=%d", stats.Count, stats.Capacity)
	}

	r.Append(TraceEvent{Detail: "first"})
	r.Append(TraceEvent{Detail: "second"})

	stats = r.Stats()
	if stats.Count != 2 {
		t.Errorf("count = %d, want 2", stats.Count)
	}
	if stats.Oldest.IsZero() || stats.Newest.IsZero() {
		t.Error("timestamps should be set")
	}
}

func TestRingIDAssignment(t *testing.T) {
	r := NewRing(10)
	id1 := r.Append(TraceEvent{Detail: "a"})
	id2 := r.Append(TraceEvent{Detail: "b"})

	if id1 == id2 {
		t.Error("IDs should be unique")
	}
	if id1 != "trace-1" {
		t.Errorf("first ID = %q, want trace-1", id1)
	}
}

func TestRingTimestampAutoSet(t *testing.T) {
	r := NewRing(10)
	r.Append(TraceEvent{Detail: "auto"})

	events := r.Last(1)
	if events[0].Timestamp.IsZero() {
		t.Error("timestamp should be auto-set")
	}
}
