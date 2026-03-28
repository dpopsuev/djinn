//go:build linux

package namespace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOverlayMount_ReadThrough(t *testing.T) {
	// Given a workspace with a file
	lower := t.TempDir()
	if err := os.WriteFile(filepath.Join(lower, "hello.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	ov, err := Mount(lower)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}
	defer ov.Unmount()

	// When reading through the merged view
	got, err := os.ReadFile(filepath.Join(ov.Merged, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}

	// Then the content matches the original
	if string(got) != "original" {
		t.Fatalf("got %q, want %q", got, "original")
	}
}

func TestOverlayMount_WriteCaptured(t *testing.T) {
	// Given a workspace with a file
	lower := t.TempDir()
	if err := os.WriteFile(filepath.Join(lower, "main.go"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	ov, err := Mount(lower)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}
	defer ov.Unmount()

	// When writing through the merged view
	if err := os.WriteFile(filepath.Join(ov.Merged, "main.go"), []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Then the actual file is untouched
	actual, _ := os.ReadFile(filepath.Join(lower, "main.go"))
	if string(actual) != "original" {
		t.Fatalf("real file modified: got %q, want %q", actual, "original")
	}

	// And the upper dir has the modified version
	upper, _ := os.ReadFile(filepath.Join(ov.Upper, "main.go"))
	if string(upper) != "modified" {
		t.Fatalf("upper missing write: got %q, want %q", upper, "modified")
	}
}

func TestOverlayMount_NewFileCaptured(t *testing.T) {
	lower := t.TempDir()

	ov, err := Mount(lower)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}
	defer ov.Unmount()

	// When creating a new file
	if err := os.WriteFile(filepath.Join(ov.Merged, "new.go"), []byte("new content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Then it doesn't exist in the real workspace
	if _, err := os.Stat(filepath.Join(lower, "new.go")); !os.IsNotExist(err) {
		t.Fatal("new file leaked to real workspace")
	}

	// But it does exist in upper
	got, _ := os.ReadFile(filepath.Join(ov.Upper, "new.go"))
	if string(got) != "new content" {
		t.Fatalf("upper missing new file: got %q", got)
	}
}

func TestOverlayMount_ReadOwnWrite(t *testing.T) {
	lower := t.TempDir()
	if err := os.WriteFile(filepath.Join(lower, "foo.go"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	ov, err := Mount(lower)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}
	defer ov.Unmount()

	// Write modified version
	if err := os.WriteFile(filepath.Join(ov.Merged, "foo.go"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Read back — should see own write (from upper), not original (from lower)
	got, _ := os.ReadFile(filepath.Join(ov.Merged, "foo.go"))
	if string(got) != "v2" {
		t.Fatalf("should see own write: got %q, want %q", got, "v2")
	}
}

func TestOverlayUnmount_CleansUp(t *testing.T) {
	lower := t.TempDir()

	ov, err := Mount(lower)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}

	// Capture paths before unmount
	merged := ov.Merged
	upper := ov.Upper

	if err := ov.Unmount(); err != nil {
		t.Fatal(err)
	}

	// Temp dirs should be gone
	if _, err := os.Stat(merged); !os.IsNotExist(err) {
		t.Fatal("merged dir not cleaned up")
	}
	if _, err := os.Stat(upper); !os.IsNotExist(err) {
		t.Fatal("upper dir not cleaned up")
	}
}

func TestOverlayMount_FailsGracefullyOnBadLower(t *testing.T) {
	_, err := Mount("/nonexistent/path")
	if err == nil {
		t.Fatal("should error on nonexistent lower dir")
	}
}

func skipIfUnsupported(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, ErrUnsupported) {
		t.Skip("overlayfs not supported on this system")
	}
}
