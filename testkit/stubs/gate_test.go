package stubs

import (
	"context"
	"errors"
	"testing"

	"github.com/dpopsuev/djinn/gate"
)

func TestStubGate_InterfaceSatisfaction(t *testing.T) {
	var _ gate.Gate = (*StubGate)(nil)
	var _ gate.Gate = (*FailOnNthGate)(nil)
}

func TestAlwaysPassGate(t *testing.T) {
	g := AlwaysPassGate()
	if err := g.Validate(context.Background(), "s1"); err != nil {
		t.Fatalf("AlwaysPassGate.Validate() = %v, want nil", err)
	}
}

func TestAlwaysFailGate(t *testing.T) {
	g := AlwaysFailGate("lint failed")
	err := g.Validate(context.Background(), "s1")
	if err == nil {
		t.Fatal("AlwaysFailGate.Validate() = nil, want error")
	}
	if err.Error() != "lint failed" {
		t.Fatalf("error = %q, want %q", err.Error(), "lint failed")
	}
}

func TestFailOnNthGate(t *testing.T) {
	injected := errors.New("gate-3-fail")
	g := NewFailOnNthGate(3, injected)

	ctx := context.Background()
	if err := g.Validate(ctx, "s1"); err != nil {
		t.Fatalf("call 1: %v", err)
	}
	if err := g.Validate(ctx, "s2"); err != nil {
		t.Fatalf("call 2: %v", err)
	}
	if err := g.Validate(ctx, "s3"); !errors.Is(err, injected) {
		t.Fatalf("call 3: got %v, want injected", err)
	}
	if err := g.Validate(ctx, "s4"); err != nil {
		t.Fatalf("call 4: %v", err)
	}
	if g.Count() != 4 {
		t.Fatalf("Count() = %d, want 4", g.Count())
	}
}
