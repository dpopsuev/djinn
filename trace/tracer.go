// tracer.go — Component-scoped tracer facade with auto-timed spans.
//
// Each component gets a Tracer via Ring.For(component). The Tracer
// auto-fills Component on every event and provides Begin() → Span
// for automatic latency measurement. Nil-safe — (*Tracer)(nil).Begin()
// returns a no-op Span.
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

// Begin starts a timed span. Call span.End() to record latency.
// Nil-safe: (*Tracer)(nil).Begin() returns a no-op Span.
func (t *Tracer) Begin(action, detail string) *Span {
	if t == nil {
		return &nopSpan
	}
	id := t.ring.Append(TraceEvent{
		Component: t.component,
		Action:    action,
		Detail:    detail,
	})
	return &Span{
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

// Span represents a timed operation. Call End() to record latency.
type Span struct {
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

// nopSpan is returned by nil Tracer.Begin() — all methods are safe no-ops.
var nopSpan = Span{nop: true}

// WithServer sets the MCP server name on the span.
func (s *Span) WithServer(server string) *Span {
	s.server = server
	return s
}

// WithTool sets the tool name on the span.
func (s *Span) WithTool(tool string) *Span {
	s.tool = tool
	return s
}

// End records the span's latency and appends a result event to the ring.
func (s *Span) End() {
	if s.nop {
		return
	}
	s.tracer.ring.Append(TraceEvent{
		ParentID:  s.id,
		Component: s.tracer.component,
		Action:    s.action + "_done",
		Server:    s.server,
		Tool:      s.tool,
		Detail:    s.detail,
		Latency:   time.Since(s.start),
	})
}

// EndWithError records the span with an error flag.
func (s *Span) EndWithError() {
	if s.nop {
		return
	}
	s.tracer.ring.Append(TraceEvent{
		ParentID:  s.id,
		Component: s.tracer.component,
		Action:    s.action + "_done",
		Server:    s.server,
		Tool:      s.tool,
		Detail:    s.detail,
		Latency:   time.Since(s.start),
		Error:     true,
	})
}

// Child creates a child span correlated to this span via ParentID.
func (s *Span) Child(action, detail string) *Span {
	if s.nop {
		return &nopSpan
	}
	id := s.tracer.ring.Append(TraceEvent{
		ParentID:  s.id,
		Component: s.tracer.component,
		Action:    action,
		Detail:    detail,
	})
	return &Span{
		tracer:   s.tracer,
		id:       id,
		parentID: s.id,
		start:    time.Now(),
		action:   action,
		detail:   detail,
	}
}

// ID returns the span's trace event ID.
func (s *Span) ID() string {
	return s.id
}
