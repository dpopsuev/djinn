package cache

var _ Cache = (*WriteThrough)(nil)

// WriteThrough wraps an L1 (per-agent) and L2 (shared) cache.
// On Put: writes to both L1 and L2.
// On Get: reads L1 first, falls back to L2 (promotes to L1 on hit).
// On Evict: evicts from L1 only (L2 retains for recovery).
// GPU write-through model.
type WriteThrough struct {
	l1 Cache
	l2 Cache
}

// NewWriteThrough creates a write-through cache wrapping L1 and L2.
func NewWriteThrough(l1, l2 Cache) *WriteThrough {
	return &WriteThrough{l1: l1, l2: l2}
}

func (w *WriteThrough) Put(scope, key string, data []byte) {
	w.l1.Put(scope, key, data)
	w.l2.Put(scope, key, data)
}

func (w *WriteThrough) Get(scope, key string) ([]byte, bool) {
	// L1 first.
	if data, ok := w.l1.Get(scope, key); ok {
		return data, true
	}
	// L2 fallback — promote to L1 on hit.
	if data, ok := w.l2.Get(scope, key); ok {
		w.l1.Put(scope, key, data)
		return data, true
	}
	return nil, false
}

func (w *WriteThrough) Keys(scope string) []string {
	// L1 keys + L2 keys (deduplicated).
	seen := make(map[string]bool)
	var keys []string
	for _, k := range w.l1.Keys(scope) {
		if !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	for _, k := range w.l2.Keys(scope) {
		if !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	return keys
}

func (w *WriteThrough) Evict(scope, key string) {
	// Evict from L1 only. L2 retains for recovery.
	w.l1.Evict(scope, key)
}

func (w *WriteThrough) EvictScope(scope string) {
	w.l1.EvictScope(scope)
	// L2 retains for recovery.
}

func (w *WriteThrough) Scopes() []string {
	seen := make(map[string]bool)
	var scopes []string
	for _, s := range w.l1.Scopes() {
		if !seen[s] {
			scopes = append(scopes, s)
			seen[s] = true
		}
	}
	for _, s := range w.l2.Scopes() {
		if !seen[s] {
			scopes = append(scopes, s)
			seen[s] = true
		}
	}
	return scopes
}

func (w *WriteThrough) Len() int {
	// L2 is the source of truth for total count (inclusive).
	return w.l2.Len()
}

// L1 returns the underlying L1 cache (for testing/inspection).
func (w *WriteThrough) L1() Cache { return w.l1 }

// L2 returns the underlying L2 cache (for testing/inspection).
func (w *WriteThrough) L2() Cache { return w.l2 }
