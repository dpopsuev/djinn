// ports.go — Hexagonal port interfaces for Day 2 external adapters (GOL-58).
//
// Day 1: all ports have nil-safe no-op behavior (hubs check nil before calling).
// Day 2: adapters implement these against MCP servers (Scribe, Locus, Limes).
// Pattern follows broker/ports.go and staff/ports.go conventions.
package daemon

import "context"

// ExecutionPlannerPort is the driven port for external planning services.
// Day 2 adapter: ScribeAdapter (via MCP).
type ExecutionPlannerPort interface {
	SyncPlan(ctx context.Context, segments []PlanSegmentDTO) error
	FetchPlan(ctx context.Context) ([]PlanSegmentDTO, error)
}

// PlanSegmentDTO is the wire format for plan segments crossing port boundaries.
type PlanSegmentDTO struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Content string `json:"content"`
}

// StructuralAnalyzerPort is the driven port for architecture analysis.
// Day 2 adapter: LocusAdapter (via MCP).
type StructuralAnalyzerPort interface {
	Analyze(ctx context.Context, paths []string) (AnalysisResult, error)
}

// AnalysisResult is the wire format for architecture analysis outcomes.
type AnalysisResult struct {
	Components []string            `json:"components"`
	Violations []string            `json:"violations,omitempty"`
	Graph      map[string][]string `json:"graph,omitempty"`
}
