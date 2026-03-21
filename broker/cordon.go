package broker

import (
	"strings"
	"sync"
	"time"
)

// Cordon represents a fenced-off scope that should not receive new work.
type Cordon struct {
	Scope     []string  // cordoned file/package paths
	Reason    string
	Source    string    // which signal/agent triggered it
	Timestamp time.Time
	Cleared   bool
}

// CordonRegistry tracks active cordons.
type CordonRegistry struct {
	mu      sync.RWMutex
	cordons []Cordon
}

// NewCordonRegistry creates a new cordon registry.
func NewCordonRegistry() *CordonRegistry {
	return &CordonRegistry{}
}

// Set adds a cordon for the given scope.
func (r *CordonRegistry) Set(scope []string, reason, source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cordons = append(r.cordons, Cordon{
		Scope:     scope,
		Reason:    reason,
		Source:    source,
		Timestamp: time.Now(),
	})
}

// Clear marks cordons overlapping the given scope as cleared.
func (r *CordonRegistry) Clear(scope []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.cordons {
		if !r.cordons[i].Cleared && pathsOverlap(r.cordons[i].Scope, scope) {
			r.cordons[i].Cleared = true
		}
	}
}

// Overlaps returns all active (non-cleared) cordons that overlap with the given paths.
// Overlap is detected via path-prefix matching: "auth/" overlaps "auth/middleware.go".
func (r *CordonRegistry) Overlaps(paths []string) []Cordon {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Cordon
	for _, c := range r.cordons {
		if !c.Cleared && pathsOverlap(c.Scope, paths) {
			out = append(out, c)
		}
	}
	return out
}

// Active returns all non-cleared cordons.
func (r *CordonRegistry) Active() []Cordon {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Cordon
	for _, c := range r.cordons {
		if !c.Cleared {
			out = append(out, c)
		}
	}
	return out
}

// pathsOverlap checks if any path in a is a prefix of (or equal to) any path in b, or vice versa.
func pathsOverlap(a, b []string) bool {
	for _, pa := range a {
		for _, pb := range b {
			if pathPrefix(pa, pb) || pathPrefix(pb, pa) {
				return true
			}
		}
	}
	return false
}

// pathPrefix checks if a is a prefix of b (directory-aware).
func pathPrefix(a, b string) bool {
	if a == b {
		return true
	}
	// Ensure directory boundary: "auth" should match "auth/middleware.go"
	// but "au" should NOT match "auth/middleware.go"
	prefix := a
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return strings.HasPrefix(b, prefix)
}
