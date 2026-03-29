// orchestrator.go — SelfHealOrchestrator wires the full circuit (TSK-493).
//
// Circuit: TraceSignalBridge detects → Signal emitted → GenSec diagnoses →
// ExecutorHarness fixes → SelfHealGate validates → result logged.
// Circuit breaker: max attempts per hour. Three-layer control enforced.
package selfheal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/trace"
)

// MaxAttemptsPerHour limits self-heal attempts to prevent loops.
const MaxAttemptsPerHour = 3

// ErrCircuitBreaker is returned when too many self-heal attempts were made recently.
var ErrCircuitBreaker = fmt.Errorf("circuit breaker: max %d attempts per hour exceeded", MaxAttemptsPerHour)

// Orchestrator coordinates the self-healing circuit.
type Orchestrator struct {
	ring    *trace.Ring
	harness *Harness

	mu       sync.Mutex
	attempts []time.Time // timestamps of recent attempts
	history  []AttemptRecord
}

// AttemptRecord captures one self-heal attempt.
type AttemptRecord struct {
	Timestamp   time.Time    `json:"ts"`
	FixID       string       `json:"fix_id"`
	Trigger     string       `json:"trigger"`
	BuildResult *BuildResult `json:"build_result"`
	GateResult  *GateResult  `json:"gate_result,omitempty"`
}

// NewOrchestrator creates a self-heal orchestrator.
func NewOrchestrator(ring *trace.Ring, harness *Harness) *Orchestrator {
	return &Orchestrator{
		ring:    ring,
		harness: harness,
	}
}

// CanAttempt checks if the circuit breaker allows another attempt.
func (o *Orchestrator) CanAttempt() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	cutoff := time.Now().Add(-time.Hour)
	recent := 0
	for _, t := range o.attempts {
		if t.After(cutoff) {
			recent++
		}
	}
	return recent < MaxAttemptsPerHour
}

// Attempt runs the self-heal circuit for a given trigger.
// Returns the attempt record. Does NOT hot-swap — that's the caller's decision.
func (o *Orchestrator) Attempt(ctx context.Context, trigger string, instructions []string) (*AttemptRecord, error) {
	if !o.CanAttempt() {
		return nil, ErrCircuitBreaker
	}

	o.mu.Lock()
	o.attempts = append(o.attempts, time.Now())
	o.mu.Unlock()

	fixID := fmt.Sprintf("selfheal-%d", time.Now().UnixMilli())

	// Snapshot trace before fix.
	beforeArchive := trace.Export(o.ring, "")

	// Run fix in worktree.
	buildResult, err := o.harness.RunFix(ctx, fixID, instructions)
	if err != nil {
		return nil, fmt.Errorf("harness: %w", err)
	}

	record := &AttemptRecord{
		Timestamp:   time.Now(),
		FixID:       fixID,
		Trigger:     trigger,
		BuildResult: buildResult,
	}

	// If build/test failed, skip gate.
	if !buildResult.Success {
		o.recordAttempt(record)
		return record, nil
	}

	// Validate via trace comparison.
	gateResult := Validate(beforeArchive, o.ring)
	record.GateResult = gateResult

	o.recordAttempt(record)
	return record, nil
}

// History returns all attempt records.
func (o *Orchestrator) History() []AttemptRecord {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]AttemptRecord, len(o.history))
	copy(out, o.history)
	return out
}

func (o *Orchestrator) recordAttempt(record *AttemptRecord) {
	o.mu.Lock()
	o.history = append(o.history, *record)
	o.mu.Unlock()
}
