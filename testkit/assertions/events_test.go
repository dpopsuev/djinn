package assertions

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/broker"
)

func TestAssertEventOrder_Pass(t *testing.T) {
	events := []broker.Event{
		{Kind: broker.StageStarted},
		{Kind: broker.GatePassed},
		{Kind: broker.StageCompleted},
		{Kind: broker.ExecutionDone},
	}
	AssertEventOrder(t, events, []broker.EventKind{
		broker.StageStarted,
		broker.StageCompleted,
		broker.ExecutionDone,
	})
}

func TestAssertNoEvent_Pass(t *testing.T) {
	events := []broker.Event{
		{Kind: broker.StageStarted},
		{Kind: broker.StageCompleted},
	}
	AssertNoEvent(t, events, broker.StageFailed)
}

func TestCollectEvents(t *testing.T) {
	ch := make(chan broker.Event, 3)
	ch <- broker.Event{Kind: broker.StageStarted}
	ch <- broker.Event{Kind: broker.StageCompleted}
	close(ch)

	events := CollectEvents(ch, time.Second)
	if len(events) != 2 {
		t.Fatalf("CollectEvents = %d, want 2", len(events))
	}
}

func TestCollectEvents_Timeout(t *testing.T) {
	ch := make(chan broker.Event)
	events := CollectEvents(ch, 50*time.Millisecond)
	if len(events) != 0 {
		t.Fatalf("CollectEvents on timeout = %d, want 0", len(events))
	}
}

func TestWaitForEvent_Found(t *testing.T) {
	events := []broker.Event{
		{Kind: broker.StageStarted, ExecID: "e1"},
		{Kind: broker.ExecutionDone, ExecID: "e1"},
	}
	e := WaitForEvent(t, func() []broker.Event { return events }, broker.ExecutionDone, time.Second)
	if e.ExecID != "e1" {
		t.Fatalf("ExecID = %q, want %q", e.ExecID, "e1")
	}
}
