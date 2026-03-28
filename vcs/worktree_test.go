package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repo with one commit for testing.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	return dir
}

func TestWorktreeManager_CreateAndList(t *testing.T) {
	repo := initTestRepo(t)
	mgr := NewWorktreeManager(repo)

	path, err := mgr.Create("TSK-108")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Worktree directory should exist.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("worktree dir not found: %v", err)
	}

	// Should appear in list.
	infos, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("list = %d, want 1", len(infos))
	}
	if infos[0].TaskID != "TSK-108" {
		t.Fatalf("taskID = %q", infos[0].TaskID)
	}
	if infos[0].Branch != "djinn/TSK-108" {
		t.Fatalf("branch = %q", infos[0].Branch)
	}
}

func TestWorktreeManager_Remove(t *testing.T) {
	repo := initTestRepo(t)
	mgr := NewWorktreeManager(repo)

	mgr.Create("TSK-109") //nolint:errcheck // error intentionally ignored

	err := mgr.Remove("TSK-109")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Should not appear in list.
	infos, _ := mgr.List()
	if len(infos) != 0 {
		t.Fatalf("list after remove = %d, want 0", len(infos))
	}

	// Directory should be gone.
	if _, err := os.Stat(mgr.Path("TSK-109")); !os.IsNotExist(err) {
		t.Fatal("worktree dir should be removed")
	}
}

func TestWorktreeManager_ParallelWorktrees(t *testing.T) {
	repo := initTestRepo(t)
	mgr := NewWorktreeManager(repo)

	path1, err := mgr.Create("TSK-110")
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	path2, err := mgr.Create("TSK-111")
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	if path1 == path2 {
		t.Fatal("parallel worktrees should have different paths")
	}

	// Write a file in worktree 1 — should NOT appear in worktree 2.
	os.WriteFile(filepath.Join(path1, "from-110.txt"), []byte("hello"), 0o644) //nolint:errcheck // best-effort write
	if _, err := os.Stat(filepath.Join(path2, "from-110.txt")); !os.IsNotExist(err) {
		t.Fatal("file from worktree 1 should NOT appear in worktree 2")
	}

	// Both should be in list.
	infos, _ := mgr.List()
	if len(infos) != 2 {
		t.Fatalf("list = %d, want 2", len(infos))
	}

	// Cleanup.
	mgr.Remove("TSK-110") //nolint:errcheck // best-effort cleanup
	mgr.Remove("TSK-111") //nolint:errcheck // best-effort cleanup
}

func TestWorktreeManager_Path(t *testing.T) {
	mgr := NewWorktreeManager("/home/user/repo")
	if mgr.Path("TSK-112") != "/home/user/repo/.worktrees/TSK-112" {
		t.Fatalf("path = %q", mgr.Path("TSK-112"))
	}
}

func TestWorktreeManager_Branch(t *testing.T) {
	mgr := NewWorktreeManager("/any")
	if mgr.Branch("TSK-113") != "djinn/TSK-113" {
		t.Fatalf("branch = %q", mgr.Branch("TSK-113"))
	}
}
