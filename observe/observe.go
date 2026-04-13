// Package observe provides the Observe tool — Djinn's self-introspection
// facade over EventLog. Collapses TraceProjection into a single query API.
// Agents call Observe for context about what happened, system health,
// and active work. Hierophant uses Observe to assemble the context sandwich.
package observe

import "time"

// TraceLine is a single entry in a trace query result.
type TraceLine struct {
	Timestamp time.Time // when it happened
	Source    string    // which agent/component
	Kind      string    // event kind (tool_call, agent_turn, error, etc.)
	Summary   string    // human-readable one-liner
	Duration  int64     // milliseconds (0 if not applicable)
	TraceID   string    // intent-level correlation (GOL-162)
}

// HealthReport summarizes system health across all agents.
type HealthReport struct {
	AgentsAlive  int       // how many agents are running
	BudgetUsed   float64   // aggregate budget consumption (0.0–1.0)
	LastActivity time.Time // most recent event across all agents
	StuckAgents  []string  // agents with no activity for > threshold
	Errors       int       // error events in recent window
}

// TraceOpts controls what Trace returns.
type TraceOpts struct {
	Last    int    // return last N events (default 50)
	Kind    string // filter by event kind (empty = all)
	Source  string // filter by agent/component (empty = all)
	TraceID string // filter by intent-level trace ID (empty = all)
}

// Observer is the introspection interface. Implementations query EventLog
// and return structured, human-readable results.
type Observer interface {
	// Trace returns recent events matching the filter.
	Trace(opts TraceOpts) ([]TraceLine, error)

	// Health returns a summary of system health.
	Health() (HealthReport, error)
}
