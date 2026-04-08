// analysis_hub.go — AnalysisHub mediates architecture analysis (GOL-53).
//
// Day 1: uses internal arch tool. Day 2: StructuralAnalyzerPort for Locus MCP.
package daemon

import (
	"context"

	"github.com/dpopsuev/djinn/telemetry"
)

// Hub and phase name constants for analysis.
const (
	analysisHubName = "analysis"
	monitorPhase    = "monitor"
)

// AnalysisHub mediates architecture analysis between internal tools and external ports.
type AnalysisHub struct {
	HubCore
	WorkDir  string
	Analyzer StructuralAnalyzerPort // nil on Day 1
}

// NewAnalysisHub creates an analysis hub for the given work directory.
func NewAnalysisHub(core HubCore, workDir string) *AnalysisHub {
	return &AnalysisHub{HubCore: core, WorkDir: workDir}
}

// Name returns the hub name.
func (h *AnalysisHub) Name() string { return analysisHubName }

// Phase returns the DevOps phase.
func (h *AnalysisHub) Phase() string { return monitorPhase }

// Analyze runs architecture analysis with full mediation.
// Day 2: delegates to StructuralAnalyzerPort if available.
func (h *AnalysisHub) Analyze(ctx context.Context, paths []string) (AnalysisResult, error) {
	rt := h.Tracer.Begin("analyze", h.WorkDir)

	var result AnalysisResult
	var err error

	if h.Analyzer != nil {
		result, err = h.Analyzer.Analyze(ctx, paths)
	} else {
		// Day 1: return empty result (internal arch tool called separately).
		result = AnalysisResult{Components: paths}
	}

	if err != nil {
		rt.EndWithError()
		h.Emit(telemetry.Signal{
			Category: analysisHubName,
			Level:    telemetry.Yellow,
			Source:   analysisHubName + "-hub",
			Message:  "analysis failed: " + err.Error(),
		})
		return result, err
	}

	rt.End()

	if len(result.Violations) > 0 {
		h.Emit(telemetry.Signal{
			Category: analysisHubName,
			Level:    telemetry.Yellow,
			Source:   analysisHubName + "-hub",
			Message:  "architecture violations detected",
			Scope:    result.Violations,
		})
	}

	h.Render(DisplayMsg{
		Source:   analysisHubName,
		Category: monitorPhase,
		Content:  AnalysisEvent{Components: len(result.Components), Violations: len(result.Violations)},
	})

	return result, nil
}

// AnalysisEvent is the display payload for analysis operations.
type AnalysisEvent struct {
	Components int `json:"components"`
	Violations int `json:"violations"`
}

// Compile-time interface check.
var _ MediatorHub = (*AnalysisHub)(nil)
