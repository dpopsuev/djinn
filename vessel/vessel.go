// Package vessel defines the agent harness — the bridge between
// an agent (LLM) and the tools/services it can use.
// One Vessel per agent session. Provides tool access, budget,
// workspace scope, and event emission.
package vessel

import (
	"context"

	"github.com/dpopsuev/battery/tool"
	"github.com/dpopsuev/troupe/signal"
)

// Vessel is the agent harness. Provides everything an agent needs
// to do work: tools, events, budget, workspace path.
type Vessel interface {
	// Tools returns the tool executor for this agent session.
	Tools() tool.Executor

	// EventLog returns the event log for this session.
	EventLog() signal.EventLog

	// WorkDir returns the workspace directory for this agent.
	WorkDir() string

	// Close releases resources when the agent session ends.
	Close(ctx context.Context) error
}
