package broker

import (
	"sync"
	"testing"
)

func TestCordonRegistry_Lifecycle(t *testing.T) {
	r := NewCordonRegistry()

	if len(r.Overlaps([]string{"auth"})) > 0 {
		t.Fatal("auth should not be cordoned initially")
	}

	r.Set([]string{"auth/middleware.go"}, "failing tests", "agent-1")
	overlaps := r.Overlaps([]string{"auth/middleware.go"})
	if len(overlaps) != 1 {
		t.Fatalf("Overlaps = %d, want 1", len(overlaps))
	}
	if overlaps[0].Reason != "failing tests" {
		t.Fatalf("Reason = %q, want %q", overlaps[0].Reason, "failing tests")
	}
	if overlaps[0].Source != "agent-1" {
		t.Fatalf("Source = %q, want %q", overlaps[0].Source, "agent-1")
	}

	active := r.Active()
	if len(active) != 1 {
		t.Fatalf("Active() = %d, want 1", len(active))
	}

	r.Clear([]string{"auth/middleware.go"})
	if len(r.Active()) != 0 {
		t.Fatalf("Active() after Clear = %d, want 0", len(r.Active()))
	}
}

func TestCordonRegistry_PathPrefixOverlap(t *testing.T) {
	r := NewCordonRegistry()
	r.Set([]string{"auth/middleware.go"}, "security", "agent-1")

	// Exact match
	if len(r.Overlaps([]string{"auth/middleware.go"})) != 1 {
		t.Fatal("exact path should overlap")
	}

	// Parent directory overlaps child
	if len(r.Overlaps([]string{"auth"})) != 1 {
		t.Fatal("parent dir 'auth' should overlap cordoned 'auth/middleware.go'")
	}

	// Unrelated path does not overlap
	if len(r.Overlaps([]string{"billing/handler.go"})) != 0 {
		t.Fatal("unrelated path should not overlap")
	}

	// Partial name should NOT match (au != auth)
	if len(r.Overlaps([]string{"au"})) != 0 {
		t.Fatal("partial name 'au' should not overlap 'auth/middleware.go'")
	}
}

func TestCordonRegistry_BroadCordonOverlapsChild(t *testing.T) {
	r := NewCordonRegistry()
	r.Set([]string{"auth"}, "whole dir cordoned", "agent-1")

	// Query with a child path — should overlap
	if len(r.Overlaps([]string{"auth/middleware.go"})) != 1 {
		t.Fatal("child path should overlap broad cordon")
	}
	if len(r.Overlaps([]string{"auth/handler.go"})) != 1 {
		t.Fatal("another child path should overlap broad cordon")
	}
}

func TestCordonRegistry_MultipleScopes(t *testing.T) {
	r := NewCordonRegistry()
	r.Set([]string{"auth/middleware.go", "auth/handler.go"}, "multi-file", "agent-1")

	if len(r.Overlaps([]string{"auth/handler.go"})) != 1 {
		t.Fatal("second scope path should overlap")
	}
}

func TestCordonRegistry_ClearedCordonNotReturned(t *testing.T) {
	r := NewCordonRegistry()
	r.Set([]string{"auth"}, "broken", "agent-1")
	r.Clear([]string{"auth"})

	overlaps := r.Overlaps([]string{"auth/middleware.go"})
	if len(overlaps) != 0 {
		t.Fatalf("cleared cordon should not appear in Overlaps, got %d", len(overlaps))
	}
}

func TestCordonRegistry_ConcurrentSafety(t *testing.T) {
	r := NewCordonRegistry()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			r.Set([]string{"scope"}, "reason", "agent")
		}(i)
		go func(n int) {
			defer wg.Done()
			r.Overlaps([]string{"scope"})
		}(i)
	}
	wg.Wait()
}
