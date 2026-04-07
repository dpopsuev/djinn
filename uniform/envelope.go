// envelope.go — Lifecycle Envelope orchestrator.
// The Envelope wraps goal execution with automatic pre/post-flight phases
// and buffer checkpoints. GenSec uses this to schedule invisible recon
// before execution and audit afterward.
package uniform

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrPreFlightFailed is returned when a required pre-flight action fails.
var ErrPreFlightFailed = errors.New("envelope: pre-flight failed")

// ErrPostFlightFailed is returned when a required post-flight action fails.
var ErrPostFlightFailed = errors.New("envelope: post-flight failed")

// PhaseExecutor runs a phase action. Caller provides the execution strategy.
type PhaseExecutor func(ctx context.Context, action PhaseAction) (string, error)

// Envelope wraps goal execution with automatic pre/post-flight phases.
type Envelope struct {
	config  EnvelopeConfig
	results EnvelopeResult
	mu      sync.Mutex
}

// NewEnvelope creates a lifecycle envelope for a goal.
func NewEnvelope(goalID string, cfg EnvelopeConfig) *Envelope {
	return &Envelope{
		config:  cfg,
		results: EnvelopeResult{GoalID: goalID},
	}
}

// PreFlightAssignment returns a read-only assignment for pre-flight recon.
func (e *Envelope) PreFlightAssignment() Assignment {
	return Assignment{
		Role:    RoleGenSec,
		Mode:    ModeAsk,
		Scope:   AssignmentScope{ReadPaths: []string{"/"}},
		Persona: RolePersona[RoleGenSec],
	}
}

// PostFlightAssignment returns a read-only assignment for post-flight audit.
func (e *Envelope) PostFlightAssignment() Assignment {
	return Assignment{
		Role:    RoleInspector,
		Mode:    ModeAsk,
		Scope:   AssignmentScope{ReadPaths: []string{"/"}},
		Persona: RolePersona[RoleInspector],
	}
}

// RunPreFlight executes pre-flight actions. Returns error if a required action fails.
func (e *Envelope) RunPreFlight(ctx context.Context, exec PhaseExecutor) error {
	if !e.config.Enabled {
		return nil
	}
	return e.runPhase(ctx, exec, e.config.PreFlightActions, &e.results.PreFlight, ErrPreFlightFailed)
}

// RunPostFlight executes post-flight actions. Returns error if a required action fails.
func (e *Envelope) RunPostFlight(ctx context.Context, exec PhaseExecutor) error {
	if !e.config.Enabled {
		return nil
	}
	return e.runPhase(ctx, exec, e.config.PostFlightActions, &e.results.PostFlight, ErrPostFlightFailed)
}

// runPhase executes a list of phase actions sequentially.
func (e *Envelope) runPhase(ctx context.Context, exec PhaseExecutor, actions []PhaseAction, results *[]PhaseResult, sentinel error) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, action := range actions {
		start := time.Now()
		output, err := exec(ctx, action)
		result := PhaseResult{
			Action:   action,
			Output:   output,
			Error:    err,
			Duration: time.Since(start),
		}
		*results = append(*results, result)

		if err != nil && action.Required {
			return fmt.Errorf("%w: %s: %w", sentinel, action.Name, err)
		}
	}
	return nil
}

// ShouldCheckpoint returns true if a buffer check is due after this task.
func (e *Envelope) ShouldCheckpoint(taskIndex int) bool {
	if !e.config.Enabled || e.config.CheckpointEvery <= 0 {
		return false
	}
	return (taskIndex+1)%e.config.CheckpointEvery == 0
}

// CheckBuffer evaluates a buffer checkpoint and records it.
func (e *Envelope) CheckBuffer(taskIndex int, budgetUsed float64) BufferCheckpoint {
	e.mu.Lock()
	defer e.mu.Unlock()

	cp := BufferCheckpoint{
		AfterTask:  taskIndex,
		DriftScore: budgetUsed, // simple heuristic: drift ~ budget consumption rate
		BudgetUsed: budgetUsed,
		Alert:      budgetUsed > e.config.DriftThreshold,
		Timestamp:  time.Now(),
	}
	e.results.Buffers = append(e.results.Buffers, cp)
	return cp
}

// Result returns a copy of the full envelope outcome.
func (e *Envelope) Result() EnvelopeResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Return copy to prevent external mutation.
	r := e.results
	r.PreFlight = append([]PhaseResult(nil), e.results.PreFlight...)
	r.PostFlight = append([]PhaseResult(nil), e.results.PostFlight...)
	r.Buffers = append([]BufferCheckpoint(nil), e.results.Buffers...)
	return r
}

// Config returns the current envelope configuration.
func (e *Envelope) Config() EnvelopeConfig {
	return e.config
}

// SetEnabled toggles the envelope on/off.
func (e *Envelope) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.Enabled = enabled
}

// SetCheckpointEvery sets buffer checkpoint frequency.
func (e *Envelope) SetCheckpointEvery(n int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.CheckpointEvery = n
}
