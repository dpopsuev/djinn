// e2e_bugle_test.go — E2E tests for Bugle v0.9.0+v0.10.0 features
// through Djinn's bugleport/ adapter layer.
//
// Tests: RoleRegistry, Ask, SendToRole, Broadcast, Staff facade,
// AgentHandle lifecycle, process supervision via facade.
// All through bugleport re-exports — proves the adapter layer works.
package acceptance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/bugleport"
)

// mockLauncherBugle tracks Start/Stop calls for Bugle E2E tests.
type mockLauncherBugle struct {
	mu      sync.Mutex
	started map[bugleport.EntityID]bool
	stopped map[bugleport.EntityID]bool
}

func newMockLauncherBugle() *mockLauncherBugle {
	return &mockLauncherBugle{
		started: make(map[bugleport.EntityID]bool),
		stopped: make(map[bugleport.EntityID]bool),
	}
}

func (m *mockLauncherBugle) Start(_ context.Context, id bugleport.EntityID, _ bugleport.LaunchConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started[id] = true
	return nil
}

func (m *mockLauncherBugle) Stop(_ context.Context, id bugleport.EntityID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped[id] = true
	return nil
}

func (m *mockLauncherBugle) Healthy(_ context.Context, id bugleport.EntityID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started[id] && !m.stopped[id]
}

// === v0.9.0: Inter-Agent Messaging through bugleport ===

func TestE2E_Bugle_AskThroughBugleport(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	agent, err := staff.Spawn(ctx, "worker", bugleport.LaunchConfig{})
	if err != nil {
		t.Fatal(err)
	}

	agent.Listen(func(content string) string {
		return "echo: " + content
	})

	response, err := agent.Ask(ctx, "hello bugle")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if response != "echo: hello bugle" {
		t.Fatalf("response = %q, want 'echo: hello bugle'", response)
	}

	staff.KillAll(ctx)
}

func TestE2E_Bugle_BroadcastThroughBugleport(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	var received atomic.Int32

	for i := range 3 {
		agent, _ := staff.Spawn(ctx, "executor", bugleport.LaunchConfig{})
		_ = i
		agent.Listen(func(content string) string {
			received.Add(1)
			return "ack"
		})
	}

	executors := staff.FindByRole("executor")
	if len(executors) != 3 {
		t.Fatalf("executors = %d, want 3", len(executors))
	}

	// Broadcast to all executors.
	err := executors[0].Broadcast(ctx, "build stage 3")
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	// Give goroutines time to process.
	time.Sleep(50 * time.Millisecond)

	if received.Load() != 3 {
		t.Fatalf("received = %d, want 3", received.Load())
	}

	staff.KillAll(ctx)
}

