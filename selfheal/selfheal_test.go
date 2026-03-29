package selfheal

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/trace"
)

func TestGateValidate_ErrorRateImproved(t *testing.T) {
	// Before: 3 errors out of 10.
	beforeRing := trace.NewRing(100)
	for i := range 10 {
		beforeRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Error:     i < 3,
			Latency:   10 * time.Millisecond,
		})
	}
	beforeArchive := trace.Export(beforeRing, "")

	// After: 1 error out of 10.
	afterRing := trace.NewRing(100)
	for i := range 10 {
		afterRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Error:     i < 1,
			Latency:   10 * time.Millisecond,
		})
	}

	result := Validate(beforeArchive, afterRing)
	if result.Verdict != GatePass {
		t.Errorf("verdict = %s, want pass (error rate improved)", result.Verdict)
	}
}

func TestGateValidate_NewErrors(t *testing.T) {
	beforeRing := trace.NewRing(100)
	for range 10 {
		beforeRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Latency:   10 * time.Millisecond,
		})
	}
	beforeArchive := trace.Export(beforeRing, "")

	// After: new errors on a different tool.
	afterRing := trace.NewRing(100)
	for range 5 {
		afterRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "locus",
			Tool:      "codograph.scan",
			Error:     true,
			Latency:   10 * time.Millisecond,
		})
	}

	result := Validate(beforeArchive, afterRing)
	if result.Verdict != GateFail {
		t.Errorf("verdict = %s, want fail (new errors on locus)", result.Verdict)
	}
}

func TestGateValidate_InsufficientData(t *testing.T) {
	beforeRing := trace.NewRing(100)
	beforeRing.Append(trace.TraceEvent{Component: trace.ComponentMCP})
	beforeArchive := trace.Export(beforeRing, "")

	afterRing := trace.NewRing(100)
	afterRing.Append(trace.TraceEvent{Component: trace.ComponentMCP})

	result := Validate(beforeArchive, afterRing)
	if result.Verdict != GateUnsure {
		t.Errorf("verdict = %s, want unsure (too few events)", result.Verdict)
	}
}

func TestOrchestratorCircuitBreaker(t *testing.T) {
	ring := trace.NewRing(100)
	harness := &Harness{} // nil worktrees — will fail but that's OK for this test
	orch := NewOrchestrator(ring, harness)

	// Exhaust the circuit breaker.
	for range MaxAttemptsPerHour {
		orch.mu.Lock()
		orch.attempts = append(orch.attempts, time.Now())
		orch.mu.Unlock()
	}

	if orch.CanAttempt() {
		t.Error("circuit breaker should be tripped after max attempts")
	}
}

func TestOrchestratorCircuitBreakerExpires(t *testing.T) {
	ring := trace.NewRing(100)
	harness := &Harness{}
	orch := NewOrchestrator(ring, harness)

	// Add old attempts (> 1 hour ago).
	oldTime := time.Now().Add(-2 * time.Hour) //nolint:mnd // 2 hours ago
	orch.mu.Lock()
	for range MaxAttemptsPerHour {
		orch.attempts = append(orch.attempts, oldTime)
	}
	orch.mu.Unlock()

	if !orch.CanAttempt() {
		t.Error("circuit breaker should allow attempts after cooldown")
	}
}

func TestHealthAnalyzerConsecutiveErrors(t *testing.T) {
	ring := trace.NewRing(100)
	for range 5 {
		ring.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Error:     true,
		})
	}

	alerts := trace.Analyze(ring, trace.DefaultHealthConfig())
	found := false
	for _, a := range alerts {
		if a.Pattern == "consecutive_errors" && a.Server == "scribe" {
			found = true
			break
		}
	}
	if !found {
		t.Error("should detect 5 consecutive errors on scribe")
	}
}

func TestHealthAnalyzerNoAlerts(t *testing.T) {
	ring := trace.NewRing(100)
	for range 10 {
		ring.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Latency:   10 * time.Millisecond,
		})
	}

	alerts := trace.Analyze(ring, trace.DefaultHealthConfig())
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts for healthy ring, got %d", len(alerts))
	}
}

func TestTraceArchiveDiff(t *testing.T) {
	beforeRing := trace.NewRing(100)
	for range 10 {
		beforeRing.Append(trace.TraceEvent{
			Server:  "scribe",
			Tool:    "artifact.list",
			Latency: 50 * time.Millisecond,
		})
	}
	before := trace.Export(beforeRing, "")

	afterRing := trace.NewRing(100)
	for range 10 {
		afterRing.Append(trace.TraceEvent{
			Server:  "scribe",
			Tool:    "artifact.list",
			Latency: 20 * time.Millisecond,
		})
	}
	after := trace.Export(afterRing, "")

	diff := trace.Diff(before, after)
	if diff.EventCountBefore != 10 || diff.EventCountAfter != 10 {
		t.Errorf("counts = %d/%d, want 10/10", diff.EventCountBefore, diff.EventCountAfter)
	}
	if len(diff.LatencyDeltas) == 0 {
		t.Error("should have latency deltas for scribe")
	}
}
