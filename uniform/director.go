// director.go — LocalDirector implements troupe/director.Director
// using config-driven transitions. The Origami seam.
//
// LocalDirector is Djinn's built-in orchestrator: linear pipeline,
// YAML-driven transitions, zero LLM cost for role routing.
// Origami provides CircuitDirector for graph-based orchestration.
// Both satisfy the same interface — swap at composition root.
//
// DJN-TSK-1059
package uniform

import (
	"context"

	troupe "github.com/dpopsuev/troupe"
	"github.com/dpopsuev/troupe/director"
)

var _ director.Director = (*LocalDirector)(nil)

// Scheduler resolves the next role for a given signal.
// Extracted interface so external orchestrators (Origami) can plug in.
type Scheduler interface {
	NextRole(signal Signal, currentRole string) string
}

// LocalDirector is Djinn's built-in Director. Linear pipeline
// driven by a config transition table. Implements troupe director.Director.
type LocalDirector struct {
	scheduler Scheduler
}

// NewLocalDirector creates a Director backed by the given Scheduler.
func NewLocalDirector(s Scheduler) *LocalDirector {
	return &LocalDirector{scheduler: s}
}

// Direct executes the orchestration plan. For LocalDirector this is
// event-driven — it emits a single Transition event per signal.
// The REPL drives the loop; Direct provides the contract.
func (d *LocalDirector) Direct(ctx context.Context, broker troupe.Broker) (<-chan troupe.Event, error) {
	ch := make(chan troupe.Event, 1)
	ch <- troupe.Event{Kind: troupe.Started, Step: "local-director"}
	close(ch)
	return ch, nil
}

// Scheduler returns the underlying Scheduler for direct lookup.
// Used by repl/model.go for synchronous NextRole calls.
func (d *LocalDirector) Scheduler() Scheduler { return d.scheduler }

// TransitionScheduler implements Scheduler using a config-driven transition table.
type TransitionScheduler struct {
	transitions []Transition
	fallback    string
}

// NewTransitionScheduler creates a Scheduler from a transition table.
func NewTransitionScheduler(transitions []Transition) *TransitionScheduler {
	return &TransitionScheduler{
		transitions: transitions,
		fallback:    defaultFallbackRole,
	}
}

// DefaultScheduler creates a Scheduler with the standard 5-role pipeline.
func DefaultScheduler() *TransitionScheduler {
	return NewTransitionScheduler(DefaultTransitions())
}

// NextRole resolves the target role for a signal.
// Checks FromRole-specific transitions first, then any-role transitions.
func (s *TransitionScheduler) NextRole(signal Signal, currentRole string) string {
	// First pass: match signal + fromRole.
	for _, t := range s.transitions {
		if t.Signal == signal && t.FromRole != "" && t.FromRole == currentRole {
			return t.ToRole
		}
	}
	// Second pass: match signal with any fromRole.
	for _, t := range s.transitions {
		if t.Signal == signal && t.FromRole == "" {
			return t.ToRole
		}
	}
	return s.fallback
}

// Interface compliance.
var _ Scheduler = (*TransitionScheduler)(nil)
