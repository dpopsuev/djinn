package telemetry

import (
	"testing"
	"time"
)

func TestAnalyzeConsecutiveErrors(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// Seed 4 consecutive errors for server="locus", tool="scan".
	for i := range 4 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "scan",
			Error:     true,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour // wide window to capture all events

	alerts := Analyze(r, cfg)

	// After the 3rd error and the 4th error, consecutive_errors alerts should fire.
	found := false
	for _, a := range alerts {
		if a.Pattern == "consecutive_errors" && a.Server == "locus" && a.Tool == "scan" {
			found = true
			if a.Severity != SeverityError {
				t.Errorf("severity = %d, want SeverityError (%d)", a.Severity, SeverityError)
			}
			if len(a.Evidence) == 0 {
				t.Error("evidence should contain event IDs")
			}
		}
	}
	if !found {
		t.Errorf("expected consecutive_errors alert for locus/scan, got %d alerts: %+v", len(alerts), alerts)
	}
}

func TestAnalyzeConsecutiveErrorsResetOnSuccess(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// 2 errors, then a success, then 2 more errors — should NOT trigger (threshold=3).
	events := []struct {
		err bool
	}{
		{true}, {true}, {false}, {true}, {true},
	}
	for i, e := range events {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "scan",
			Error:     e.err,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour

	alerts := Analyze(r, cfg)
	for _, a := range alerts {
		if a.Pattern == "consecutive_errors" {
			t.Error("success should have reset the streak, no consecutive_errors alert expected")
		}
	}
}

func TestAnalyzeBelowThreshold(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// Only 2 consecutive errors (threshold is 3).
	for i := range 2 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "scan",
			Error:     true,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour

	alerts := Analyze(r, cfg)
	for _, a := range alerts {
		if a.Pattern == "consecutive_errors" {
			t.Error("2 errors should be below threshold of 3")
		}
	}
}

func TestAnalyzeErrorRate(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// 10 calls: 5 errors, 5 successes = 50% error rate (threshold=20%).
	for i := range 10 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.put",
			Error:     i < 5,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour

	alerts := Analyze(r, cfg)
	found := false
	for _, a := range alerts {
		if a.Pattern == "error_rate" && a.Server == "scribe" && a.Tool == "artifact.put" {
			found = true
			if a.Severity != SeverityWarning {
				t.Errorf("severity = %d, want SeverityWarning (%d)", a.Severity, SeverityWarning)
			}
		}
	}
	if !found {
		t.Errorf("expected error_rate alert, got %d alerts: %+v", len(alerts), alerts)
	}
}

func TestAnalyzeErrorRateBelowMinCalls(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// Only 4 calls (minimum is 5) — all errors, but still shouldn't fire.
	for i := range 4 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.put",
			Error:     true,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour

	alerts := Analyze(r, cfg)
	for _, a := range alerts {
		if a.Pattern == "error_rate" {
			t.Error("should not fire error_rate with fewer than 5 calls")
		}
	}
}

func TestAnalyzeLatencySpike(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// First half (baseline): 6 events with ~10ms latency.
	for i := range 6 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "codograph.scan",
			Latency:   10 * time.Millisecond,
		})
	}

	// Second half (recent): 6 events with ~100ms latency (10x spike, threshold is 2x).
	for i := range 6 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(6+i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "codograph.scan",
			Latency:   100 * time.Millisecond,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour

	alerts := Analyze(r, cfg)
	found := false
	for _, a := range alerts {
		if a.Pattern == "latency_spike" && a.Server == "locus" && a.Tool == "codograph.scan" {
			found = true
			if a.Severity != SeverityWarning {
				t.Errorf("severity = %d, want SeverityWarning", a.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected latency_spike alert, got %d alerts: %+v", len(alerts), alerts)
	}
}

func TestAnalyzeLatencyNoSpike(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// 12 events, all with ~10ms latency — no spike.
	for i := range 12 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "codograph.scan",
			Latency:   10 * time.Millisecond,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour

	alerts := Analyze(r, cfg)
	for _, a := range alerts {
		if a.Pattern == "latency_spike" {
			t.Error("uniform latency should not trigger spike alert")
		}
	}
}

func TestAnalyzeCleanRing(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// All successful, normal latency.
	for i := range 10 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call" + ActionDoneSuffix,
			Server:    "locus",
			Tool:      "scan",
			Latency:   10 * time.Millisecond,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour

	alerts := Analyze(r, cfg)
	if len(alerts) != 0 {
		t.Errorf("clean ring should produce 0 alerts, got %d: %+v", len(alerts), alerts)
	}
}

func TestAnalyzeNilRing(t *testing.T) {
	cfg := DefaultHealthConfig()
	alerts := Analyze(nil, cfg)
	if alerts != nil {
		t.Errorf("nil ring should return nil alerts, got %d", len(alerts))
	}
}

func TestAnalyzeEmptyRing(t *testing.T) {
	r := NewTraceProjection(10)
	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour
	alerts := Analyze(r, cfg)
	if len(alerts) != 0 {
		t.Errorf("empty ring should return 0 alerts, got %d", len(alerts))
	}
}

func TestDefaultHealthConfig(t *testing.T) {
	cfg := DefaultHealthConfig()

	if cfg.ConsecutiveErrors != DefaultConsecutiveErrors {
		t.Errorf("ConsecutiveErrors = %d, want %d", cfg.ConsecutiveErrors, DefaultConsecutiveErrors)
	}
	if cfg.LatencyMultiplier != DefaultLatencyMultiplier {
		t.Errorf("LatencyMultiplier = %f, want %f", cfg.LatencyMultiplier, DefaultLatencyMultiplier)
	}
	if cfg.ErrorRatePercent != DefaultErrorRatePercent {
		t.Errorf("ErrorRatePercent = %d, want %d", cfg.ErrorRatePercent, DefaultErrorRatePercent)
	}
	if cfg.Window != 5*time.Minute {
		t.Errorf("Window = %v, want 5m", cfg.Window)
	}
}

func TestAnalyzeSkipsNonCallDoneEvents(t *testing.T) {
	r := NewTraceProjection(100)
	now := time.Now()

	// Add events with action="call" (not "call_done") — should be ignored.
	for i := range 10 {
		r.Append(TraceEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Component: ComponentMCP,
			Action:    "call",
			Server:    "locus",
			Tool:      "scan",
			Error:     true,
		})
	}

	cfg := DefaultHealthConfig()
	cfg.Window = time.Hour

	alerts := Analyze(r, cfg)
	for _, a := range alerts {
		if a.Pattern == "consecutive_errors" || a.Pattern == "error_rate" {
			t.Errorf("events without ActionDoneSuffix should be skipped, got alert: %+v", a)
		}
	}
}
