// bottleneck.go — Bottleneck detection for the Djinn staffing model.
// Analyzes agent metrics and tool latency to identify performance issues.
package staff

import (
	"fmt"
	"time"

	"github.com/dpopsuev/djinn/tools"
)

// BottleneckKind classifies the type of performance bottleneck.
type BottleneckKind string

const (
	BottleneckToolLatency   BottleneckKind = "tool_latency"   // p95 > 2s
	BottleneckRateLimit     BottleneckKind = "rate_limit"     // TTFT spike 10x
	BottleneckContextThrash BottleneckKind = "context_thrash" // relays > 3
	BottleneckGateLoop      BottleneckKind = "gate_loop"      // retries > 3
	BottleneckIdleWait      BottleneckKind = "idle_wait"      // idle > 50%
	BottleneckToolSpam      BottleneckKind = "tool_spam"      // > 20 calls/trip
)

// Bottleneck describes a detected performance issue.
type Bottleneck struct {
	Kind   BottleneckKind
	Agent  string
	Detail string
}

// Detection thresholds.
const (
	toolLatencyThreshold = 2 * time.Second // p95 > 2s
	rateLimitMultiplier  = 10              // TTFT spike 10x over average
	gateLimitThreshold   = 3               // gate failures > 3
	toolSpamThreshold    = 20              // > 20 tool calls per round-trip
)

// DetectBottlenecks analyzes agent metrics and tool latency to find performance issues.
func DetectBottlenecks(metrics *AgentMetrics, latency *tools.ToolLatencyTracker) []Bottleneck {
	var results []Bottleneck

	if metrics == nil {
		return results
	}

	metrics.mu.Lock()
	agentID := metrics.AgentID
	ttft := make([]time.Duration, len(metrics.TTFT))
	copy(ttft, metrics.TTFT)
	rtt := make([]time.Duration, len(metrics.RTT))
	copy(rtt, metrics.RTT)
	gateFails := metrics.GateFails
	roundTrips := metrics.RoundTrips
	metrics.mu.Unlock()

	// Tool latency: p95 > 2s for any tool.
	if latency != nil {
		for _, tool := range latency.AllTools() {
			p95 := latency.P95(tool)
			if p95 > toolLatencyThreshold {
				results = append(results, Bottleneck{
					Kind:   BottleneckToolLatency,
					Agent:  agentID,
					Detail: fmt.Sprintf("tool %q p95=%v exceeds %v", tool, p95, toolLatencyThreshold),
				})
			}
		}
	}

	// Rate limit: last TTFT is 10x the average.
	if len(ttft) >= 2 {
		avg := avgDuration(ttft[:len(ttft)-1])
		last := ttft[len(ttft)-1]
		if avg > 0 && last > time.Duration(rateLimitMultiplier)*avg {
			results = append(results, Bottleneck{
				Kind:   BottleneckRateLimit,
				Agent:  agentID,
				Detail: fmt.Sprintf("TTFT spike: last=%v, avg=%v (%dx)", last, avg, rateLimitMultiplier),
			})
		}
	}

	// Gate loop: too many gate failures.
	if gateFails > gateLimitThreshold {
		results = append(results, Bottleneck{
			Kind:   BottleneckGateLoop,
			Agent:  agentID,
			Detail: fmt.Sprintf("gate failures=%d exceeds threshold=%d", gateFails, gateLimitThreshold),
		})
	}

	// Idle wait: if average RTT is long relative to TTFT, the agent is idle.
	// "Idle" = TTFT accounts for > 50% of RTT on average.
	if len(ttft) > 0 && len(rtt) > 0 {
		avgT := avgDuration(ttft)
		avgR := avgDuration(rtt)
		if avgR > 0 && float64(avgT)/float64(avgR) > 0.50 {
			results = append(results, Bottleneck{
				Kind:   BottleneckIdleWait,
				Agent:  agentID,
				Detail: fmt.Sprintf("TTFT/RTT ratio=%.0f%% (idle waiting)", float64(avgT)/float64(avgR)*100),
			})
		}
	}

	// Tool spam: more tool calls than round-trips * threshold.
	// We approximate using latency tracker counts.
	if latency != nil && roundTrips > 0 {
		totalCalls := 0
		for _, tool := range latency.AllTools() {
			totalCalls += latency.Count(tool)
		}
		callsPerTrip := float64(totalCalls) / float64(roundTrips)
		if callsPerTrip > float64(toolSpamThreshold) {
			results = append(results, Bottleneck{
				Kind:   BottleneckToolSpam,
				Agent:  agentID,
				Detail: fmt.Sprintf("%.1f tool calls/round-trip exceeds %d", callsPerTrip, toolSpamThreshold),
			})
		}
	}

	return results
}
