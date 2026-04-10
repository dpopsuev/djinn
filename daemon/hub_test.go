package daemon

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/review"
	"github.com/dpopsuev/djinn/telemetry"
)

// --- HubCore nil-safety ---

func TestHubCore_NilSafe(t *testing.T) {
	var core HubCore
	// All methods must not panic with zero-value core.
	core.Trace("action", "detail")
	core.Emit(telemetry.Signal{Message: "test"})
	core.Render(DisplayMsg{Source: "test"})
}

func TestHubCore_Trace(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	core := HubCore{Tracer: ring.For(telemetry.ComponentTool)}
	core.Trace("build", "hub test")

	events := ring.Last(10)
	if len(events) == 0 {
		t.Fatal("expected trace event")
	}
	if events[0].Action != "build" {
		t.Errorf("action = %q, want %q", events[0].Action, "build")
	}
}

func TestHubCore_Emit(t *testing.T) {
	bus := telemetry.NewSignalBus()
	core := HubCore{Signals: bus}
	core.Emit(telemetry.Signal{Category: "test", Level: telemetry.Green, Source: "hub", Message: "ok"})

	signals := bus.Signals()
	if len(signals) != 1 {
		t.Fatalf("signals = %d, want 1", len(signals))
	}
	if signals[0].Category != "test" {
		t.Errorf("category = %q, want %q", signals[0].Category, "test")
	}
}

func TestHubCore_Render(t *testing.T) {
	spy := &spyDisplay{}
	core := HubCore{Display: spy}
	core.Render(DisplayMsg{Source: "plan", Category: "plan", Content: "payload"})

	if len(spy.msgs) != 1 {
		t.Fatalf("msgs = %d, want 1", len(spy.msgs))
	}
	if spy.msgs[0].Source != "plan" {
		t.Errorf("source = %q, want %q", spy.msgs[0].Source, "plan")
	}
}

// --- NopDisplaySender ---

func TestNopDisplaySender(t *testing.T) {
	nop := NopDisplaySender{}
	nop.Send(DisplayMsg{Source: "test"}) // must not panic
}

// --- HubRegistry ---

func TestHubRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubHub{name: "plan", phase: "plan"})
	reg.Register(&stubHub{name: "code", phase: "code"})

	h, ok := reg.Get("plan")
	if !ok || h.Name() != "plan" {
		t.Errorf("Get(plan) = %v, %v", h, ok)
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestHubRegistry_All(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubHub{name: "code", phase: "code"})
	reg.Register(&stubHub{name: "plan", phase: "plan"})

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("All() = %d, want 2", len(all))
	}
	// Sorted by name.
	if all[0].Name() != "code" || all[1].Name() != "plan" {
		t.Errorf("All() order: %s, %s", all[0].Name(), all[1].Name())
	}
}

func TestHubRegistry_ByPhase(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubHub{name: "plan", phase: "plan"})
	reg.Register(&stubHub{name: "code", phase: "code"})
	reg.Register(&stubHub{name: "tool", phase: "execute"})

	plan := reg.ByPhase("plan")
	if len(plan) != 1 {
		t.Errorf("ByPhase(plan) = %d, want 1", len(plan))
	}

	none := reg.ByPhase("deploy")
	if len(none) != 0 {
		t.Errorf("ByPhase(deploy) = %d, want 0", len(none))
	}
}

func TestHubRegistry_Names(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubHub{name: "code", phase: "code"})
	reg.Register(&stubHub{name: "plan", phase: "plan"})

	names := reg.Names()
	if len(names) != 2 || names[0] != "code" || names[1] != "plan" {
		t.Errorf("Names() = %v, want [code plan]", names)
	}
}

// --- PlanHub mediation ---

func TestPlanHub_AddSegment_Mediation(t *testing.T) {
	bus := telemetry.NewSignalBus()
	ring := telemetry.NewTraceProjection(100)
	spy := &spyDisplay{}
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Signals: bus,
		Display: spy,
	}

	ph := NewPlanHub(core, artifact.NewGraph("test", artifact.DefaultRegistry()))
	id := ph.AddSegment(artifact.Artifact{Title: "implement auth"})

	if id == "" {
		t.Fatal("AddSegment returned empty ID")
	}

	// Trace event recorded.
	events := ring.Last(10)
	found := false
	for _, e := range events {
		if e.Action == "segment-add" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected trace event with action 'segment-add'")
	}

	// Signal emitted.
	signals := bus.Signals()
	if len(signals) == 0 {
		t.Fatal("expected signal emission after AddSegment")
	}
	if signals[0].Category != "plan" {
		t.Errorf("signal category = %q, want %q", signals[0].Category, "plan")
	}

	// Display message sent.
	if len(spy.msgs) == 0 {
		t.Fatal("expected display message after AddSegment")
	}
	if spy.msgs[0].Source != "plan" {
		t.Errorf("display source = %q, want %q", spy.msgs[0].Source, "plan")
	}
}

