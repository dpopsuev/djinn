package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/dpopsuev/djinn/telemetry"
)

// Sentinel errors for mount operations.
var (
	// ErrMountConflict is returned when a virtual path is already mounted.
	ErrMountConflict = errors.New("vfs: mount conflict — virtual path already mounted")

	// ErrNotMounted is returned when a virtual path has no mount entry.
	ErrNotMounted = errors.New("vfs: path not mounted")

	// ErrPathEscape is returned when a translated path escapes the mount root.
	ErrPathEscape = errors.New("vfs: path escapes mount root")
)

// MountEntry maps a virtual path (what the agent sees) to a host path (real filesystem).
type MountEntry struct {
	VirtualPath string    // what the agent sees (e.g., "/workspace/djinn")
	HostPath    string    // real path on host (e.g., "/home/user/Workspace/djinn")
	ReadOnly    bool      // true for global/reference mounts
	ScopeType   ScopeType // scope semantics (system, operations, global, etc.)
}

// MountTable manages the set of active mounts. Thread-safe.
type MountTable struct {
	entries []MountEntry
	mu      sync.RWMutex
	log     *slog.Logger
}

// NewMountTable creates an empty mount table with the given logger.
func NewMountTable(log *slog.Logger) *MountTable {
	if log == nil {
		log = slog.Default()
	}
	return &MountTable{
		log: log,
	}
}

// Mount adds a virtual-to-host path mapping. Returns ErrMountConflict if the
// virtual path is already mounted.
func (t *MountTable) Mount(virtual, host string, readOnly bool, scopeType ScopeType) error {
	virtual = cleanMountPath(virtual)
	host = cleanMountPath(host)

	t.mu.Lock()
	defer t.mu.Unlock()

	// Check for conflicts.
	for _, e := range t.entries {
		if e.VirtualPath == virtual {
			t.log.WarnContext(context.Background(), "mount conflict",
				slog.String(telemetry.KeyPath, virtual),
				"existing_host", e.HostPath,
				"requested_host", host,
			)
			return fmt.Errorf("%w: %s already maps to %s", ErrMountConflict, virtual, e.HostPath)
		}
	}

	t.entries = append(t.entries, MountEntry{
		VirtualPath: virtual,
		HostPath:    host,
		ReadOnly:    readOnly,
		ScopeType:   scopeType,
	})

	t.log.InfoContext(context.Background(), "mounted",
		slog.String(telemetry.KeyPath, virtual),
		"host", host,
		"readonly", readOnly,
		"scope_type", string(scopeType),
	)

	return nil
}

// Unmount removes the mount entry for the given virtual path.
// Returns ErrNotMounted if no entry exists.
func (t *MountTable) Unmount(virtual string) error {
	virtual = cleanMountPath(virtual)

	t.mu.Lock()
	defer t.mu.Unlock()

	for i, e := range t.entries {
		if e.VirtualPath == virtual {
			t.entries = append(t.entries[:i], t.entries[i+1:]...)
			t.log.InfoContext(context.Background(), "unmounted",
				slog.String(telemetry.KeyPath, virtual),
				"host", e.HostPath,
			)
			return nil
		}
	}

	t.log.WarnContext(context.Background(), "unmount not found", slog.String(telemetry.KeyPath, virtual))
	return fmt.Errorf("%w: %s", ErrNotMounted, virtual)
}

