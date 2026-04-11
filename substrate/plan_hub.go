// plan_hub.go — PlanHub mediates planning operations (GOL-58).
//
// Wraps artifact.Graph with five-step mediation: execute -> trace -> signal -> render -> sync.
// Day 1: internal artifact.Graph only. Day 2: ExecutionPlannerPort for external sync.
package substrate

import (
	"context"

	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/telemetry"
)

// Hub and phase name constants.
const (
	planHubName = "plan"
	codeHubName = "code"
	planPhase   = "plan"
	codePhase   = "code"
)

// PlanHub mediates between the plan tool, trace ring, signal bus, and display.
type PlanHub struct {
	HubCore
	Graph   *artifact.Graph
	Planner ExecutionPlannerPort // nil on Day 1
}

// NewPlanHub creates a plan hub backed by the given artifact graph.
func NewPlanHub(core HubCore, graph *artifact.Graph) *PlanHub {
	return &PlanHub{HubCore: core, Graph: graph}
}

// Name returns the hub name.
func (h *PlanHub) Name() string { return planHubName }

// Phase returns the DevOps phase.
func (h *PlanHub) Phase() string { return planPhase }

// AddSegment wraps Graph.Add with five-step mediation.
func (h *PlanHub) AddSegment(s artifact.Artifact) string { //nolint:gocritic // value copy intentional
	s.Kind = artifact.KindPlanSegment
	id, _ := h.Graph.Add(s) //nolint:errcheck // mediation layer, errors traced

	h.Trace("segment-add", s.Title)

	h.Emit(telemetry.Signal{
		Category: "plan",
		Level:    telemetry.Green,
		Source:   planHubName + "-hub",
		Message:  "segment added: " + s.Title,
	})

	h.Render(DisplayMsg{
		Source:   planHubName,
		Category: planHubName,
		Content:  PlanEvent{Action: "add", SegmentID: id, Title: s.Title},
	})

	h.syncPlanner(context.Background())

	return id
}

// Claim wraps PlanGraph.Claim with mediation.
func (h *PlanHub) Claim(segmentID, owner string) error {
	if err := h.Graph.Claim(segmentID, owner); err != nil {
		return err
	}

	h.Trace("segment-claim", segmentID+" by "+owner)

	h.Emit(telemetry.Signal{
		Category: "plan",
		Level:    telemetry.Green,
		Source:   planHubName + "-hub",
		Message:  "segment claimed: " + segmentID + " by " + owner,
	})

	h.Render(DisplayMsg{
		Source:   planHubName,
		Category: planHubName,
		Content:  PlanEvent{Action: "claim", SegmentID: segmentID, Owner: owner},
	})

	return nil
}

// Complete wraps PlanGraph.Complete with mediation.
func (h *PlanHub) Complete(segmentID string) error {
	if err := h.Graph.Complete(segmentID); err != nil {
		return err
	}

	h.Trace("segment-complete", segmentID)

	h.Emit(telemetry.Signal{
		Category: "plan",
		Level:    telemetry.Green,
		Source:   planHubName + "-hub",
		Message:  "segment completed: " + segmentID,
	})

	h.Render(DisplayMsg{
		Source:   planHubName,
		Category: planHubName,
		Content:  PlanEvent{Action: "complete", SegmentID: segmentID},
	})

	h.syncPlanner(context.Background())

	return nil
}

// Ready returns segments whose dependencies are all complete.
func (h *PlanHub) Ready() []artifact.Artifact {
	return h.Graph.Ready()
}

// syncPlanner syncs with the external planner port if available.
func (h *PlanHub) syncPlanner(ctx context.Context) {
	if h.Planner == nil {
		return
	}

	segments := h.Graph.All()
	dtos := make([]PlanSegmentDTO, len(segments))
	for i := range segments {
		dtos[i] = PlanSegmentDTO{
			ID:      segments[i].ID,
			Title:   segments[i].Title,
			Status:  string(segments[i].Status),
			Content: segments[i].Content,
		}
	}

	if err := h.Planner.SyncPlan(ctx, dtos); err != nil {
		h.Trace("sync-error", err.Error())
	}
}

// PlanEvent is the display payload for plan operations.
type PlanEvent struct {
	Action    string `json:"action"`
	SegmentID string `json:"segment_id"`
	Title     string `json:"title,omitempty"`
	Owner     string `json:"owner,omitempty"`
}

// Compile-time interface check.
var _ MediatorHub = (*PlanHub)(nil)
