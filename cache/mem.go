package cache

import "sync"

var _ Cache = (*MemCache)(nil)

// MemCache is an in-memory Cache. Thread-safe. Used for both L1 and L2.
type MemCache struct {
	mu   sync.RWMutex
	data map[string]map[string][]byte // scope → key → data
}

// NewMemCache creates an empty in-memory cache.
func NewMemCache() *MemCache {
	return &MemCache{data: make(map[string]map[string][]byte)}
}

func (c *MemCache) Put(scope, key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data[scope] == nil {
		c.data[scope] = make(map[string][]byte)
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	c.data[scope][key] = cp
}

func (c *MemCache) Get(scope, key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.data[scope] == nil {
		return nil, false
	}
	d, ok := c.data[scope][key]
	if !ok {
		return nil, false
	}
	cp := make([]byte, len(d))
	copy(cp, d)
	return cp, true
}

func (c *MemCache) Keys(scope string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := c.data[scope]
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (c *MemCache) Evict(scope, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if m := c.data[scope]; m != nil {
		delete(m, key)
		if len(m) == 0 {
			delete(c.data, scope)
		}
	}
}

func (c *MemCache) EvictScope(scope string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, scope)
}

func (c *MemCache) Scopes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	scopes := make([]string, 0, len(c.data))
	for s := range c.data {
		scopes = append(scopes, s)
	}
	return scopes
}

func (c *MemCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := 0
	for _, m := range c.data {
		total += len(m)
	}
	return total
}
