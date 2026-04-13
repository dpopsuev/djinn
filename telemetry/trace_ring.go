// ring.go — Bounded ring buffer for trace events (TSK-476).
//
// Thread-safe, O(1) append, O(n) query. Adapts djinnlog.RingHandler
// pattern for structured TraceEvents instead of log entries.
package telemetry

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dpopsuev/troupe/signal"
)

// TraceProjection is a bounded CQRS read model over the event log.
// Optimized for recent-event queries (Last, ByComponent, Get, Stats).
// When EventLog is set, every Append also emits to it —
// bridging Djinn's trace system to the unified event log.
type TraceProjection struct {
	mu       sync.RWMutex
	events   []TraceEvent
	cap      int
	pos      int
	count    int
	nextID   atomic.Int64
	traceID  string          // current intent-level trace ID (GOL-162)
	eventLog signal.EventLog // optional: unified event log bridge
}

// NewTraceProjection creates a bounded projection with the given capacity.
func NewTraceProjection(capacity int) *TraceProjection {
	return &TraceProjection{
		events: make([]TraceEvent, capacity),
		cap:    capacity,
	}
}

// WithEventLog bridges the Ring to a unified EventLog.
// Every Append also emits to the EventLog. Call once at composition time.
func (r *TraceProjection) WithEventLog(log signal.EventLog) *TraceProjection {
	r.eventLog = log
	return r
}

// SetTraceID sets the intent-level trace ID for all subsequent events.
// Call when a new operator prompt arrives. Pass "" to clear.
func (r *TraceProjection) SetTraceID(traceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.traceID = traceID
}

// TraceID returns the current intent-level trace ID.
func (r *TraceProjection) TraceID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.traceID
}

// Append adds an event to the ring, assigning an ID and timestamp.
// If the event has no TraceID, inherits the current one from SetTraceID.
// Returns the assigned event ID.
func (r *TraceProjection) Append(e TraceEvent) string {
	id := r.nextID.Add(1)
	e.ID = fmt.Sprintf("trace-%d", id)
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	r.mu.Lock()
	// Inherit trace ID if not explicitly set.
	if e.TraceID == "" && r.traceID != "" {
		e.TraceID = r.traceID
	}
	r.events[r.pos] = e
	r.pos = (r.pos + 1) % r.cap
	if r.count < r.cap {
		r.count++
	}
	log := r.eventLog
	r.mu.Unlock()

	// Bridge to unified event log (outside lock).
	if log != nil {
		log.Emit(traceToEvent(e))
	}

	return e.ID
}

// Last returns the most recent n events (oldest first).
func (r *TraceProjection) Last(n int) []TraceEvent {
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
func (r *TraceProjection) Since(t time.Time) []TraceEvent {
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
func (r *TraceProjection) ByParent(parentID string) []TraceEvent {
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
func (r *TraceProjection) ByComponent(c Component) []TraceEvent {
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
func (r *TraceProjection) Get(id string) (TraceEvent, bool) {
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
func (r *TraceProjection) Stats() RingStats {
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

// traceToEvent converts a TraceEvent to a signal Event.
// TraceEvent itself becomes the typed Data payload.
func traceToEvent(te TraceEvent) signal.Event {
	return signal.Event{
		ID:        te.ID,
		ParentID:  te.ParentID,
		TraceID:   te.TraceID,
		Timestamp: te.Timestamp,
		Source:    string(te.Component),
		Kind:      te.Action,
		Data:      te,
	}
}

// walk iterates events oldest-first. Caller must hold read lock.
func (r *TraceProjection) walk(fn func(TraceEvent)) {
	start := (r.pos - r.count + r.cap) % r.cap
	for i := range r.count {
		fn(r.events[(start+i)%r.cap])
	}
}

// --- Deprecated aliases for backward compatibility ---

// Ring is the old name for TraceProjection. Use TraceProjection.
//
// Deprecated: use TraceProjection.
type Ring = TraceProjection

// NewRing is the old name for NewTraceProjection.
//
// Deprecated: use NewTraceProjection.
var NewRing = NewTraceProjection
