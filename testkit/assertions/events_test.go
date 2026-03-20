package assertions

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/orchestrator"
)

func TestAssertEventOrder_Pass(t *testing.T) {
	events := []orchestrator.Event{
		{Kind: orchestrator.StageStarted},
		{Kind: orchestrator.GatePassed},
		{Kind: orchestrator.StageCompleted},
		{Kind: orchestrator.ExecutionDone},
	}
	AssertEventOrder(t, events, []orchestrator.EventKind{
		orchestrator.StageStarted,
		orchestrator.StageCompleted,
		orchestrator.ExecutionDone,
	})
}

func TestAssertNoEvent_Pass(t *testing.T) {
	events := []orchestrator.Event{
		{Kind: orchestrator.StageStarted},
		{Kind: orchestrator.StageCompleted},
	}
	AssertNoEvent(t, events, orchestrator.StageFailed)
}

func TestCollectEvents(t *testing.T) {
	ch := make(chan orchestrator.Event, 3)
	ch <- orchestrator.Event{Kind: orchestrator.StageStarted}
	ch <- orchestrator.Event{Kind: orchestrator.StageCompleted}
	close(ch)

	events := CollectEvents(ch, time.Second)
	if len(events) != 2 {
		t.Fatalf("CollectEvents = %d, want 2", len(events))
	}
}

func TestCollectEvents_Timeout(t *testing.T) {
	ch := make(chan orchestrator.Event)
	events := CollectEvents(ch, 50*time.Millisecond)
	if len(events) != 0 {
		t.Fatalf("CollectEvents on timeout = %d, want 0", len(events))
	}
}

func TestWaitForEvent_Found(t *testing.T) {
	events := []orchestrator.Event{
		{Kind: orchestrator.StageStarted, ExecID: "e1"},
		{Kind: orchestrator.ExecutionDone, ExecID: "e1"},
	}
	e := WaitForEvent(t, func() []orchestrator.Event { return events }, orchestrator.ExecutionDone, time.Second)
	if e.ExecID != "e1" {
		t.Fatalf("ExecID = %q, want %q", e.ExecID, "e1")
	}
}
