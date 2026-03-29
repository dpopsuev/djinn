// tracer.go — Component-scoped tracer facade with auto-timed round-trips.
//
// Each component gets a Tracer via Ring.For(component). The Tracer
// auto-fills Component on every event and provides Begin() → RoundTrip
// for automatic latency measurement. Nil-safe — (*Tracer)(nil).Begin()
// returns a no-op RoundTrip.
package trace

import "time"

// Tracer is a component-scoped facade over a Ring.
// Created via Ring.For(component).
type Tracer struct {
	ring      *Ring
	component Component
}

// For creates a component-scoped Tracer. The returned Tracer auto-fills
// Component on every event. Safe to store and reuse.
func (r *Ring) For(component Component) *Tracer {
	return &Tracer{ring: r, component: component}
}

// Begin starts a timed round-trip. Call rt.End() to record latency.
// Nil-safe: (*Tracer)(nil).Begin() returns a no-op RoundTrip.
func (t *Tracer) Begin(action, detail string) *RoundTrip {
	if t == nil {
		return &nopRoundTrip
	}
	id := t.ring.Append(TraceEvent{
		Component: t.component,
		Action:    action,
		Detail:    detail,
	})
	return &RoundTrip{
		tracer: t,
		id:     id,
		start:  time.Now(),
		action: action,
		detail: detail,
	}
}

// Event records a point-in-time event (no duration).
// Nil-safe: (*Tracer)(nil).Event() is a no-op.
func (t *Tracer) Event(action, detail string) {
	if t == nil {
		return
	}
	t.ring.Append(TraceEvent{
		Component: t.component,
		Action:    action,
		Detail:    detail,
	})
}

// RoundTrip represents a timed operation. Call End() to record latency.
type RoundTrip struct {
	tracer   *Tracer
	id       string
	parentID string
	start    time.Time
	action   string
	detail   string
	server   string
	tool     string
	nop      bool
}

// nopRoundTrip is returned by nil Tracer.Begin() \u2014 all methods are safe no-ops.
var nopRoundTrip = RoundTrip{nop: true}

// WithServer sets the MCP server name on the round-trip.
func (s *RoundTrip) WithServer(server string) *RoundTrip {
	s.server = server
	return s
}

// WithTool sets the tool name on the round-trip.
func (s *RoundTrip) WithTool(tool string) *RoundTrip {
	s.tool = tool
	return s
}

// End records the round-trip's latency and appends a result event to the ring.
func (s *RoundTrip) End() {
	if s.nop {
		return
	}
	s.tracer.ring.Append(TraceEvent{
		ParentID:  s.id,
		Component: s.tracer.component,
		Action:    s.action + ActionDoneSuffix,
		Server:    s.server,
		Tool:      s.tool,
		Detail:    s.detail,
		Latency:   time.Since(s.start),
	})
}

// EndWithError records the round-trip with an error flag.
func (s *RoundTrip) EndWithError() {
	if s.nop {
		return
	}
	s.tracer.ring.Append(TraceEvent{
		ParentID:  s.id,
		Component: s.tracer.component,
		Action:    s.action + ActionDoneSuffix,
		Server:    s.server,
		Tool:      s.tool,
		Detail:    s.detail,
		Latency:   time.Since(s.start),
		Error:     true,
	})
}

// Child creates a child round-trip correlated to this one via ParentID.
func (s *RoundTrip) Child(action, detail string) *RoundTrip {
	if s.nop {
		return &nopRoundTrip
	}
	id := s.tracer.ring.Append(TraceEvent{
		ParentID:  s.id,
		Component: s.tracer.component,
		Action:    action,
		Detail:    detail,
	})
	return &RoundTrip{
		tracer:   s.tracer,
		id:       id,
		parentID: s.id,
		start:    time.Now(),
		action:   action,
		detail:   detail,
	}
}

// ID returns the round-trip's trace event ID.
func (s *RoundTrip) ID() string {
	return s.id
}
