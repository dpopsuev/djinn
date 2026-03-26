package staff

import (
	"testing"
	"time"
)

func TestCheckCordon_NilMetrics(t *testing.T) {
	c := CheckCordon(nil, DefaultCordonConfig())
	if c != nil {
		t.Fatal("nil metrics should produce nil cordon")
	}
}

func TestCheckCordon_NoCordonWhenWithinBudget(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)

	cfg := DefaultCordonConfig()
	cfg.MaxTokens = 10000

	c := CheckCordon(m, cfg)
	if c != nil {
		t.Fatalf("should not cordon within budget, got %+v", c)
	}
}

func TestCheckCordon_TokenBudget(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 5000, 5000)

	cfg := DefaultCordonConfig()
	cfg.MaxTokens = 1000

	c := CheckCordon(m, cfg)
	if c == nil {
		t.Fatal("expected cordon for token budget exceeded")
	}
	if c.Reason != CordonTokenBudget {
		t.Fatalf("reason = %q, want %q", c.Reason, CordonTokenBudget)
	}
	if c.AgentID != "agent-1" {
		t.Fatalf("agentID = %q, want %q", c.AgentID, "agent-1")
	}
}

func TestCheckCordon_CostBudget(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.mu.Lock()
	m.Cost = 5.0
	m.mu.Unlock()

	cfg := DefaultCordonConfig()
	cfg.MaxCost = 1.0

	c := CheckCordon(m, cfg)
	if c == nil {
		t.Fatal("expected cordon for cost budget exceeded")
	}
	if c.Reason != CordonCostBudget {
		t.Fatalf("reason = %q, want %q", c.Reason, CordonCostBudget)
	}
}

func TestCheckCordon_GateLoop(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	for i := 0; i < 5; i++ {
		m.RecordGateResult(false)
	}

	cfg := DefaultCordonConfig()
	// MaxGateRetries defaults to 3, 5 failures > 3

	c := CheckCordon(m, cfg)
	if c == nil {
		t.Fatal("expected cordon for gate loop exceeded")
	}
	if c.Reason != CordonGateLoop {
		t.Fatalf("reason = %q, want %q", c.Reason, CordonGateLoop)
	}
}

func TestCheckCordon_NoCordonBelowGateThreshold(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordGateResult(false)
	m.RecordGateResult(false)

	cfg := DefaultCordonConfig()
	c := CheckCordon(m, cfg)
	if c != nil {
		t.Fatalf("should not cordon with only 2 gate failures, got %+v", c)
	}
}

func TestCheckCordon_UnlimitedTokens(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 999999, 999999)

	cfg := DefaultCordonConfig()
	// MaxTokens = 0 means unlimited.

	c := CheckCordon(m, cfg)
	if c != nil {
		t.Fatalf("unlimited tokens should not cordon, got %+v", c)
	}
}

func TestCheckCordon_TokenBudgetPrioritized(t *testing.T) {
	// When both token and cost budgets are exceeded, token is returned first.
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 5000, 5000)
	m.mu.Lock()
	m.Cost = 10.0
	m.mu.Unlock()

	cfg := DefaultCordonConfig()
	cfg.MaxTokens = 1000
	cfg.MaxCost = 1.0

	c := CheckCordon(m, cfg)
	if c == nil {
		t.Fatal("expected cordon")
	}
	if c.Reason != CordonTokenBudget {
		t.Fatalf("reason = %q, want %q (token budget checked first)", c.Reason, CordonTokenBudget)
	}
}

func TestDefaultCordonConfig(t *testing.T) {
	cfg := DefaultCordonConfig()
	if cfg.MaxGateRetries != 3 {
		t.Fatalf("MaxGateRetries = %d, want 3", cfg.MaxGateRetries)
	}
	if cfg.SilenceTimeout != 30*time.Second {
		t.Fatalf("SilenceTimeout = %v, want 30s", cfg.SilenceTimeout)
	}
	if cfg.MaxFilesChanged != 50 {
		t.Fatalf("MaxFilesChanged = %d, want 50", cfg.MaxFilesChanged)
	}
}

func TestCordonTimestamp(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 5000, 5000)

	cfg := DefaultCordonConfig()
	cfg.MaxTokens = 1000

	before := time.Now()
	c := CheckCordon(m, cfg)
	after := time.Now()

	if c == nil {
		t.Fatal("expected cordon")
	}
	if c.Timestamp.Before(before) || c.Timestamp.After(after) {
		t.Fatalf("timestamp %v not between %v and %v", c.Timestamp, before, after)
	}
}
