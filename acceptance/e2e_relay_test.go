package acceptance

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
)

// TestE2E_RelayContextSwap exercises the full relay lifecycle end-to-end:
//
//	idle → record 80% → spawn background → record 95% → swap → verify
//
// Uses ScriptedDriver as the factory, a real Store, and a real ContextMonitor
// with maxTokens=1000 so thresholds are reached quickly.
func TestE2E_RelayContextSwap(t *testing.T) {
	ctx := context.Background()

	// --- Setup: Store, Monitor, Session, ScriptedDriver ---
	store, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	monitor := session.NewContextMonitor(
		session.WithMaxTokens(1000),
		session.WithSpawnAt(0.80),
		session.WithSwapAt(0.95),
	)

	// Build a session with enough history to trigger compaction during seed.
	oldSess := session.New("e2e-relay", "opus-4", "/workspace")
	oldSess.Name = "e2e-relay"
	oldSess.Driver = "acp"
	oldSess.Workspace = "aeon"
	oldSess.Append(session.Entry{Role: "user", Content: "implement auth module"})
	oldSess.Append(session.Entry{Role: "assistant", Content: "Created auth package with JWT middleware."})
	oldSess.Append(session.Entry{Role: "user", Content: "add unit tests"})
	oldSess.Append(session.Entry{Role: "assistant", Content: "Added 12 tests, all green."})
	oldSess.Append(session.Entry{Role: "user", Content: "add rate limiting"})
	oldSess.Append(session.Entry{Role: "assistant", Content: "Token bucket rate limiter added."})

	if err := store.Save(oldSess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Old driver is a ScriptedDriver (no steps needed — relay only calls Start/Stop/Send).
	oldDriver := stubs.NewScriptedDriver()

	// New driver produced by factory — also a ScriptedDriver.
	newDriver := stubs.NewScriptedDriver()
	factoryCalled := false

	relay := session.NewRelayManager(session.RelayConfig{
		Monitor: monitor,
		Store:   store,
		Session: oldSess,
		Driver:  oldDriver,
		DriverFactory: func() (driver.ChatDriver, error) {
			factoryCalled = true
			return newDriver, nil
		},
		Log: slog.Default(),
	})

	// --- Phase 1: Below threshold — no action ---
	monitor.Record(200, 100) // 300/1000 = 30%

	sess, drv, err := relay.CheckAndRelay(ctx)
	if err != nil {
		t.Fatalf("CheckAndRelay at 30%%: %v", err)
	}
	if sess.ID != "e2e-relay" {
		t.Fatal("session should not change below threshold")
	}
	if drv != oldDriver {
		t.Fatal("driver should not change below threshold")
	}
	if monitor.State() != session.MonitorIdle {
		t.Fatalf("state = %d, want MonitorIdle at 30%%", monitor.State())
	}
	if factoryCalled {
		t.Fatal("factory should not be called below threshold")
	}

	// --- Phase 2: Push past 80% → verify background session spawned ---
	monitor.Record(300, 200) // total 800/1000 = 80%

	sess, drv, err = relay.CheckAndRelay(ctx)
	if err != nil {
		t.Fatalf("CheckAndRelay at 80%%: %v", err)
	}
	// Active session still the old one during spawn.
	if sess.ID != "e2e-relay" {
		t.Fatal("active session should remain old during spawn")
	}
	if drv != oldDriver {
		t.Fatal("active driver should remain old during spawn")
	}
	if monitor.State() != session.MonitorReady {
		t.Fatalf("state = %d, want MonitorReady after spawn", monitor.State())
	}
	if !factoryCalled {
		t.Fatal("factory should have been called at 80%%")
	}
	// New driver should have been started.
	if !newDriver.Started() {
		t.Fatal("backup driver should have been started")
	}
	// New driver should have received seed messages via Send.
	if len(newDriver.SendLog) == 0 {
		t.Fatal("backup driver should have received seed messages")
	}
	// All replayed messages should be user role.
	for _, msg := range newDriver.SendLog {
		if msg.Role != driver.RoleUser {
			t.Errorf("replayed msg role = %q, want user", msg.Role)
		}
	}

	// --- Phase 3: Queue prompts during the ready state ---
	relay.QueuePrompt("continue with rate limiter")
	relay.QueuePrompt("also add metrics")

	// --- Phase 4: Push past 95% → verify swap executed ---
	monitor.Record(100, 50) // total 950/1000 = 95%

	sess, drv, err = relay.CheckAndRelay(ctx)
	if err != nil {
		t.Fatalf("CheckAndRelay at 95%%: %v", err)
	}

	// Active session should now be the new one.
	if sess.ID == "e2e-relay" {
		t.Fatal("session should have changed after swap")
	}
	if drv != newDriver {
		t.Fatal("driver should be the new ScriptedDriver after swap")
	}

	// Old driver should be stopped.
	if !oldDriver.Stopped() {
		t.Fatal("old driver should have been stopped after swap")
	}

	// Monitor should be reset.
	if monitor.State() != session.MonitorIdle {
		t.Fatalf("state = %d after swap, want MonitorIdle (reset)", monitor.State())
	}
	if monitor.TotalTokens() != 0 {
		t.Errorf("tokens = %d after swap, want 0 (reset)", monitor.TotalTokens())
	}

	// --- Phase 5: Verify old session archived ---
	archived, err := store.ListArchived()
	if err != nil {
		t.Fatalf("ListArchived: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("archived count = %d, want 1", len(archived))
	}
	if archived[0].Name != "e2e-relay" {
		t.Errorf("archived name = %q, want e2e-relay", archived[0].Name)
	}

	// Verify archived session is loadable with full history.
	loadedArchive, err := store.LoadArchived("e2e-relay")
	if err != nil {
		t.Fatalf("LoadArchived: %v", err)
	}
	if loadedArchive.History.Len() != 6 {
		t.Errorf("archived history = %d, want 6", loadedArchive.History.Len())
	}

	// Old session should be removed from active list.
	active, _ := store.List()
	for _, s := range active {
		if s.Name == "e2e-relay" {
			t.Fatal("old session should not be in active list after archive")
		}
	}

	// --- Phase 6: Verify new session has seeded entries ---
	entries := sess.Entries()
	if len(entries) == 0 {
		t.Fatal("new session should have seeded entries")
	}
	// First entry should be the compacted summary.
	if !strings.Contains(entries[0].Content, "[Session context]") {
		t.Errorf("first entry should be session context, got: %q", entries[0].Content)
	}
	// New session should inherit metadata from old session.
	if sess.Driver != "acp" {
		t.Errorf("new session Driver = %q, want acp", sess.Driver)
	}
	if sess.Workspace != "aeon" {
		t.Errorf("new session Workspace = %q, want aeon", sess.Workspace)
	}

	// --- Phase 7: Verify queued prompts drained ---
	q := relay.DrainQueue()
	if len(q) != 2 {
		t.Fatalf("queue = %d, want 2", len(q))
	}
	if q[0] != "continue with rate limiter" {
		t.Errorf("queue[0] = %q", q[0])
	}
	if q[1] != "also add metrics" {
		t.Errorf("queue[1] = %q", q[1])
	}

	// Drain again — should be empty.
	q2 := relay.DrainQueue()
	if len(q2) != 0 {
		t.Errorf("queue after second drain = %d, want 0", len(q2))
	}

	// --- Phase 8: Verify new session continues normally ---
	monitor.Record(50, 50) // 100/1000 = 10%
	if monitor.Usage() < 0.09 || monitor.Usage() > 0.11 {
		t.Errorf("usage after reset = %f, want ~0.10", monitor.Usage())
	}
}

// TestE2E_RelayFallbackCompact verifies that when no driver factory is
// configured, the relay falls back to in-place compaction and the monitor
// returns to idle.
func TestE2E_RelayFallbackCompact(t *testing.T) {
	ctx := context.Background()

	store, _ := session.NewStore(t.TempDir())
	monitor := session.NewContextMonitor(
		session.WithMaxTokens(1000),
		session.WithSpawnAt(0.80),
		session.WithSwapAt(0.95),
	)

	// Build a session with enough entries to compact.
	sess := session.New("fallback-e2e", "model", "/work")
	for i := range 10 {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		sess.Append(session.Entry{Role: role, Content: "message about topic " + string(rune('A'+i))})
	}
	beforeEntries := sess.History.Len()

	oldDriver := stubs.NewScriptedDriver()

	relay := session.NewRelayManager(session.RelayConfig{
		Monitor:       monitor,
		Store:         store,
		Session:       sess,
		Driver:        oldDriver,
		DriverFactory: nil, // no factory — will force fallback
		Log:           slog.Default(),
	})

	// Push past 80%.
	monitor.Record(850, 0) // 850/1000 = 85%

	newSess, newDrv, err := relay.CheckAndRelay(ctx)
	if err != nil {
		t.Fatalf("CheckAndRelay: %v", err)
	}

	// Should fallback to compact — same session, same driver.
	if newSess.ID != "fallback-e2e" {
		t.Fatal("session should not change on fallback")
	}
	if newDrv != oldDriver {
		t.Fatal("driver should not change on fallback")
	}

	// Session should be compacted (fewer entries).
	if newSess.History.Len() >= beforeEntries {
		t.Fatalf("history should be compacted: %d -> %d", beforeEntries, newSess.History.Len())
	}

	// Monitor should be back to idle (ready to try again next cycle).
	if monitor.State() != session.MonitorIdle {
		t.Fatalf("state = %d, want MonitorIdle after fallback", monitor.State())
	}
}
