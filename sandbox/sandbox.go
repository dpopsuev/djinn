// Package sandbox defines the Strategy interface for execution isolation.
// Backends (Misbah, bubblewrap, podman, etc.) implement this interface.
// The workspace manifest declares the desired backend + level.
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Handle identifies a running sandbox instance.
type Handle string

// ExecResult holds the output of a command executed inside a sandbox.
type ExecResult struct {
	ExitCode int32
	Stdout   string
	Stderr   string
}

// Sandbox is the Strategy interface for execution isolation.
type Sandbox interface {
	Create(ctx context.Context, level string, repos []string) (Handle, error)
	Destroy(ctx context.Context, handle Handle) error
	Exec(ctx context.Context, handle Handle, cmd []string, timeout int64) (ExecResult, error)
	Name() string
}

// Levels.
const (
	LevelNone      = "none"
	LevelNamespace = "namespace"
	LevelContainer = "container"
	LevelKata      = "kata"
)

// Sentinel errors.
var (
	ErrBackendNotFound = errors.New("sandbox backend not found")
	ErrBackendFailed   = errors.New("sandbox backend failed to start")
)

// Registry of sandbox backends.
var (
	mu       sync.RWMutex
	backends = make(map[string]func() (Sandbox, error))
)

// Register adds a sandbox backend factory.
func Register(name string, factory func() (Sandbox, error)) {
	mu.Lock()
	defer mu.Unlock()
	backends[name] = factory
}

// Get returns a sandbox backend by name.
func Get(name string) (Sandbox, error) {
	mu.RLock()
	factory, ok := backends[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q (available: %v)", ErrBackendNotFound, name, Available())
	}
	return factory()
}

// Available returns registered backend names.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(backends))
	for name := range backends {
		names = append(names, name)
	}
	return names
}
