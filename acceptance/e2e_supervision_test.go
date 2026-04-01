package acceptance

import (
	"context"
	"sync"
	"testing"

	"github.com/dpopsuev/djinn/jerichoport"
	"github.com/dpopsuev/djinn/staff"
)

// mockLauncher tracks Start/Stop calls for process supervision E2E testing.
type mockLauncher struct {
	mu      sync.Mutex
	started map[jerichoport.EntityID]bool
	stopped map[jerichoport.EntityID]bool
}

func newMockLauncher() *mockLauncher {
	return &mockLauncher{
		started: make(map[jerichoport.EntityID]bool),
		stopped: make(map[jerichoport.EntityID]bool),
	}
}

func (m *mockLauncher) Start(_ context.Context, id jerichoport.EntityID, _ jerichoport.AgentConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started[id] = true
	return nil
}

func (m *mockLauncher) Stop(_ context.Context, id jerichoport.EntityID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped[id] = true
	return nil
}

func (m *mockLauncher) Healthy(_ context.Context, id jerichoport.EntityID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started[id] && !m.stopped[id]
}

// TestE2E_ProcessSupervision exercises the full Djinn staff process supervision
// lifecycle through StaffWorld using the Bugle facade: spawn, tree, orphan
// reparenting, exit codes, zombie reaping, and clean shutdown.
func TestE2E_ProcessSupervision(t *testing.T) {
	ctx := context.Background()
	launcher := newMockLauncher()
	sw := staff.NewStaffWorld(launcher)

	// Escape hatch for operations the facade doesn't expose yet.
	pool := sw.Pool()

	// --- Phase 1: Spawn GenSec as root ---
	gensec, err := sw.Spawn(ctx, "gensec", jerichoport.AgentConfig{
		Role:  "gensec",
		Model: "haiku",
	})
	if err != nil {
		t.Fatalf("Spawn gensec: %v", err)
	}
	if gensec.ID() == 0 {
		t.Fatal("gensec ID should not be 0")
	}

	// --- Phase 2: SetSubreaper(gensec) ---
	sw.SetSubreaper(gensec)
	pool.SetAutoReap(gensec.ID(), false) // GenSec explicitly reaps

	// --- Phase 3: Spawn Scheduler under GenSec ---
	scheduler, err := gensec.Spawn(ctx, "scheduler", jerichoport.AgentConfig{
		Role:  "scheduler",
		Model: "sonnet",
	})
	if err != nil {
		t.Fatalf("Spawn scheduler: %v", err)
	}
	pool.SetAutoReap(scheduler.ID(), false) // Scheduler explicitly reaps

	// --- Phase 4: Spawn 3 Executors under Scheduler ---
	exec1, err := scheduler.Spawn(ctx, "executor", jerichoport.AgentConfig{
		Role:  "executor",
		Model: "opus",
	})
	if err != nil {
		t.Fatalf("Spawn executor-1: %v", err)
	}
	exec2, err := scheduler.Spawn(ctx, "executor", jerichoport.AgentConfig{
		Role:  "executor",
		Model: "opus",
	})
	if err != nil {
		t.Fatalf("Spawn executor-2: %v", err)
	}
	exec3, err := scheduler.Spawn(ctx, "executor", jerichoport.AgentConfig{
		Role:  "executor",
		Model: "opus",
	})
	if err != nil {
		t.Fatalf("Spawn executor-3: %v", err)
	}

	// --- Phase 5: Verify Tree shows 3-level hierarchy ---
	// gensec -> scheduler -> [exec1, exec2, exec3]
	if sw.Count() != 5 {
		t.Fatalf("count = %d, want 5", sw.Count())
	}

	tree := sw.Tree(gensec)
	if tree == nil {
		t.Fatal("tree should not be nil")
	}
	if tree.Role != "gensec" {
		t.Fatalf("root role = %q, want gensec", tree.Role)
	}
	if len(tree.Children) != 1 {
		t.Fatalf("gensec children = %d, want 1 (scheduler)", len(tree.Children))
	}
	schedulerNode := tree.Children[0]
	if schedulerNode.Role != "scheduler" {
		t.Fatalf("mid-level role = %q, want scheduler", schedulerNode.Role)
	}
	if len(schedulerNode.Children) != 3 {
		t.Fatalf("scheduler children = %d, want 3", len(schedulerNode.Children))
	}

	// --- Phase 6: Kill Scheduler -> verify orphans reparented to GenSec ---
	if err := scheduler.Kill(ctx); err != nil {
		t.Fatalf("Kill scheduler: %v", err)
	}

	// Executors should now be children of GenSec (subreaper).
	if pool.ParentOf(exec1.ID()) != gensec.ID() {
		t.Fatalf("exec1 parent = %d, want gensec %d", pool.ParentOf(exec1.ID()), gensec.ID())
	}
	if pool.ParentOf(exec2.ID()) != gensec.ID() {
		t.Fatalf("exec2 parent = %d, want gensec %d", pool.ParentOf(exec2.ID()), gensec.ID())
	}
	if pool.ParentOf(exec3.ID()) != gensec.ID() {
		t.Fatalf("exec3 parent = %d, want gensec %d", pool.ParentOf(exec3.ID()), gensec.ID())
	}

	// GenSec should now have 3 adopted children.
	children := gensec.Children()
	if len(children) != 3 {
		t.Fatalf("gensec children after reparenting = %d, want 3", len(children))
	}

	// Reap the scheduler zombie.
	schedStatus := pool.WaitAny(gensec.ID())
	if schedStatus == nil {
		t.Fatal("scheduler should be a zombie under gensec")
	}
	if schedStatus.Role != "scheduler" {
		t.Fatalf("scheduler exit role = %q", schedStatus.Role)
	}

	// --- Phase 7: KillWithReason each executor with different exit codes ---
	if err := exec1.KillWithReason(ctx, jerichoport.ExitSuccess); err != nil {
		t.Fatalf("KillWithReason exec1: %v", err)
	}
	if err := exec2.KillWithReason(ctx, jerichoport.ExitBudget); err != nil {
		t.Fatalf("KillWithReason exec2: %v", err)
	}
	if err := exec3.KillWithReason(ctx, jerichoport.ExitError); err != nil {
		t.Fatalf("KillWithReason exec3: %v", err)
	}

	// --- Phase 8: Wait for each -> verify correct ExitStatus codes ---
	status1, err := exec1.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait exec1: %v", err)
	}
	if status1.Code != jerichoport.ExitSuccess {
		t.Fatalf("exec1 exit code = %d, want ExitSuccess (%d)", status1.Code, jerichoport.ExitSuccess)
	}
	if status1.Duration <= 0 {
		t.Fatal("exec1 duration should be positive")
	}

	status2, err := exec2.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait exec2: %v", err)
	}
	if status2.Code != jerichoport.ExitBudget {
		t.Fatalf("exec2 exit code = %d, want ExitBudget (%d)", status2.Code, jerichoport.ExitBudget)
	}

	status3, err := exec3.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait exec3: %v", err)
	}
	if status3.Code != jerichoport.ExitError {
		t.Fatalf("exec3 exit code = %d, want ExitError (%d)", status3.Code, jerichoport.ExitError)
	}

	// --- Phase 9: Verify zero zombies ---
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombie count = %d, want 0 after reaping all", pool.ZombieCount())
	}

	// --- Phase 10: Verify launcher tracked all starts and stops ---
	launcher.mu.Lock()
	for _, h := range []*jerichoport.Solo{gensec, scheduler, exec1, exec2, exec3} {
		if !launcher.started[h.ID()] {
			t.Errorf("entity %d was not started", h.ID())
		}
	}
	for _, h := range []*jerichoport.Solo{scheduler, exec1, exec2, exec3} {
		if !launcher.stopped[h.ID()] {
			t.Errorf("entity %d was not stopped", h.ID())
		}
	}
	launcher.mu.Unlock()

	// --- Phase 11: KillAll -> verify clean shutdown ---
	sw.KillAll(ctx)
	if sw.Count() != 0 {
		t.Fatalf("count = %d after KillAll, want 0", sw.Count())
	}
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombie count = %d after KillAll, want 0", pool.ZombieCount())
	}

	// Verify GenSec was also stopped by KillAll.
	launcher.mu.Lock()
	if !launcher.stopped[gensec.ID()] {
		t.Fatal("gensec should have been stopped by KillAll")
	}
	launcher.mu.Unlock()
}
