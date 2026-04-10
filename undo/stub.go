package undo

import (
	"sync"

	"github.com/dpopsuev/troupe/signal"
)

var _ Manager = (*StubManager)(nil)

// StubManager implements Manager for testing.
// Records all checkpoint/rollback calls. Returns configurable events on rollback.
type StubManager struct {
	mu          sync.Mutex
	checkpoints []Checkpoint
	current     int
	log         signal.EventLog

	// RollbackEvents is returned by Rollback. Set this in tests.
	RollbackEvents []signal.Event
	// RollbackError is returned by Rollback. Set this to simulate errors.
	RollbackError error

	// CallLog records method calls for assertion.
	CheckpointCalls []string // names passed to Checkpoint
	RollbackCalls   []int    // indices passed to Rollback
}

// NewStubManager creates a stub backed by the given EventLog.
func NewStubManager(log signal.EventLog) *StubManager {
	return &StubManager{log: log, current: -1}
}

func (s *StubManager) Checkpoint(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.log.Len()
	s.checkpoints = append(s.checkpoints, Checkpoint{Index: idx, Name: name})
	s.current = idx
	s.CheckpointCalls = append(s.CheckpointCalls, name)
	return idx
}

func (s *StubManager) Rollback(index int) ([]signal.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.RollbackCalls = append(s.RollbackCalls, index)
	if s.RollbackError != nil {
		return nil, s.RollbackError
	}
	s.current = index
	if s.RollbackEvents != nil {
		return s.RollbackEvents, nil
	}
	return s.log.Since(index), nil
}

func (s *StubManager) Current() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *StubManager) Checkpoints() []Checkpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Checkpoint, len(s.checkpoints))
	copy(out, s.checkpoints)
	return out
}
