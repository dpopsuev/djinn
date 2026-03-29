//go:build linux

package namespace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/mirage"
)

func createSpace(t *testing.T, workspace string) mirage.Space {
	t.Helper()
	builder := mirage.NewOverlayBuilder()
	space, err := builder.Create(workspace)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}
	t.Cleanup(func() { space.Destroy() })
	return space
}

func TestMirageSpace_ReadThrough(t *testing.T) {
	lower := t.TempDir()
	os.WriteFile(filepath.Join(lower, "hello.txt"), []byte("original"), 0o644)

	space := createSpace(t, lower)
	got, err := os.ReadFile(filepath.Join(space.WorkDir(), "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatalf("got %q, want %q", got, "original")
	}
}

func TestMirageSpace_WriteCaptured(t *testing.T) {
	lower := t.TempDir()
	os.WriteFile(filepath.Join(lower, "main.go"), []byte("original"), 0o644)

	space := createSpace(t, lower)
	os.WriteFile(filepath.Join(space.WorkDir(), "main.go"), []byte("modified"), 0o644)

	actual, _ := os.ReadFile(filepath.Join(lower, "main.go"))
	if string(actual) != "original" {
		t.Fatalf("real file modified: got %q", actual)
	}
}

func TestMirageSpace_Diff(t *testing.T) {
	lower := t.TempDir()
	os.WriteFile(filepath.Join(lower, "existing.go"), []byte("v1"), 0o644)

	space := createSpace(t, lower)
	os.WriteFile(filepath.Join(space.WorkDir(), "existing.go"), []byte("v2"), 0o644)
	os.WriteFile(filepath.Join(space.WorkDir(), "new.go"), []byte("new"), 0o644)

	changes, err := space.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	kinds := make(map[string]mirage.ChangeKind)
	for _, c := range changes {
		kinds[c.Path] = c.Kind
	}
	if kinds["existing.go"] != mirage.Modified {
		t.Errorf("existing.go kind = %s, want modified", kinds["existing.go"])
	}
	if kinds["new.go"] != mirage.Created {
		t.Errorf("new.go kind = %s, want created", kinds["new.go"])
	}
}

func TestMirageSpace_Commit(t *testing.T) {
	lower := t.TempDir()
	space := createSpace(t, lower)

	os.WriteFile(filepath.Join(space.WorkDir(), "promoted.go"), []byte("promoted"), 0o644)
	if err := space.Commit([]string{"promoted.go"}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(lower, "promoted.go"))
	if err != nil {
		t.Fatal("committed file not in workspace")
	}
	if string(got) != "promoted" {
		t.Errorf("content = %q", got)
	}
}

func TestMirageSpace_Reset(t *testing.T) {
	lower := t.TempDir()
	space := createSpace(t, lower)

	os.WriteFile(filepath.Join(space.WorkDir(), "discard.go"), []byte("gone"), 0o644)
	if err := space.Reset(); err != nil {
		t.Fatal(err)
	}

	changes, _ := space.Diff()
	if len(changes) != 0 {
		t.Errorf("expected 0 changes after reset, got %d", len(changes))
	}
}

func TestMirageSpace_Destroy(t *testing.T) {
	lower := t.TempDir()
	builder := mirage.NewOverlayBuilder()
	space, err := builder.Create(lower)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}

	workDir := space.WorkDir()
	if err := space.Destroy(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatal("workdir not cleaned up after destroy")
	}
}

func skipIfUnsupported(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, ErrUnsupported) || errors.Is(err, mirage.ErrFuseNotAvailable) {
		t.Skip("fuse-overlayfs not available")
	}
}
