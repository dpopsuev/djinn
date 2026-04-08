// workspace.go — TestWorkspace: isolated workspace for all test harnesses.
//
// Wraps Mirage (fuse-overlayfs) when available, falls back to StubSpace
// (plain temp dir) when FUSE is unavailable (CI, quick tests).
//
// Usage:
//
//	ws := testkit.NewTestWorkspace(t)
//	defer ws.Destroy()  // auto-called via t.Cleanup
//	// write files to ws.Dir()
//	// check ws.Diff() for changes
//	// ws.Reset() to discard changes
package testkit

import (
	"os"
	"testing"

	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/mirage"
)

// TestWorkspace provides an isolated directory for tests.
// Uses Mirage overlay when available, falls back to StubSpace.
type TestWorkspace struct {
	space mirage.Space
	real  bool // true if Mirage overlay, false if stub
}

// NewTestWorkspace creates an isolated workspace.
// Uses Mirage overlay if FUSE available, otherwise falls back to StubSpace.
func NewTestWorkspace(t *testing.T, projectRoot ...string) *TestWorkspace {
	t.Helper()

	// Skip Mirage if explicitly disabled
	if os.Getenv("DJINN_TEST_NO_FUSE") != "" {
		return newStubWorkspace(t)
	}

	// Try Mirage overlay
	root := ""
	if len(projectRoot) > 0 {
		root = projectRoot[0]
	}
	if root == "" {
		root = t.TempDir()
	}

	space, err := mirage.Create(mirage.Spec{
		Workspace: root,
		Backend:   mirage.Overlay,
	})
	if err != nil {
		// FUSE not available — fall back to stub
		t.Logf("testkit: Mirage unavailable (%v), using StubSpace", err)
		return newStubWorkspace(t)
	}

	t.Cleanup(func() { space.Destroy() }) //nolint:errcheck // test cleanup
	return &TestWorkspace{space: space, real: true}
}

func newStubWorkspace(t *testing.T) *TestWorkspace {
	t.Helper()
	return &TestWorkspace{space: stubs.NewStubSpace(t), real: false}
}

// Dir returns the workspace directory where tests should write files.
func (w *TestWorkspace) Dir() string { return w.space.WorkDir() }

// Diff returns files changed since workspace creation.
func (w *TestWorkspace) Diff() ([]mirage.Change, error) { return w.space.Diff() }

// Reset discards all changes, returning workspace to clean state.
func (w *TestWorkspace) Reset() error { return w.space.Reset() }

// Destroy tears down the workspace. Auto-called by t.Cleanup.
func (w *TestWorkspace) Destroy() error { return w.space.Destroy() }

// IsReal returns true if backed by a real Mirage overlay (not a stub).
func (w *TestWorkspace) IsReal() bool { return w.real }
