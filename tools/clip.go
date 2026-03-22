package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	clipDirName = "clips"
	clipPerm    = 0600
	clipDirPerm = 0700
)

// Sentinel errors for clipboard.
var (
	ErrClipNotFound = errors.New("clip not found")
	ErrEmptyKey     = errors.New("clip key cannot be empty")
)

// Clipboard provides file-backed key-value storage for cross-session
// data sharing between agents.
type Clipboard struct {
	mu  sync.RWMutex
	dir string
}

// NewClipboard creates a clipboard at the given base directory.
// The clips subdirectory is created if it doesn't exist.
func NewClipboard(baseDir string) (*Clipboard, error) {
	dir := filepath.Join(baseDir, clipDirName)
	if err := os.MkdirAll(dir, clipDirPerm); err != nil {
		return nil, fmt.Errorf("create clip dir: %w", err)
	}
	return &Clipboard{dir: dir}, nil
}

// Set stores a value under the given key.
func (c *Clipboard) Set(key, value string) error {
	if key == "" {
		return ErrEmptyKey
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return os.WriteFile(c.keyPath(key), []byte(value), clipPerm)
}

// Get retrieves a value by key.
func (c *Clipboard) Get(key string) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, err := os.ReadFile(c.keyPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrClipNotFound
		}
		return "", err
	}
	return string(data), nil
}

// Delete removes a clip by key.
func (c *Clipboard) Delete(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	err := os.Remove(c.keyPath(key))
	if err != nil && os.IsNotExist(err) {
		return ErrClipNotFound
	}
	return err
}

// List returns all clip keys sorted alphabetically.
func (c *Clipboard) List() ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, e := range entries {
		if !e.IsDir() {
			keys = append(keys, e.Name())
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (c *Clipboard) keyPath(key string) string {
	// Sanitize key: replace path separators
	safe := strings.ReplaceAll(key, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	return filepath.Join(c.dir, safe)
}
