// tool_hub.go — ToolHub mediates all tool execution through hubs (GOL-37).
//
// Wraps builtin.ToolExecutor with five-step mediation: trace → execute →
// measure → signal → render. Transparently drops in where raw Registry was used.
package substrate

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// Hub and phase name constants for tool execution.
const (
	toolHubName  = "tool"
	executePhase = "execute"
)

// ToolHub mediates all tool execution with tracing, SLA checks, and signal emission.
type ToolHub struct {
	HubCore
	Executor builtin.ToolExecutor
	Tracker  *tools.ToolLatencyTracker
	SLAs     map[string]ToolSLA
}

// NewToolHub creates a tool hub wrapping the given executor.
func NewToolHub(core HubCore, executor builtin.ToolExecutor, tracker *tools.ToolLatencyTracker) *ToolHub {
	return &ToolHub{
		HubCore:  core,
		Executor: executor,
		Tracker:  tracker,
		SLAs:     DefaultSLAs(),
	}
}

// Name returns the hub name.
func (h *ToolHub) Name() string { return toolHubName }

// Phase returns the DevOps phase.
func (h *ToolHub) Phase() string { return executePhase }

// Execute runs a tool with full mediation: trace → execute → measure → signal → render.
func (h *ToolHub) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	rt := h.Tracer.Begin("tool-exec", name)

	start := time.Now()
	result, err := h.Executor.Execute(ctx, name, input)
	elapsed := time.Since(start)

	if err != nil {
		rt.EndWithError()
	} else {
		rt.End()
	}

	// Record latency.
	if h.Tracker != nil {
		h.Tracker.Record(name, elapsed)
	}

	// Check SLA.
	if sla, ok := h.SLAs[name]; ok && h.Tracker != nil {
		slaResult := CheckSLA(sla, h.Tracker.P50(name), h.Tracker.P95(name), 0)
		if !slaResult.Overall {
			h.Emit(telemetry.Signal{
				Category: toolHubName,
				Level:    telemetry.Yellow,
				Source:   toolHubName + "-hub",
				Message:  name + " SLA breach: P95=" + elapsed.String(),
			})
		}
	}

	// Render.
	h.Render(DisplayMsg{
		Source:   toolHubName,
		Category: executePhase,
		Content:  ToolExecEvent{Name: name, Elapsed: elapsed, Error: err != nil},
	})

	return result, err
}

// All returns all registered tools.
func (h *ToolHub) All() []builtin.Tool {
	return h.Executor.All()
}

// Names returns all registered tool names.
func (h *ToolHub) Names() []string {
	return h.Executor.Names()
}

// ToolExecEvent is the display payload for tool execution.
type ToolExecEvent struct {
	Name    string        `json:"name"`
	Elapsed time.Duration `json:"elapsed"`
	Error   bool          `json:"error"`
}

// Compile-time interface checks.
var (
	_ MediatorHub          = (*ToolHub)(nil)
	_ builtin.ToolExecutor = (*ToolHub)(nil)
)
