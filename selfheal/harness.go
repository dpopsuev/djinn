// harness.go — ExecutorHarness: worktree → build → test → result (TSK-489).
//
// Orchestrates a fix attempt in an isolated worktree: create worktree,
// apply fix instructions, build, test, collect result. Clean up on failure.
package selfheal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dpopsuev/djinn/vcs"
)

// BuildResult captures the outcome of a fix attempt.
type BuildResult struct {
	Success  bool          `json:"success"`
	FixID    string        `json:"fix_id"`
	Output   string        `json:"output"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// Harness manages fix execution in worktrees.
type Harness struct {
	worktrees *vcs.WorktreeManager
}

// NewHarness creates an executor harness with the given worktree manager.
func NewHarness(wm *vcs.WorktreeManager) *Harness {
	return &Harness{worktrees: wm}
}

// RunFix executes a fix in an isolated worktree.
// Instructions are shell commands to apply the fix (e.g., "sed -i 's/old/new/' file.go").
func (h *Harness) RunFix(ctx context.Context, fixID string, instructions []string) (*BuildResult, error) {
	start := time.Now()

	// Create isolated worktree.
	wtPath, err := h.worktrees.Create(fixID)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	// Apply fix instructions.
	for _, instr := range instructions {
		if err := runInDir(ctx, wtPath, "sh", "-c", instr); err != nil {
			_ = h.worktrees.Remove(fixID)
			return &BuildResult{
				FixID:    fixID,
				Duration: time.Since(start),
				Error:    "apply failed: " + err.Error(),
			}, nil
		}
	}

	// Build.
	buildOut, buildErr := runOutput(ctx, wtPath, "go", "build", "./...")
	if buildErr != nil {
		_ = h.worktrees.Remove(fixID)
		return &BuildResult{
			FixID:    fixID,
			Output:   buildOut,
			Duration: time.Since(start),
			Error:    "build failed: " + buildErr.Error(),
		}, nil
	}

	// Test.
	testOut, testErr := runOutput(ctx, wtPath, "go", "test", "./...", "-count=1")
	if testErr != nil {
		_ = h.worktrees.Remove(fixID)
		return &BuildResult{
			FixID:    fixID,
			Output:   testOut,
			Duration: time.Since(start),
			Error:    "test failed: " + testErr.Error(),
		}, nil
	}

	return &BuildResult{
		Success:  true,
		FixID:    fixID,
		Output:   buildOut + "\n" + testOut,
		Duration: time.Since(start),
	}, nil
}

// Cleanup removes the worktree for a fix.
func (h *Harness) Cleanup(fixID string) error {
	return h.worktrees.Remove(fixID)
}

func runInDir(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // instructions from trusted GenSec
	cmd.Dir = dir
	return cmd.Run()
}

func runOutput(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // build/test commands
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
