// ring.go — Bounded ring buffer for trace events (TSK-476).
//
// Thread-safe, O(1) append, O(n) query. Adapts djinnlog.RingHandler
// pattern for structured TraceEvents instead of log entries.
package trace

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Ring is a bounded circular buffer of TraceEvents.
type Ring struct {
	mu     sync.RWMutex
	events []TraceEvent
	cap    int
	pos    int
	count  int
	nextID atomic.Int64
}

// NewRing creates a ring buffer with the given capacity.
func NewRing(capacity int) *Ring {
	return &Ring{
		events: make([]TraceEvent, capacity),
		cap:    capacity,
	}
}

// Append adds an event to the ring, assigning an ID and timestamp.
// Returns the assigned event ID.
func (r *Ring) Append(e TraceEvent) string {
	id := r.nextID.Add(1)
	e.ID = fmt.Sprintf("trace-%d", id)
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	r.mu.Lock()
	r.events[r.pos] = e
	r.pos = (r.pos + 1) % r.cap
	if r.count < r.cap {
		r.count++
	}
	r.mu.Unlock()

	return e.ID
}

// Last returns the most recent n events (oldest first).
func (r *Ring) Last(n int) []TraceEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n > r.count {
		n = r.count
	}
	if n == 0 {
		return nil
	}

	out := make([]TraceEvent, n)
	start := (r.pos - n + r.cap) % r.cap
	for i := range n {
		out[i] = r.events[(start+i)%r.cap]
	}
	return out
}

// Since returns all events after the given time (oldest first).
func (r *Ring) Since(t time.Time) []TraceEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []TraceEvent
	r.walk(func(e TraceEvent) {
		if e.Timestamp.After(t) {
			out = append(out, e)
		}
	})
	return out
}

// ByParent returns all events with the given ParentID.
func (r *Ring) ByParent(parentID string) []TraceEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []TraceEvent
	r.walk(func(e TraceEvent) {
		if e.ParentID == parentID {
			out = append(out, e)
		}
	})
	return out
}

// ByComponent returns events filtered by component.
func (r *Ring) ByComponent(c Component) []TraceEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []TraceEvent
	r.walk(func(e TraceEvent) {
		if e.Component == c {
			out = append(out, e)
		}
	})
	return out
}

// Get returns a single event by ID.
func (r *Ring) Get(id string) (TraceEvent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var found TraceEvent
	var ok bool
	r.walk(func(e TraceEvent) {
		if e.ID == id {
			found = e
			ok = true
		}
	})
	return found, ok
}

// Stats returns current ring buffer statistics.
func (r *Ring) Stats() RingStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RingStats{
		Capacity: r.cap,
		Count:    r.count,
	}
	if r.count > 0 {
		oldest := (r.pos - r.count + r.cap) % r.cap
		stats.Oldest = r.events[oldest].Timestamp
		newest := (r.pos - 1 + r.cap) % r.cap
		stats.Newest = r.events[newest].Timestamp
	}
	return stats
}

// walk iterates events oldest-first. Caller must hold read lock.
func (r *Ring) walk(fn func(TraceEvent)) {
	start := (r.pos - r.count + r.cap) % r.cap
	for i := range r.count {
		fn(r.events[(start+i)%r.cap])
	}
}
