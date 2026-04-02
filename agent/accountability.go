// accountability.go — Agent accountability metrics (TSK-561).
//
// Tracks attempted/approved/denied counts per MetricKind.
// ComplianceRate = approved / attempted. Extensible via MetricKind constants.
// New domains add constants without changing the struct (OCP).
package agent

import "sync"

// MetricKind identifies an accountability domain.
type MetricKind string

// Built-in metric kinds (Day 1).
const (
	MetricArchive    MetricKind = "archive"    // agent tried to archive a task
	MetricDeferral   MetricKind = "deferral"   // agent deferred work
	MetricBlocker    MetricKind = "blocker"    // agent claimed a blocker exists
	MetricOverride   MetricKind = "override"   // sovereign override used
	MetricEscalation MetricKind = "escalation" // agent escalated to operator
)

// AccountabilityMetric tracks attempted/approved/denied for one kind.
type AccountabilityMetric struct {
	Kind      MetricKind
	Attempted int
	Approved  int
	Denied    int
}

// ComplianceRate returns approved / attempted. Returns 1.0 if no attempts.
func (m *AccountabilityMetric) ComplianceRate() float64 {
	if m.Attempted == 0 {
		return 1.0
	}
	return float64(m.Approved) / float64(m.Attempted)
}

// AgentAccountability holds per-kind compliance metrics for an agent.
// Thread-safe: all methods are safe for concurrent use.
type AgentAccountability struct {
	agentID string
	metrics map[MetricKind]*AccountabilityMetric
	mu      sync.RWMutex
}

// NewAgentAccountability creates accountability tracking for an agent.
func NewAgentAccountability(agentID string) *AgentAccountability {
	return &AgentAccountability{
		agentID: agentID,
		metrics: make(map[MetricKind]*AccountabilityMetric),
	}
}

// AgentID returns the tracked agent's identifier.
func (a *AgentAccountability) AgentID() string { return a.agentID }

// Record increments the attempted count for a kind.
// Auto-creates the metric on first use.
func (a *AgentAccountability) Record(kind MetricKind) {
	a.mu.Lock()
	defer a.mu.Unlock()
	m := a.getOrCreate(kind)
	m.Attempted++
}

// Approve increments the approved count for a kind.
func (a *AgentAccountability) Approve(kind MetricKind) {
	a.mu.Lock()
	defer a.mu.Unlock()
	m := a.getOrCreate(kind)
	m.Approved++
}

// Deny increments the denied count for a kind.
func (a *AgentAccountability) Deny(kind MetricKind) {
	a.mu.Lock()
	defer a.mu.Unlock()
	m := a.getOrCreate(kind)
	m.Denied++
}

// Rate returns the compliance rate for a kind. Returns 1.0 if no attempts.
func (a *AgentAccountability) Rate(kind MetricKind) float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	m, ok := a.metrics[kind]
	if !ok {
		return 1.0
	}
	return m.ComplianceRate()
}

// OverallRate returns the average compliance rate across all kinds.
// Returns 1.0 if no metrics recorded.
func (a *AgentAccountability) OverallRate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.metrics) == 0 {
		return 1.0
	}
	var sum float64
	for _, m := range a.metrics {
		sum += m.ComplianceRate()
	}
	return sum / float64(len(a.metrics))
}

// Metric returns a copy of the metric for a kind. Returns nil if not tracked.
func (a *AgentAccountability) Metric(kind MetricKind) *AccountabilityMetric {
	a.mu.RLock()
	defer a.mu.RUnlock()
	m, ok := a.metrics[kind]
	if !ok {
		return nil
	}
	dup := *m
	return &dup
}

// Kinds returns all tracked metric kinds.
func (a *AgentAccountability) Kinds() []MetricKind {
	a.mu.RLock()
	defer a.mu.RUnlock()
	kinds := make([]MetricKind, 0, len(a.metrics))
	for k := range a.metrics {
		kinds = append(kinds, k)
	}
	return kinds
}

func (a *AgentAccountability) getOrCreate(kind MetricKind) *AccountabilityMetric {
	m, ok := a.metrics[kind]
	if !ok {
		m = &AccountabilityMetric{Kind: kind}
		a.metrics[kind] = m
	}
	return m
}
