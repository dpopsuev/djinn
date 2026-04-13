package substrate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools"
	"github.com/dpopsuev/djinn/uniform/control"
	"github.com/dpopsuev/djinn/uniform/quality"
)

func TestMetricsHandler_RecordsRoundTrips(t *testing.T) {
	metrics := quality.NewAgentMetrics("test-agent", "developer")
	police := quality.NewAgentPolice(quality.DefaultCordonConfig())
	bus := telemetry.NewSignalBus()
	latency := tools.NewToolLatencyTracker()

	h := NewMetricsHandler(metrics, police, bus, latency, quality.DefaultCordonConfig())
	h.StartTurn()

	h.OnText("hello")
	h.OnDone(&driver.Usage{InputTokens: 100, OutputTokens: 50})

	if metrics.RoundTrips != 1 {
		t.Fatalf("RoundTrips = %d, want 1", metrics.RoundTrips)
	}
	if metrics.TotalIn != 100 || metrics.TotalOut != 50 {
		t.Fatalf("tokens = %d/%d, want 100/50", metrics.TotalIn, metrics.TotalOut)
	}
}

func TestMetricsHandler_PoliceEmitsBudgetViolation(t *testing.T) {
	cfg := quality.CordonConfig{MaxTokens: 100}
	metrics := quality.NewAgentMetrics("test-agent", "developer")
	police := quality.NewAgentPolice(cfg)
	bus := telemetry.NewSignalBus()
	latency := tools.NewToolLatencyTracker()

	h := NewMetricsHandler(metrics, police, bus, latency, cfg)

	// Simulate a round-trip that blows the budget.
	h.StartTurn()
	h.OnDone(&driver.Usage{InputTokens: 80, OutputTokens: 80})

	// Police should have detected budget_exceeded and emitted to bus.
	signals := bus.Signals()
	found := false
	for _, s := range signals {
		if s.Category == telemetry.CategoryBudget && s.Level >= telemetry.Yellow {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected budget violation signal on bus, got none")
	}

	if police.ViolationCount() == 0 {
		t.Fatal("expected police to record violation")
	}
}

func TestMetricsHandler_CordonEmitsBlackSignal(t *testing.T) {
	cfg := quality.CordonConfig{MaxTokens: 50}
	metrics := quality.NewAgentMetrics("test-agent", "developer")
	police := quality.NewAgentPolice(cfg)
	bus := telemetry.NewSignalBus()

	h := NewMetricsHandler(metrics, police, bus, nil, cfg)

	h.StartTurn()
	h.OnDone(&driver.Usage{InputTokens: 100, OutputTokens: 100})

	// Cordon should emit Black level signal.
	signals := bus.Signals()
	found := false
	for _, s := range signals {
		if s.Level == telemetry.Black {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected Black cordon signal, got none")
	}
}

func TestMetricsHandler_InterpreterReceivesSignal(t *testing.T) {
	cfg := quality.CordonConfig{MaxTokens: 50}
	metrics := quality.NewAgentMetrics("test-agent", "developer")
	police := quality.NewAgentPolice(cfg)
	bus := telemetry.NewSignalBus()

	h := NewMetricsHandler(metrics, police, bus, nil, cfg)

	// Wire interpreter with a stub GenSec.
	stub := &stubGenSec{response: `{"action":"throttle","reason":"budget","confidence":0.9}`}
	interpreter := control.NewSignalInterpreter(bus, stub)
	interpreter.Start(context.Background())

	// Blow budget — should flow through pipeline.
	h.StartTurn()
	h.OnDone(&driver.Usage{InputTokens: 100, OutputTokens: 100})

	// Give interpreter time to process.
	time.Sleep(50 * time.Millisecond)
	interpreter.Stop()

	// Interpreter should have audit entries.
	entries := interpreter.AuditEntries()
	if len(entries) == 0 {
		t.Fatal("expected interpreter to record audit entries from budget violation")
	}

	// GenSec should have been asked.
	if stub.asked == 0 {
		t.Fatal("expected GenSec to be asked about yellow+ signal")
	}

	// Decision should be parsed.
	if entries[0].Decision.Action != control.ActionThrottle {
		t.Fatalf("decision action = %q, want throttle", entries[0].Decision.Action)
	}
}

func TestMetricsHandler_ErrorEmitsRedSignal(t *testing.T) {
	metrics := quality.NewAgentMetrics("test-agent", "developer")
	police := quality.NewAgentPolice(quality.DefaultCordonConfig())
	bus := telemetry.NewSignalBus()

	h := NewMetricsHandler(metrics, police, bus, nil, quality.DefaultCordonConfig())
	h.OnError(errors.New("no choices in response"))

	signals := bus.Signals()
	if len(signals) == 0 {
		t.Fatal("expected error signal")
	}
	if signals[0].Level != telemetry.Red {
		t.Fatalf("level = %v, want Red", signals[0].Level)
	}
}

// stubGenSec implements Asker for testing.
type stubGenSec struct {
	response string
	asked    int
}

func (s *stubGenSec) Ask(_ context.Context, _ string) (string, error) {
	s.asked++
	return s.response, nil
}
