// metrics.go — Per-agent telemetry for the Djinn staffing model.
// Tracks round-trips, token usage, cost, gate outcomes, and latency samples.
package staff

import (
	"sync"
	"time"
)

// AgentMetrics collects telemetry for a single agent instance.
type AgentMetrics struct {
	mu         sync.Mutex
	AgentID    string
	Role       string
	RoundTrips int
	TotalIn    int // input tokens
	TotalOut   int // output tokens
	Cost       float64
	GatePasses int
	GateFails  int
	TTFT       []time.Duration // time to first token samples
	RTT        []time.Duration // round-trip time samples
}

// NewAgentMetrics creates a new metrics collector for the given agent.
func NewAgentMetrics(agentID, role string) *AgentMetrics {
	return &AgentMetrics{
		AgentID: agentID,
		Role:    role,
	}
}

// RecordRoundTrip records a single round-trip with timing and token data.
func (m *AgentMetrics) RecordRoundTrip(ttft, rtt time.Duration, tokensIn, tokensOut int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RoundTrips++
	m.TotalIn += tokensIn
	m.TotalOut += tokensOut
	m.TTFT = append(m.TTFT, ttft)
	m.RTT = append(m.RTT, rtt)
}

// RecordGateResult records whether an agent passed or failed a quality gate.
func (m *AgentMetrics) RecordGateResult(passed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if passed {
		m.GatePasses++
	} else {
		m.GateFails++
	}
}

// TPM returns tokens per minute based on the most recent RTT samples.
// Uses up to the last 10 RTTs for the estimate.
func (m *AgentMetrics) TPM() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.RTT) == 0 {
		return 0
	}

	// Use up to last 10 samples.
	start := 0
	if len(m.RTT) > 10 {
		start = len(m.RTT) - 10
	}
	recent := m.RTT[start:]

	var totalDuration time.Duration
	for _, d := range recent {
		totalDuration += d
	}
	if totalDuration == 0 {
		return 0
	}

	totalTokens := m.TotalIn + m.TotalOut
	minutes := totalDuration.Minutes()
	if minutes == 0 {
		return 0
	}

	// Scale by the fraction of samples used.
	sampleFraction := float64(len(recent)) / float64(len(m.RTT))
	tokensInWindow := float64(totalTokens) * sampleFraction
	return tokensInWindow / minutes
}

// AvgTTFT returns the average time to first token.
func (m *AgentMetrics) AvgTTFT() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return avgDuration(m.TTFT)
}

// AvgRTT returns the average round-trip time.
func (m *AgentMetrics) AvgRTT() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return avgDuration(m.RTT)
}

// GatePassRate returns the fraction of gate passes (0.0-1.0).
// Returns 0 if no gates have been evaluated.
func (m *AgentMetrics) GatePassRate() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	total := m.GatePasses + m.GateFails
	if total == 0 {
		return 0
	}
	return float64(m.GatePasses) / float64(total)
}

// avgDuration computes the mean of a slice of durations.
func avgDuration(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range ds {
		total += d
	}
	return total / time.Duration(len(ds))
}
