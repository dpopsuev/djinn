// lifecycle.go — Lifecycle phases for the Envelope system.
// The Envelope wraps every agent goal with invisible pre-flight (recon),
// buffer checkpoints (during execution), and post-flight (audit) phases.
// GenSec schedules these automatically.
package staff

import "time"

// Phase represents a lifecycle stage around goal execution.
type Phase string

const (
	PhaseRecon   Phase = "recon"   // pre-flight: architecture scan, test health
	PhaseExecute Phase = "execute" // main work
	PhaseBuffer  Phase = "buffer"  // mid-flight checkpoint: drift, budget
	PhaseVerify  Phase = "verify"  // post-flight: audit, coverage
)

// allPhases for iteration.
var allPhases = []Phase{PhaseRecon, PhaseExecute, PhaseBuffer, PhaseVerify}

// ParsePhase parses a string into a Phase. Returns false if invalid.
func ParsePhase(s string) (Phase, bool) {
	for _, p := range allPhases {
		if string(p) == s {
			return p, true
		}
	}
	return "", false
}

// String returns the phase name.
func (p Phase) String() string { return string(p) }

// IsReadOnly returns true for phases that should use read-only scope.
func (p Phase) IsReadOnly() bool { return p == PhaseRecon || p == PhaseVerify }

// PhaseAction is a single action within a lifecycle phase.
type PhaseAction struct {
	Name     string // human-readable (e.g., "architecture scan")
	Tool     string // built-in tool name (e.g., "arch", "test")
	Args     string // tool arguments (JSON string)
	Required bool   // if true, failure blocks the phase
}

// PhaseResult captures the outcome of a single phase action.
type PhaseResult struct {
	Action   PhaseAction
	Output   string
	Error    error
	Duration time.Duration
}

// Passed returns true if the action completed without error.
func (r PhaseResult) Passed() bool { return r.Error == nil }

// EnvelopeResult captures the full lifecycle outcome.
type EnvelopeResult struct {
	GoalID     string
	PreFlight  []PhaseResult
	PostFlight []PhaseResult
	Buffers    []BufferCheckpoint
}

// BufferCheckpoint is a mid-execution health check.
type BufferCheckpoint struct {
	AfterTask  int     // task index that triggered this checkpoint
	DriftScore float64 // 0.0 = on track, 1.0 = completely off
	BudgetUsed float64 // fraction of budget consumed (0.0 to 1.0)
	Alert      bool    // true if drift > threshold
	Timestamp  time.Time
}

// EnvelopeConfig controls lifecycle behavior.
type EnvelopeConfig struct {
	Enabled           bool          // false = skip all envelope phases
	CheckpointEvery   int           // buffer check frequency (default: 3)
	DriftThreshold    float64       // alert if drift > this (default: 0.2)
	PreFlightActions  []PhaseAction // customizable pre-flight
	PostFlightActions []PhaseAction // customizable post-flight
}

// DefaultEnvelopeConfig returns sensible defaults.
func DefaultEnvelopeConfig() EnvelopeConfig {
	return EnvelopeConfig{
		Enabled:           true,
		CheckpointEvery:   3,
		DriftThreshold:    0.2,
		PreFlightActions:  DefaultPreFlight(),
		PostFlightActions: DefaultPostFlight(),
	}
}

// DefaultPreFlight returns the standard recon actions.
func DefaultPreFlight() []PhaseAction {
	return []PhaseAction{
		{Name: "architecture scan", Tool: "arch", Args: `{"action":"analyze"}`, Required: false},
		{Name: "test health", Tool: "test", Args: `{"action":"run"}`, Required: false},
		{Name: "lint baseline", Tool: "arch", Args: `{"action":"lint"}`, Required: false},
	}
}

// DefaultPostFlight returns the standard audit actions.
func DefaultPostFlight() []PhaseAction {
	return []PhaseAction{
		{Name: "architecture audit", Tool: "arch", Args: `{"action":"cycles"}`, Required: true},
		{Name: "test coverage", Tool: "test", Args: `{"action":"coverage"}`, Required: true},
		{Name: "lint hygiene", Tool: "arch", Args: `{"action":"lint"}`, Required: false},
	}
}
