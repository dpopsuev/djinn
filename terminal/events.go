package terminal

import "time"

// ViewEventKind identifies the type of output event.
type ViewEventKind string

const (
	EventOutput      ViewEventKind = "output"       // text appended to output stream
	EventThinking    ViewEventKind = "thinking"     // agent reasoning/thinking text
	EventToolCall    ViewEventKind = "tool_call"    // agent invoked a tool
	EventToolResult  ViewEventKind = "tool_result"  // tool returned a result
	EventAgentStatus ViewEventKind = "agent_status" // agent state changed
	EventDashboard   ViewEventKind = "dashboard"    // dashboard metrics updated
	EventError       ViewEventKind = "error"        // error occurred
	EventDone        ViewEventKind = "done"         // agent turn completed
)

// ViewEvent is the universal output event from the Terminal.
// Subscribers receive these on their registered channels.
type ViewEvent struct {
	Kind      ViewEventKind `json:"kind"`
	Text      string        `json:"text,omitempty"`     // output/thinking/error text
	Tool      string        `json:"tool,omitempty"`     // tool name (tool_call/tool_result)
	AgentID   string        `json:"agent_id,omitempty"` // which agent (multi-agent)
	Role      string        `json:"role,omitempty"`     // agent role
	State     string        `json:"state,omitempty"`    // agent state (idle/streaming/done)
	IsError   bool          `json:"is_error,omitempty"` // tool result was an error
	Timestamp time.Time     `json:"ts"`
}

// RunState captures the current terminal state for Status() queries.
type RunState struct {
	Operation   string `json:"operation"`    // ask/plan/agent
	AgentCount  int    `json:"agent_count"`  // currently running
	AgentCap    int    `json:"agent_cap"`    // maximum capacity
	Turns       int    `json:"turns"`        // conversation turns
	TokensIn    int    `json:"tokens_in"`    // cumulative input tokens
	TokensOut   int    `json:"tokens_out"`   // cumulative output tokens
	ActiveRole  string `json:"active_role"`  // current agent role
	ScopePath   string `json:"scope_path"`   // current scope position
	ScopeType   string `json:"scope_type"`   // scope type (system/ops/etc)
	EnvelopeOn  bool   `json:"envelope_on"`  // lifecycle envelope enabled
	IsStreaming bool   `json:"is_streaming"` // agent currently responding
}

// IntrospectionReport provides detailed observability data.
type IntrospectionReport struct {
	RunState

	ToolLevel    string        `json:"tool_level"`    // L0-L4
	EnabledTools []string      `json:"enabled_tools"` // list of available tool names
	Agents       []AgentInfo   `json:"agents"`        // per-agent detail
	Uptime       time.Duration `json:"uptime"`        // time since Start()

	// Resource usage (populated when available)
	PeakMemoryMB float64 `json:"peak_memory_mb,omitempty"`
	CPUPercent   float64 `json:"cpu_percent,omitempty"`
}

// AgentInfo describes one running agent.
type AgentInfo struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	State     string `json:"state"`
	TokensIn  int    `json:"tokens_in"`
	TokensOut int    `json:"tokens_out"`
	Color     string `json:"color,omitempty"` // hex from Display
}
