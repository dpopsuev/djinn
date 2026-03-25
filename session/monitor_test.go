package session

import (
	"sync"
	"testing"
)

func TestNewContextMonitor_defaults(t *testing.T) {
	m := NewContextMonitor()
	if m.maxTokens != DefaultMaxTokens {
		t.Errorf("maxTokens = %d, want %d", m.maxTokens, DefaultMaxTokens)
	}
	if m.spawnAt != DefaultSpawnAt {
		t.Errorf("spawnAt = %f, want %f", m.spawnAt, DefaultSpawnAt)
	}
	if m.swapAt != DefaultSwapAt {
		t.Errorf("swapAt = %f, want %f", m.swapAt, DefaultSwapAt)
	}
	if m.State() != MonitorIdle {
		t.Errorf("state = %d, want MonitorIdle", m.State())
	}
}

func TestNewContextMonitor_options(t *testing.T) {
	m := NewContextMonitor(
		WithMaxTokens(100_000),
		WithSpawnAt(0.70),
		WithSwapAt(0.90),
	)
	if m.maxTokens != 100_000 {
		t.Errorf("maxTokens = %d, want 100000", m.maxTokens)
	}
	if m.spawnAt != 0.70 {
		t.Errorf("spawnAt = %f, want 0.70", m.spawnAt)
	}
	if m.swapAt != 0.90 {
		t.Errorf("swapAt = %f, want 0.90", m.swapAt)
	}
}

func TestContextMonitor_Usage(t *testing.T) {
	m := NewContextMonitor(WithMaxTokens(1000))
	m.Record(100, 100) // 200/1000 = 0.2
	got := m.Usage()
	if got != 0.2 {
		t.Errorf("Usage() = %f, want 0.2", got)
	}
}

func TestContextMonitor_Usage_zero_max(t *testing.T) {
	m := NewContextMonitor(WithMaxTokens(0))
	m.Record(100, 100)
	if m.Usage() != 0 {
		t.Error("Usage() should be 0 when maxTokens is 0")
	}
}

func TestContextMonitor_TotalTokens(t *testing.T) {
	m := NewContextMonitor()
	m.Record(500, 300)
	m.Record(200, 100)
	if m.TotalTokens() != 1100 {
		t.Errorf("TotalTokens() = %d, want 1100", m.TotalTokens())
	}
}

func TestContextMonitor_ShouldSpawn(t *testing.T) {
	m := NewContextMonitor(WithMaxTokens(1000), WithSpawnAt(0.80))

	// Below threshold.
	m.totalIn = 400
	if m.ShouldSpawn() {
		t.Error("ShouldSpawn() true at 40%, want false")
	}

	// At threshold, state still idle.
	m.totalIn = 800
	if !m.ShouldSpawn() {
		t.Error("ShouldSpawn() false at 80%, want true")
	}

	// After state transitions, ShouldSpawn returns false.
	m.SetState(MonitorSpawning)
	if m.ShouldSpawn() {
		t.Error("ShouldSpawn() true when spawning, want false")
	}
}

func TestContextMonitor_ShouldSwap(t *testing.T) {
	m := NewContextMonitor(WithMaxTokens(1000), WithSwapAt(0.95))

	// Set to ready state and tokens directly (bypass Record's state transitions).
	m.SetState(MonitorReady)
	m.totalIn = 950
	if !m.ShouldSwap() {
		t.Error("ShouldSwap() false at 95% and ready, want true")
	}
}

func TestContextMonitor_ShouldSwap_not_ready(t *testing.T) {
	m := NewContextMonitor(WithMaxTokens(1000), WithSwapAt(0.95))

	// Idle state — even above threshold, don't swap.
	m.Record(960, 0) // 0.96
	if m.ShouldSwap() {
		t.Error("ShouldSwap() true when idle, want false")
	}
}

func TestContextMonitor_Record_fires_spawn(t *testing.T) {
	var fired bool
	m := NewContextMonitor(
		WithMaxTokens(1000),
		WithSpawnAt(0.80),
		WithOnSpawn(func() { fired = true }),
	)

	// First call below threshold.
	m.Record(700, 0)
	if fired {
		t.Fatal("onSpawn fired below threshold")
	}

	// This pushes over 80%.
	if !m.Record(200, 0) {
		t.Error("Record() should return true when callback fired")
	}
	if !fired {
		t.Error("onSpawn not fired at 90%")
	}
	if m.State() != MonitorSpawning {
		t.Errorf("state = %d, want MonitorSpawning", m.State())
	}
}

func TestContextMonitor_Record_fires_swap(t *testing.T) {
	var fired bool
	m := NewContextMonitor(
		WithMaxTokens(1000),
		WithSwapAt(0.95),
		WithOnSwap(func() { fired = true }),
	)

	m.SetState(MonitorReady)
	if !m.Record(960, 0) {
		t.Error("Record() should return true when swap callback fired")
	}
	if !fired {
		t.Error("onSwap not fired at 96%")
	}
	if m.State() != MonitorSwapping {
		t.Errorf("state = %d, want MonitorSwapping", m.State())
	}
}

func TestContextMonitor_Record_no_double_spawn(t *testing.T) {
	count := 0
	m := NewContextMonitor(
		WithMaxTokens(1000),
		WithSpawnAt(0.80),
		WithOnSpawn(func() { count++ }),
	)

	m.Record(900, 0) // fires spawn
	m.Record(50, 0)  // already spawning, no second fire

	if count != 1 {
		t.Errorf("onSpawn called %d times, want 1", count)
	}
}

func TestContextMonitor_Reset(t *testing.T) {
	m := NewContextMonitor(WithMaxTokens(1000))
	m.Record(500, 500)
	m.SetState(MonitorSpawning)

	m.Reset()
	if m.TotalTokens() != 0 {
		t.Errorf("TotalTokens() = %d after Reset, want 0", m.TotalTokens())
	}
	if m.State() != MonitorIdle {
		t.Errorf("state = %d after Reset, want MonitorIdle", m.State())
	}
}

func TestContextMonitor_state_transitions(t *testing.T) {
	m := NewContextMonitor(
		WithMaxTokens(1000),
		WithSpawnAt(0.50),
		WithSwapAt(0.90),
		WithOnSpawn(func() {}),
		WithOnSwap(func() {}),
	)

	// idle → spawning
	m.Record(600, 0)
	if m.State() != MonitorSpawning {
		t.Fatalf("expected MonitorSpawning, got %d", m.State())
	}

	// Spawn won't fire again.
	m.Record(100, 0)
	if m.State() != MonitorSpawning {
		t.Fatalf("expected MonitorSpawning (no change), got %d", m.State())
	}

	// External transition to ready.
	m.SetState(MonitorReady)

	// ready → swapping
	m.Record(300, 0) // total = 1000, usage = 1.0
	if m.State() != MonitorSwapping {
		t.Fatalf("expected MonitorSwapping, got %d", m.State())
	}
}

func TestContextMonitor_concurrent(t *testing.T) {
	m := NewContextMonitor(WithMaxTokens(1_000_000))
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Record(100, 100)
			m.Usage()
			m.TotalTokens()
			m.State()
			m.ShouldSpawn()
			m.ShouldSwap()
		}()
	}
	wg.Wait()

	if m.TotalTokens() != 20_000 {
		t.Errorf("TotalTokens() = %d after concurrent, want 20000", m.TotalTokens())
	}
}