func TestPlanHub_Complete_Mediation(t *testing.T) {
	bus := telemetry.NewSignalBus()
	core := HubCore{Signals: bus, Display: NopDisplaySender{}}
	graph := artifact.NewGraph("test", artifact.DefaultRegistry())
	ph := NewPlanHub(core, graph)

	id, _ := graph.Add(artifact.Artifact{Kind: artifact.KindPlanSegment, Title: "task", Status: artifact.StatusReady})
	if err := graph.Claim(id, "executor"); err != nil {
		t.Fatal(err)
	}
	if err := graph.Start(id); err != nil {
		t.Fatal(err)
	}

	if err := ph.Complete(id); err != nil {
		t.Fatal(err)
	}

	signals := bus.Signals()
	if len(signals) == 0 {
		t.Fatal("expected signal after Complete")
	}
}

func TestPlanHub_WithExternalPlanner(t *testing.T) {
	spy := &spyPlanner{}
	core := HubCore{Display: NopDisplaySender{}}
	ph := NewPlanHub(core, artifact.NewGraph("test", artifact.DefaultRegistry()))
	ph.Planner = spy

	ph.AddSegment(artifact.Artifact{Title: "synced segment"})

	if spy.syncCount != 1 {
		t.Errorf("syncCount = %d, want 1", spy.syncCount)
	}
}

// --- CodeHub mediation ---

func TestCodeHub_RecordChange_Mediation(t *testing.T) {
	ring := telemetry.NewTraceProjection(100)
	spy := &spyDisplay{}
	core := HubCore{
		Tracer:  ring.For(telemetry.ComponentTool),
		Display: spy,
	}

	window := review.NewReviewWindow("test request", nil)
	ch := NewCodeHub(core, window)

	ch.RecordChange("main.go")

	// Trace recorded.
	events := ring.Last(10)
	found := false
	for _, e := range events {
		if e.Action == "file-change" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected trace event with action 'file-change'")
	}

	// Display sent.
	if len(spy.msgs) == 0 {
		t.Fatal("expected display message after RecordChange")
	}
}

func TestCodeHub_CheckBudget_NilWindow(t *testing.T) {
	core := HubCore{Display: NopDisplaySender{}}
	ch := NewCodeHub(core, nil)

	exceeded, signals := ch.CheckBudget(context.Background())
	if exceeded {
		t.Error("nil window should not exceed budget")
	}
	if signals != nil {
		t.Error("nil window should return nil signals")
	}
}

// --- TeaDisplayAdapter nil-safety ---

func TestTeaDisplayAdapter_NilSafe(t *testing.T) {
	var adapter *TeaDisplayAdapter
	adapter.Send(DisplayMsg{Source: "test"}) // must not panic

	adapter2 := NewTeaDisplayAdapter(nil)
	adapter2.Send(DisplayMsg{Source: "test"}) // must not panic
}

// --- Port DTO roundtrips ---

func TestPlanSegmentDTO(t *testing.T) {
	dto := PlanSegmentDTO{
		ID:      "seg-1",
		Title:   "implement auth",
		Status:  "ready",
		Content: "acceptance criteria here",
	}
	if dto.ID != "seg-1" || dto.Title != "implement auth" {
		t.Errorf("DTO fields not set correctly: %+v", dto)
	}
}

func TestAnalysisResult(t *testing.T) {
	result := AnalysisResult{
		Components: []string{"hub", "signal"},
		Violations: []string{"cycle detected"},
		Graph:      map[string][]string{"hub": {"signal", "trace"}},
	}
	if len(result.Components) != 2 {
		t.Errorf("components = %d, want 2", len(result.Components))
	}
}

// TestGateResult removed — GateResult type moved to staff/ports.go (canonical location).

// --- Test helpers ---

type spyDisplay struct {
	msgs []DisplayMsg
}

func (s *spyDisplay) Send(msg DisplayMsg) {
	s.msgs = append(s.msgs, msg)
}

type stubHub struct {
	name  string
	phase string
}

func (h *stubHub) Name() string  { return h.name }
func (h *stubHub) Phase() string { return h.phase }

type spyPlanner struct {
	syncCount  int
	fetchCount int
}

func (p *spyPlanner) SyncPlan(_ context.Context, _ []PlanSegmentDTO) error {
	p.syncCount++
	return nil
}

func (p *spyPlanner) FetchPlan(_ context.Context) ([]PlanSegmentDTO, error) {
	p.fetchCount++
	return nil, nil
}
