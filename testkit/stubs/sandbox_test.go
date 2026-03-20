package stubs

import (
	"context"
	"errors"
	"testing"

	"github.com/dpopsuev/djinn/tier"
)

func TestStubSandbox_IDGeneration(t *testing.T) {
	s := NewStubSandbox()
	ctx := context.Background()
	scope := tier.Scope{Level: tier.Mod, Name: "auth"}

	id1, err := s.Create(ctx, scope)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id1 != "sandbox-1" {
		t.Fatalf("first ID = %q, want %q", id1, "sandbox-1")
	}

	id2, _ := s.Create(ctx, scope)
	if id2 != "sandbox-2" {
		t.Fatalf("second ID = %q, want %q", id2, "sandbox-2")
	}
}

func TestStubSandbox_LifecycleTracking(t *testing.T) {
	s := NewStubSandbox()
	ctx := context.Background()
	scope := tier.Scope{Level: tier.Mod, Name: "auth"}

	id, _ := s.Create(ctx, scope)
	if err := s.Destroy(ctx, id); err != nil {
		t.Fatalf("Destroy: %v", err)
	}

	created := s.Created()
	if len(created) != 1 || created[0] != id {
		t.Fatalf("Created() = %v, want [%q]", created, id)
	}
	destroyed := s.Destroyed()
	if len(destroyed) != 1 || destroyed[0] != id {
		t.Fatalf("Destroyed() = %v, want [%q]", destroyed, id)
	}
}

func TestStubSandbox_ErrorInjection(t *testing.T) {
	s := NewStubSandbox()
	ctx := context.Background()
	scope := tier.Scope{Level: tier.Mod, Name: "auth"}
	injected := errors.New("injected")

	s.SetCreateErr(injected)
	if _, err := s.Create(ctx, scope); !errors.Is(err, injected) {
		t.Fatalf("Create err = %v, want injected", err)
	}

	s.SetCreateErr(nil)
	id, _ := s.Create(ctx, scope)

	s.SetDestroyErr(injected)
	if err := s.Destroy(ctx, id); !errors.Is(err, injected) {
		t.Fatalf("Destroy err = %v, want injected", err)
	}
}
