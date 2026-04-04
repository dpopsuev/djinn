package workspace

import (
	"errors"
	"log/slog"
	"testing"
)

func newTestTable() *MountTable {
	return NewMountTable(slog.Default())
}

func TestMountTable_MountAndTranslate(t *testing.T) {
	mt := newTestTable()

	err := mt.Mount("/workspace/djinn", "/home/user/Workspace/djinn", false, ScopeSystem)
	if err != nil {
		t.Fatalf("Mount: %v", err)
	}

	// Translate a file inside the mount.
	got, err := mt.Translate("/workspace/djinn/agent/loop.go")
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	want := "/home/user/Workspace/djinn/agent/loop.go"
	if got != want {
		t.Errorf("Translate = %q, want %q", got, want)
	}

	// Translate the mount root itself.
	got, err = mt.Translate("/workspace/djinn")
	if err != nil {
		t.Fatalf("Translate root: %v", err)
	}
	if got != "/home/user/Workspace/djinn" {
		t.Errorf("Translate root = %q", got)
	}
}

func TestMountTable_Unmount(t *testing.T) {
	mt := newTestTable()

	err := mt.Mount("/workspace/djinn", "/home/user/Workspace/djinn", false, ScopeSystem)
	if err != nil {
		t.Fatal(err)
	}

	// Verify mount works.
	if _, err := mt.Translate("/workspace/djinn/main.go"); err != nil {
		t.Fatalf("before unmount: %v", err)
	}

	// Unmount.
	if err := mt.Unmount("/workspace/djinn"); err != nil {
		t.Fatalf("Unmount: %v", err)
	}

	// Translate should now fail.
	_, err = mt.Translate("/workspace/djinn/main.go")
	if !errors.Is(err, ErrNotMounted) {
		t.Fatalf("after unmount: err = %v, want ErrNotMounted", err)
	}

	// Double unmount should fail.
	err = mt.Unmount("/workspace/djinn")
	if !errors.Is(err, ErrNotMounted) {
		t.Fatalf("double unmount: err = %v, want ErrNotMounted", err)
	}
}

func TestMountTable_TranslatePathEscape(t *testing.T) {
	mt := newTestTable()

	err := mt.Mount("/workspace/djinn", "/home/user/Workspace/djinn", false, ScopeSystem)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt to escape via ../../etc/passwd.
	_, err = mt.Translate("/workspace/djinn/../../etc/passwd")
	if !errors.Is(err, ErrPathEscape) {
		t.Fatalf("path escape: err = %v, want ErrPathEscape", err)
	}
}

func TestMountTable_MountConflict(t *testing.T) {
	mt := newTestTable()

	err := mt.Mount("/workspace/djinn", "/home/user/Workspace/djinn", false, ScopeSystem)
	if err != nil {
		t.Fatal(err)
	}

	// Same virtual path, different host — conflict.
	err = mt.Mount("/workspace/djinn", "/other/path", false, ScopeSystem)
	if !errors.Is(err, ErrMountConflict) {
		t.Fatalf("conflict: err = %v, want ErrMountConflict", err)
	}

	// Same virtual path, same host — still conflict (idempotent not supported).
	err = mt.Mount("/workspace/djinn", "/home/user/Workspace/djinn", false, ScopeSystem)
	if !errors.Is(err, ErrMountConflict) {
		t.Fatalf("same-host conflict: err = %v, want ErrMountConflict", err)
	}
}

func TestMountTable_LongestPrefixMatch(t *testing.T) {
	mt := newTestTable()

	// Mount parent and child with different host paths.
	if err := mt.Mount("/workspace", "/mnt/workspace", true, ScopeGlobal); err != nil {
		t.Fatal(err)
	}
	if err := mt.Mount("/workspace/djinn", "/home/user/Workspace/djinn", false, ScopeSystem); err != nil {
		t.Fatal(err)
	}

	// File under /workspace/djinn should use the more specific mount.
	got, err := mt.Translate("/workspace/djinn/main.go")
	if err != nil {
		t.Fatalf("Translate djinn: %v", err)
	}
	want := "/home/user/Workspace/djinn/main.go"
	if got != want {
		t.Errorf("longest prefix: got %q, want %q", got, want)
	}

	// File under /workspace but NOT under /workspace/djinn should use parent.
	got, err = mt.Translate("/workspace/misbah/jail.go")
	if err != nil {
		t.Fatalf("Translate misbah: %v", err)
	}
	want = "/mnt/workspace/misbah/jail.go"
	if got != want {
		t.Errorf("parent mount: got %q, want %q", got, want)
	}
}

