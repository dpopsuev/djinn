package capability

import (
	"fmt"
	"sync"
)

// Registry holds capability port adapters keyed by port name.
type Registry struct {
	mu    sync.RWMutex
	ports map[string]any
}

// NewRegistry creates an empty capability registry.
func NewRegistry() *Registry {
	return &Registry{ports: make(map[string]any)}
}

// Register adds a port adapter to the registry.
func (r *Registry) Register(name string, port any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ports[name] = port
}

// Get retrieves a port adapter by name.
func (r *Registry) Get(name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.ports[name]
	return p, ok
}

// MustGet retrieves a port adapter or panics if not registered.
func (r *Registry) MustGet(name string) any {
	p, ok := r.Get(name)
	if !ok {
		panic(fmt.Sprintf("%v: %s", ErrPortNotRegistered, name))
	}
	return p
}

// Has reports whether a port is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.ports[name]
	return ok
}

// Names returns all registered port names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.ports))
	for name := range r.ports {
		out = append(out, name)
	}
	return out
}
