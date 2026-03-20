package broker

import (
	"sync"
	"testing"
)

func TestCordonRegistry_Lifecycle(t *testing.T) {
	r := NewCordonRegistry()

	if r.Overlaps("auth") {
		t.Fatal("auth should not be cordoned initially")
	}

	r.Set("auth", "failing tests")
	if !r.Overlaps("auth") {
		t.Fatal("auth should be cordoned after Set")
	}

	active := r.Active()
	if len(active) != 1 {
		t.Fatalf("Active() = %d, want 1", len(active))
	}
	if active[0].Reason != "failing tests" {
		t.Fatalf("Reason = %q, want %q", active[0].Reason, "failing tests")
	}

	r.Clear("auth")
	if r.Overlaps("auth") {
		t.Fatal("auth should not be cordoned after Clear")
	}
	if len(r.Active()) != 0 {
		t.Fatalf("Active() = %d after Clear, want 0", len(r.Active()))
	}
}

func TestCordonRegistry_OverlapDetection(t *testing.T) {
	r := NewCordonRegistry()
	r.Set("auth", "r1")
	r.Set("payments", "r2")

	if !r.Overlaps("auth") {
		t.Fatal("auth should overlap")
	}
	if !r.Overlaps("payments") {
		t.Fatal("payments should overlap")
	}
	if r.Overlaps("billing") {
		t.Fatal("billing should not overlap")
	}
}

func TestCordonRegistry_ConcurrentSafety(t *testing.T) {
	r := NewCordonRegistry()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			r.Set("scope", "reason")
		}(i)
		go func(n int) {
			defer wg.Done()
			r.Overlaps("scope")
		}(i)
	}
	wg.Wait()
}
