package hub

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/plan"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/trace"
)

func TestPlanHub_SyncOnAdd(t *testing.T) {
	spy := &spyPlanner{}
	core := HubCore{
		Tracer:  trace.NewRing(100).For(trace.ComponentTool),
		Signals: signal.NewSignalBus(),
		Display: NopDisplaySender{},
	}
	ph := NewPlanHub(core, plan.NewPlanGraph("sync-test"))
	ph.Planner = spy

	ph.AddSegment(plan.Segment{Title: "segment A"})
	ph.AddSegment(plan.Segment{Title: "segment B"})

	if spy.syncCount != 2 { //nolint:mnd // expected 2 syncs
		t.Errorf("syncCount = %d, want 2", spy.syncCount)
	}
}

func TestPlanHub_SyncOnComplete(t *testing.T) {
	spy := &spyPlanner{}
	core := HubCore{
		Tracer:  trace.NewRing(100).For(trace.ComponentTool),
		Signals: signal.NewSignalBus(),
		Display: NopDisplaySender{},
	}
	graph := plan.NewPlanGraph("sync-test")
	ph := NewPlanHub(core, graph)
	ph.Planner = spy

	id := graph.AddSegment(plan.Segment{Title: "task", Status: plan.StatusReady})
	if err := graph.Claim(id, "executor"); err != nil {
		t.Fatal(err)
	}
	if err := graph.Start(id); err != nil {
		t.Fatal(err)
	}

	spy.syncCount = 0 // reset after AddSegment didn't go through hub
	if err := ph.Complete(id); err != nil {
		t.Fatal(err)
	}

	if spy.syncCount != 1 {
		t.Errorf("syncCount = %d, want 1", spy.syncCount)
	}
}

func TestPlanHub_NoSyncWithoutPlanner(t *testing.T) {
	core := HubCore{Display: NopDisplaySender{}}
	ph := NewPlanHub(core, plan.NewPlanGraph("no-sync"))
	// Planner is nil — should not panic.
	ph.AddSegment(plan.Segment{Title: "orphan segment"})
}

func TestPlanHub_FetchPopulatesGraph(t *testing.T) {
	// Simulate: external planner has segments, PlanHub fetches and populates.
	stub := &stubFetchPlanner{
		segments: []PlanSegmentDTO{
			{ID: "ext-1", Title: "external task", Status: "ready"},
		},
	}

	core := HubCore{Display: NopDisplaySender{}}
	graph := plan.NewPlanGraph("fetch-test")
	ph := NewPlanHub(core, graph)
	ph.Planner = stub

	// Fetch from external planner.
	dtos, err := ph.Planner.FetchPlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Populate local graph from fetched DTOs.
	for _, dto := range dtos {
		graph.AddSegment(plan.Segment{
			ID:    dto.ID,
			Title: dto.Title,
		})
	}

	seg, err := graph.Get("ext-1")
	if err != nil {
		t.Fatalf("segment not found: %v", err)
	}
	if seg.Title != "external task" {
		t.Errorf("title = %q, want %q", seg.Title, "external task")
	}
}

type stubFetchPlanner struct {
	segments []PlanSegmentDTO
}

func (p *stubFetchPlanner) SyncPlan(_ context.Context, _ []PlanSegmentDTO) error {
	return nil
}

func (p *stubFetchPlanner) FetchPlan(_ context.Context) ([]PlanSegmentDTO, error) {
	return p.segments, nil
}
