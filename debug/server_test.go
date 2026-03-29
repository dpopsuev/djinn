package debug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/trace"
)

func seedRing() *trace.Ring {
	r := trace.NewRing(100)
	parentID := r.Append(trace.TraceEvent{
		Component: trace.ComponentAgent,
		Action:    "turn",
		Detail:    "turn 1/5",
	})
	r.Append(trace.TraceEvent{
		ParentID:  parentID,
		Component: trace.ComponentMCP,
		Action:    "call",
		Server:    "scribe",
		Tool:      "artifact.list",
		Detail:    "artifact.list on scribe",
	})
	r.Append(trace.TraceEvent{
		ParentID:  parentID,
		Component: trace.ComponentMCP,
		Action:    "result",
		Server:    "scribe",
		Tool:      "artifact.list",
		Latency:   42 * time.Millisecond,
	})
	r.Append(trace.TraceEvent{
		ParentID:  parentID,
		Component: trace.ComponentMCP,
		Action:    "call",
		Server:    "locus",
		Tool:      "codograph.scan",
	})
	r.Append(trace.TraceEvent{
		ParentID:  parentID,
		Component: trace.ComponentMCP,
		Action:    "result",
		Server:    "locus",
		Tool:      "codograph.scan",
		Latency:   1200 * time.Millisecond,
		Error:     true,
	})
	r.Append(trace.TraceEvent{
		Component: trace.ComponentSignal,
		Action:    "emit",
		Detail:    "budget yellow from budget-watchdog",
	})
	return r
}

func TestHandleStats(t *testing.T) {
	s := NewServer(seedRing())
	out, err := s.Handle(TraceInput{Action: "stats"})
	if err != nil {
		t.Fatal(err)
	}

	var stats trace.RingStats
	if err := json.Unmarshal([]byte(out), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.Count != 6 {
		t.Errorf("count = %d, want 6", stats.Count)
	}
}

func TestHandleList(t *testing.T) {
	s := NewServer(seedRing())
	out, err := s.Handle(TraceInput{Action: "list", Limit: 3})
	if err != nil {
		t.Fatal(err)
	}

	var events []trace.TraceEvent
	if err := json.Unmarshal([]byte(out), &events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestHandleListByComponent(t *testing.T) {
	s := NewServer(seedRing())
	out, err := s.Handle(TraceInput{Action: "list", Component: "signal"})
	if err != nil {
		t.Fatal(err)
	}

	var events []trace.TraceEvent
	if err := json.Unmarshal([]byte(out), &events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 signal event, got %d", len(events))
	}
}

func TestHandleGet(t *testing.T) {
	s := NewServer(seedRing())
	out, err := s.Handle(TraceInput{Action: "get", ID: "trace-1"})
	if err != nil {
		t.Fatal(err)
	}

	var event trace.TraceEvent
	if err := json.Unmarshal([]byte(out), &event); err != nil {
		t.Fatal(err)
	}
	if event.Action != "turn" {
		t.Errorf("action = %q, want turn", event.Action)
	}
}

func TestHandleGetNotFound(t *testing.T) {
	s := NewServer(seedRing())
	_, err := s.Handle(TraceInput{Action: "get", ID: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent event")
	}
}

func TestHandleTree(t *testing.T) {
	s := NewServer(seedRing())
	out, err := s.Handle(TraceInput{Action: "tree", ParentID: "trace-1"})
	if err != nil {
		t.Fatal(err)
	}

	var tree struct {
		Root     *trace.TraceEvent  `json:"root"`
		Children []trace.TraceEvent `json:"children"`
	}
	if err := json.Unmarshal([]byte(out), &tree); err != nil {
		t.Fatal(err)
	}
	if tree.Root == nil {
		t.Fatal("root should be present")
	}
	// 4 child events (2 MCP calls + 2 MCP results).
	if len(tree.Children) != 4 {
		t.Errorf("expected 4 children, got %d", len(tree.Children))
	}
}

func TestHandleHealth(t *testing.T) {
	s := NewServer(seedRing())
	out, err := s.Handle(TraceInput{Action: "health"})
	if err != nil {
		t.Fatal(err)
	}

	var health []struct {
		Server string  `json:"server"`
		Calls  int     `json:"calls"`
		Errors int     `json:"errors"`
		AvgMs  float64 `json:"avg_ms"`
	}
	if err := json.Unmarshal([]byte(out), &health); err != nil {
		t.Fatal(err)
	}
	if len(health) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(health))
	}
	// Find locus — should have 1 error.
	for _, h := range health {
		if h.Server == "locus" {
			if h.Errors != 1 {
				t.Errorf("locus errors = %d, want 1", h.Errors)
			}
		}
	}
}

func TestHandleUnknownAction(t *testing.T) {
	s := NewServer(seedRing())
	_, err := s.Handle(TraceInput{Action: "bogus"})
	if err == nil {
		t.Error("expected error for unknown action")
	}
}
