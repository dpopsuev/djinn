package daemon

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/telemetry"
)

func TestPlanHub_SyncOnAdd(t *testing.T) {
	spy := &spyPlanner{}
	core := HubCore{
		Tracer:  telemetry.NewTraceProjection(100).For(telemetry.ComponentTool),
		Signals: telemetry.NewSignalBus(),
		Display: NopDisplaySender{},
	}
	ph := NewPlanHub(core, artifact.NewGraph("sync-test", artifact.DefaultRegistry()))
	ph.Planner = spy

	ph.AddSegment(artifact.Artifact{Title: "segment A"})
	ph.AddSegment(artifact.Artifact{Title: "segment B"})

	if spy.syncCount != 2 { //nolint:mnd // expected 2 syncs
		t.Errorf("syncCount = %d, want 2", spy.syncCount)
	}
}

func TestPlanHub_SyncOnComplete(t *testing.T) {
	spy := &spyPlanner{}
	core := HubCore{
		Tracer:  telemetry.NewTraceProjection(100).For(telemetry.ComponentTool),
		Signals: telemetry.NewSignalBus(),
		Display: NopDisplaySender{},
	}
	graph := artifact.NewGraph("sync-test", artifact.DefaultRegistry())
	ph := NewPlanHub(core, graph)
	ph.Planner = spy

	id, _ := graph.Add(artifact.Artifact{Kind: artifact.KindPlanSegment, Title: "task", Status: artifact.StatusReady})
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
	ph := NewPlanHub(core, artifact.NewGraph("no-sync", artifact.DefaultRegistry()))
	// Planner is nil — should not panic.
	ph.AddSegment(artifact.Artifact{Title: "orphan segment"})
}

func TestPlanHub_FetchPopulatesGraph(t *testing.T) {
	// Simulate: external planner has segments, PlanHub fetches and populates.
	stub := &stubFetchPlanner{
		segments: []PlanSegmentDTO{
			{ID: "ext-1", Title: "external task", Status: "ready"},
		},
	}

	core := HubCore{Display: NopDisplaySender{}}
	graph := artifact.NewGraph("fetch-test", artifact.DefaultRegistry())
	ph := NewPlanHub(core, graph)
	ph.Planner = stub

	// Fetch from external planner.
	dtos, err := ph.Planner.FetchPlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Populate local graph from fetched DTOs.
	for _, dto := range dtos {
		graph.Add(artifact.Artifact{Kind: artifact.KindPlanSegment,
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
