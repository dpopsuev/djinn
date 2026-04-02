package tui

import (
	"testing"
	"time"
)

func TestAlertQueue_Push_RingBuffer(t *testing.T) {
	q := NewAlertQueue(3)

	// Push 3 alerts — should all fit.
	for i := 0; i < 3; i++ {
		q.Push(Alert{
			Level:     AndonGreen,
			Source:    "test",
			Message:   "msg",
			Timestamp: time.Now(),
		})
	}
	if got := len(q.Alerts()); got != 3 {
		t.Fatalf("after 3 pushes: len = %d, want 3", got)
	}

	// Push a 4th — oldest should be dropped.
	q.Push(Alert{
		Level:     AndonRed,
		Source:    "overflow",
		Message:   "dropped oldest",
		Timestamp: time.Now(),
	})
	alerts := q.Alerts()
	if len(alerts) != 3 {
		t.Fatalf("after 4 pushes: len = %d, want 3", len(alerts))
	}
	// The newest (4th) should be at the end.
	if alerts[2].Source != "overflow" {
		t.Fatalf("newest alert source = %q, want 'overflow'", alerts[2].Source)
	}
	// The first alert (index 0) should have been dropped; index 0 is now what was index 1.
	if alerts[0].Source != "test" {
		t.Fatalf("oldest alert should still be 'test', got %q", alerts[0].Source)
	}
}

func TestAlertQueue_Push_DefaultCapacity(t *testing.T) {
	q := NewAlertQueue(0)
	// Verify default capacity of 50.
	for i := 0; i < 55; i++ {
		q.Push(Alert{Level: AndonGreen, Source: "bulk"})
	}
	if got := len(q.Alerts()); got != 50 {
		t.Fatalf("default capacity: len = %d, want 50", got)
	}
}

func TestAlertQueue_Ack(t *testing.T) {
	q := NewAlertQueue(10)
	q.Push(Alert{Level: AndonYellow, Source: "a"})
	q.Push(Alert{Level: AndonRed, Source: "b"})

	if got := q.Unacked(); got != 2 {
		t.Fatalf("before ack: unacked = %d, want 2", got)
	}

	q.Ack(0)
	if got := q.Unacked(); got != 1 {
		t.Fatalf("after ack(0): unacked = %d, want 1", got)
	}

	// Verify the acked alert is marked.
	alerts := q.Alerts()
	if !alerts[0].Acked {
		t.Fatal("alert 0 should be acked")
	}
	if alerts[1].Acked {
		t.Fatal("alert 1 should not be acked")
	}

	// Out-of-range index should not panic.
	q.Ack(-1)
	q.Ack(100)
}

func TestAlertQueue_Worst_UnackedOnly(t *testing.T) {
	q := NewAlertQueue(10)

	// Empty queue — worst is Green.
	if got := q.Worst(); got != AndonGreen {
		t.Fatalf("empty queue: worst = %v, want Green", got)
	}

	q.Push(Alert{Level: AndonGreen, Source: "ok"})
	q.Push(Alert{Level: AndonRed, Source: "critical"})
	q.Push(Alert{Level: AndonYellow, Source: "warn"})

	// Worst unacked should be Red.
	if got := q.Worst(); got != AndonRed {
		t.Fatalf("with red unacked: worst = %v, want Red", got)
	}

	// Ack the red one (index 1).
	q.Ack(1)

	// Now worst unacked should be Yellow.
	if got := q.Worst(); got != AndonYellow {
		t.Fatalf("after acking red: worst = %v, want Yellow", got)
	}

	// Ack everything.
	q.Ack(0)
	q.Ack(2)
	if got := q.Worst(); got != AndonGreen {
		t.Fatalf("all acked: worst = %v, want Green", got)
	}
}

func TestAlertQueue_Alerts_ReturnsCopy(t *testing.T) {
	q := NewAlertQueue(10)
	q.Push(Alert{Level: AndonGreen, Source: "orig"})

	alerts := q.Alerts()
	alerts[0].Source = "mutated"

	// Original should be unaffected.
	if q.Alerts()[0].Source != "orig" {
		t.Fatal("Alerts() should return a copy, not a reference")
	}
}
