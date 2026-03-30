package hub

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tools"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/trace"
)

func TestToolHub_Execute_Mediation(t *testing.T) {
	ring := trace.NewRing(100)
	bus := signal.NewSignalBus()
	spy := &spyDisplay{}
	core := HubCore{
		Tracer:  ring.For(trace.ComponentTool),
		Signals: bus,
		Display: spy,
	}

	executor := &stubExecutor{result: "ok"}
	tracker := tools.NewToolLatencyTracker()
	th := NewToolHub(core, executor, tracker)

	result, err := th.Execute(context.Background(), "plan", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}

	// Trace events recorded.
	events := ring.Last(10)
	if len(events) == 0 {
		t.Error("expected trace events")
	}

	// Latency tracked.
	if tracker.Count("plan") != 1 {
		t.Errorf("tracker.Count(plan) = %d, want 1", tracker.Count("plan"))
	}

	// Display sent.
	if len(spy.msgs) == 0 {
		t.Error("expected display message")
	}
}

func TestToolHub_Execute_SLABreach(t *testing.T) {
	bus := signal.NewSignalBus()
	core := HubCore{
		Tracer:  trace.NewRing(100).For(trace.ComponentTool),
		Signals: bus,
		Display: NopDisplaySender{},
	}

	// Executor that always takes 200ms (breaches P95=50ms for "plan").
	executor := &slowExecutor{delay: 200 * time.Millisecond, result: "ok"} //nolint:mnd // test value
	tracker := tools.NewToolLatencyTracker()
	th := NewToolHub(core, executor, tracker)

	for range 3 { //nolint:mnd // warm up tracker
		th.Execute(context.Background(), "plan", json.RawMessage(`{}`)) //nolint:errcheck // test
	}

	// Should have emitted yellow signals for SLA breach.
	signals := bus.Signals()
	found := false
	for _, s := range signals {
		if s.Level == signal.Yellow && s.Category == toolHubName {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected yellow signal for SLA breach")
	}
}

func TestToolHub_ImplementsToolExecutor(t *testing.T) {
	var _ builtin.ToolExecutor = (*ToolHub)(nil)
}

func TestToolHub_All(t *testing.T) {
	executor := &stubExecutor{
		tools: []builtin.Tool{&stubTool{name: "read"}, &stubTool{name: "write"}},
	}
	th := NewToolHub(HubCore{Display: NopDisplaySender{}}, executor, nil)

	all := th.All()
	if len(all) != 2 { //nolint:mnd // expected 2 tools
		t.Errorf("All() = %d, want 2", len(all))
	}
}

func TestToolHub_Names(t *testing.T) {
	executor := &stubExecutor{
		names: []string{"read", "write"},
	}
	th := NewToolHub(HubCore{Display: NopDisplaySender{}}, executor, nil)

	names := th.Names()
	if len(names) != 2 { //nolint:mnd // expected 2 names
		t.Errorf("Names() = %d, want 2", len(names))
	}
}

// --- Test helpers ---

type stubExecutor struct {
	result string
	tools  []builtin.Tool
	names  []string
}

func (e *stubExecutor) Execute(_ context.Context, _ string, _ json.RawMessage) (string, error) {
	return e.result, nil
}

func (e *stubExecutor) All() []builtin.Tool { return e.tools }
func (e *stubExecutor) Names() []string     { return e.names }

type slowExecutor struct {
	delay  time.Duration
	result string
}

func (e *slowExecutor) Execute(_ context.Context, _ string, _ json.RawMessage) (string, error) {
	time.Sleep(e.delay)
	return e.result, nil
}

func (e *slowExecutor) All() []builtin.Tool { return nil }
func (e *slowExecutor) Names() []string     { return nil }

type stubTool struct {
	name string
}

func (t *stubTool) Name() string                                                 { return t.name }
func (t *stubTool) Description() string                                          { return "" }
func (t *stubTool) InputSchema() json.RawMessage                                 { return nil }
func (t *stubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return "", nil }
