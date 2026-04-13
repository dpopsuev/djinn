// director.go — Scheduler: config-driven role routing.
//
// Scheduler is a pure function: signal + currentRole → nextRole.
// Reads from a YAML-driven transition table. No LLM, no lifecycle.
//
// UnifiedDirector (unified_director.go) uses Scheduler for role routing
// and adds full agent orchestration (Run, Direct, sandbox, policies).
//
// DJN-TSK-1059, GOL-163
package substrate

import "github.com/dpopsuev/djinn/uniform"

// Scheduler resolves the next role for a given signal.
// Extracted interface so external orchestrators (Origami) can plug in.
type Scheduler interface {
	NextRole(signal uniform.Signal, currentRole string) string
}

// TransitionScheduler implements Scheduler using a config-driven transition table.
type TransitionScheduler struct {
	transitions []uniform.Transition
	fallback    string
}

// NewTransitionScheduler creates a Scheduler from a transition table.
func NewTransitionScheduler(transitions []uniform.Transition) *TransitionScheduler {
	return &TransitionScheduler{
		transitions: transitions,
		fallback:    uniform.DefaultFallbackRole,
	}
}

// DefaultScheduler creates a Scheduler with the standard 5-role pipeline.
func DefaultScheduler() *TransitionScheduler {
	return NewTransitionScheduler(uniform.DefaultTransitions())
}

// NextRole resolves the target role for a signal.
// Checks FromRole-specific transitions first, then any-role transitions.
func (s *TransitionScheduler) NextRole(signal uniform.Signal, currentRole string) string {
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
