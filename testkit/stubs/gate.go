package stubs

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/dpopsuev/djinn/gate"
)

// StubGate is a configurable gate stub.
type StubGate struct {
	err error
}

// NewStubGate creates a gate that returns the given error (nil = pass).
func NewStubGate(err error) *StubGate {
	return &StubGate{err: err}
}

func (g *StubGate) Validate(ctx context.Context, sandboxID string) error {
	return g.err
}

// AlwaysPassGate always passes validation.
func AlwaysPassGate() gate.Gate {
	return &StubGate{}
}

// AlwaysFailGate always fails with the given message.
func AlwaysFailGate(msg string) gate.Gate {
	return &StubGate{err: errors.New(msg)}
}

// FailOnNthGate fails on the Nth call (1-indexed), passes otherwise.
type FailOnNthGate struct {
	failOn int64
	count  atomic.Int64
	err    error
}

// NewFailOnNthGate creates a gate that fails on the Nth call.
func NewFailOnNthGate(n int, err error) *FailOnNthGate {
	return &FailOnNthGate{failOn: int64(n), err: err}
}

func (g *FailOnNthGate) Validate(ctx context.Context, sandboxID string) error {
	call := g.count.Add(1)
	if call == g.failOn {
		return g.err
	}
	return nil
}

// Count returns the number of Validate calls made.
func (g *FailOnNthGate) Count() int {
	return int(g.count.Load())
}
