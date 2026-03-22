package config

import "fmt"

// Registry aggregates Configurable components for bulk snapshot/restore.
type Registry struct {
	components map[string]Configurable
	order      []string // insertion order for deterministic output
}

// NewRegistry creates an empty config registry.
func NewRegistry() *Registry {
	return &Registry{components: make(map[string]Configurable)}
}

// Register adds a configurable component. Replaces if key already exists.
func (r *Registry) Register(c Configurable) {
	key := c.ConfigKey()
	if _, exists := r.components[key]; !exists {
		r.order = append(r.order, key)
	}
	r.components[key] = c
}

// Get returns a registered component by key.
func (r *Registry) Get(key string) (Configurable, bool) {
	c, ok := r.components[key]
	return c, ok
}

// Keys returns registered keys in insertion order.
func (r *Registry) Keys() []string {
	return r.order
}

// Dump snapshots all registered components into a map.
func (r *Registry) Dump() map[string]any {
	out := make(map[string]any, len(r.components))
	for _, key := range r.order {
		out[key] = r.components[key].Snapshot()
	}
	return out
}

// Load applies config values to registered components.
// Unknown keys are silently ignored.
func (r *Registry) Load(cfg map[string]any) error {
	for key, v := range cfg {
		c, ok := r.components[key]
		if !ok {
			continue
		}
		if err := c.Apply(v); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfigApply, key, err)
		}
	}
	return nil
}
