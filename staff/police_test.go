package staff

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/tools"
)

func TestAgentPolice_NoViolationsWhenClean(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)

	cfg := DefaultCordonConfig()
	cfg.MaxTokens = 10000

	p := NewAgentPolice(cfg)
	violations := p.Observe(m, nil)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
	}
	if p.ViolationCount() != 0 {
		t.Fatalf("ViolationCount = %d, want 0", p.ViolationCount())
	}
}

func TestAgentPolice_TokenBudgetViolation(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 5000, 5000)

	cfg := DefaultCordonConfig()
	cfg.MaxTokens = 1000

	p := NewAgentPolice(cfg)
	violations := p.Observe(m, nil)

	found := false
	for _, v := range violations {
		if v.Kind == "budget_exceeded" && v.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected budget_exceeded violation")
	}
}

func TestAgentPolice_ToolSpamViolation(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)

	lat := tools.NewToolLatencyTracker()
	for i := 0; i < 25; i++ {
		lat.Record("Read", 50*time.Millisecond)
	}

	cfg := DefaultCordonConfig()
	p := NewAgentPolice(cfg)
	violations := p.Observe(m, lat)

	found := false
	for _, v := range violations {
		if v.Kind == "tool_spam" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected tool_spam violation for 25 calls/trip")
	}
}

func TestAgentPolice_NoToolSpamBelowThreshold(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)

	lat := tools.NewToolLatencyTracker()
	for i := 0; i < 10; i++ {
		lat.Record("Read", 50*time.Millisecond)
	}

	cfg := DefaultCordonConfig()
	p := NewAgentPolice(cfg)
	violations := p.Observe(m, lat)

	for _, v := range violations {
		if v.Kind == "tool_spam" {
			t.Fatal("should not flag tool_spam for 10 calls/trip")
		}
	}
}

func TestAgentPolice_AccumulatesViolations(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 5000, 5000)

	cfg := DefaultCordonConfig()
	cfg.MaxTokens = 1000

	p := NewAgentPolice(cfg)
	p.Observe(m, nil)
	p.Observe(m, nil)

	report := p.Report()
	if len(report) < 2 {
		t.Fatalf("expected at least 2 accumulated violations, got %d", len(report))
	}
	if p.ViolationCount() != len(report) {
		t.Fatalf("ViolationCount = %d, Report len = %d", p.ViolationCount(), len(report))
	}
}

func TestAgentPolice_NilMetrics(t *testing.T) {
	p := NewAgentPolice(DefaultCordonConfig())
	violations := p.Observe(nil, nil)
	if violations != nil {
		t.Fatalf("nil metrics should produce nil violations, got %+v", violations)
	}
}

func TestAgentPolice_GateLoopViolation(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	for i := 0; i < 5; i++ {
		m.RecordGateResult(false)
	}

	cfg := DefaultCordonConfig()
	p := NewAgentPolice(cfg)
	violations := p.Observe(m, nil)

	found := false
	for _, v := range violations {
		if v.Kind == "gate_loop" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected gate_loop violation for 5 gate failures")
	}
}

func TestAgentPolice_CostBudgetViolation(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.mu.Lock()
	m.Cost = 5.0
	m.mu.Unlock()

	cfg := DefaultCordonConfig()
	cfg.MaxCost = 1.0

	p := NewAgentPolice(cfg)
	violations := p.Observe(m, nil)

	found := false
	for _, v := range violations {
		if v.Kind == "budget_exceeded" && v.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected budget_exceeded violation for cost")
	}
}

func TestAgentPolice_ReportReturnsCopy(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 5000, 5000)

	cfg := DefaultCordonConfig()
	cfg.MaxTokens = 1000

	p := NewAgentPolice(cfg)
	p.Observe(m, nil)

	report1 := p.Report()
	report2 := p.Report()

	if len(report1) != len(report2) {
		t.Fatalf("reports differ: %d vs %d", len(report1), len(report2))
	}

	// Mutating report1 should not affect report2 or internal state.
	if len(report1) > 0 {
		report1[0].Kind = "mutated"
		report3 := p.Report()
		if report3[0].Kind == "mutated" {
			t.Fatal("Report should return a copy, not a reference")
		}
	}
}
