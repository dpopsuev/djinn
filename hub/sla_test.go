package hub

import (
	"testing"
	"time"
)

func TestCheckSLA_AllMet(t *testing.T) {
	sla := ToolSLA{
		Name:        "plan",
		P50Target:   10 * time.Millisecond,
		P95Target:   50 * time.Millisecond,
		MaxErrorPct: 0.01,
	}
	result := CheckSLA(sla, 5*time.Millisecond, 30*time.Millisecond, 0.005)

	if !result.Overall {
		t.Error("expected Overall=true when all targets met")
	}
	if !result.P50Met || !result.P95Met || !result.ErrorMet {
		t.Errorf("individual checks: P50=%v P95=%v Error=%v", result.P50Met, result.P95Met, result.ErrorMet)
	}
}

func TestCheckSLA_P95Breach(t *testing.T) {
	sla := ToolSLA{
		Name:        "test",
		P50Target:   5 * time.Second,
		P95Target:   30 * time.Second,
		MaxErrorPct: 0.05,
	}
	result := CheckSLA(sla, 3*time.Second, 45*time.Second, 0.01)

	if result.Overall {
		t.Error("expected Overall=false when P95 exceeds target")
	}
	if !result.P50Met {
		t.Error("P50 should be met")
	}
	if result.P95Met {
		t.Error("P95 should NOT be met")
	}
}

func TestCheckSLA_ErrorBreach(t *testing.T) {
	sla := ToolSLA{
		Name:        "git",
		P50Target:   100 * time.Millisecond,
		P95Target:   1 * time.Second,
		MaxErrorPct: 0.01,
	}
	result := CheckSLA(sla, 50*time.Millisecond, 500*time.Millisecond, 0.05)

	if result.Overall {
		t.Error("expected Overall=false when error rate exceeds target")
	}
	if result.ErrorMet {
		t.Error("ErrorMet should be false since 0.05 > 0.01")
	}
}

func TestCheckSLA_ExactBoundary(t *testing.T) {
	sla := ToolSLA{
		Name:        "latency",
		P50Target:   5 * time.Millisecond,
		P95Target:   20 * time.Millisecond,
		MaxErrorPct: 0.01,
	}
	// Exactly at boundary — should pass (<=)
	result := CheckSLA(sla, 5*time.Millisecond, 20*time.Millisecond, 0.01)
	if !result.Overall {
		t.Error("expected Overall=true at exact boundary")
	}
}

func TestDefaultSLAs_ReasonableValues(t *testing.T) {
	for name, sla := range DefaultSLAs() {
		if sla.P50Target >= sla.P95Target {
			t.Errorf("%s: P50 (%v) should be < P95 (%v)", name, sla.P50Target, sla.P95Target)
		}
		if sla.MaxErrorPct <= 0 {
			t.Errorf("%s: MaxErrorPct should be > 0, got %f", name, sla.MaxErrorPct)
		}
		if sla.MaxErrorPct > 0.1 {
			t.Errorf("%s: MaxErrorPct should be <= 10%%, got %f", name, sla.MaxErrorPct)
		}
		if sla.Name != name {
			t.Errorf("SLA Name = %q, map key = %q", sla.Name, name)
		}
	}
}

func TestDefaultSLAs_AllToolsCovered(t *testing.T) {
	slas := DefaultSLAs()
	expected := []string{"plan", "test", "git", "arch", "discourse", "reconcile", "latency", "render"}

	for _, name := range expected {
		if _, ok := slas[name]; !ok {
			t.Errorf("missing SLA for tool %q", name)
		}
	}
}
