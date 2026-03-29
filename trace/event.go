// event.go — TraceEvent type for cross-component tracing (TSK-476).
//
// Every MCP call, agent turn, signal emission, and tool execution
// produces a TraceEvent. Events are correlated via ParentID to
// reconstruct full round-trips.
package trace

import "time"

// Component identifies the system layer that produced an event.
type Component string

const (
	ComponentMCP    Component = "mcp"
	ComponentAgent  Component = "agent"
	ComponentSignal Component = "signal"
	ComponentTool   Component = "tool"
	ComponentTUI    Component = "tui"
	ComponentReview Component = "review"
)

// TraceEvent captures a single traced operation.
type TraceEvent struct {
	ID        string            `json:"id"`                  // unique: trace-1, trace-2, ...
	ParentID  string            `json:"parent_id,omitempty"` // round-trip correlation
	Timestamp time.Time         `json:"ts"`
	Component Component         `json:"component"`
	Action    string            `json:"action"`           // call, result, emit, prompt, response, claim
	Server    string            `json:"server,omitempty"` // MCP server name
	Tool      string            `json:"tool,omitempty"`   // tool name
	Detail    string            `json:"detail"`           // human-readable summary
	Latency   time.Duration     `json:"latency,omitzero"`
	Error     bool              `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ActionDoneSuffix is appended to action names when a RoundTrip completes.
const ActionDoneSuffix = "_done"

// RingStats summarizes ring buffer state.
type RingStats struct {
	Capacity int       `json:"capacity"`
	Count    int       `json:"count"`
	Oldest   time.Time `json:"oldest,omitzero"`
	Newest   time.Time `json:"newest,omitzero"`
}
