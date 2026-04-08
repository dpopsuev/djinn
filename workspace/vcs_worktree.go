// Package vcs provides version control operations for task isolation.
//
// WorktreeManager creates git worktrees so each executor task works
// on its own branch in its own directory. No conflicts between
// parallel executors. Clean diffs for inspector review. Gate runs
// scoped to the task's changes only.
//
// Worktree location: <repoRoot>/.worktrees/<taskID>/
// Branch naming: djinn/<taskID>
package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

const (
	worktreeDir    = ".worktrees"
	branchPrefix   = "djinn/"
	defaultTimeout = 30 * time.Second
)

// WorktreeInfo describes an active worktree.
type WorktreeInfo struct {
	TaskID string
	Path   string
	Branch string
}

// WorktreeManager manages git worktrees for executor task isolation.
type WorktreeManager struct {
	repoRoot string
	log      *slog.Logger
}

// NewWorktreeManager creates a manager for the given repo root.
// Pass nil for log to discard all output.
func NewWorktreeManager(repoRoot string, log *slog.Logger) *WorktreeManager {
	if log == nil {
		log = telemetry.Nop()
	}
	return &WorktreeManager{repoRoot: repoRoot, log: log}
}

// Create makes a new worktree and branch for a task.
// Returns the absolute path to the worktree directory.
func (m *WorktreeManager) Create(taskID string) (string, error) {
	wtPath := m.Path(taskID)
	branch := m.Branch(taskID)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	start := time.Now()

	// Create the worktree with a new branch from current HEAD.
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", wtPath, "-b", branch)
	cmd.Dir = m.repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Orange: git command failure
		m.log.WarnContext(ctx, "worktree create failed",
			slog.String(telemetry.KeyAction, "create"),
			slog.String(telemetry.KeyPath, wtPath),
			slog.String(telemetry.KeyError, strings.TrimSpace(string(out))),
		)
		return "", fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Yellow: worktree created
	m.log.InfoContext(ctx, "worktree created",
		slog.String(telemetry.KeyAction, "create"),
		slog.String(telemetry.KeyPath, wtPath),
		slog.String(telemetry.KeyBranch, branch),
		slog.String(telemetry.KeyTaskID, taskID),
		slog.Duration(telemetry.KeyDuration, time.Since(start)),
	)

	return wtPath, nil
}

// Remove deletes a worktree and its branch.
func (m *WorktreeManager) Remove(taskID string) error {
	wtPath := m.Path(taskID)
	branch := m.Branch(taskID)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Remove the worktree.
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", wtPath, "--force")
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		// If worktree dir doesn't exist, try to prune stale entries.
		if _, statErr := os.Stat(wtPath); os.IsNotExist(statErr) {
			// Orange: stale worktree detected
			m.log.WarnContext(ctx, "stale worktree detected",
				slog.String(telemetry.KeyAction, "prune"),
				slog.String(telemetry.KeyPath, wtPath),
				slog.String(telemetry.KeyReason, "directory does not exist"),
			)
			exec.CommandContext(ctx, "git", "worktree", "prune").Run() //nolint:errcheck // test helper, error checked elsewhere
		} else {
			// Orange: git command failure
			m.log.WarnContext(ctx, "worktree remove failed",
				slog.String(telemetry.KeyAction, "remove"),
				slog.String(telemetry.KeyPath, wtPath),
				slog.String(telemetry.KeyError, strings.TrimSpace(string(out))),
			)
			return fmt.Errorf("git worktree remove: %s: %w", strings.TrimSpace(string(out)), err)
		}
	}

	// Delete the branch.
	branchCmd := exec.CommandContext(ctx, "git", "branch", "-D", branch)
	branchCmd.Dir = m.repoRoot
	branchCmd.Run() //nolint:errcheck // branch may already be gone

	// Yellow: worktree removed
	m.log.InfoContext(ctx, "worktree removed",
		slog.String(telemetry.KeyAction, "remove"),
		slog.String(telemetry.KeyPath, wtPath),
	)

	return nil
}

// List returns all active djinn worktrees.
func (m *WorktreeManager) List() ([]WorktreeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	start := time.Now()

	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = m.repoRoot
	out, err := cmd.Output()
	if err != nil {
		// Orange: git command failure
		m.log.WarnContext(ctx, "worktree list failed",
			slog.String(telemetry.KeyAction, "list"),
			slog.String(telemetry.KeyError, err.Error()),
		)
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var infos []WorktreeInfo
	var currentPath, currentBranch string

	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			currentPath = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			currentBranch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "" && currentPath != "":
			// Only include djinn-managed worktrees.
			if strings.HasPrefix(currentBranch, branchPrefix) {
				taskID := strings.TrimPrefix(currentBranch, branchPrefix)
				infos = append(infos, WorktreeInfo{
					TaskID: taskID,
					Path:   currentPath,
					Branch: currentBranch,
				})
			}
			currentPath = ""
			currentBranch = ""
		}
	}

	// Yellow: worktree listed
	m.log.DebugContext(ctx, "worktrees listed",
		slog.String(telemetry.KeyAction, "list"),
		slog.Int(telemetry.KeyCount, len(infos)),
		slog.Duration(telemetry.KeyDuration, time.Since(start)),
	)

	return infos, nil
}

// Path returns the filesystem path for a task's worktree.
func (m *WorktreeManager) Path(taskID string) string {
	return filepath.Join(m.repoRoot, worktreeDir, taskID)
}

// Branch returns the git branch name for a task.
func (m *WorktreeManager) Branch(taskID string) string {
	return branchPrefix + taskID
}
