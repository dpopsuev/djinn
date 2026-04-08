package testkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTestWorkspace_CreatesDir(t *testing.T) {
	ws := NewTestWorkspace(t)
	if ws.Dir() == "" {
		t.Fatal("Dir() should not be empty")
	}
	info, err := os.Stat(ws.Dir())
	if err != nil {
		t.Fatalf("Dir() should exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("Dir() should be a directory")
	}
}

func TestNewTestWorkspace_WriteAndDiff(t *testing.T) {
	ws := NewTestWorkspace(t)

	if err := os.WriteFile(filepath.Join(ws.Dir(), "hello.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	changes, err := ws.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) == 0 {
		t.Fatal("expected at least 1 change after write")
	}
}

func TestNewTestWorkspace_Reset(t *testing.T) {
	ws := NewTestWorkspace(t)

	os.WriteFile(filepath.Join(ws.Dir(), "temp.txt"), []byte("data"), 0o644) //nolint:errcheck // test
	if err := ws.Reset(); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(ws.Dir())
	if len(entries) != 0 {
		t.Fatalf("expected empty after reset, got %d", len(entries))
	}
}

func TestNewTestWorkspace_FallbackWhenNoFuse(t *testing.T) {
	t.Setenv("DJINN_TEST_NO_FUSE", "1")
	ws := NewTestWorkspace(t)
	if ws.IsReal() {
		t.Fatal("should fall back to stub when DJINN_TEST_NO_FUSE is set")
	}
	// Should still work
	if ws.Dir() == "" {
		t.Fatal("stub workspace Dir should not be empty")
	}
}