// Translate converts a virtual path to a host path using longest prefix match.
//
// Given virtual "/workspace/djinn/agent/loop.go" and mount entry
// "/workspace/djinn" -> "/home/user/Workspace/djinn", the result is
// "/home/user/Workspace/djinn/agent/loop.go".
//
// Returns ErrNotMounted if no mount matches. Returns ErrPathEscape if the
// translated path escapes the mount's host root (prevents ../../../etc/passwd).
func (t *MountTable) Translate(virtualPath string) (string, error) {
	// First, find a mount using the raw (uncleaned) path segments.
	// This lets us detect escape attempts like "/workspace/djinn/../../etc/passwd"
	// where cleaning would lose the mount prefix match.
	rawPath := virtualPath
	virtualPath = cleanMountPath(virtualPath)

	t.mu.RLock()
	defer t.mu.RUnlock()

	// Try matching the cleaned path first.
	entry, ok := t.longestMatch(virtualPath)
	if !ok {
		// Cleaned path doesn't match — check if the raw path contained a mount
		// prefix that got stripped by ".." resolution. This is an escape.
		if t.wasEscape(rawPath) {
			t.log.WarnContext(context.Background(), "path escape attempt",
				"virtual", rawPath,
				"cleaned", virtualPath,
			)
			return "", fmt.Errorf("%w: %s resolves outside mount root", ErrPathEscape, rawPath)
		}
		return "", fmt.Errorf("%w: %s", ErrNotMounted, virtualPath)
	}

	// Compute the relative path from the mount point.
	rel := strings.TrimPrefix(virtualPath, entry.VirtualPath)
	if rel == "" {
		rel = "/"
	}

	// Join host path with relative path.
	hostPath := filepath.Join(entry.HostPath, rel)

	// Clean and verify no escape.
	hostPath = filepath.Clean(hostPath)
	hostRoot := filepath.Clean(entry.HostPath)
	if !isMountUnder(hostPath, hostRoot) {
		t.log.WarnContext(context.Background(), "path escape attempt",
			"virtual", virtualPath,
			"translated", hostPath,
			"mount_root", hostRoot,
		)
		return "", fmt.Errorf("%w: %s resolves outside %s", ErrPathEscape, virtualPath, entry.VirtualPath)
	}

	t.log.DebugContext(context.Background(), "translated",
		"virtual", virtualPath,
		"host", hostPath,
	)

	return hostPath, nil
}

// wasEscape checks if a raw (uncleaned) path originally started with a mounted
// virtual prefix but ".." segments caused it to escape. Must be called under
// read lock.
func (t *MountTable) wasEscape(rawPath string) bool {
	// Check each mount — if the raw path starts with the mount's virtual prefix
	// (before ".." processing), that means it tried to escape.
	for _, e := range t.entries {
		vp := e.VirtualPath
		if strings.HasPrefix(rawPath, vp+"/") || rawPath == vp {
			return true
		}
	}
	return false
}

// Resolve performs reverse translation: given a host path, returns the
// corresponding virtual path. Uses longest prefix match on host paths.
// Returns ErrNotMounted if no mount entry covers the host path.
func (t *MountTable) Resolve(hostPath string) (string, error) {
	hostPath = cleanMountPath(hostPath)

	t.mu.RLock()
	defer t.mu.RUnlock()

	// Find the longest host path prefix match.
	var best *MountEntry
	bestLen := 0
	for i := range t.entries {
		hp := t.entries[i].HostPath
		if isMountUnder(hostPath, hp) && len(hp) > bestLen {
			e := t.entries[i]
			best = &e
			bestLen = len(hp)
		}
	}

	if best == nil {
		return "", fmt.Errorf("%w: host path %s", ErrNotMounted, hostPath)
	}

	// Compute the relative path from the host mount root.
	rel := strings.TrimPrefix(hostPath, best.HostPath)
	if rel == "" {
		rel = "/"
	}

	virtualPath := filepath.Join(best.VirtualPath, rel)
	return filepath.Clean(virtualPath), nil
}

// List returns a snapshot of all current mount entries, sorted by virtual path.
func (t *MountTable) List() []MountEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]MountEntry, len(t.entries))
	copy(result, t.entries)
	sort.Slice(result, func(i, j int) bool {
		return result[i].VirtualPath < result[j].VirtualPath
	})
	return result
}

// longestMatch finds the mount entry with the longest VirtualPath prefix
// that matches the given path. Must be called under read lock.
func (t *MountTable) longestMatch(virtualPath string) (MountEntry, bool) {
	var best MountEntry
	bestLen := 0
	for _, e := range t.entries {
		if isMountUnder(virtualPath, e.VirtualPath) && len(e.VirtualPath) > bestLen {
			best = e
			bestLen = len(e.VirtualPath)
		}
	}
	return best, bestLen > 0
}

// isMountUnder returns true if path is equal to or nested under root.
// Both paths must be clean.
func isMountUnder(path, root string) bool {
	if path == root {
		return true
	}
	// Ensure root ends with separator for prefix matching,
	// so "/workspace/djinn" does not match "/workspace/djinn2".
	prefix := root
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return strings.HasPrefix(path, prefix)
}

// cleanMountPath normalizes a path: cleans it and ensures consistent format.
func cleanMountPath(p string) string {
	if p == "" {
		return "/"
	}
	return filepath.Clean(p)
}
