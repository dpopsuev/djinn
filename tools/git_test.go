package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repository for testing.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s %v", args, out, err)
		}
	}
	return dir
}

func TestGitRepo_CurrentBranch(t *testing.T) {
	dir := initTestRepo(t)
	repo := NewGitRepo(dir)

	branch, err := repo.CurrentBranch(context.Background())
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	// git init creates "main" or "master" depending on git version.
	if branch != "main" && branch != "master" {
		t.Fatalf("branch = %q, want main or master", branch)
	}
}

func TestGitRepo_StatusClean(t *testing.T) {
	dir := initTestRepo(t)
	repo := NewGitRepo(dir)

	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Clean {
		t.Fatal("expected clean repo")
	}
	if len(status.Staged) != 0 {
		t.Fatalf("Staged = %d, want 0", len(status.Staged))
	}
	if len(status.Unstaged) != 0 {
		t.Fatalf("Unstaged = %d, want 0", len(status.Unstaged))
	}
}

func TestGitRepo_StatusWithUntrackedFile(t *testing.T) {
	dir := initTestRepo(t)
	repo := NewGitRepo(dir)

	// Create an untracked file.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Clean {
		t.Fatal("expected dirty repo")
	}
	if len(status.Untracked) != 1 || status.Untracked[0] != "new.txt" {
		t.Fatalf("Untracked = %v, want [new.txt]", status.Untracked)
	}
}

func TestGitRepo_StatusWithStagedFile(t *testing.T) {
	dir := initTestRepo(t)
	repo := NewGitRepo(dir)

	// Create and stage a file.
	path := filepath.Join(dir, "staged.go")
	if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cmd := exec.Command("git", "add", "staged.go")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %s %v", out, err)
	}

	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Clean {
		t.Fatal("expected dirty repo")
	}
	if len(status.Staged) != 1 {
		t.Fatalf("Staged = %d, want 1", len(status.Staged))
	}
	if status.Staged[0].Status != "added" {
		t.Fatalf("Staged[0].Status = %q, want added", status.Staged[0].Status)
	}
}

func TestGitRepo_Diff(t *testing.T) {
	dir := initTestRepo(t)
	repo := NewGitRepo(dir)

	// Create, commit, then modify a file.
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	for _, args := range [][]string{
		{"git", "add", "hello.txt"},
		{"git", "commit", "-m", "add hello"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s %v", args, out, err)
		}
	}

	// Modify.
	if err := os.WriteFile(path, []byte("hello world\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	diff, err := repo.Diff(context.Background())
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if diff == "" {
		t.Fatal("expected non-empty diff")
	}
}

func TestGitRepo_Log(t *testing.T) {
	dir := initTestRepo(t)
	repo := NewGitRepo(dir)

	// Add a second commit.
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	for _, args := range [][]string{
		{"git", "add", "f.txt"},
		{"git", "commit", "-m", "second commit"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s %v", args, out, err)
		}
	}

	commits, err := repo.Log(context.Background(), 5)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("len = %d, want 2", len(commits))
	}
	if commits[0].Subject != "second commit" {
		t.Fatalf("commits[0].Subject = %q, want 'second commit'", commits[0].Subject)
	}
	if commits[0].Hash == "" {
		t.Fatal("commit hash should not be empty")
	}
	if commits[0].Author != "Test" {
		t.Fatalf("Author = %q, want Test", commits[0].Author)
	}
}

func TestGitRepo_StatusBranchPopulated(t *testing.T) {
	dir := initTestRepo(t)
	repo := NewGitRepo(dir)

	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Branch == "" {
		t.Fatal("Branch should be populated in status")
	}
}

func TestStatusLetter(t *testing.T) {
	tests := []struct {
		in   byte
		want string
	}{
		{'M', "modified"},
		{'A', "added"},
		{'D', "deleted"},
		{'R', "renamed"},
		{'C', "copied"},
		{'U', "unmerged"},
		{'X', "X"},
	}
	for _, tc := range tests {
		got := statusLetter(tc.in)
		if got != tc.want {
			t.Errorf("statusLetter(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
