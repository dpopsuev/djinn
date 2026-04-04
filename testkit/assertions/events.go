package assertions

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/broker"
)

// AssertEventOrder checks that events contain the expected kinds in order.
func AssertEventOrder(t *testing.T, events []broker.Event, expectedKinds []broker.EventKind) {
	t.Helper()
	idx := 0
	for _, e := range events {
		if idx < len(expectedKinds) && e.Kind == expectedKinds[idx] {
			idx++
		}
	}
	if idx != len(expectedKinds) {
		t.Fatalf("event order: matched %d of %d expected kinds", idx, len(expectedKinds))
	}
}

// WaitForEvent polls a function until an event with the given kind appears.
func WaitForEvent(t *testing.T, getEvents func() []broker.Event, kind broker.EventKind, timeout time.Duration) broker.Event {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, e := range getEvents() {
			if e.Kind == kind {
				return e
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for event kind %v", kind)
	return broker.Event{}
}

// AssertNoEvent checks that no event with the given kind exists.
func AssertNoEvent(t *testing.T, events []broker.Event, kind broker.EventKind) {
	t.Helper()
	for _, e := range events {
		if e.Kind == kind {
			t.Fatalf("unexpected event kind %v found", kind)
		}
	}
}

// CollectEvents collects events from a channel until it's closed or timeout.
func CollectEvents(ch <-chan broker.Event, timeout time.Duration) []broker.Event {
	var events []broker.Event
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, e)
		case <-timer.C:
			return events
		}
	}
}
