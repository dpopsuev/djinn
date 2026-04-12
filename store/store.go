// Package store defines the KeyStore interface — the simplest storage primitive.
// Put/Get/List/Delete by key. Backed by in-memory map (MemStore) or
// file-per-key (FileStore). When Reliquary ships, swap for ReliquaryStore.
package store

import "errors"

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = errors.New("store: key not found")

// KeyStore is a key-value store. Keys are strings, values are byte slices.
// Thread-safe. Implementations must be safe for concurrent use.
type KeyStore interface {
	// Put writes a value for the given key. Overwrites if exists.
	Put(key string, data []byte) error

	// Get reads the value for a key. Returns ErrNotFound if missing.
	Get(key string) ([]byte, error)

	// List returns all keys. Order is not guaranteed.
	List() ([]string, error)

	// Delete removes a key. Returns ErrNotFound if missing.
	Delete(key string) error

	// Close releases resources.
	Close() error
}
