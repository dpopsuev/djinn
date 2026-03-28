package stubs

import (
	"context"
	"fmt"
	"sync"

	"github.com/dpopsuev/djinn/tier"
)

// StubSandbox implements broker.SandboxPort with deterministic IDs.
type StubSandbox struct {
	mu         sync.Mutex
	counter    int
	created    []string
	destroyed  []string
	createErr  error
	destroyErr error
}

// NewStubSandbox creates a new stub sandbox port.
func NewStubSandbox() *StubSandbox {
	return &StubSandbox{}
}

// Create generates a deterministic sandbox ID ("sandbox-N") and tracks it.
func (s *StubSandbox) Create(ctx context.Context, scope tier.Scope) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.createErr != nil {
		return "", s.createErr
	}
	s.counter++
	id := fmt.Sprintf("sandbox-%d", s.counter)
	s.created = append(s.created, id)
	return id, nil
}

// Destroy records the destruction of a sandbox.
func (s *StubSandbox) Destroy(ctx context.Context, sandboxID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.destroyErr != nil {
		return s.destroyErr
	}
	s.destroyed = append(s.destroyed, sandboxID)
	return nil
}

// Created returns a copy of all created sandbox IDs.
func (s *StubSandbox) Created() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.created))
	copy(out, s.created)
	return out
}

// Destroyed returns a copy of all destroyed sandbox IDs.
func (s *StubSandbox) Destroyed() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.destroyed))
	copy(out, s.destroyed)
	return out
}

// SetCreateErr injects an error for the next Create call.
func (s *StubSandbox) SetCreateErr(err error) { s.mu.Lock(); s.createErr = err; s.mu.Unlock() }

// SetDestroyErr injects an error for the next Destroy call.
func (s *StubSandbox) SetDestroyErr(err error) { s.mu.Lock(); s.destroyErr = err; s.mu.Unlock() }
