package lector

import (
	"testing"

	djinncache "github.com/dpopsuev/djinn/cache"
)

func TestCachedLector_WriteThroughOnRead(t *testing.T) {
	inner := NewStubLector()
	l2 := djinncache.NewMemCache()
	cached := NewCachedLector(inner, l2, "coder-1")

	// Seed inner with a file.
	inner.SeedFile(FileEntry{Path: "/main.go", Package: "main", Language: "go"})

	// Read through cached — should write to L2.
	fe, ok := cached.FileInfo("/main.go")
	if !ok {
		t.Fatal("expected hit from inner")
	}
	if fe.Package != "main" {
		t.Fatalf("package = %q", fe.Package)
	}

	// L2 should now have it.
	if _, ok := l2.Get("coder-1", "file:/main.go"); !ok {
		t.Fatal("L2 should have file after read-through")
	}
}

func TestCachedLector_L2HitSkipsInner(t *testing.T) {
	inner := NewStubLector()
	l2 := djinncache.NewMemCache()
	cached := NewCachedLector(inner, l2, "coder-1")

	// Seed inner and do a read to populate L2.
	inner.SeedFile(FileEntry{Path: "/main.go", Package: "main"})
	cached.FileInfo("/main.go")

	// Clear inner — L2 should still serve.
	inner2 := NewStubLector() // fresh, empty
	cached2 := NewCachedLector(inner2, l2, "coder-1")

	fe, ok := cached2.FileInfo("/main.go")
	if !ok {
		t.Fatal("expected L2 cache hit")
	}
	if fe.Package != "main" {
		t.Fatalf("package from L2 = %q", fe.Package)
	}
}

func TestCachedLector_OnFileWrite_InvalidatesL2(t *testing.T) {
	inner := NewStubLector()
	l2 := djinncache.NewMemCache()
	cached := NewCachedLector(inner, l2, "coder-1")

	// Read to populate L2.
	inner.SeedFile(FileEntry{Path: "/main.go", Package: "main"})
	cached.FileInfo("/main.go")

	// Write invalidates + re-caches.
	cached.OnFileWrite("/main.go")

	// L2 should have updated entry (re-indexed by inner).
	if _, ok := l2.Get("coder-1", "file:/main.go"); !ok {
		t.Fatal("L2 should have re-cached after write")
	}
}

func TestCachedLector_OnFileDelete_EvictsL2(t *testing.T) {
	inner := NewStubLector()
	l2 := djinncache.NewMemCache()
	cached := NewCachedLector(inner, l2, "coder-1")

	inner.SeedFile(FileEntry{Path: "/main.go"})
	cached.FileInfo("/main.go")

	cached.OnFileDelete("/main.go")

	if _, ok := l2.Get("coder-1", "file:/main.go"); ok {
		t.Fatal("L2 should NOT have file after delete")
	}
}

func TestCachedLector_ScopeIsolation(t *testing.T) {
	l2 := djinncache.NewMemCache()
	inner1 := NewStubLector()
	inner2 := NewStubLector()
	cached1 := NewCachedLector(inner1, l2, "coder-1")
	cached2 := NewCachedLector(inner2, l2, "coder-2")

	// coder-1 reads a file.
	inner1.SeedFile(FileEntry{Path: "/main.go", Package: "main"})
	cached1.FileInfo("/main.go")

	// coder-2 can't see it via its own scope.
	_, ok := cached2.FileInfo("/main.go")
	if ok {
		t.Fatal("coder-2 should NOT see coder-1's cached file via own scope")
	}

	// But L2 has it under coder-1's scope (for recovery).
	if _, ok := l2.Get("coder-1", "file:/main.go"); !ok {
		t.Fatal("L2 should have coder-1's file")
	}
}

func TestCachedLector_AgentRecovery(t *testing.T) {
	l2 := djinncache.NewMemCache()
	inner := NewStubLector()
	cached := NewCachedLector(inner, l2, "coder-1")

	// coder-1 reads 2 files.
	inner.SeedFile(FileEntry{Path: "/main.go", Package: "main"})
	inner.SeedFile(FileEntry{Path: "/go.mod", Package: ""})
	cached.FileInfo("/main.go")
	cached.FileInfo("/go.mod")

	// coder-1 dies. New coder-1 spawns with same scope — L2 pre-warms.
	inner2 := NewStubLector() // fresh, empty
	cached2 := NewCachedLector(inner2, l2, "coder-1")

	fe, ok := cached2.FileInfo("/main.go")
	if !ok {
		t.Fatal("new coder-1 should recover /main.go from L2")
	}
	if fe.Package != "main" {
		t.Fatalf("recovered package = %q", fe.Package)
	}

	_, ok = cached2.FileInfo("/go.mod")
	if !ok {
		t.Fatal("new coder-1 should recover /go.mod from L2")
	}
}
