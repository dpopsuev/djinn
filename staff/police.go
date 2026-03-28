// police.go — AgentPolice observes agent behavior and flags violations.
// Works alongside the Cordon system: Cordon halts, Police records evidence.
package staff

import (
	"fmt"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/tools"
)

// Violation records a single policy or behavioral violation.
type Violation struct {
	Kind      string // "sandbox_breach", "budget_exceeded", "tool_spam", "dangerous_output"
	AgentID   string
	Detail    string
	Severity  string // "warning", "critical"
	Timestamp time.Time
}

// AgentPolice observes agent metrics and tool usage, accumulating violations.
type AgentPolice struct {
	mu         sync.Mutex
	violations []Violation
	config     CordonConfig
}

// NewAgentPolice creates a police instance with the given cordon config.
func NewAgentPolice(config CordonConfig) *AgentPolice {
	return &AgentPolice{
		config: config,
	}
}

// toolSpamPerTripThreshold is the number of tool calls per round-trip
// that triggers a tool_spam violation.
const toolSpamPerTripThreshold = 20

// Observe inspects the current metrics and latency tracker for violations.
// Any new violations are appended to the internal log and returned.
func (p *AgentPolice) Observe(metrics *AgentMetrics, latency *tools.ToolLatencyTracker) []Violation {
	if metrics == nil {
		return nil
	}

	metrics.mu.Lock()
	agentID := metrics.AgentID
	totalTokens := metrics.TotalIn + metrics.TotalOut
	cost := metrics.Cost
	roundTrips := metrics.RoundTrips
	gateFails := metrics.GateFails
	metrics.mu.Unlock()

	now := time.Now()
	var found []Violation

	// Budget exceeded (tokens).
	if p.config.MaxTokens > 0 && totalTokens > p.config.MaxTokens {
		found = append(found, Violation{
			Kind:      "budget_exceeded",
			AgentID:   agentID,
			Detail:    fmt.Sprintf("token usage %d exceeds budget %d", totalTokens, p.config.MaxTokens),
			Severity:  "critical",
			Timestamp: now,
		})
	}

	// Budget exceeded (cost).
	if p.config.MaxCost > 0 && cost > p.config.MaxCost {
		found = append(found, Violation{
			Kind:      "budget_exceeded",
			AgentID:   agentID,
			Detail:    fmt.Sprintf("cost $%.4f exceeds budget $%.4f", cost, p.config.MaxCost),
			Severity:  "critical",
			Timestamp: now,
		})
	}

	// Gate loop — too many consecutive failures.
	if p.config.MaxGateRetries > 0 && gateFails > p.config.MaxGateRetries {
		found = append(found, Violation{
			Kind:      "gate_loop",
			AgentID:   agentID,
			Detail:    fmt.Sprintf("gate failures %d exceed max retries %d", gateFails, p.config.MaxGateRetries),
			Severity:  "warning",
			Timestamp: now,
		})
	}

	// Tool spam: >20 tool calls per round-trip.
	if latency != nil && roundTrips > 0 {
		totalCalls := 0
		for _, tool := range latency.AllTools() {
			totalCalls += latency.Count(tool)
		}
		callsPerTrip := float64(totalCalls) / float64(roundTrips)
		if callsPerTrip > float64(toolSpamPerTripThreshold) {
			found = append(found, Violation{
				Kind:      "tool_spam",
				AgentID:   agentID,
				Detail:    fmt.Sprintf("%.1f tool calls/trip exceeds %d", callsPerTrip, toolSpamPerTripThreshold),
				Severity:  "warning",
				Timestamp: now,
			})
		}
	}

	// Record any new violations.
	if len(found) > 0 {
		p.mu.Lock()
		p.violations = append(p.violations, found...)
		p.mu.Unlock()
	}

	return found
}

// Report returns all accumulated violations.
func (p *AgentPolice) Report() []Violation {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Violation, len(p.violations))
	copy(out, p.violations)
	return out
}

// ViolationCount returns the total number of accumulated violations.
func (p *AgentPolice) ViolationCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.violations)
}
