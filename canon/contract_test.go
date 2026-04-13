package canon

import (
	"errors"
	"testing"
	"time"
)

// RunCanonContract runs the Liskov contract test suite against any Canon.
// Every implementation must pass these tests.
func RunCanonContract(t *testing.T, factory func(t *testing.T) Canon) {
	t.Helper()

	t.Run("ContentHash_returns_hash", func(t *testing.T) {
		c := factory(t)
		hash, err := c.ContentHash("README.md")
		if err != nil {
			t.Fatalf("ContentHash: %v", err)
		}
		if hash == "" {
			t.Fatal("hash should not be empty")
		}
		if len(hash) != 64 { // SHA256 hex
			t.Fatalf("hash length = %d, want 64 (SHA256 hex)", len(hash))
		}
	})

	t.Run("ContentHash_same_file_same_hash", func(t *testing.T) {
		c := factory(t)
		h1, _ := c.ContentHash("README.md")
		h2, _ := c.ContentHash("README.md")
		if h1 != h2 {
			t.Fatalf("same file different hash: %q vs %q", h1, h2)
		}
	})

	t.Run("ContentHash_missing_file", func(t *testing.T) {
		c := factory(t)
		_, err := c.ContentHash("nonexistent.go")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		var fnf *FileNotFoundError
		if !errors.As(err, &fnf) {
			t.Fatalf("expected FileNotFoundError, got %T: %v", err, err)
		}
	})

	t.Run("DirtyFiles_returns_list", func(t *testing.T) {
		c := factory(t)
		files, err := c.DirtyFiles()
		if err != nil {
			t.Fatalf("DirtyFiles: %v", err)
		}
		// May be empty or non-empty depending on factory setup.
		_ = files
	})

	t.Run("RecentCommits_returns_commits", func(t *testing.T) {
		c := factory(t)
		commits, err := c.RecentCommits(5)
		if err != nil {
			t.Fatalf("RecentCommits: %v", err)
		}
		if len(commits) == 0 {
			t.Fatal("expected at least one commit")
		}
		if commits[0].Hash == "" {
			t.Fatal("commit hash should not be empty")
		}
		if commits[0].Message == "" {
			t.Fatal("commit message should not be empty")
		}
	})

	t.Run("Blame_returns_lines", func(t *testing.T) {
		c := factory(t)
		lines, err := c.Blame("README.md")
		if err != nil {
			t.Fatalf("Blame: %v", err)
		}
		if len(lines) == 0 {
			t.Fatal("expected at least one blame line")
		}
		if lines[0].Hash == "" {
			t.Fatal("blame line hash should not be empty")
		}
		if lines[0].Line != 1 {
			t.Fatalf("first blame line = %d, want 1", lines[0].Line)
		}
	})

	t.Run("Blame_missing_file", func(t *testing.T) {
		c := factory(t)
		_, err := c.Blame("nonexistent.go")
		if err == nil {
			t.Fatal("expected error for missing file blame")
		}
	})
}

// --- Run contract against StubCanon ---

func TestStubCanon_Contract(t *testing.T) {
	RunCanonContract(t, func(t *testing.T) Canon {
		t.Helper()
		s := NewStubCanon()
		s.SeedHash("README.md", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
		s.SeedCommits([]Commit{
			{Hash: "abc123", Author: "test", Date: time.Now(), Message: "initial"},
		})
		s.SeedBlame("README.md", []BlameLine{
			{Line: 1, Hash: "abc123", Author: "test"},
		})
		return s
	})
}

// --- Run contract against FakeCanon ---

func TestFakeCanon_Contract(t *testing.T) {
	RunCanonContract(t, func(t *testing.T) Canon {
		t.Helper()
		dir := t.TempDir()
		fake, err := NewFakeCanon(dir)
		if err != nil {
			t.Fatalf("NewFakeCanon: %v", err)
		}
		return fake
	})
}

// --- Additional FakeCanon tests ---

func TestFakeCanon_DirtyFiles(t *testing.T) {
	dir := t.TempDir()
	fake, err := NewFakeCanon(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Clean repo — no dirty files.
	dirty, err := fake.DirtyFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirty) != 0 {
		t.Fatalf("expected 0 dirty, got %d", len(dirty))
	}

	// Write uncommitted file → dirty.
	if err := fake.WriteFile("new.go", "package main"); err != nil {
		t.Fatal(err)
	}

	dirty, err = fake.DirtyFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirty) != 1 {
		t.Fatalf("expected 1 dirty, got %d", len(dirty))
	}
	if dirty[0] != "new.go" {
		t.Fatalf("dirty file = %q, want new.go", dirty[0])
	}
}

func TestFakeCanon_ContentHash_ChangesOnEdit(t *testing.T) {
	dir := t.TempDir()
	fake, err := NewFakeCanon(dir)
	if err != nil {
		t.Fatal(err)
	}

	h1, _ := fake.ContentHash("README.md")

	// Edit the file.
	if err := fake.WriteFile("README.md", "# changed\n"); err != nil {
		t.Fatal(err)
	}

	h2, _ := fake.ContentHash("README.md")

	if h1 == h2 {
		t.Fatal("hash should change after edit")
	}
}
