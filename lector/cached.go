// cached.go — CachedLector wraps a Lector with L2 write-through.
// On every file observation (read/write/delete), the index data
// is written to the L2 cache scope-tagged by agent ID.
// On cache miss, falls back to the inner Lector and promotes to L2.
package lector

import (
	"encoding/json"

	djinncache "github.com/dpopsuev/djinn/cache"
)

var _ Lector = (*CachedLector)(nil)

// CachedLector wraps an inner Lector with L2 cache write-through.
type CachedLector struct {
	inner Lector
	l2    djinncache.Cache
	scope string // agent ID for scope tagging
}

// NewCachedLector creates a Lector that writes through to L2.
func NewCachedLector(inner Lector, l2 djinncache.Cache, agentScope string) *CachedLector {
	return &CachedLector{inner: inner, l2: l2, scope: agentScope}
}

// --- Index (read) ---

func (c *CachedLector) FileInfo(path string) (FileEntry, bool) {
	// L2 cache hit?
	if data, ok := c.l2.Get(c.scope, "file:"+path); ok {
		var fe FileEntry
		if err := json.Unmarshal(data, &fe); err == nil {
			return fe, true
		}
	}
	// Inner fallback — promote to L2 on hit.
	fe, ok := c.inner.FileInfo(path)
	if ok {
		c.cacheFile(path, fe)
	}
	return fe, ok
}

func (c *CachedLector) Symbols(scope string) []Symbol {
	// L2 cache hit?
	if data, ok := c.l2.Get(c.scope, "syms:"+scope); ok {
		var syms []Symbol
		if err := json.Unmarshal(data, &syms); err == nil {
			return syms
		}
	}
	// Inner fallback — promote to L2.
	syms := c.inner.Symbols(scope)
	if len(syms) > 0 {
		if data, err := json.Marshal(syms); err == nil {
			c.l2.Put(c.scope, "syms:"+scope, data)
		}
	}
	return syms
}

func (c *CachedLector) Imports(pkg string) []string {
	return c.inner.Imports(pkg)
}

func (c *CachedLector) Dependents(pkg string) []string {
	return c.inner.Dependents(pkg)
}

func (c *CachedLector) SymbolsForFile(file string) []Symbol {
	return c.inner.SymbolsForFile(file)
}

func (c *CachedLector) FuzzyFiles(query string) []FileEntry {
	return c.inner.FuzzyFiles(query)
}

func (c *CachedLector) FuzzySymbols(query string) []Symbol {
	return c.inner.FuzzySymbols(query)
}

// --- Observer (write) — write-through to L2 ---

func (c *CachedLector) OnFileRead(path string) {
	c.inner.OnFileRead(path)
	// After inner indexes, cache the result in L2.
	if fe, ok := c.inner.FileInfo(path); ok {
		c.cacheFile(path, fe)
	}
}

func (c *CachedLector) OnFileWrite(path string) {
	c.inner.OnFileWrite(path)
	// Invalidate + re-cache.
	c.l2.Evict(c.scope, "file:"+path)
	c.l2.Evict(c.scope, "syms:"+path)
	if fe, ok := c.inner.FileInfo(path); ok {
		c.cacheFile(path, fe)
	}
}

func (c *CachedLector) OnFileDelete(path string) {
	c.inner.OnFileDelete(path)
	c.l2.Evict(c.scope, "file:"+path)
	c.l2.Evict(c.scope, "syms:"+path)
}

func (c *CachedLector) cacheFile(path string, fe FileEntry) {
	if data, err := json.Marshal(fe); err == nil {
		c.l2.Put(c.scope, "file:"+path, data)
	}
}
