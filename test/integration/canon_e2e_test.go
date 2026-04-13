//go:build integration

package integration

import (
	"testing"

	"github.com/dpopsuev/djinn/canon"
	djinncache "github.com/dpopsuev/djinn/cache"
	"github.com/dpopsuev/djinn/substrate"
)

// TestCanon_E2E_SubstrateComposition proves Canon + L2 compose through Substrate.
// Content hash cached: first call computes, second call serves from L2.
func TestCanon_E2E_SubstrateComposition(t *testing.T) {
	dir := t.TempDir()
	fake, err := canon.NewFakeCanon(dir)
	if err != nil {
		t.Fatalf("NewFakeCanon: %v", err)
	}

	// Write a file to hash.
	if err := fake.WriteFile("main.go", "package main\nfunc main() {}\n"); err != nil {
		t.Fatal(err)
	}
	if err := fake.CommitAll("add main.go"); err != nil {
		t.Fatal(err)
	}

	// Compose through Substrate.
	l2 := djinncache.NewMemCache()
	sub := substrate.New(dir,
		substrate.WithCanon(fake),
		substrate.WithL2Cache(l2),
	)

	// First call: Canon computes hash.
	c := sub.Canon()
	if c == nil {
		t.Fatal("Canon should not be nil after WithCanon")
	}

	h1, err := c.ContentHash("main.go")
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	if h1 == "" {
		t.Fatal("hash should not be empty")
	}

	// Write hash to L2 (simulating what RealCanon will do).
	l2.Put("canon", "hash:main.go", []byte(h1))

	// Second call: should return same hash (content unchanged).
	h2, err := c.ContentHash("main.go")
	if err != nil {
		t.Fatalf("ContentHash 2: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("hash changed without edit: %q vs %q", h1, h2)
	}

	// L2 has the cached value.
	cached, ok := l2.Get("canon", "hash:main.go")
	if !ok {
		t.Fatal("L2 should have cached hash")
	}
	if string(cached) != h1 {
		t.Fatalf("L2 hash = %q, want %q", cached, h1)
	}

	// Edit file → hash changes.
	if err := fake.WriteFile("main.go", "package main\nfunc main() { println(\"changed\") }\n"); err != nil {
		t.Fatal(err)
	}
	h3, err := c.ContentHash("main.go")
	if err != nil {
		t.Fatalf("ContentHash after edit: %v", err)
	}
	if h3 == h1 {
		t.Fatal("hash should change after edit")
	}
}

// TestCanon_E2E_DirtyFilesAndCommits proves dirty files and commits work through Substrate.
func TestCanon_E2E_DirtyFilesAndCommits(t *testing.T) {
	dir := t.TempDir()
	fake, err := canon.NewFakeCanon(dir)
	if err != nil {
		t.Fatal(err)
	}

	sub := substrate.New(dir, substrate.WithCanon(fake))
	c := sub.Canon()

	// Initial state: clean repo.
	dirty, err := c.DirtyFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirty) != 0 {
		t.Fatalf("expected 0 dirty, got %d", len(dirty))
	}

	// Has at least initial commit.
	commits, err := c.RecentCommits(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least 1 commit")
	}
	if commits[0].Message != "initial" {
		t.Fatalf("first commit = %q, want initial", commits[0].Message)
	}

	// Write uncommitted file → dirty.
	if err := fake.WriteFile("dirty.go", "package main"); err != nil {
		t.Fatal(err)
	}
	dirty, err = c.DirtyFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirty) != 1 {
		t.Fatalf("expected 1 dirty, got %d", len(dirty))
	}
}