func TestE2E_Bugle_TellFireAndForget(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	var got atomic.Value

	agent, _ := staff.Spawn(ctx, "worker", bugleport.LaunchConfig{})
	agent.Listen(func(content string) string {
		got.Store(content)
		return ""
	})

	err := agent.Tell("fire and forget")
	if err != nil {
		t.Fatalf("Tell: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if got.Load() != "fire and forget" {
		t.Fatalf("got = %v, want 'fire and forget'", got.Load())
	}

	staff.KillAll(ctx)
}

// === v0.10.0: Facade through bugleport ===

func TestE2E_Bugle_FacadeSpawnHierarchy(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	gensec, _ := staff.Spawn(ctx, "gensec", bugleport.LaunchConfig{})
	executor, _ := gensec.Spawn(ctx, "executor", bugleport.LaunchConfig{})
	inspector, _ := gensec.Spawn(ctx, "inspector", bugleport.LaunchConfig{})

	// Verify hierarchy.
	if executor.Parent() == nil || executor.Parent().ID() != gensec.ID() {
		t.Fatal("executor parent should be gensec")
	}
	if inspector.Parent() == nil || inspector.Parent().ID() != gensec.ID() {
		t.Fatal("inspector parent should be gensec")
	}

	children := gensec.Children()
	if len(children) != 2 {
		t.Fatalf("gensec children = %d, want 2", len(children))
	}

	// Verify state.
	if !executor.IsAlive() {
		t.Fatal("executor should be alive")
	}
	if !executor.IsHealthy() {
		t.Fatal("executor should be healthy")
	}

	// Verify roles via FindByRole.
	execs := staff.FindByRole("executor")
	if len(execs) != 1 {
		t.Fatalf("FindByRole executor = %d, want 1", len(execs))
	}

	// Verify String() format.
	s := executor.String()
	if s == "" {
		t.Fatal("String() should not be empty")
	}

	staff.KillAll(ctx)
	if staff.Count() != 0 {
		t.Fatalf("count = %d after KillAll", staff.Count())
	}
}

func TestE2E_Bugle_FacadeKillWaitExitStatus(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	gensec, _ := staff.Spawn(ctx, "gensec", bugleport.LaunchConfig{})
	staff.Pool().SetAutoReap(gensec.ID(), false)

	executor, _ := gensec.Spawn(ctx, "executor", bugleport.LaunchConfig{})

	executor.KillWithReason(ctx, bugleport.ExitBudget)

	status, err := executor.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if status.Code != bugleport.ExitBudget {
		t.Fatalf("exit code = %d, want ExitBudget(%d)", status.Code, bugleport.ExitBudget)
	}
	if status.Duration <= 0 {
		t.Fatal("duration should be positive")
	}

	staff.KillAll(ctx)
}

func TestE2E_Bugle_FacadeOrphanReparenting(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	gensec, _ := staff.Spawn(ctx, "gensec", bugleport.LaunchConfig{})
	staff.SetSubreaper(gensec)

	scheduler, _ := gensec.Spawn(ctx, "scheduler", bugleport.LaunchConfig{})
	exec1, _ := scheduler.Spawn(ctx, "executor", bugleport.LaunchConfig{})
	exec2, _ := scheduler.Spawn(ctx, "executor", bugleport.LaunchConfig{})

	// Kill scheduler — orphans should be reparented to gensec.
	scheduler.Kill(ctx)

	// Verify reparenting via pool escape hatch.
	if staff.Pool().ParentOf(exec1.ID()) != gensec.ID() {
		t.Fatal("exec1 should be reparented to gensec")
	}
	if staff.Pool().ParentOf(exec2.ID()) != gensec.ID() {
		t.Fatal("exec2 should be reparented to gensec")
	}

	// Gensec now has 2 adopted children.
	if len(gensec.Children()) != 2 {
		t.Fatalf("gensec children = %d, want 2 (adopted)", len(gensec.Children()))
	}

	staff.KillAll(ctx)
}

func TestE2E_Bugle_FacadeProgressTracking(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	agent, _ := staff.Spawn(ctx, "executor", bugleport.LaunchConfig{})

	agent.SetProgress(3, 10)

	progress, ok := agent.Progress()
	if !ok {
		t.Fatal("progress should be attached")
	}
	if progress.Current != 3 || progress.Total != 10 {
		t.Fatalf("progress = %d/%d, want 3/10", progress.Current, progress.Total)
	}

	staff.KillAll(ctx)
}

func TestE2E_Bugle_FacadeSignalObservation(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	var signals atomic.Int32
	staff.OnSignal(func(sig bugleport.Signal) {
		signals.Add(1)
	})

	staff.Spawn(ctx, "worker", bugleport.LaunchConfig{})
	staff.Spawn(ctx, "worker", bugleport.LaunchConfig{})

	// Each spawn emits EventWorkerStarted.
	if signals.Load() < 2 {
		t.Fatalf("signals = %d, want >= 2", signals.Load())
	}

	staff.KillAll(ctx)
}

func TestE2E_Bugle_FacadeConcurrent(t *testing.T) {
	staff := bugleport.NewStaff(newMockLauncherBugle())
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			agent, err := staff.Spawn(ctx, fmt.Sprintf("worker-%d", n), bugleport.LaunchConfig{})
			if err != nil {
				t.Error(err)
				return
			}
			agent.Listen(func(content string) string { return "ack" })
			agent.Ask(ctx, "ping")
			agent.Kill(ctx)
		}(i)
	}
	wg.Wait()

	staff.KillAll(ctx)
	if staff.Count() != 0 {
		t.Fatalf("count = %d after concurrent KillAll", staff.Count())
	}
}
