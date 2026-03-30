//go:build linux

package namespace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Sandbox Escape Canary Tests ---
//
// These tests verify that sandbox isolation cannot be bypassed.
// Each test targets a specific escape vector.

// TestCanary_WriteDoesNotEscapeOverlay creates a sandbox, writes a file
// inside the overlay, and verifies the file does NOT exist on the host
// filesystem outside the overlay mount.
func TestCanary_WriteDoesNotEscapeOverlay(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"existing.go": "package existing"})
	sb, handle := createSandbox(t, dir)

	ctx := context.Background()

	// Write a brand-new file through the sandbox.
	result, err := sb.Exec(ctx, handle,
		[]string{"bash", "-c", "echo -n 'escaped content' > canary_escape.txt"}, 10)
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("write failed: exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}

	// Verify the file exists inside the sandbox.
	result, err = sb.Exec(ctx, handle, []string{"cat", "canary_escape.txt"}, 10)
	if err != nil {
		t.Fatalf("read inside sandbox failed: %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "escaped content" {
		t.Fatalf("sandbox read = %q, want %q", result.Stdout, "escaped content")
	}

	// Verify the file does NOT exist on the host.
	hostPath := filepath.Join(dir, "canary_escape.txt")
	if _, err := os.Stat(hostPath); !os.IsNotExist(err) {
		t.Fatalf("canary file escaped to host at %s (err=%v)", hostPath, err)
	}

	// Also verify an existing file modified in overlay stays untouched on host.
	result, err = sb.Exec(ctx, handle,
		[]string{"bash", "-c", "echo -n 'overlay modified' > existing.go"}, 10)
	if err != nil {
		t.Fatalf("overwrite existing failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("overwrite failed: exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}

	hostContent, err := os.ReadFile(filepath.Join(dir, "existing.go"))
	if err != nil {
		t.Fatalf("reading host file: %v", err)
	}
	if string(hostContent) != "package existing" {
		t.Fatalf("host file modified: got %q, want %q", hostContent, "package existing")
	}
}

// TestCanary_CommandArrayPreventsInjection verifies that Exec with []string
// does not perform shell interpretation. A semicolon and dangerous command
// passed as arguments to echo must be treated as literal strings.
func TestCanary_CommandArrayPreventsInjection(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"precious.txt": "do not delete"})
	sb, handle := createSandbox(t, dir)

	ctx := context.Background()

	// The dangerous payload: if shell-interpreted, "rm -rf /" would execute.
	// With []string, "hello; rm -rf /" is a single argument to echo.
	result, err := sb.Exec(ctx, handle,
		[]string{"echo", "hello; rm -rf /"}, 10)
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("echo failed: exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}

	// Stdout must contain the literal injection string, proving no interpretation.
	got := strings.TrimSpace(result.Stdout)
	if got != "hello; rm -rf /" {
		t.Fatalf("stdout = %q, want %q (shell interpreted the payload?)", got, "hello; rm -rf /")
	}

	// Verify precious.txt was not deleted inside the sandbox.
	result, err = sb.Exec(ctx, handle, []string{"cat", "precious.txt"}, 10)
	if err != nil {
		t.Fatalf("cat precious.txt failed: %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "do not delete" {
		t.Fatalf("precious.txt content = %q, want %q", result.Stdout, "do not delete")
	}

	// Also test with backtick injection attempt.
	result, err = sb.Exec(ctx, handle,
		[]string{"echo", "`rm -rf /`"}, 10)
	if err != nil {
		t.Fatalf("backtick exec failed: %v", err)
	}
	got = strings.TrimSpace(result.Stdout)
	if got != "`rm -rf /`" {
		t.Fatalf("backtick stdout = %q, want %q", got, "`rm -rf /`")
	}

	// And $() subshell injection attempt.
	result, err = sb.Exec(ctx, handle,
		[]string{"echo", "$(rm -rf /)"}, 10)
	if err != nil {
		t.Fatalf("subshell exec failed: %v", err)
	}
	got = strings.TrimSpace(result.Stdout)
	if got != "$(rm -rf /)" {
		t.Fatalf("subshell stdout = %q, want %q", got, "$(rm -rf /)")
	}
}

// TestCanary_DiffContainsRelativePaths writes a file in the sandbox, calls
// Diff, and verifies all returned paths are relative (no absolute host paths
// leak through the overlay abstraction).
func TestCanary_DiffContainsRelativePaths(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"base.go": "package base"})
	sb, handle := createSandbox(t, dir)

	ctx := context.Background()

	// Create a new file and modify an existing one.
	sb.Exec(ctx, handle,
		[]string{"bash", "-c", "echo -n 'new file' > added.go"}, 10)
	sb.Exec(ctx, handle,
		[]string{"bash", "-c", "echo -n 'changed' > base.go"}, 10)

	diff, err := sb.Diff(handle)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if len(diff) == 0 {
		t.Fatal("Diff returned no changes, expected at least 2")
	}

	for _, change := range diff {
		// Path must not be absolute.
		if filepath.IsAbs(change.Path) {
			t.Errorf("Diff path is absolute (host path leak): %q", change.Path)
		}
		// Path must not contain the host workspace directory.
		if strings.Contains(change.Path, dir) {
			t.Errorf("Diff path contains host workspace dir: %q (workspace=%q)", change.Path, dir)
		}
		// Path must not start with "/" or contain "..".
		if strings.HasPrefix(change.Path, "/") {
			t.Errorf("Diff path starts with /: %q", change.Path)
		}
		if strings.Contains(change.Path, "..") {
			t.Errorf("Diff path contains ..: %q", change.Path)
		}
	}

	// Verify expected files are present.
	pathSet := make(map[string]bool)
	for _, c := range diff {
		pathSet[c.Path] = true
	}
	if !pathSet["added.go"] {
		t.Errorf("Diff missing added.go: %v", diff)
	}
	if !pathSet["base.go"] {
		t.Errorf("Diff missing base.go: %v", diff)
	}
}