func TestMountTable_Resolve_ReverseTranslate(t *testing.T) {
	mt := newTestTable()

	if err := mt.Mount("/workspace/djinn", "/home/user/Workspace/djinn", false, ScopeSystem); err != nil {
		t.Fatal(err)
	}

	// Reverse translate a host path.
	got, err := mt.Resolve("/home/user/Workspace/djinn/agent/loop.go")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := "/workspace/djinn/agent/loop.go"
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}

	// Reverse translate the mount root.
	got, err = mt.Resolve("/home/user/Workspace/djinn")
	if err != nil {
		t.Fatalf("Resolve root: %v", err)
	}
	if got != "/workspace/djinn" {
		t.Errorf("Resolve root = %q", got)
	}

	// Unknown host path.
	_, err = mt.Resolve("/opt/somewhere/else")
	if !errors.Is(err, ErrNotMounted) {
		t.Fatalf("Resolve unknown: err = %v, want ErrNotMounted", err)
	}
}

func TestMountTable_List(t *testing.T) {
	mt := newTestTable()

	// Empty table.
	if got := mt.List(); len(got) != 0 {
		t.Fatalf("empty List = %d entries", len(got))
	}

	// Add entries in non-alphabetical order.
	mt.Mount("/workspace/misbah", "/home/user/misbah", false, ScopeSystem)   //nolint:errcheck // test setup, error not relevant
	mt.Mount("/workspace/djinn", "/home/user/djinn", false, ScopeSystem)     //nolint:errcheck // test setup, error not relevant
	mt.Mount("/workspace/scribe", "/home/user/scribe", true, ScopeEcosystem) //nolint:errcheck // test setup, error not relevant

	list := mt.List()
	if len(list) != 3 {
		t.Fatalf("List = %d entries, want 3", len(list))
	}

	// Should be sorted by virtual path.
	if list[0].VirtualPath != "/workspace/djinn" {
		t.Errorf("list[0].VirtualPath = %q", list[0].VirtualPath)
	}
	if list[1].VirtualPath != "/workspace/misbah" {
		t.Errorf("list[1].VirtualPath = %q", list[1].VirtualPath)
	}
	if list[2].VirtualPath != "/workspace/scribe" {
		t.Errorf("list[2].VirtualPath = %q", list[2].VirtualPath)
	}

	// Verify metadata preserved.
	if !list[2].ReadOnly {
		t.Error("scribe should be ReadOnly")
	}
	if list[2].ScopeType != ScopeEcosystem {
		t.Errorf("scribe ScopeType = %q", list[2].ScopeType)
	}
}

func TestMountTable_TranslateNoMount(t *testing.T) {
	mt := newTestTable()

	_, err := mt.Translate("/workspace/djinn/main.go")
	if !errors.Is(err, ErrNotMounted) {
		t.Fatalf("empty table translate: err = %v, want ErrNotMounted", err)
	}
}

func TestMountTable_NilLogger(t *testing.T) {
	// Should not panic with nil logger.
	mt := NewMountTable(nil)
	if err := mt.Mount("/a", "/b", false, ScopeSystem); err != nil {
		t.Fatal(err)
	}
}

func TestMountTable_PathEscapeWithDotDot(t *testing.T) {
	mt := newTestTable()

	if err := mt.Mount("/workspace/djinn", "/home/user/Workspace/djinn", false, ScopeSystem); err != nil {
		t.Fatal(err)
	}

	// Various escape attempts.
	cases := []string{
		"/workspace/djinn/../../../etc/passwd",
		"/workspace/djinn/../../../../etc/shadow",
	}
	for _, c := range cases {
		_, err := mt.Translate(c)
		if !errors.Is(err, ErrPathEscape) {
			t.Errorf("Translate(%q): err = %v, want ErrPathEscape", c, err)
		}
	}
}

func TestMountTable_SimilarPrefixNoFalseMatch(t *testing.T) {
	mt := newTestTable()

	// Mount /workspace/djinn but NOT /workspace/djinn2.
	if err := mt.Mount("/workspace/djinn", "/home/user/djinn", false, ScopeSystem); err != nil {
		t.Fatal(err)
	}

	// /workspace/djinn2/foo.go should NOT match /workspace/djinn.
	_, err := mt.Translate("/workspace/djinn2/foo.go")
	if !errors.Is(err, ErrNotMounted) {
		t.Fatalf("similar prefix: err = %v, want ErrNotMounted", err)
	}
}
