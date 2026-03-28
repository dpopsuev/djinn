//go:build linux

// Package namespace implements an invisible filesystem proxy using
// overlayfs. The agent's own built-in tools hit the filesystem —
// this package controls what the filesystem returns.
//
// Each Overlay mounts a copy-on-write layer over a real workspace:
//   - Reads pass through to the real workspace (lower)
//   - Writes are captured in a temp dir (upper) — real workspace untouched
//   - Diff compares upper vs lower to show what changed
//   - Commit selectively promotes files from upper to real workspace
//
// Uses fuse-overlayfs for unprivileged operation. No root required.
// Production implementation will live in Misbah (MSB-GOL-8).
package namespace

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Sentinel errors.
var (
	ErrUnsupported  = errors.New("namespace: fuse-overlayfs not available")
	ErrNotDirectory = errors.New("namespace: not a directory")
	ErrNotMounted   = errors.New("namespace: overlay not mounted")
	ErrUnmount      = errors.New("namespace: unmount failed")
)

// Overlay represents a mounted overlayfs instance.
type Overlay struct {
	Lower  string // real workspace (read-only in overlay)
	Upper  string // writable layer (agent's changes land here)
	Work   string // overlayfs scratch directory
	Merged string // what the agent sees (lower + upper merged)

	tempDir string
	mu      sync.Mutex
	mounted bool
}

// Mount creates a fuse-overlayfs overlay on the given workspace directory.
// Reads pass through to lower. Writes go to upper. Real workspace is untouched.
// Caller must call Unmount() when done.
func Mount(lower string) (*Overlay, error) {
	info, err := os.Stat(lower)
	if err != nil {
		return nil, fmt.Errorf("namespace: lower dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, lower)
	}

	if _, err := exec.LookPath("fuse-overlayfs"); err != nil {
		return nil, ErrUnsupported
	}

	tempDir, err := os.MkdirTemp("", "djinn-overlay-*")
	if err != nil {
		return nil, fmt.Errorf("namespace: create temp: %w", err)
	}

	upper := filepath.Join(tempDir, "upper")
	work := filepath.Join(tempDir, "work")
	merged := filepath.Join(tempDir, "merged")

	for _, d := range []string{upper, work, merged} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			os.RemoveAll(tempDir)
			return nil, fmt.Errorf("namespace: mkdir %s: %w", d, err)
		}
	}

	cmd := exec.Command("fuse-overlayfs", //nolint:gosec // paths are sandbox-controlled, not user input
		"-o", fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work),
		merged)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("namespace: fuse-overlayfs mount: %s: %w", out, err)
	}

	return &Overlay{
		Lower:   lower,
		Upper:   upper,
		Work:    work,
		Merged:  merged,
		tempDir: tempDir,
		mounted: true,
	}, nil
}

// Unmount tears down the overlay and removes all temp directories.
// The real workspace (lower) is untouched.
func (o *Overlay) Unmount() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.mounted {
		return nil
	}
	o.mounted = false

	// fusermount3 is the standard FUSE unmount tool.
	if out, err := exec.Command("fusermount3", "-u", o.Merged).CombinedOutput(); err != nil { //nolint:gosec // path is sandbox-controlled
		// Try fusermount (older systems).
		if out2, err2 := exec.Command("fusermount", "-u", o.Merged).CombinedOutput(); err2 != nil { //nolint:gosec // path is sandbox-controlled
			return fmt.Errorf("%w: %s / %s", ErrUnmount, out, out2)
		}
	}

	return os.RemoveAll(o.tempDir)
}

// Diff returns files that were modified or created in the overlay.
// Paths are relative to the workspace root.
func (o *Overlay) Diff() ([]string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.mounted {
		return nil, ErrNotMounted
	}

	var changed []string
	err := filepath.Walk(o.Upper, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == o.Upper {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(o.Upper, path)
		changed = append(changed, rel)
		return nil
	})
	return changed, err
}

// Commit copies selected files from the overlay upper dir to the real workspace.
// Only listed paths are promoted. Others stay in the overlay only.
func (o *Overlay) Commit(paths []string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.mounted {
		return ErrNotMounted
	}

	for _, p := range paths {
		src := filepath.Join(o.Upper, p)
		dst := filepath.Join(o.Lower, p)

		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("namespace: commit read %s: %w", p, err)
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("namespace: commit mkdir: %w", err)
		}

		info, _ := os.Stat(src)
		if err := os.WriteFile(dst, data, info.Mode()); err != nil {
			return fmt.Errorf("namespace: commit write %s: %w", p, err)
		}
	}
	return nil
}
