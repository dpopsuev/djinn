// registry.go — MediatorHub interface and HubRegistry (GOL-58).
//
// Each DevOps phase registers a hub. The registry provides lookup by name or phase.
package hub

import "sort"

// MediatorHub is the common interface for all phase-specific hubs.
type MediatorHub interface {
	Name() string
	Phase() string // DevOps phase: "plan", "code", "test", "build", "monitor", "execute"
}

// HubRegistry stores and retrieves phase-specific hubs.
type HubRegistry struct {
	hubs map[string]MediatorHub
}

// NewRegistry creates an empty hub registry.
func NewRegistry() *HubRegistry {
	return &HubRegistry{hubs: make(map[string]MediatorHub)}
}

// Register adds a hub to the registry, keyed by Name().
func (r *HubRegistry) Register(h MediatorHub) {
	r.hubs[h.Name()] = h
}

// Get returns a hub by name.
func (r *HubRegistry) Get(name string) (MediatorHub, bool) {
	h, ok := r.hubs[name]
	return h, ok
}

// All returns all registered hubs sorted by name.
func (r *HubRegistry) All() []MediatorHub {
	out := make([]MediatorHub, 0, len(r.hubs))
	for _, h := range r.hubs {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// ByPhase returns all hubs matching the given DevOps phase.
func (r *HubRegistry) ByPhase(phase string) []MediatorHub {
	var out []MediatorHub
	for _, h := range r.hubs {
		if h.Phase() == phase {
			out = append(out, h)
		}
	}
	return out
}

// Names returns all registered hub names sorted.
func (r *HubRegistry) Names() []string {
	names := make([]string, 0, len(r.hubs))
	for name := range r.hubs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
