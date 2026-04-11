package substrate

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/telemetry"
)

func TestAnalysisHub_Day1_InternalOnly(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	spy := &spyDisplay{}
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: telemetry.NewSignalBus(),
		Display: spy,
	}

	ah := NewAnalysisHub(core, "/tmp/test")
	result, err := ah.Analyze(context.Background(), []string{"hub", "signal"})
	if err != nil {
		t.Fatal(err)
	}

	// Day 1: components echo paths.
	if len(result.Components) != 2 { //nolint:mnd // expected 2 components
		t.Errorf("components = %d, want 2", len(result.Components))
	}

	// Trace recorded.
	events := ring.Last(10)
	found := false
	for _, e := range events {
		if e.Action == "analyze" || e.Action == "analyze_done" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected trace event for analyze")
	}

	// Display sent.
	if len(spy.msgs) == 0 {
		t.Error("expected display message")
	}
}

func TestAnalysisHub_Day2_PortDelegation(t *testing.T) {
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  telemetry.NewTraceProjection(100).For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}

	ah := NewAnalysisHub(core, "/tmp/test")
	ah.Analyzer = &stubAnalyzer{
		result: AnalysisResult{
			Components: []string{"hub", "signal", "trace"},
			Violations: []string{"hub -> tui (skip layer)"},
		},
	}

	result, err := ah.Analyze(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Components) != 3 { //nolint:mnd // expected 3 from stub
		t.Errorf("components = %d, want 3", len(result.Components))
	}
	if len(result.Violations) != 1 {
		t.Errorf("violations = %d, want 1", len(result.Violations))
	}

	// Violations should emit a yellow signal.
	signals := bus.Signals()
	found := false
	for _, s := range signals {
		if s.Level == telemetry.Yellow && s.Category == analysisHubName {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected yellow signal for violations")
	}
}

func TestAnalysisHub_AnalyzerError(t *testing.T) {
	bus := telemetry.NewSignalBus()
	core := HubCore{
		Tracer:  telemetry.NewTraceProjection(100).For(telemetry.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}

	ah := NewAnalysisHub(core, "/tmp/test")
	ah.Analyzer = &failingAnalyzer{}

	_, err := ah.Analyze(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error from failing analyzer")
	}

	// Should emit signal on error.
	signals := bus.Signals()
	if len(signals) == 0 {
		t.Error("expected signal on analysis error")
	}
}

// --- Test helpers ---

type stubAnalyzer struct {
	result AnalysisResult
}

func (a *stubAnalyzer) Analyze(_ context.Context, _ []string) (AnalysisResult, error) {
	return a.result, nil
}

type failingAnalyzer struct{}

func (a *failingAnalyzer) Analyze(_ context.Context, _ []string) (AnalysisResult, error) {
	return AnalysisResult{}, context.DeadlineExceeded
}
