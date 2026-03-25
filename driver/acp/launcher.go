// launcher.go — ACPLauncher implements bugle/pool.Launcher for Djinn.
// Spawns ACP agent processes. One process per entity.
package acp

import (
	"context"
	"fmt"
	"sync"

	"github.com/dpopsuev/djinn/bugleport"
)

// ACPLauncher implements bugleport.Launcher by spawning ACP agent processes.
type ACPLauncher struct {
	mu      sync.RWMutex
	drivers map[bugleport.EntityID]*ACPDriver
}

// NewACPLauncher creates a launcher for ACP-based agents.
func NewACPLauncher() *ACPLauncher {
	return &ACPLauncher{
		drivers: make(map[bugleport.EntityID]*ACPDriver),
	}
}

// Start spawns an ACP agent process for the given entity.
func (l *ACPLauncher) Start(ctx context.Context, id bugleport.EntityID, config bugleport.LaunchConfig) error {
	// Determine which ACP agent to use from the model field.
	agentName := "cursor" // default
	if config.Model != "" {
		agentName = config.Model
	}

	driver, err := New(agentName,
		WithModel(config.Model),
		WithLogger(nil),
	)
	if err != nil {
		return fmt.Errorf("create ACP driver for entity %d: %w", id, err)
	}

	if err := driver.Start(ctx, ""); err != nil {
		return fmt.Errorf("start ACP agent for entity %d: %w", id, err)
	}

	l.mu.Lock()
	l.drivers[id] = driver
	l.mu.Unlock()

	return nil
}

// Stop kills the ACP agent process for the given entity.
func (l *ACPLauncher) Stop(ctx context.Context, id bugleport.EntityID) error {
	l.mu.Lock()
	driver, ok := l.drivers[id]
	if ok {
		delete(l.drivers, id)
	}
	l.mu.Unlock()

	if !ok {
		return nil
	}
	return driver.Stop(ctx)
}

// Healthy returns true if the ACP agent process is still running.
func (l *ACPLauncher) Healthy(_ context.Context, id bugleport.EntityID) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.drivers[id]
	return ok
}

// Driver returns the ACPDriver for an entity (for sending messages).
func (l *ACPLauncher) Driver(id bugleport.EntityID) (*ACPDriver, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	d, ok := l.drivers[id]
	return d, ok
}

// Compile-time check.
var _ bugleport.Launcher = (*ACPLauncher)(nil)
