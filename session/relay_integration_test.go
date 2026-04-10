package session

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

// TestRelayIntegration_FullLifecycle wires Monitor + RelayManager + Store + DriverFactory
// with tiny maxTokens (1000) and drives the complete relay lifecycle:
//
//	idle → record → 80% spawn → ready → record → 95% swap → new session active
//	                                                        → old session archived
//	                                                        → queued prompts drained
//	                                                        → monitor reset
//
// Covers SPC-57 Gherkin scenarios. No real LLM calls.
func TestRelayIntegration_FullLifecycle(t *testing.T) {
	ctx := context.Background()

	// --- Setup ---
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	monitor := NewContextMonitor(
		WithMaxTokens(1000),
		WithSpawnAt(0.80),
		WithSwapAt(0.95),
	)

	// Old session with enough history to compact.
	oldSess := New("relay-old", "claude-4", "/workspace")
	oldSess.Name = "relay-old"
	oldSess.Driver = "acp"
	oldSess.Workspace = "aeon"
	oldSess.Append(Entry{Role: "user", Content: "implement the auth module"})
	oldSess.Append(Entry{Role: "assistant", Content: "I'll create the auth package with JWT middleware."})
	oldSess.Append(Entry{Role: "user", Content: "add tests"})
	oldSess.Append(Entry{Role: "assistant", Content: "Done. 12 tests, all passing."})
	oldSess.Append(Entry{Role: "user", Content: "now add rate limiting"})
	oldSess.Append(Entry{Role: "assistant", Content: "Added token bucket rate limiter."})

	// Save old session to store.
	if err := store.Save(oldSess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	oldDriver := &mockChatDriver{}
	newDriver := &mockChatDriver{}
	factoryCalled := false

	relay := NewRelayManager(RelayConfig{
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
	if sess.ID != "relay-old" {
		t.Fatal("session should not change below threshold")
	}
	if drv != oldDriver {
		t.Fatal("driver should not change below threshold")
	}
	if monitor.State() != MonitorIdle {
		t.Fatalf("state = %d, want idle at 30%%", monitor.State())
	}
	if factoryCalled {
		t.Fatal("factory should not be called below threshold")
	}

	// --- Phase 2: Hit 80% — spawn background ---
	monitor.Record(300, 200) // total 800/1000 = 80%

	sess, drv, err = relay.CheckAndRelay(ctx)
	if err != nil {
		t.Fatalf("CheckAndRelay at 80%%: %v", err)
	}
	if sess.ID != "relay-old" {
		t.Fatal("active session should still be old during spawn")
	}
	if drv != oldDriver {
		t.Fatal("active driver should still be old during spawn")
	}
	if monitor.State() != MonitorReady {
		t.Fatalf("state = %d, want MonitorReady after spawn", monitor.State())
	}
	if !factoryCalled {
		t.Fatal("factory should have been called at 80%")
	}
	if !newDriver.started {
		t.Fatal("backup driver should have been started")
	}
	if relay.backupSession == nil {
		t.Fatal("backup session should exist")
	}

	// Verify backup session is seeded with summary + recent entries.
	backupEntries := relay.backupSession.Entries()
	if len(backupEntries) == 0 {
		t.Fatal("backup session should have seeded entries")
	}
	// First entry should be the compacted summary.
	if !strings.Contains(backupEntries[0].Content, "[Session context]") {
		t.Errorf("first entry should be session context, got: %q", backupEntries[0].Content)
	}
	// Backup session should inherit metadata from old session.
	if relay.backupSession.Driver != "acp" {
		t.Errorf("backup Driver = %q, want acp", relay.backupSession.Driver)
	}
	if relay.backupSession.Workspace != "aeon" {
		t.Errorf("backup Workspace = %q, want aeon", relay.backupSession.Workspace)
	}

	// Verify backup driver received seed messages.
	if len(newDriver.messages) == 0 {
		t.Fatal("backup driver should have received seed messages via Send")
	}

	// --- Phase 3: Queue prompts during the ready state ---
	relay.QueuePrompt("continue with the rate limiter")
	relay.QueuePrompt("also add metrics")

	// --- Phase 4: Hit 95% — execute swap ---
	monitor.Record(100, 50) // total 950/1000 = 95%

	sess, drv, err = relay.CheckAndRelay(ctx)
	if err != nil {
		t.Fatalf("CheckAndRelay at 95%%: %v", err)
	}

	// Active session should now be the backup.
	if sess.ID == "relay-old" {
		t.Fatal("session should have changed after swap")
	}
	if drv != newDriver {
		t.Fatal("driver should be the new driver after swap")
	}

	// Old driver should be stopped.
	if !oldDriver.stopped {
		t.Fatal("old driver should have been stopped after swap")
	}

	// Monitor should be reset.
	if monitor.State() != MonitorIdle {
		t.Fatalf("state = %d after swap, want MonitorIdle (reset)", monitor.State())
	}
	if monitor.TotalTokens() != 0 {
		t.Errorf("tokens = %d after swap, want 0 (reset)", monitor.TotalTokens())
	}

	// Backup references should be nil.
	if relay.backupSession != nil {
		t.Fatal("backup session should be nil after swap")
	}
	if relay.backupDriver != nil {
		t.Fatal("backup driver should be nil after swap")
	}

	// --- Phase 5: Verify old session archived ---
	archived, err := store.ListArchived()
	if err != nil {
		t.Fatalf("ListArchived: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("archived = %d, want 1", len(archived))
	}
	if archived[0].Name != "relay-old" {
		t.Errorf("archived name = %q, want relay-old", archived[0].Name)
	}

	// Verify archived session is loadable with full history.
	loadedArchive, err := store.LoadArchived("relay-old")
	if err != nil {
		t.Fatalf("LoadArchived: %v", err)
	}
	if loadedArchive.History.Len() != 6 {
		t.Errorf("archived history = %d, want 6", loadedArchive.History.Len())
	}

	// Old session should be removed from active list.
	active, _ := store.List()
	for _, s := range active {
		if s.Name == "relay-old" {
			t.Fatal("old session should not be in active list after archive")
		}
	}

	// --- Phase 6: Verify queued prompts preserved ---
	q := relay.DrainQueue()
	if len(q) != 2 {
		t.Fatalf("queue = %d, want 2", len(q))
	}
	if q[0] != "continue with the rate limiter" {
		t.Errorf("queue[0] = %q", q[0])
	}
	if q[1] != "also add metrics" {
		t.Errorf("queue[1] = %q", q[1])
	}

	// --- Phase 7: Verify new session can continue ---
	// Record more tokens on the fresh monitor — should work normally.
	monitor.Record(50, 50) // 100/1000 = 10%
	if monitor.Usage() < 0.09 || monitor.Usage() > 0.11 {
		t.Errorf("usage after reset = %f, want ~0.10", monitor.Usage())
	}
}

// TestRelayIntegration_FallbackCompact verifies that when the driver factory
// fails, the relay falls back to in-place compaction without crashing.
func TestRelayIntegration_FallbackCompact(t *testing.T) {
	ctx := context.Background()

	store, _ := NewStore(t.TempDir())
	monitor := NewContextMonitor(
		WithMaxTokens(1000),
		WithSpawnAt(0.80),
	)

	sess := New("fallback-test", "model", "/work")
	for i := range 10 {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		sess.Append(Entry{Role: role, Content: "message about topic " + string(rune('A'+i))})
	}
	beforeEntries := sess.History.Len()

	oldDriver := &mockChatDriver{}

	relay := NewRelayManager(RelayConfig{
		Monitor:       monitor,
		Store:         store,
		Session:       sess,
		Driver:        oldDriver,
		DriverFactory: nil, // no factory — will fail
		Log:           slog.Default(),
	})

	// Push past 80%.
	monitor.Record(850, 0)

	newSess, newDrv, err := relay.CheckAndRelay(ctx)
	if err != nil {
		t.Fatalf("CheckAndRelay: %v", err)
	}

	// Should fallback to compact — same session, same driver.
	if newSess.ID != "fallback-test" {
		t.Fatal("session should not change on fallback")
	}
	if newDrv != oldDriver {
		t.Fatal("driver should not change on fallback")
	}

	// Session should be compacted (fewer entries).
	if newSess.History.Len() >= beforeEntries {
		t.Fatalf("history should be compacted: %d → %d", beforeEntries, newSess.History.Len())
	}

	// Monitor should be back to idle (ready to try again next time).
	if monitor.State() != MonitorIdle {
		t.Fatalf("state = %d, want idle after fallback", monitor.State())
	}
}

// TestRelayIntegration_SpawnDoesNotFireTwice verifies that crossing the 80%
// threshold multiple times doesn't spawn multiple background sessions.
func TestRelayIntegration_SpawnDoesNotFireTwice(t *testing.T) {
	ctx := context.Background()

	factoryCount := 0
	monitor := NewContextMonitor(WithMaxTokens(1000), WithSpawnAt(0.80), WithSwapAt(0.95))
	sess := New("no-double", "model", "/work")
	sess.Append(Entry{Role: "user", Content: "hello"})
	sess.Append(Entry{Role: "assistant", Content: "hi"})

	relay := NewRelayManager(RelayConfig{
		Monitor: monitor,
		Session: sess,
		Driver:  &mockChatDriver{},
		DriverFactory: func() (driver.ChatDriver, error) {
			factoryCount++
			return &mockChatDriver{}, nil
		},
		Log: slog.Default(),
	})

	// First check at 85% — spawns.
	monitor.Record(850, 0)
	relay.CheckAndRelay(ctx)

	// Second check at 90% — should NOT spawn again.
	monitor.Record(50, 0)
	relay.CheckAndRelay(ctx)

	// Third check at 92% — still no.
	monitor.Record(20, 0)
	relay.CheckAndRelay(ctx)

	if factoryCount != 1 {
		t.Fatalf("factory called %d times, want 1", factoryCount)
	}
}
