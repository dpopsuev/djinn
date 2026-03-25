package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dpopsuev/djinn/driver"
)

// RelayManager orchestrates context relay: monitors usage, spawns a background
// session when approaching the limit, and seamlessly swaps when the limit hits.
type RelayManager struct {
	mu           sync.Mutex
	monitor      *ContextMonitor
	store        *Store
	driverFactory func() (driver.ChatDriver, error)
	log          *slog.Logger

	// Active session state (owned by caller, swapped atomically).
	activeSession *Session
	activeDriver  driver.ChatDriver

	// Background session state (prepared during spawn).
	backupSession *Session
	backupDriver  driver.ChatDriver

	// Queue: prompts received during swap.
	queue []string
}

// RelayConfig configures a RelayManager.
type RelayConfig struct {
	Monitor       *ContextMonitor
	Store         *Store
	Session       *Session
	Driver        driver.ChatDriver
	DriverFactory func() (driver.ChatDriver, error)
	Log           *slog.Logger
}

// NewRelayManager creates a relay manager wired to monitor callbacks.
func NewRelayManager(cfg RelayConfig) *RelayManager {
	r := &RelayManager{
		monitor:       cfg.Monitor,
		store:         cfg.Store,
		activeSession: cfg.Session,
		activeDriver:  cfg.Driver,
		driverFactory: cfg.DriverFactory,
		log:           cfg.Log,
	}

	// Wire monitor callbacks.
	cfg.Monitor.SetState(MonitorIdle)

	return r
}

// CheckAndRelay is called after each agent turn. It checks the monitor
// and initiates spawn or swap as needed. Returns the active session and
// driver (which may have been swapped).
func (r *RelayManager) CheckAndRelay(ctx context.Context) (*Session, driver.ChatDriver, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.monitor.State()

	switch state {
	case MonitorIdle:
		if r.monitor.ShouldSpawn() {
			r.log.Info("relay: spawning background session",
				"usage", fmt.Sprintf("%.0f%%", r.monitor.Usage()*100))
			r.monitor.SetState(MonitorSpawning)
			if err := r.spawnBackground(ctx); err != nil {
				r.log.Warn("relay: background spawn failed, falling back to compact", "error", err)
				r.monitor.SetState(MonitorIdle)
				before, after := Compact(r.activeSession, DefaultKeepRecent)
				r.log.Info("relay: fallback compact", "before", before, "after", after)
			}
		}

	case MonitorSpawning:
		// Record() pre-emptively transitioned idle→spawning when it detected
		// the 80% threshold. CheckAndRelay is the actor that performs the spawn.
		r.log.Info("relay: spawning background session (deferred)",
			"usage", fmt.Sprintf("%.0f%%", r.monitor.Usage()*100))
		if err := r.spawnBackground(ctx); err != nil {
			r.log.Warn("relay: background spawn failed, falling back to compact", "error", err)
			r.monitor.SetState(MonitorIdle)
			before, after := Compact(r.activeSession, DefaultKeepRecent)
			r.log.Info("relay: fallback compact", "before", before, "after", after)
		}

	case MonitorReady:
		if r.monitor.ShouldSwap() {
			r.log.Info("relay: executing swap",
				"usage", fmt.Sprintf("%.0f%%", r.monitor.Usage()*100))
			if err := r.executeSwap(ctx); err != nil {
				r.log.Warn("relay: swap failed", "error", err)
			}
		}

	case MonitorSwapping:
		// Record() pre-emptively transitioned ready→swapping. Execute now.
		r.log.Info("relay: executing swap (deferred)",
			"usage", fmt.Sprintf("%.0f%%", r.monitor.Usage()*100))
		if err := r.executeSwap(ctx); err != nil {
			r.log.Warn("relay: swap failed", "error", err)
		}
	}

	return r.activeSession, r.activeDriver, nil
}

// QueuePrompt adds a prompt to the relay queue (used during swap).
func (r *RelayManager) QueuePrompt(prompt string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queue = append(r.queue, prompt)
}

// DrainQueue returns and clears all queued prompts.
func (r *RelayManager) DrainQueue() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	q := r.queue
	r.queue = nil
	return q
}

// spawnBackground creates a new session seeded with compacted context.
// Must be called with r.mu held.
func (r *RelayManager) spawnBackground(ctx context.Context) error {
	if r.driverFactory == nil {
		return fmt.Errorf("no driver factory configured")
	}

	// Extract summary from old entries (all except recent).
	entries := r.activeSession.Entries()
	keepRecent := DefaultKeepRecent
	if len(entries) > keepRecent {
		oldEntries := entries[:len(entries)-keepRecent]
		summaryText := ExtractSummaryText(oldEntries, 50_000)

		newID := fmt.Sprintf("%s-relay-%d", r.activeSession.ID, r.monitor.TotalTokens())
		r.backupSession = SeedSession(newID, r.activeSession, summaryText, keepRecent)
	} else {
		// Not enough history to compact — just clone.
		newID := fmt.Sprintf("%s-relay-%d", r.activeSession.ID, r.monitor.TotalTokens())
		r.backupSession = SeedSession(newID, r.activeSession, "", keepRecent)
	}

	// Create new driver.
	newDriver, err := r.driverFactory()
	if err != nil {
		return fmt.Errorf("create backup driver: %w", err)
	}

	// Start the driver.
	if err := newDriver.Start(ctx, ""); err != nil {
		return fmt.Errorf("start backup driver: %w", err)
	}

	// Replay seed entries to warm up the new driver's context.
	for _, e := range r.backupSession.Entries() {
		if e.Role == driver.RoleUser {
			if err := newDriver.Send(ctx, driver.Message{Role: e.Role, Content: e.Content}); err != nil {
				newDriver.Stop(ctx) //nolint:errcheck
				return fmt.Errorf("seed replay: %w", err)
			}
		}
	}

	r.backupDriver = newDriver
	r.monitor.SetState(MonitorReady)
	r.log.Info("relay: background session ready",
		"id", r.backupSession.ID,
		"entries", r.backupSession.History.Len())

	return nil
}

// executeSwap atomically swaps active ↔ backup, archives old session.
// Must be called with r.mu held.
func (r *RelayManager) executeSwap(ctx context.Context) error {
	if r.backupDriver == nil || r.backupSession == nil {
		return fmt.Errorf("no backup available")
	}

	// Archive old session.
	oldSession := r.activeSession
	oldDriver := r.activeDriver

	// Swap.
	r.activeSession = r.backupSession
	r.activeDriver = r.backupDriver
	r.backupSession = nil
	r.backupDriver = nil

	// Archive the old session if store is available.
	if r.store != nil {
		if err := r.store.Archive(oldSession); err != nil {
			r.log.Warn("relay: archive old session failed", "error", err)
		}
	}

	// Stop old driver.
	if err := oldDriver.Stop(ctx); err != nil {
		r.log.Warn("relay: stop old driver failed", "error", err)
	}

	// Reset monitor for new context window.
	r.monitor.Reset()

	r.log.Info("relay: swap complete",
		"new_id", r.activeSession.ID,
		"queued", len(r.queue))

	return nil
}
