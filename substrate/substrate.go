// Package substrate defines the Djinn daemon (djinnd) interface.
//
// The Substrate manages agent lifecycle, Space isolation, enrichment,
// and observation. It does NOT intercept tool calls — agents act freely
// inside their Space. Value is added through Envelope middleware baked
// into the tools, not as an interception layer.
//
// Responsibilities: Cache, Envelope, Observe, Plan, Spawn, Space.
package substrate

import (
	"context"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/battery/service"
	"github.com/dpopsuev/battery/tool"
	djinncache "github.com/dpopsuev/djinn/cache"
	"github.com/dpopsuev/djinn/vessel"
	"github.com/dpopsuev/troupe/signal"
)

// Substrate is the node mediator interface. Wires all node-local services.
// Consumers take what they need (ISP).
type Substrate interface {
	// Tools returns the tool executor with Envelope middleware applied.
	Tools() tool.Executor

	// Envelope returns the Gate/Enrich/Execute/Record pipeline.
	Envelope() *middleware.Envelope

	// Observe records a tool call observation.
	Observe(ctx context.Context, obs Observation)

	// EventLog returns the unified event log. All agent actions flow through it.
	EventLog() signal.EventLog

	// L2 returns the shared L2 cache. All Vessels write-through to this.
	// Scope-tagged by agent ID for recovery.
	L2() djinncache.Cache

	// Vessel creates an agent harness for the given config.
	// The Vessel wraps Tools, EventLog, and workspace into one interface.
	Vessel(cfg SpawnConfig) vessel.Vessel

	// Spawn starts a new agent with the given config. Returns agent ID.
	Spawn(ctx context.Context, cfg SpawnConfig) (string, error)

	// Kill stops an agent by ID.
	Kill(ctx context.Context, agentID string) error

	// Health returns the Substrate's overall health.
	Health() service.HealthReport
}

// Observation is a recorded tool call event.
type Observation struct {
	AgentID  string `json:"agent_id"`
	Tool     string `json:"tool"`
	Action   string `json:"action,omitempty"`
	Duration int64  `json:"duration_ms"`
	Error    string `json:"error,omitempty"`
}

// SpawnConfig defines what an agent needs to start.
type SpawnConfig struct {
	Role       string   `json:"role"`
	Model      string   `json:"model,omitempty"`
	Tools      []string `json:"tools,omitempty"` // tool capability names
	ReadPaths  []string `json:"read_paths,omitempty"`
	WritePaths []string `json:"write_paths,omitempty"`
}
