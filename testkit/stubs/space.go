// space.go — StubSpace: in-memory mirage.Space for testing without FUSE.
//
// Records all operations for assertion. Uses a real temp dir as WorkDir
// so file operations work, but no overlay — just a plain directory.
package stubs

import (
	"os"
	"sync"
	"testing"

	"github.com/dpopsuev/mirage"
)

// StubSpace implements mirage.Space with a plain temp directory.
// No overlay, no FUSE — just records operations for assertion.
type StubSpace struct {
	mu        sync.Mutex
	workDir   string
	changes   []mirage.Change
	committed []string
	resets    int
	destroyed bool
}

var _ mirage.Space = (*StubSpace)(nil)

// NewStubSpace creates a Space backed by a temp directory.
// Cleaned up automatically via t.Cleanup.
func NewStubSpace(t *testing.T) *StubSpace {
	t.Helper()
	dir := t.TempDir()
	s := &StubSpace{workDir: dir}
	t.Cleanup(func() { s.Destroy() }) //nolint:errcheck // test cleanup
	return s
}

// NewStubSpaceAt creates a Space backed by the given directory.
func NewStubSpaceAt(dir string) *StubSpace {
	return &StubSpace{workDir: dir}
}

func (s *StubSpace) WorkDir() string { return s.workDir }

func (s *StubSpace) Diff() ([]mirage.Change, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Scan workDir for actual files (real I/O, no overlay)
	entries, err := os.ReadDir(s.workDir)
	if err != nil {
		return nil, err
	}
	changes := make([]mirage.Change, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		changes = append(changes, mirage.Change{
			Path: e.Name(),
			Kind: mirage.Created,
			Size: info.Size(),
		})
	}
	s.changes = changes
	return changes, nil
}

func (s *StubSpace) Commit(paths []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.committed = append(s.committed, paths...)
	return nil
}

func (s *StubSpace) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resets++
	// Clear workDir contents
	entries, _ := os.ReadDir(s.workDir)
	for _, e := range entries {
		os.RemoveAll(s.workDir + "/" + e.Name()) //nolint:errcheck // best-effort reset
	}
	return nil
}

func (s *StubSpace) Destroy() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.destroyed = true
	return nil
}

// Committed returns paths that were committed.
func (s *StubSpace) Committed() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.committed
}

// Resets returns how many times Reset was called.
func (s *StubSpace) Resets() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resets
}

// Destroyed returns whether Destroy was called.
func (s *StubSpace) Destroyed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.destroyed
}
