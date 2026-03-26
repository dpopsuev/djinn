// git.go — structured Git operations.
//
// GitRepo wraps git CLI commands with structured output parsing.
// Every method uses exec.CommandContext for timeout-safe execution.
// Thread-safe: a mutex serialises git calls on the same repo.
package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GitRepo provides structured access to a git repository.
type GitRepo struct {
	workDir string
	mu      sync.Mutex
}

// GitStatus describes the working tree state.
type GitStatus struct {
	Branch    string
	Clean     bool
	Staged    []FileChange
	Unstaged  []FileChange
	Untracked []string
}

// FileChange records a path and its modification type.
type FileChange struct {
	Path   string
	Status string // modified, added, deleted, renamed, copied
}

// Commit records a single git log entry.
type Commit struct {
	Hash    string
	Author  string
	Date    time.Time
	Subject string
}

// NewGitRepo creates a GitRepo rooted at workDir.
func NewGitRepo(workDir string) *GitRepo {
	return &GitRepo{workDir: workDir}
}

// Status returns the structured status of the working tree.
func (g *GitRepo) Status(ctx context.Context) (*GitStatus, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	branch, _ := g.currentBranch(ctx) //nolint:errcheck

	out, err := g.run(ctx, "status", "--porcelain=v1", "-b")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	status := &GitStatus{Branch: branch, Clean: true}

	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		// The branch line starts with ##
		if strings.HasPrefix(line, "##") {
			continue
		}

		status.Clean = false

		x := line[0] // index (staged) indicator
		y := line[1] // worktree (unstaged) indicator
		path := strings.TrimSpace(line[3:])

		// Staged changes (index column).
		if x != ' ' && x != '?' {
			status.Staged = append(status.Staged, FileChange{
				Path:   path,
				Status: statusLetter(x),
			})
		}

		// Unstaged changes (worktree column).
		if y != ' ' && y != '?' {
			status.Unstaged = append(status.Unstaged, FileChange{
				Path:   path,
				Status: statusLetter(y),
			})
		}

		// Untracked.
		if x == '?' && y == '?' {
			status.Untracked = append(status.Untracked, path)
		}
	}

	return status, nil
}

// Diff returns the working-tree diff (unstaged changes).
func (g *GitRepo) Diff(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	out, err := g.run(ctx, "diff")
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return out, nil
}

// Log returns the last n commits.
func (g *GitRepo) Log(ctx context.Context, n int) ([]Commit, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Format: hash<TAB>author<TAB>unix-timestamp<TAB>subject
	format := "%H\t%an\t%at\t%s"
	out, err := g.run(ctx, "log", fmt.Sprintf("-%d", n), fmt.Sprintf("--format=%s", format))
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}

		epoch, _ := strconv.ParseInt(parts[2], 10, 64) //nolint:errcheck
		commits = append(commits, Commit{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    time.Unix(epoch, 0),
			Subject: parts[3],
		})
	}

	return commits, nil
}

// CurrentBranch returns the name of the currently checked-out branch.
func (g *GitRepo) CurrentBranch(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.currentBranch(ctx)
}

// currentBranch is the unlocked implementation.
func (g *GitRepo) currentBranch(ctx context.Context) (string, error) {
	out, err := g.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// run executes a git command in the repo's working directory.
func (g *GitRepo) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// statusLetter converts a porcelain status letter to a human-readable string.
func statusLetter(c byte) string {
	switch c {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'U':
		return "unmerged"
	default:
		return string(c)
	}
}
