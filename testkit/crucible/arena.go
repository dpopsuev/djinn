// Package arena provides the tournament harness for testing Djinn programmatically.
//
// The Arena is the forge for Djinn itself — deterministic scenarios with
// a referee that verifies results. "Can Djinn build an HTTP server?" is
// the acceptance test, not a feature added after.
//
// Three axes: Scenario (what to build), ToolLevel (what Djinn provides),
// Operator (who drives). Scoring: time × tokens × compute.
package crucible

import (
	"context"
	"time"
)

// Referee verifies a scenario's acceptance criteria against a built project.
// Deterministic — no LLM in the referee, only programmatic checks.
//
// For event-driven scoring during a run, use referee.Referee (GOL-164).
// This interface remains valid for post-run workspace verification.
// Bridge via referee.EmitWorkspaceResult() to feed results into the scorecard.
type Referee interface {
	Check(ctx context.Context, scenarioID, projectPath string) (CheckResult, error)
}

// CheckResult is the outcome of a referee check.
type CheckResult struct {
	Pass   bool     `json:"pass"`
	Score  float64  `json:"score"`
	Errors []string `json:"errors,omitempty"`
}

// Scenario defines what the agent should build and how to verify it.
type Scenario interface {
	ID() string
	Spec() string // human-readable description of what to build
	Timeout() time.Duration
	Budget() Budget
}

// Budget defines resource limits for a scenario run.
type Budget struct {
	MaxTokens int           `json:"max_tokens"`
	MaxCost   float64       `json:"max_cost"`
	MaxTime   time.Duration `json:"max_time"`
}

// ToolLevel defines what Djinn tools are available during a run.
type ToolLevel string

const (
	L0Mothballed ToolLevel = "L0" // nothing — raw agent passthrough
	L1AgentSpace ToolLevel = "L1" // invisible overlay (Mirage)
	L2Shell      ToolLevel = "L2" // 8 built-in tools
	L3MultiAgent ToolLevel = "L3" // GenSec + Executor
	L4FullStack  ToolLevel = "L4" // Day 2 MCPs (Scribe, Locus, etc.)
)

// RunMetrics captures measurements from a single arena run.
type RunMetrics struct {
	ScenarioID  string        `json:"scenario_id"`
	ToolLevel   ToolLevel     `json:"tool_level"`
	TimeElapsed time.Duration `json:"time_elapsed"`
	TokensIn    int           `json:"tokens_in"`
	TokensOut   int           `json:"tokens_out"`
	PeakRSS     int64         `json:"peak_rss_bytes"`
	CPUSeconds  float64       `json:"cpu_seconds"`
	Pass        bool          `json:"pass"`
	Score       float64       `json:"score"`

	// Artifacts maps relative file paths to content for post-mortem audit.
	// Populated after the run completes, before workspace cleanup.
	Artifacts map[string]string `json:"artifacts,omitempty"`
}

// Operator drives the agent — feeds prompts, responds to approvals.
// MockOperator uses canned prompts. Real operator uses TUI.
type Operator interface {
	Perform(ctx context.Context, message string) (string, error)
}