// TestCanary_TwoSandboxesIsolated creates two sandboxes on the same workspace,
// writes different content to the same filename in each, and verifies that
// each sandbox sees only its own content while the host file is untouched.
func TestCanary_TwoSandboxesIsolated(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"shared.txt": "original"})
	sb := New(dir)

	ctx := context.Background()

	// Create two independent sandboxes.
	handleA, err := sb.Create(ctx, "", nil)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}
	defer sb.Destroy(ctx, handleA)

	handleB, err := sb.Create(ctx, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sb.Destroy(ctx, handleB)

	// Write different content to the same file in each sandbox.
	result, err := sb.Exec(ctx, handleA,
		[]string{"bash", "-c", "echo -n 'sandbox-A' > shared.txt"}, 10)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("A write failed: err=%v exit=%d stderr=%q", err, result.ExitCode, result.Stderr)
	}

	result, err = sb.Exec(ctx, handleB,
		[]string{"bash", "-c", "echo -n 'sandbox-B' > shared.txt"}, 10)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("B write failed: err=%v exit=%d stderr=%q", err, result.ExitCode, result.Stderr)
	}

	// Create unique files in each sandbox.
	sb.Exec(ctx, handleA,
		[]string{"bash", "-c", "echo -n 'only-in-A' > a_only.txt"}, 10)
	sb.Exec(ctx, handleB,
		[]string{"bash", "-c", "echo -n 'only-in-B' > b_only.txt"}, 10)

	// Sandbox A sees its own content.
	resultA, err := sb.Exec(ctx, handleA, []string{"cat", "shared.txt"}, 10)
	if err != nil {
		t.Fatalf("A read failed: %v", err)
	}
	if strings.TrimSpace(resultA.Stdout) != "sandbox-A" {
		t.Fatalf("A sees %q, want %q", resultA.Stdout, "sandbox-A")
	}

	// Sandbox B sees its own content.
	resultB, err := sb.Exec(ctx, handleB, []string{"cat", "shared.txt"}, 10)
	if err != nil {
		t.Fatalf("B read failed: %v", err)
	}
	if strings.TrimSpace(resultB.Stdout) != "sandbox-B" {
		t.Fatalf("B sees %q, want %q", resultB.Stdout, "sandbox-B")
	}

	// A does NOT see B's unique file.
	resultA, err = sb.Exec(ctx, handleA, []string{"cat", "b_only.txt"}, 10)
	if err != nil {
		t.Fatalf("A cat b_only.txt unexpected error: %v", err)
	}
	if resultA.ExitCode == 0 {
		t.Fatalf("A can see B's b_only.txt (cross-sandbox leak): %q", resultA.Stdout)
	}

	// B does NOT see A's unique file.
	resultB, err = sb.Exec(ctx, handleB, []string{"cat", "a_only.txt"}, 10)
	if err != nil {
		t.Fatalf("B cat a_only.txt unexpected error: %v", err)
	}
	if resultB.ExitCode == 0 {
		t.Fatalf("B can see A's a_only.txt (cross-sandbox leak): %q", resultB.Stdout)
	}

	// Host file is untouched.
	hostContent, err := os.ReadFile(filepath.Join(dir, "shared.txt"))
	if err != nil {
		t.Fatalf("reading host file: %v", err)
	}
	if string(hostContent) != "original" {
		t.Fatalf("host shared.txt = %q, want %q", hostContent, "original")
	}

	// Host does not have sandbox-only files.
	if _, err := os.Stat(filepath.Join(dir, "a_only.txt")); !os.IsNotExist(err) {
		t.Fatal("a_only.txt leaked to host")
	}
	if _, err := os.Stat(filepath.Join(dir, "b_only.txt")); !os.IsNotExist(err) {
		t.Fatal("b_only.txt leaked to host")
	}
}

