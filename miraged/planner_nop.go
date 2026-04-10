// planner_nop.go — Day 1 no-op adapter for ExecutionPlannerPort (GOL-57).
//
// All methods return success with empty data. Used when no external planner
// (e.g., Scribe) is connected.
package miraged

import "context"

// NopExecutionPlanner is the Day 1 no-op adapter.
type NopExecutionPlanner struct{}

// SyncPlan is a no-op.
func (NopExecutionPlanner) SyncPlan(_ context.Context, _ []PlanSegmentDTO) error {
	return nil
}

// FetchPlan returns nil (no external plan data).
func (NopExecutionPlanner) FetchPlan(_ context.Context) ([]PlanSegmentDTO, error) {
	return nil, nil
}

// Compile-time interface check.
var _ ExecutionPlannerPort = NopExecutionPlanner{}
