package observe

import (
	"sync"
	"time"
)

var _ Observer = (*StubObserver)(nil)

// StubObserver is an in-memory Observer for testing.
// Seed it with canned trace lines and health data.
type StubObserver struct {
	mu     sync.RWMutex
	traces []TraceLine
	health HealthReport

	// TraceCalls records how many times Trace was called.
	TraceCalls int
	// HealthCalls records how many times Health was called.
	HealthCalls int
}

// NewStubObserver creates a StubObserver with empty data.
func NewStubObserver() *StubObserver {
	return &StubObserver{
		health: HealthReport{
			LastActivity: time.Now(),
		},
	}
}

func (s *StubObserver) Trace(opts TraceOpts) ([]TraceLine, error) {
	s.mu.Lock()
	s.TraceCalls++
	s.mu.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := s.traces

	// Filter by kind.
	if opts.Kind != "" {
		var filtered []TraceLine
		for _, t := range result {
			if t.Kind == opts.Kind {
				filtered = append(filtered, t)
			}
		}
		result = filtered
	}

	// Filter by source.
	if opts.Source != "" {
		var filtered []TraceLine
		for _, t := range result {
			if t.Source == opts.Source {
				filtered = append(filtered, t)
			}
		}
		result = filtered
	}

	// Limit to last N.
	last := opts.Last
	if last <= 0 {
		last = 50
	}
	if len(result) > last {
		result = result[len(result)-last:]
	}

	out := make([]TraceLine, len(result))
	copy(out, result)
	return out, nil
}

func (s *StubObserver) Health() (HealthReport, error) {
	s.mu.Lock()
	s.HealthCalls++
	s.mu.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.health, nil
}

// --- Test helpers ---

// SeedTrace adds trace lines for testing.
func (s *StubObserver) SeedTrace(lines ...TraceLine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.traces = append(s.traces, lines...)
}

// SeedHealth sets the health report for testing.
func (s *StubObserver) SeedHealth(h HealthReport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health = h
}
