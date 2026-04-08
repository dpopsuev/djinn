package stubs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/mirage"
)

func TestStubSpace_WorkDir(t *testing.T) {
	s := NewStubSpace(t)
	if s.WorkDir() == "" {
		t.Fatal("WorkDir should not be empty")
	}
}

func TestStubSpace_WriteAndDiff(t *testing.T) {
	s := NewStubSpace(t)

	// Write a file into the workspace
	if err := os.WriteFile(filepath.Join(s.WorkDir(), "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	changes, err := s.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("changes = %d, want 1", len(changes))
	}
	if changes[0].Path != "main.go" {
		t.Fatalf("path = %q, want main.go", changes[0].Path)
	}
	if changes[0].Kind != mirage.Created {
		t.Fatalf("kind = %q, want created", changes[0].Kind)
	}
}

func TestStubSpace_Reset(t *testing.T) {
	s := NewStubSpace(t)

	// Write then reset
	os.WriteFile(filepath.Join(s.WorkDir(), "temp.txt"), []byte("data"), 0o644) //nolint:errcheck // test
	s.Reset()                                                                   //nolint:errcheck // test

	// WorkDir should be empty after reset
	entries, _ := os.ReadDir(s.WorkDir())
	if len(entries) != 0 {
		t.Fatalf("expected empty dir after reset, got %d entries", len(entries))
	}
	if s.Resets() != 1 {
		t.Fatalf("resets = %d, want 1", s.Resets())
	}
}

func TestStubSpace_Commit(t *testing.T) {
	s := NewStubSpace(t)
	s.Commit([]string{"main.go", "go.mod"}) //nolint:errcheck // test

	committed := s.Committed()
	if len(committed) != 2 {
		t.Fatalf("committed = %d, want 2", len(committed))
	}
}

func TestStubSpace_Destroy(t *testing.T) {
	s := NewStubSpace(t)
	if s.Destroyed() {
		t.Fatal("should not be destroyed yet")
	}
	s.Destroy() //nolint:errcheck // test
	if !s.Destroyed() {
		t.Fatal("should be destroyed")
	}
}

func TestStubSpace_WritesDoNotEscape(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "project")
	os.MkdirAll(projectDir, 0o755) //nolint:errcheck // test

	// Write a marker file in project dir
	os.WriteFile(filepath.Join(projectDir, "original.txt"), []byte("keep"), 0o644) //nolint:errcheck // test

	// StubSpace uses its own temp dir, not projectDir
	s := NewStubSpace(t)
	os.WriteFile(filepath.Join(s.WorkDir(), "agent_file.txt"), []byte("new"), 0o644) //nolint:errcheck // test

	// Original project dir should be untouched
	if _, err := os.Stat(filepath.Join(projectDir, "agent_file.txt")); err == nil {
		t.Fatal("agent writes should not escape to project dir")
	}
}
