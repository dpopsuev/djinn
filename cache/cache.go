// Package cache provides the L1/L2 cache hierarchy for Djinn agents.
// L1 (per-Vessel) and L2 (Substrate-shared) hold identical content types
// at different scopes. Write-through: every L1 write propagates to L2.
// Agent dies → L1 gone but L2 has scope-tagged entries → new agent pre-warmed.
// GPU-inspired: NVIDIA L1 write-through to L2, Intel inclusive L3.
package cache

import "time"

// Entry is a cached item with scope tag for recovery.
type Entry struct {
	Key       string    `json:"key"`
	Data      []byte    `json:"data"`
	Scope     string    `json:"scope"` // agent ID that owns this entry
	Timestamp time.Time `json:"timestamp"`
}

// Cache is the interface for both L1 (per-agent) and L2 (shared).
// Same interface, different scope. Thread-safe.
type Cache interface {
	// Put writes an entry tagged with a scope. Overwrites if exists.
	Put(scope, key string, data []byte)

	// Get reads an entry by scope + key. Returns false if missing.
	Get(scope, key string) ([]byte, bool)

	// Keys returns all keys for a scope.
	Keys(scope string) []string

	// Evict removes a single entry.
	Evict(scope, key string)

	// EvictScope removes all entries for a scope.
	EvictScope(scope string)

	// Scopes returns all scopes that have entries.
	Scopes() []string

	// Len returns total entry count across all scopes.
	Len() int
}
