package staff

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/tools"
)

func TestDetectBottlenecks_ToolLatency(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)

	lat := tools.NewToolLatencyTracker()
	// Add a slow tool with p95 > 2s.
	for i := 0; i < 20; i++ {
		lat.Record("SlowTool", 3*time.Second)
	}

	bottlenecks := DetectBottlenecks(m, lat)
	found := false
	for _, b := range bottlenecks {
		if b.Kind == BottleneckToolLatency {
			found = true
		}
	}
	if !found {
		t.Fatal("expected BottleneckToolLatency for slow tool")
	}
}

func TestDetectBottlenecks_NoToolLatency(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)

	lat := tools.NewToolLatencyTracker()
	for i := 0; i < 20; i++ {
		lat.Record("FastTool", 100*time.Millisecond)
	}

	bottlenecks := DetectBottlenecks(m, lat)
	for _, b := range bottlenecks {
		if b.Kind == BottleneckToolLatency {
			t.Fatal("should not detect tool latency for fast tool")
		}
	}
}

func TestDetectBottlenecks_RateLimit(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	// Normal TTFTs then a spike.
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)
	m.RecordRoundTrip(5*time.Second, 6*time.Second, 100, 50) // 50x spike

	bottlenecks := DetectBottlenecks(m, nil)
	found := false
	for _, b := range bottlenecks {
		if b.Kind == BottleneckRateLimit {
			found = true
		}
	}
	if !found {
		t.Fatal("expected BottleneckRateLimit for TTFT spike")
	}
}

func TestDetectBottlenecks_GateLoop(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)
	// 4 gate failures (> 3 threshold)
	for i := 0; i < 4; i++ {
		m.RecordGateResult(false)
	}

	bottlenecks := DetectBottlenecks(m, nil)
	found := false
	for _, b := range bottlenecks {
		if b.Kind == BottleneckGateLoop {
			found = true
		}
	}
	if !found {
		t.Fatal("expected BottleneckGateLoop for repeated failures")
	}
}

func TestDetectBottlenecks_NoGateLoop_BelowThreshold(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)
	// Only 2 failures (below threshold of 3)
	m.RecordGateResult(false)
	m.RecordGateResult(false)

	bottlenecks := DetectBottlenecks(m, nil)
	for _, b := range bottlenecks {
		if b.Kind == BottleneckGateLoop {
			t.Fatal("should not detect gate loop with only 2 failures")
		}
	}
}

func TestDetectBottlenecks_IdleWait(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	// TTFT is 60% of RTT → idle waiting.
	m.RecordRoundTrip(600*time.Millisecond, time.Second, 100, 50)
	m.RecordRoundTrip(600*time.Millisecond, time.Second, 100, 50)

	bottlenecks := DetectBottlenecks(m, nil)
	found := false
	for _, b := range bottlenecks {
		if b.Kind == BottleneckIdleWait {
			found = true
		}
	}
	if !found {
		t.Fatal("expected BottleneckIdleWait when TTFT/RTT > 50%")
	}
}

func TestDetectBottlenecks_ToolSpam(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)

	lat := tools.NewToolLatencyTracker()
	// 25 tool calls in 1 round-trip (> 20 threshold).
	for i := 0; i < 25; i++ {
		lat.Record("Read", 50*time.Millisecond)
	}

	bottlenecks := DetectBottlenecks(m, lat)
	found := false
	for _, b := range bottlenecks {
		if b.Kind == BottleneckToolSpam {
			found = true
		}
	}
	if !found {
		t.Fatal("expected BottleneckToolSpam for 25 calls/trip")
	}
}

func TestDetectBottlenecks_NilMetrics(t *testing.T) {
	bottlenecks := DetectBottlenecks(nil, nil)
	if len(bottlenecks) != 0 {
		t.Fatal("nil metrics should produce no bottlenecks")
	}
}

func TestDetectBottlenecks_NilLatency(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)

	// Should not panic with nil latency.
	bottlenecks := DetectBottlenecks(m, nil)
	_ = bottlenecks
}
