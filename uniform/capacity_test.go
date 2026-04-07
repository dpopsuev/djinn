package uniform

import (
	"errors"
	"sync"
	"testing"
)

func TestNewAgentCapacity(t *testing.T) {
	ac := NewAgentCapacity(4)
	if ac.Cap() != 4 {
		t.Fatalf("Cap() = %d, want 4", ac.Cap())
	}
	if ac.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", ac.Len())
	}
}

func TestNewAgentCapacity_Negative(t *testing.T) {
	ac := NewAgentCapacity(-5)
	if ac.Cap() != 0 {
		t.Fatalf("Cap() = %d, want 0 (negative clamped)", ac.Cap())
	}
}

func TestSetCap(t *testing.T) {
	ac := NewAgentCapacity(1)
	if err := ac.SetCap(3); err != nil {
		t.Fatalf("SetCap(3) error: %v", err)
	}
	if ac.Cap() != 3 {
		t.Fatalf("Cap() = %d, want 3", ac.Cap())
	}
}

func TestSetCap_Negative(t *testing.T) {
	ac := NewAgentCapacity(1)
	err := ac.SetCap(-1)
	if err == nil {
		t.Fatal("expected error for negative SetCap")
	}
	if !errors.Is(err, ErrCapacityNegative) {
		t.Fatalf("err = %v, want ErrCapacityNegative", err)
	}
}

func TestInc(t *testing.T) {
	ac := NewAgentCapacity(1)
	ac.Inc()
	if ac.Cap() != 2 {
		t.Fatalf("Cap() = %d, want 2 after Inc", ac.Cap())
	}
}

func TestDec(t *testing.T) {
	ac := NewAgentCapacity(2)
	if err := ac.Dec(); err != nil {
		t.Fatalf("Dec() error: %v", err)
	}
	if ac.Cap() != 1 {
		t.Fatalf("Cap() = %d, want 1 after Dec", ac.Cap())
	}
}

func TestDec_AtZero(t *testing.T) {
	ac := NewAgentCapacity(0)
	err := ac.Dec()
	if err == nil {
		t.Fatal("expected error for Dec at zero")
	}
	if !errors.Is(err, ErrCapacityZero) {
		t.Fatalf("err = %v, want ErrCapacityZero", err)
	}
}

func TestCanSpawn(t *testing.T) {
	ac := NewAgentCapacity(2)

	// Nothing running, should be able to spawn.
	if !ac.CanSpawn() {
		t.Fatal("CanSpawn() = false, want true (0/2)")
	}

	// Acquire one slot.
	ac.Acquire()
	if !ac.CanSpawn() {
		t.Fatal("CanSpawn() = false, want true (1/2)")
	}

	// Acquire second slot — now at capacity.
	ac.Acquire()
	if ac.CanSpawn() {
		t.Fatal("CanSpawn() = true, want false (2/2)")
	}
}

func TestAcquire(t *testing.T) {
	ac := NewAgentCapacity(2)

	if !ac.Acquire() {
		t.Fatal("first Acquire() = false, want true")
	}
	if !ac.Acquire() {
		t.Fatal("second Acquire() = false, want true")
	}
	if ac.Acquire() {
		t.Fatal("third Acquire() = true, want false (at capacity)")
	}
	if ac.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", ac.Len())
	}
}

func TestRelease(t *testing.T) {
	ac := NewAgentCapacity(1)
	ac.Acquire()
	if ac.Len() != 1 {
		t.Fatalf("Len() = %d, want 1 after Acquire", ac.Len())
	}

	ac.Release()
	if ac.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 after Release", ac.Len())
	}
}

func TestRelease_AtZero(t *testing.T) {
	ac := NewAgentCapacity(1)
	// Release without any Acquire — should not panic.
	ac.Release()
	if ac.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 after Release at zero", ac.Len())
	}
}

func TestExcessRunning(t *testing.T) {
	ac := NewAgentCapacity(3)
	ac.Acquire()
	ac.Acquire()
	ac.Acquire()

	if ac.ExcessRunning() != 0 {
		t.Fatalf("ExcessRunning() = %d, want 0 (3/3)", ac.ExcessRunning())
	}

	// Shrink cap below running count.
	if err := ac.SetCap(1); err != nil {
		t.Fatalf("SetCap(1) error: %v", err)
	}
	if ac.ExcessRunning() != 2 {
		t.Fatalf("ExcessRunning() = %d, want 2 (3 running, cap 1)", ac.ExcessRunning())
	}
}

func TestCapacityString(t *testing.T) {
	ac := NewAgentCapacity(4)
	ac.Acquire()
	ac.Acquire()

	got := ac.String()
	if got != "2/4" {
		t.Fatalf("String() = %q, want %q", got, "2/4")
	}
}

func TestCapacityConcurrency(t *testing.T) {
	ac := NewAgentCapacity(100)
	var wg sync.WaitGroup

	// 100 goroutines all trying to Acquire.
	for range 100 {
		wg.Go(func() {
			ac.Acquire()
		})
	}
	wg.Wait()

	if ac.Len() != 100 {
		t.Fatalf("Len() = %d, want 100 after concurrent Acquire", ac.Len())
	}

	// 100 goroutines all releasing.
	for range 100 {
		wg.Go(func() {
			ac.Release()
		})
	}
	wg.Wait()

	if ac.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 after concurrent Release", ac.Len())
	}

	// Mixed Acquire/Release — should not race or panic.
	for range 100 {
		wg.Go(func() {
			ac.Acquire()
		})
		wg.Go(func() {
			ac.Release()
		})
	}
	wg.Wait()

	// State is non-deterministic but must be valid.
	if ac.Len() < 0 {
		t.Fatalf("Len() = %d, must not be negative", ac.Len())
	}
}
