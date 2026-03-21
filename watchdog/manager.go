package watchdog

import (
	"context"
	"sync"

	"github.com/dpopsuev/djinn/signal"
)

// Manager manages the lifecycle of multiple watchdogs.
type Manager struct {
	mu        sync.Mutex
	bus       *signal.SignalBus
	watchdogs []Watchdog
	cancel    context.CancelFunc
}

// NewManager creates a watchdog manager attached to a signal bus.
func NewManager(bus *signal.SignalBus) *Manager {
	return &Manager{bus: bus}
}

// Register adds a watchdog to the manager.
func (m *Manager) Register(w Watchdog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchdogs = append(m.watchdogs, w)
}

// StartAll starts all registered watchdogs.
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	watchCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	for _, w := range m.watchdogs {
		if err := w.Start(watchCtx); err != nil {
			cancel()
			return err
		}
	}
	return nil
}

// StopAll stops all registered watchdogs.
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	var firstErr error
	for _, w := range m.watchdogs {
		if err := w.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Watchdogs returns a copy of all registered watchdogs.
func (m *Manager) Watchdogs() []Watchdog {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Watchdog, len(m.watchdogs))
	copy(out, m.watchdogs)
	return out
}
