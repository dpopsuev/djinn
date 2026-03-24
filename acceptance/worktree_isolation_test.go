// worktree_isolation_test.go — E2E tests for git worktree task isolation.
//
// Verifies that executor tasks work in isolated worktrees:
// - Changes in worktree don't affect main repo
// - Parallel worktrees don't conflict
// - Cleanup removes worktree and branch
package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpopsuev/djinn/vcs"
)

func initAcceptanceRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	// Create a file and commit so we have a HEAD.
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644) //nolint:errcheck
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run() //nolint:errcheck
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	cmd.Run() //nolint:errcheck
	return dir
}

func TestWorktree_CreateWriteVerifyMainUnaffected(t *testing.T) {
	repo := initAcceptanceRepo(t)
	mgr := vcs.NewWorktreeManager(repo)

	// Create worktree for a task.
	wtPath, err := mgr.Create("TSK-200")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer mgr.Remove("TSK-200") //nolint:errcheck

	// Write a new file in the worktree.
	os.WriteFile(filepath.Join(wtPath, "new-feature.go"), []byte("package feature\n"), 0644) //nolint:errcheck

	// Verify the file exists in the worktree.
	if _, err := os.Stat(filepath.Join(wtPath, "new-feature.go")); err != nil {
		t.Fatal("file should exist in worktree")
	}

	// Verify the file does NOT exist in the main repo.
	if _, err := os.Stat(filepath.Join(repo, "new-feature.go")); !os.IsNotExist(err) {
		t.Fatal("file should NOT exist in main repo — worktree isolation failed")
	}

	// Verify main.go from the initial commit exists in BOTH.
	if _, err := os.Stat(filepath.Join(wtPath, "main.go")); err != nil {
		t.Fatal("main.go should exist in worktree (inherited from HEAD)")
	}
	if _, err := os.Stat(filepath.Join(repo, "main.go")); err != nil {
		t.Fatal("main.go should exist in main repo")
	}
}

func TestWorktree_ParallelWorktreesDontConflict(t *testing.T) {
	repo := initAcceptanceRepo(t)
	mgr := vcs.NewWorktreeManager(repo)

	wt1, err := mgr.Create("TSK-201")
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	defer mgr.Remove("TSK-201") //nolint:errcheck

	wt2, err := mgr.Create("TSK-202")
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	defer mgr.Remove("TSK-202") //nolint:errcheck

	// Write different files in each worktree.
	os.WriteFile(filepath.Join(wt1, "from-201.txt"), []byte("hello from 201"), 0644) //nolint:errcheck
	os.WriteFile(filepath.Join(wt2, "from-202.txt"), []byte("hello from 202"), 0644) //nolint:errcheck

	// Each worktree should only see its own file.
	if _, err := os.Stat(filepath.Join(wt1, "from-202.txt")); !os.IsNotExist(err) {
		t.Fatal("worktree 1 should NOT see file from worktree 2")
	}
	if _, err := os.Stat(filepath.Join(wt2, "from-201.txt")); !os.IsNotExist(err) {
		t.Fatal("worktree 2 should NOT see file from worktree 1")
	}

	// Main repo should see neither.
	if _, err := os.Stat(filepath.Join(repo, "from-201.txt")); !os.IsNotExist(err) {
		t.Fatal("main repo should NOT see worktree 1 file")
	}
}

func TestWorktree_CleanupOnRemove(t *testing.T) {
	repo := initAcceptanceRepo(t)
	mgr := vcs.NewWorktreeManager(repo)

	wtPath, err := mgr.Create("TSK-203")
	if err != nil {
		t.Fatal(err)
	}

	// Verify exists.
	if _, err := os.Stat(wtPath); err != nil {
		t.Fatal("worktree should exist after create")
	}

	// Remove.
	if err := mgr.Remove("TSK-203"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Directory should be gone.
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Fatal("worktree dir should be removed after cleanup")
	}

	// Branch should be gone.
	cmd := exec.Command("git", "branch", "--list", "djinn/TSK-203")
	cmd.Dir = repo
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) != "" {
		t.Fatal("branch should be deleted after cleanup")
	}
}
