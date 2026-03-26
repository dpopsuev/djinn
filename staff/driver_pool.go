// driver_pool.go — ChatDriver pool with per-model caching and hot-swap.
//
// DriverPool manages ChatDriver instances keyed by "driverName/model",
// reusing existing drivers and tracking the most recently accessed one.
package staff

import (
	"context"
	"sync"

	"github.com/dpopsuev/djinn/driver"
)

// DriverPool manages ChatDriver instances per model, reusing existing drivers.
type DriverPool struct {
	mu      sync.Mutex
	factory func(driverName, model, prompt string) (driver.ChatDriver, error)
	drivers map[string]driver.ChatDriver // keyed by "driverName/model"
	active  string                       // current active key
}

// NewDriverPool creates a pool with the given factory function.
func NewDriverPool(factory func(driverName, model, prompt string) (driver.ChatDriver, error)) *DriverPool {
	return &DriverPool{
		factory: factory,
		drivers: make(map[string]driver.ChatDriver),
	}
}

// GetOrCreate returns a cached driver if one exists for this driverName/model
// combination, or creates and starts a new one otherwise.
func (p *DriverPool) GetOrCreate(ctx context.Context, driverName, model, prompt string) (driver.ChatDriver, error) {
	key := driverName + "/" + model

	p.mu.Lock()
	if d, ok := p.drivers[key]; ok {
		p.active = key
		p.mu.Unlock()
		return d, nil
	}
	p.mu.Unlock()

	// Create outside the lock to avoid holding it during potentially slow factory calls.
	d, err := p.factory(driverName, model, prompt)
	if err != nil {
		return nil, err
	}

	if err := d.Start(ctx, ""); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check: another goroutine may have created it while we were unlocked.
	if existing, ok := p.drivers[key]; ok {
		// Stop the one we just created; use the existing one.
		_ = d.Stop(ctx)
		p.active = key
		return existing, nil
	}

	p.drivers[key] = d
	p.active = key
	return d, nil
}

// Active returns the most recently accessed driver, or nil if none.
func (p *DriverPool) Active() driver.ChatDriver {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.drivers[p.active]
}

// StopAll stops all cached drivers. Called on shutdown.
func (p *DriverPool) StopAll(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, d := range p.drivers {
		_ = d.Stop(ctx)
		delete(p.drivers, key)
	}
	p.active = ""
}

// Count returns the number of cached drivers.
func (p *DriverPool) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.drivers)
}
