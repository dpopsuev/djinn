package acceptance

import (
	"context"
	"sync"
	"testing"

	"github.com/dpopsuev/djinn/bugleport"
	"github.com/dpopsuev/djinn/staff"
)

// mockLauncher tracks Start/Stop calls for process supervision E2E testing.
type mockLauncher struct {
	mu      sync.Mutex
	started map[bugleport.EntityID]bool
	stopped map[bugleport.EntityID]bool
}

func newMockLauncher() *mockLauncher {
	return &mockLauncher{
		started: make(map[bugleport.EntityID]bool),
		stopped: make(map[bugleport.EntityID]bool),
	}
}

func (m *mockLauncher) Start(_ context.Context, id bugleport.EntityID, _ bugleport.LaunchConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started[id] = true
	return nil
}

func (m *mockLauncher) Stop(_ context.Context, id bugleport.EntityID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped[id] = true
	return nil
}

func (m *mockLauncher) Healthy(_ context.Context, id bugleport.EntityID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started[id] && !m.stopped[id]
}

// TestE2E_ProcessSupervision exercises the full Djinn staff process supervision
// lifecycle through StaffWorld: fork, tree, orphan reparenting, exit codes,
// zombie reaping, and clean shutdown.
func TestE2E_ProcessSupervision(t *testing.T) {
	ctx := context.Background()
	launcher := newMockLauncher()
	sw := staff.NewStaffWorld(launcher)
	pool := sw.Pool

	// --- Phase 1: Fork GenSec as root (parentID=0) ---
	gensecID, err := pool.Fork(ctx, "gensec", bugleport.LaunchConfig{
		Role:  "gensec",
		Model: "haiku",
	}, 0)
	if err != nil {
		t.Fatalf("Fork gensec: %v", err)
	}
	if gensecID == 0 {
		t.Fatal("gensec ID should not be 0")
	}

	// --- Phase 2: SetSubreaper(gensecID) ---
	pool.SetSubreaper(gensecID)
	pool.SetAutoReap(gensecID, false) // GenSec explicitly reaps

	// --- Phase 3: Fork Scheduler under GenSec ---
	schedulerID, err := pool.Fork(ctx, "scheduler", bugleport.LaunchConfig{
		Role:  "scheduler",
		Model: "sonnet",
	}, gensecID)
	if err != nil {
		t.Fatalf("Fork scheduler: %v", err)
	}
	pool.SetAutoReap(schedulerID, false) // Scheduler explicitly reaps

	// --- Phase 4: Fork 3 Executors under Scheduler ---
	exec1, err := pool.Fork(ctx, "executor-1", bugleport.LaunchConfig{
		Role:  "executor",
		Model: "opus",
	}, schedulerID)
	if err != nil {
		t.Fatalf("Fork executor-1: %v", err)
	}
	exec2, err := pool.Fork(ctx, "executor-2", bugleport.LaunchConfig{
		Role:  "executor",
		Model: "opus",
	}, schedulerID)
	if err != nil {
		t.Fatalf("Fork executor-2: %v", err)
	}
	exec3, err := pool.Fork(ctx, "executor-3", bugleport.LaunchConfig{
		Role:  "executor",
		Model: "opus",
	}, schedulerID)
	if err != nil {
		t.Fatalf("Fork executor-3: %v", err)
	}

	// --- Phase 5: Verify Tree shows 3-level hierarchy ---
	// gensec -> scheduler -> [exec1, exec2, exec3]
	if pool.Count() != 5 {
		t.Fatalf("count = %d, want 5", pool.Count())
	}

	tree := pool.Tree(gensecID)
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
	if err := pool.Kill(ctx, schedulerID); err != nil {
		t.Fatalf("Kill scheduler: %v", err)
	}

	// Executors should now be children of GenSec (subreaper).
	if pool.ParentOf(exec1) != gensecID {
		t.Fatalf("exec1 parent = %d, want gensec %d", pool.ParentOf(exec1), gensecID)
	}
	if pool.ParentOf(exec2) != gensecID {
		t.Fatalf("exec2 parent = %d, want gensec %d", pool.ParentOf(exec2), gensecID)
	}
	if pool.ParentOf(exec3) != gensecID {
		t.Fatalf("exec3 parent = %d, want gensec %d", pool.ParentOf(exec3), gensecID)
	}

	// GenSec should now have 3 adopted children.
	children := pool.Children(gensecID)
	if len(children) != 3 {
		t.Fatalf("gensec children after reparenting = %d, want 3", len(children))
	}

	// Reap the scheduler zombie.
	schedStatus := pool.WaitAny(gensecID)
	if schedStatus == nil {
		t.Fatal("scheduler should be a zombie under gensec")
	}
	if schedStatus.Role != "scheduler" {
		t.Fatalf("scheduler exit role = %q", schedStatus.Role)
	}

	// --- Phase 7: KillWithCode each executor with different exit codes ---
	if err := pool.KillWithCode(ctx, exec1, bugleport.ExitSuccess); err != nil {
		t.Fatalf("KillWithCode exec1: %v", err)
	}
	if err := pool.KillWithCode(ctx, exec2, bugleport.ExitBudget); err != nil {
		t.Fatalf("KillWithCode exec2: %v", err)
	}
	if err := pool.KillWithCode(ctx, exec3, bugleport.ExitError); err != nil {
		t.Fatalf("KillWithCode exec3: %v", err)
	}

	// --- Phase 8: Wait for each -> verify correct ExitStatus codes ---
	status1, err := pool.Wait(ctx, exec1)
	if err != nil {
		t.Fatalf("Wait exec1: %v", err)
	}
	if status1.Code != bugleport.ExitSuccess {
		t.Fatalf("exec1 exit code = %d, want ExitSuccess (%d)", status1.Code, bugleport.ExitSuccess)
	}
	if status1.Duration <= 0 {
		t.Fatal("exec1 duration should be positive")
	}

	status2, err := pool.Wait(ctx, exec2)
	if err != nil {
		t.Fatalf("Wait exec2: %v", err)
	}
	if status2.Code != bugleport.ExitBudget {
		t.Fatalf("exec2 exit code = %d, want ExitBudget (%d)", status2.Code, bugleport.ExitBudget)
	}

	status3, err := pool.Wait(ctx, exec3)
	if err != nil {
		t.Fatalf("Wait exec3: %v", err)
	}
	if status3.Code != bugleport.ExitError {
		t.Fatalf("exec3 exit code = %d, want ExitError (%d)", status3.Code, bugleport.ExitError)
	}

	// --- Phase 9: Verify zero zombies ---
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombie count = %d, want 0 after reaping all", pool.ZombieCount())
	}

	// --- Phase 10: Verify launcher tracked all starts and stops ---
	launcher.mu.Lock()
	for _, id := range []bugleport.EntityID{gensecID, schedulerID, exec1, exec2, exec3} {
		if !launcher.started[id] {
			t.Errorf("entity %d was not started", id)
		}
	}
	for _, id := range []bugleport.EntityID{schedulerID, exec1, exec2, exec3} {
		if !launcher.stopped[id] {
			t.Errorf("entity %d was not stopped", id)
		}
	}
	launcher.mu.Unlock()

	// --- Phase 11: KillAll -> verify clean shutdown ---
	pool.KillAll(ctx)
	if pool.Count() != 0 {
		t.Fatalf("count = %d after KillAll, want 0", pool.Count())
	}
	if pool.ZombieCount() != 0 {
		t.Fatalf("zombie count = %d after KillAll, want 0", pool.ZombieCount())
	}

	// Verify GenSec was also stopped by KillAll.
	launcher.mu.Lock()
	if !launcher.stopped[gensecID] {
		t.Fatal("gensec should have been stopped by KillAll")
	}
	launcher.mu.Unlock()
}
