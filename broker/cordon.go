package broker

import "sync"

// Cordon represents a fenced-off scope that should not receive new work.
type Cordon struct {
	Scope  string
	Reason string
}

// CordonRegistry tracks active cordons.
type CordonRegistry struct {
	mu      sync.RWMutex
	cordons map[string]Cordon
}

// NewCordonRegistry creates a new cordon registry.
func NewCordonRegistry() *CordonRegistry {
	return &CordonRegistry{
		cordons: make(map[string]Cordon),
	}
}

// Set adds or replaces a cordon for the given scope.
func (r *CordonRegistry) Set(scope, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cordons[scope] = Cordon{Scope: scope, Reason: reason}
}

// Clear removes a cordon for the given scope.
func (r *CordonRegistry) Clear(scope string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cordons, scope)
}

// Overlaps checks if a given scope is cordoned.
func (r *CordonRegistry) Overlaps(scope string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.cordons[scope]
	return ok
}

// Active returns all active cordons.
func (r *CordonRegistry) Active() []Cordon {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Cordon, 0, len(r.cordons))
	for _, c := range r.cordons {
		out = append(out, c)
	}
	return out
}
