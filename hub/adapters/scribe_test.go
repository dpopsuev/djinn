package adapters

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dpopsuev/djinn/hub"
)

func TestScribeAdapter_InterfaceCompliance(t *testing.T) {
	var _ hub.ExecutionPlannerPort = (*ScribeAdapter)(nil)
}

func TestScribeAdapter_Construction(t *testing.T) {
	adapter := NewScribeAdapter(nil, "scribe")
	if adapter.Server != "scribe" {
		t.Errorf("server = %q, want %q", adapter.Server, "scribe")
	}
}

func TestPlanSegmentDTO_JSONRoundTrip(t *testing.T) {
	dto := hub.PlanSegmentDTO{
		ID:      "seg-1",
		Title:   "implement auth",
		Status:  "ready",
		Content: "acceptance criteria",
	}

	data, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}

	var decoded hub.PlanSegmentDTO
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ID != dto.ID || decoded.Title != dto.Title {
		t.Errorf("roundtrip mismatch: %+v vs %+v", decoded, dto)
	}
}

func TestNopExecutionPlanner(t *testing.T) {
	nop := hub.NopExecutionPlanner{}

	if err := nop.SyncPlan(context.Background(), []hub.PlanSegmentDTO{{ID: "1"}}); err != nil {
		t.Errorf("SyncPlan error: %v", err)
	}

	dtos, err := nop.FetchPlan(context.Background())
	if err != nil {
		t.Errorf("FetchPlan error: %v", err)
	}
	if dtos != nil {
		t.Errorf("FetchPlan = %v, want nil", dtos)
	}
}
