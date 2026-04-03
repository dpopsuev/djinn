package vfs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dpopsuev/djinn/scope"
)

// Security tests for VFS mount table (TSK-596, TSK-597).
// Trust boundary: agent virtual paths ↔ host filesystem.
// STRIDE: Elevation of Privilege via path traversal or mount injection.

// --- TSK-596: Path escape tests ---

func TestVFS_PathEscape_DotDot(t *testing.T) {
	mt := newTestTable()
	if err := mt.Mount("/workspace/djinn", "/home/user/djinn", false, scope.ScopeSystem); err != nil {
		t.Fatal(err)
	}

	// Classic directory traversal: /../../../etc/passwd
	_, err := mt.Translate("/workspace/djinn/../../../etc/passwd")
	if !errors.Is(err, ErrPathEscape) {
		t.Fatalf("dot-dot escape: err = %v, want ErrPathEscape", err)
	}
}

func TestVFS_PathEscape_Symlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	// Set up a real temp directory structure with a symlink escape.
	hostRoot := t.TempDir()
	secretDir := t.TempDir()
	secretFile := filepath.Join(secretDir, "passwd")
	if err := os.WriteFile(secretFile, []byte("root:x:0:0"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create symlink inside hostRoot that points outside it.
	symlinkPath := filepath.Join(hostRoot, "escape")
	if err := os.Symlink(secretDir, symlinkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	mt := newTestTable()
	if err := mt.Mount("/workspace", hostRoot, false, scope.ScopeSystem); err != nil {
		t.Fatal(err)
	}

	// The Translate call itself won't follow symlinks (it's pure string manipulation).
	// But the resolved host path "/hostRoot/escape/passwd" does exist and the string
	// path IS under hostRoot — this shows MountTable.Translate is string-based.
	// A production system should EvalSymlinks before serving content.
	hostPath, err := mt.Translate("/workspace/escape/passwd")
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}

	// The returned path should start with hostRoot (string-based it does).
	// But if we resolve symlinks, it escapes.
	resolved, err := filepath.EvalSymlinks(hostPath)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}

	if !isUnder(resolved, hostRoot) {
		// Expected: resolved path escapes hostRoot.
		// This documents that callers MUST EvalSymlinks after Translate
		// and re-check containment to prevent symlink-based escape.
		t.Logf("SECURITY NOTE: symlink escape detected — resolved %q is outside %q", resolved, hostRoot)
		t.Logf("Callers must EvalSymlinks + re-check containment after Translate")
	}
}

func TestVFS_PathEscape_AbsoluteOutsideMount(t *testing.T) {
	mt := newTestTable()
	if err := mt.Mount("/workspace/djinn", "/home/user/djinn", false, scope.ScopeSystem); err != nil {
		t.Fatal(err)
	}

	// /etc/passwd has no mount — should be ErrNotMounted.
	_, err := mt.Translate("/etc/passwd")
	if !errors.Is(err, ErrNotMounted) {
		t.Fatalf("absolute outside mount: err = %v, want ErrNotMounted", err)
	}
}

func TestVFS_ValidRelativePath(t *testing.T) {
	mt := newTestTable()
	if err := mt.Mount("/workspace/djinn", "/home/user/djinn", false, scope.ScopeSystem); err != nil {
		t.Fatal(err)
	}

	got, err := mt.Translate("/workspace/djinn/agent/loop.go")
	if err != nil {
		t.Fatalf("valid path: %v", err)
	}
	want := "/home/user/djinn/agent/loop.go"
	if got != want {
		t.Errorf("Translate = %q, want %q", got, want)
	}
}

// --- TSK-597: Mount injection tests ---

func TestVFS_MountInjection_ConflictRejected(t *testing.T) {
	mt := newTestTable()

	// First mount succeeds.
	if err := mt.Mount("/workspace/djinn", "/home/user/djinn", false, scope.ScopeSystem); err != nil {
		t.Fatal(err)
	}

	// Second mount to same virtual path but different host → conflict.
	err := mt.Mount("/workspace/djinn", "/tmp/evil", false, scope.ScopeSystem)
	if !errors.Is(err, ErrMountConflict) {
		t.Fatalf("mount injection: err = %v, want ErrMountConflict", err)
	}
}

func TestVFS_MountInjection_NestedMountAllowed(t *testing.T) {
	mt := newTestTable()

	// Parent mount.
	if err := mt.Mount("/workspace", "/home/user/workspace", true, scope.ScopeGlobal); err != nil {
		t.Fatal(err)
	}

	// Nested mount under parent — different virtual path, allowed.
	if err := mt.Mount("/workspace/djinn", "/home/user/djinn", false, scope.ScopeSystem); err != nil {
		t.Fatalf("nested mount should be allowed: %v", err)
	}

	// Verify longest prefix match routes correctly.
	got, err := mt.Translate("/workspace/djinn/main.go")
	if err != nil {
		t.Fatalf("translate nested: %v", err)
	}
	// Should use the more specific /workspace/djinn mount.
	want := "/home/user/djinn/main.go"
	if got != want {
		t.Errorf("nested mount: got %q, want %q", got, want)
	}

	// Path outside nested mount but under parent should use parent.
	got, err = mt.Translate("/workspace/other/file.go")
	if err != nil {
		t.Fatalf("translate parent: %v", err)
	}
	want = "/home/user/workspace/other/file.go"
	if got != want {
		t.Errorf("parent mount: got %q, want %q", got, want)
	}
}
