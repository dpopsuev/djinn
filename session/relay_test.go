package session

import (
	"context"
	"log/slog"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

// mockChatDriver implements driver.ChatDriver for testing.
type mockChatDriver struct {
	started  bool
	stopped  bool
	messages []driver.Message
}

func (d *mockChatDriver) Start(_ context.Context, _ driver.SandboxHandle) error {
	d.started = true
	return nil
}
func (d *mockChatDriver) Stop(_ context.Context) error {
	d.stopped = true
	return nil
}
func (d *mockChatDriver) Send(_ context.Context, msg driver.Message) error {
	d.messages = append(d.messages, msg)
	return nil
}
func (d *mockChatDriver) SendRich(_ context.Context, _ driver.RichMessage) error { return nil }
func (d *mockChatDriver) Chat(_ context.Context) (<-chan driver.StreamEvent, error) {
	return nil, nil
}
func (d *mockChatDriver) AppendAssistant(_ driver.RichMessage)   {}
func (d *mockChatDriver) SetSystemPrompt(_ string)               {}

func newTestRelay(maxTokens int) (*RelayManager, *mockChatDriver, *mockChatDriver) {
	monitor := NewContextMonitor(
		WithMaxTokens(maxTokens),
		WithSpawnAt(0.80),
		WithSwapAt(0.95),
	)

	activeDriver := &mockChatDriver{}
	backupDriver := &mockChatDriver{}

	sess := New("test-session", "claude-4", "/workspace")
	sess.Append(Entry{Role: "user", Content: "hello"})
	sess.Append(Entry{Role: "assistant", Content: "hi there"})
	sess.Append(Entry{Role: "user", Content: "do something"})
	sess.Append(Entry{Role: "assistant", Content: "done"})
	sess.Append(Entry{Role: "user", Content: "more work"})
	sess.Append(Entry{Role: "assistant", Content: "completed"})

	r := NewRelayManager(RelayConfig{
		Monitor: monitor,
		Session: sess,
		Driver:  activeDriver,
		DriverFactory: func() (driver.ChatDriver, error) {
			return backupDriver, nil
		},
		Log: slog.Default(),
	})

	return r, activeDriver, backupDriver
}

func TestRelayManager_no_action_below_threshold(t *testing.T) {
	r, _, _ := newTestRelay(10000)

	// Record tokens below spawn threshold (80% of 10000 = 8000).
	r.monitor.Record(1000, 1000) // 20%

	sess, drv, err := r.CheckAndRelay(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID != "test-session" {
		t.Error("session should not have changed")
	}
	if drv == nil {
		t.Error("driver should not be nil")
	}
	if r.monitor.State() != MonitorIdle {
		t.Errorf("state = %d, want MonitorIdle", r.monitor.State())
	}
}

func TestRelayManager_spawns_at_threshold(t *testing.T) {
	r, _, backupDriver := newTestRelay(10000)

	// Push past 80%.
	r.monitor.totalIn = 8500

	_, _, err := r.CheckAndRelay(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if r.monitor.State() != MonitorReady {
		t.Errorf("state = %d, want MonitorReady", r.monitor.State())
	}
	if !backupDriver.started {
		t.Error("backup driver should have been started")
	}
	if r.backupSession == nil {
		t.Error("backup session should exist")
	}
}

func TestRelayManager_swaps_at_threshold(t *testing.T) {
	r, activeDriver, backupDriver := newTestRelay(10000)

	// Spawn first.
	r.monitor.totalIn = 8500
	r.CheckAndRelay(context.Background())

	// Now push past 95%.
	r.monitor.totalIn = 9600

	sess, drv, err := r.CheckAndRelay(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Should have swapped.
	if sess.ID == "test-session" {
		t.Error("session should have changed after swap")
	}
	if drv != backupDriver {
		t.Error("driver should be the backup driver after swap")
	}
	if !activeDriver.stopped {
		t.Error("old driver should have been stopped")
	}
	if r.monitor.State() != MonitorIdle {
		t.Errorf("state = %d after swap, want MonitorIdle (reset)", r.monitor.State())
	}
	if r.monitor.TotalTokens() != 0 {
		t.Errorf("tokens = %d after swap, want 0 (reset)", r.monitor.TotalTokens())
	}
}

func TestRelayManager_queue(t *testing.T) {
	r, _, _ := newTestRelay(10000)

	r.QueuePrompt("first")
	r.QueuePrompt("second")

	q := r.DrainQueue()
	if len(q) != 2 {
		t.Fatalf("queue = %d, want 2", len(q))
	}
	if q[0] != "first" || q[1] != "second" {
		t.Errorf("queue = %v", q)
	}

	// Drain again — should be empty.
	q2 := r.DrainQueue()
	if len(q2) != 0 {
		t.Errorf("queue after drain = %d, want 0", len(q2))
	}
}

func TestRelayManager_fallback_compact_on_factory_nil(t *testing.T) {
	monitor := NewContextMonitor(
		WithMaxTokens(1000),
		WithSpawnAt(0.80),
	)
	activeDriver := &mockChatDriver{}
	sess := New("test", "model", "/work")
	for i := range 10 {
		sess.Append(Entry{Role: "user", Content: "msg " + string(rune('A'+i))})
	}

	r := NewRelayManager(RelayConfig{
		Monitor:       monitor,
		Session:       sess,
		Driver:        activeDriver,
		DriverFactory: nil, // no factory
		Log:           slog.Default(),
	})

	// Push past threshold.
	monitor.totalIn = 900

	_, _, err := r.CheckAndRelay(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Should have fallen back to compact, state back to idle.
	if monitor.State() != MonitorIdle {
		t.Errorf("state = %d, want MonitorIdle after fallback", monitor.State())
	}
}

func TestRelayManager_backup_seed_replays_entries(t *testing.T) {
	r, _, backupDriver := newTestRelay(10000)

	// Trigger spawn.
	r.monitor.totalIn = 8500
	r.CheckAndRelay(context.Background())

	// Backup driver should have received seed entries via Send.
	if len(backupDriver.messages) == 0 {
		t.Error("backup driver should have received seed messages")
	}
	// All replayed messages should be user role.
	for _, msg := range backupDriver.messages {
		if msg.Role != driver.RoleUser {
			t.Errorf("replayed msg role = %q, want user", msg.Role)
		}
	}
}
