package staff

import (
	"sync"
	"testing"
	"time"
)

func TestAgentMetrics_RecordRoundTrip(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")

	m.RecordRoundTrip(100*time.Millisecond, 2*time.Second, 500, 200)
	m.RecordRoundTrip(150*time.Millisecond, 3*time.Second, 600, 300)

	if m.RoundTrips != 2 {
		t.Fatalf("RoundTrips = %d, want 2", m.RoundTrips)
	}
	if m.TotalIn != 1100 {
		t.Fatalf("TotalIn = %d, want 1100", m.TotalIn)
	}
	if m.TotalOut != 500 {
		t.Fatalf("TotalOut = %d, want 500", m.TotalOut)
	}
	if len(m.TTFT) != 2 {
		t.Fatalf("TTFT samples = %d, want 2", len(m.TTFT))
	}
	if len(m.RTT) != 2 {
		t.Fatalf("RTT samples = %d, want 2", len(m.RTT))
	}
}

func TestAgentMetrics_GatePassRate(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")

	// No gates yet — should be 0.
	if rate := m.GatePassRate(); rate != 0 {
		t.Fatalf("empty GatePassRate = %f, want 0", rate)
	}

	m.RecordGateResult(true)
	m.RecordGateResult(true)
	m.RecordGateResult(false)

	rate := m.GatePassRate()
	// 2/3 = 0.6666...
	if rate < 0.66 || rate > 0.67 {
		t.Fatalf("GatePassRate = %f, want ~0.667", rate)
	}
}

func TestAgentMetrics_GatePassRate_AllPass(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordGateResult(true)
	m.RecordGateResult(true)

	if rate := m.GatePassRate(); rate != 1.0 {
		t.Fatalf("all pass rate = %f, want 1.0", rate)
	}
}

func TestAgentMetrics_AvgTTFT(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")

	// No samples — should be 0.
	if avg := m.AvgTTFT(); avg != 0 {
		t.Fatalf("empty AvgTTFT = %v, want 0", avg)
	}

	m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)
	m.RecordRoundTrip(200*time.Millisecond, time.Second, 100, 50)

	avg := m.AvgTTFT()
	if avg != 150*time.Millisecond {
		t.Fatalf("AvgTTFT = %v, want 150ms", avg)
	}
}

func TestAgentMetrics_AvgRTT(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	m.RecordRoundTrip(100*time.Millisecond, 2*time.Second, 100, 50)
	m.RecordRoundTrip(100*time.Millisecond, 4*time.Second, 100, 50)

	avg := m.AvgRTT()
	if avg != 3*time.Second {
		t.Fatalf("AvgRTT = %v, want 3s", avg)
	}
}

func TestAgentMetrics_TPM(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")

	// No data — should be 0.
	if tpm := m.TPM(); tpm != 0 {
		t.Fatalf("empty TPM = %f, want 0", tpm)
	}

	// Record a round-trip: 1000 tokens in 1 second = 60000 tpm.
	m.RecordRoundTrip(50*time.Millisecond, time.Second, 500, 500)
	tpm := m.TPM()
	if tpm < 50000 || tpm > 70000 {
		t.Fatalf("TPM = %f, want ~60000", tpm)
	}
}

func TestAgentMetrics_ConcurrentAccess(t *testing.T) {
	m := NewAgentMetrics("agent-1", "executor")
	var wg sync.WaitGroup

	// Hammer from multiple goroutines.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.RecordRoundTrip(100*time.Millisecond, time.Second, 100, 50)
			m.RecordGateResult(true)
			_ = m.GatePassRate()
			_ = m.AvgTTFT()
			_ = m.AvgRTT()
			_ = m.TPM()
		}()
	}
	wg.Wait()

	if m.RoundTrips != 50 {
		t.Fatalf("RoundTrips = %d, want 50 after concurrent writes", m.RoundTrips)
	}
}

func TestAgentMetrics_NewHasCorrectIdentity(t *testing.T) {
	m := NewAgentMetrics("agent-42", "inspector")
	if m.AgentID != "agent-42" {
		t.Fatalf("AgentID = %q", m.AgentID)
	}
	if m.Role != "inspector" {
		t.Fatalf("Role = %q", m.Role)
	}
}