// TestCanary_DestroyCleanup creates a sandbox, writes files, destroys it,
// and verifies the overlay mount point is cleaned up and no artifacts remain.
func TestCanary_DestroyCleanup(t *testing.T) {
	dir := setupWorkspace(t, map[string]string{"keep.go": "package keep"})
	sb := New(dir)

	ctx := context.Background()

	handle, err := sb.Create(ctx, "", nil)
	if err != nil {
		skipIfUnsupported(t, err)
		t.Fatal(err)
	}

	// Get the overlay WorkDir before destroy so we can check cleanup.
	space, err := sb.GetSpace(handle)
	if err != nil {
		t.Fatal(err)
	}
	overlayWorkDir := space.WorkDir()

	// Write files through the sandbox.
	result, err := sb.Exec(ctx, handle,
		[]string{"bash", "-c", "echo -n 'sandbox data' > temp.txt"}, 10)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("write failed: err=%v exit=%d", err, result.ExitCode)
	}

	// Verify the overlay directory exists before destroy.
	if _, err := os.Stat(overlayWorkDir); os.IsNotExist(err) {
		t.Fatal("overlay WorkDir does not exist before destroy")
	}

	// Destroy the sandbox.
	if err := sb.Destroy(ctx, handle); err != nil {
		t.Fatalf("destroy failed: %v", err)
	}

	// Overlay mount point must be cleaned up.
	if _, err := os.Stat(overlayWorkDir); !os.IsNotExist(err) {
		t.Fatalf("overlay WorkDir still exists after destroy: %s (err=%v)", overlayWorkDir, err)
	}

	// Host workspace is untouched — original file still there.
	hostContent, err := os.ReadFile(filepath.Join(dir, "keep.go"))
	if err != nil {
		t.Fatalf("host file missing after destroy: %v", err)
	}
	if string(hostContent) != "package keep" {
		t.Fatalf("host file modified: %q", hostContent)
	}

	// Sandbox-only file did not leak to host.
	if _, err := os.Stat(filepath.Join(dir, "temp.txt")); !os.IsNotExist(err) {
		t.Fatal("temp.txt leaked to host after destroy")
	}

	// Exec on destroyed handle should fail.
	_, err = sb.Exec(ctx, handle, []string{"echo", "should fail"}, 10)
	if err == nil {
		t.Fatal("Exec on destroyed handle should error")
	}
}
