// agent_handler.go — Bridges agent.EventHandler into the uniform pipeline.
//
// MetricsHandler implements agent.EventHandler and feeds AgentMetrics,
// AgentPolice, quality.DetectBottlenecks, and quality.CheckCordon on every round-trip.
// Violations and cordons emit to the SignalBus for downstream consumers
// (SignalInterpreter, TUI, etc.).
//
// DJN-TSK-1056
package substrate

import (
	"sync"
	"time"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools"
	"github.com/dpopsuev/djinn/uniform/quality"
)

// MetricsHandler bridges agent loop events into the uniform pipeline.
// Implements agent.EventHandler (same method set, no import needed).
type MetricsHandler struct {
	mu      sync.Mutex
	metrics *quality.AgentMetrics
	police  *quality.AgentPolice
	cordon  quality.CordonConfig
	bus     *telemetry.SignalBus
	latency *tools.ToolLatencyTracker

	// Per-turn state
	turnStart time.Time
	ttft      time.Duration
	turnIn    int
	turnOut   int
	gotFirst  bool
}

// NewMetricsHandler creates a handler that feeds the uniform pipeline.
func NewMetricsHandler(
	metrics *quality.AgentMetrics,
	police *quality.AgentPolice,
	bus *telemetry.SignalBus,
	latency *tools.ToolLatencyTracker,
	cordonCfg quality.CordonConfig,
) *MetricsHandler {
	return &MetricsHandler{
		metrics: metrics,
		police:  police,
		cordon:  cordonCfg,
		bus:     bus,
		latency: latency,
	}
}

func (h *MetricsHandler) OnText(_ string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.gotFirst {
		h.ttft = time.Since(h.turnStart)
		h.gotFirst = true
	}
}

func (h *MetricsHandler) OnThinking(_ string) {}

func (h *MetricsHandler) OnToolCall(_ driver.ToolCall) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.gotFirst {
		h.ttft = time.Since(h.turnStart)
		h.gotFirst = true
	}
}

func (h *MetricsHandler) OnToolResult(_, _, _ string, _ bool) {}

func (h *MetricsHandler) OnDone(usage *driver.Usage) {
	h.mu.Lock()
	rtt := time.Since(h.turnStart)
	ttft := h.ttft
	if usage != nil {
		h.turnIn = usage.InputTokens
		h.turnOut = usage.OutputTokens
	}
	tokensIn := h.turnIn
	tokensOut := h.turnOut

	// Reset per-turn state for next turn.
	h.turnStart = time.Now()
	h.gotFirst = false
	h.turnIn = 0
	h.turnOut = 0
	h.mu.Unlock()

	// Record metrics.
	h.metrics.RecordRoundTrip(ttft, rtt, tokensIn, tokensOut)

	// Check police violations.
	violations := h.police.Observe(h.metrics, h.latency)
	for _, v := range violations {
		level := telemetry.Yellow
		if v.Severity == quality.SeverityCritical {
			level = telemetry.Red
		}
		h.bus.Emit(telemetry.Signal{
			Level:    level,
			Source:   h.metrics.AgentID,
			Category: telemetry.CategoryBudget,
			Message:  v.Kind + ": " + v.Detail,
		})
	}

	// Check cordon thresholds.
	if c := quality.CheckCordon(h.metrics, h.cordon); c != nil {
		h.bus.Emit(telemetry.Signal{
			Level:    telemetry.Black,
			Source:   h.metrics.AgentID,
			Category: telemetry.CategoryBudget,
			Message:  string(c.Reason) + ": " + c.Detail,
		})
	}

	// Detect performance bottlenecks.
	bottlenecks := quality.DetectBottlenecks(h.metrics, h.latency)
	for _, b := range bottlenecks {
		h.bus.Emit(telemetry.Signal{
			Level:    telemetry.Yellow,
			Source:   h.metrics.AgentID,
			Category: telemetry.CategoryPerformance,
			Message:  string(b.Kind) + ": " + b.Detail,
		})
	}
}

func (h *MetricsHandler) OnError(err error) {
	h.bus.Emit(telemetry.Signal{
		Level:    telemetry.Red,
		Source:   h.metrics.AgentID,
		Category: telemetry.CategoryLifecycle,
		Message:  "agent error: " + err.Error(),
	})
}

// StartTurn marks the beginning of a new agent turn.
// Call this before each Chat() call in the agent loop.
func (h *MetricsHandler) StartTurn() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.turnStart = time.Now()
	h.gotFirst = false
}

// Metrics returns the underlying metrics for inspection.
func (h *MetricsHandler) Metrics() *quality.AgentMetrics { return h.metrics }

// Police returns the underlying police for inspection.
func (h *MetricsHandler) Police() *quality.AgentPolice { return h.police }
