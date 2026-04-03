package selfheal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/trace"
	"github.com/dpopsuev/djinn/vcs"
)

func TestGateValidate_ErrorRateImproved(t *testing.T) {
	// Before: 3 errors out of 10.
	beforeRing := trace.NewRing(100)
	for i := range 10 {
		beforeRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Error:     i < 3,
			Latency:   10 * time.Millisecond,
		})
	}
	beforeArchive := trace.Export(beforeRing, "")

	// After: 1 error out of 10.
	afterRing := trace.NewRing(100)
	for i := range 10 {
		afterRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Error:     i < 1,
			Latency:   10 * time.Millisecond,
		})
	}

	result := Validate(beforeArchive, afterRing)
	if result.Verdict != GatePass {
		t.Errorf("verdict = %s, want pass (error rate improved)", result.Verdict)
	}
}

func TestGateValidate_NewErrors(t *testing.T) {
	beforeRing := trace.NewRing(100)
	for range 10 {
		beforeRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Latency:   10 * time.Millisecond,
		})
	}
	beforeArchive := trace.Export(beforeRing, "")

	// After: new errors on a different tool.
	afterRing := trace.NewRing(100)
	for range 5 {
		afterRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "locus",
			Tool:      "codograph.scan",
			Error:     true,
			Latency:   10 * time.Millisecond,
		})
	}

	result := Validate(beforeArchive, afterRing)
	if result.Verdict != GateFail {
		t.Errorf("verdict = %s, want fail (new errors on locus)", result.Verdict)
	}
}

func TestGateValidate_InsufficientData(t *testing.T) {
	beforeRing := trace.NewRing(100)
	beforeRing.Append(trace.TraceEvent{Component: trace.ComponentMCP})
	beforeArchive := trace.Export(beforeRing, "")

	afterRing := trace.NewRing(100)
	afterRing.Append(trace.TraceEvent{Component: trace.ComponentMCP})

	result := Validate(beforeArchive, afterRing)
	if result.Verdict != GateUnsure {
		t.Errorf("verdict = %s, want unsure (too few events)", result.Verdict)
	}
}

func TestOrchestratorCircuitBreaker(t *testing.T) {
	ring := trace.NewRing(100)
	harness := &Harness{} // nil worktrees — will fail but that's OK for this test
	orch := NewOrchestrator(ring, harness)

	// Exhaust the circuit breaker.
	for range MaxAttemptsPerHour {
		orch.mu.Lock()
		orch.attempts = append(orch.attempts, time.Now())
		orch.mu.Unlock()
	}

	if orch.CanAttempt() {
		t.Error("circuit breaker should be tripped after max attempts")
	}
}

func TestOrchestratorCircuitBreakerExpires(t *testing.T) {
	ring := trace.NewRing(100)
	harness := &Harness{}
	orch := NewOrchestrator(ring, harness)

	// Add old attempts (> 1 hour ago).
	oldTime := time.Now().Add(-2 * time.Hour) //nolint:mnd // 2 hours ago
	orch.mu.Lock()
	for range MaxAttemptsPerHour {
		orch.attempts = append(orch.attempts, oldTime)
	}
	orch.mu.Unlock()

	if !orch.CanAttempt() {
		t.Error("circuit breaker should allow attempts after cooldown")
	}
}

func TestHealthAnalyzerConsecutiveErrors(t *testing.T) {
	ring := trace.NewRing(100)
	for range 5 {
		ring.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Error:     true,
		})
	}

	alerts := trace.Analyze(ring, trace.DefaultHealthConfig())
	found := false
	for _, a := range alerts {
		if a.Pattern == "consecutive_errors" && a.Server == "scribe" {
			found = true
			break
		}
	}
	if !found {
		t.Error("should detect 5 consecutive errors on scribe")
	}
}

func TestHealthAnalyzerNoAlerts(t *testing.T) {
	ring := trace.NewRing(100)
	for range 10 {
		ring.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Latency:   10 * time.Millisecond,
		})
	}

	alerts := trace.Analyze(ring, trace.DefaultHealthConfig())
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts for healthy ring, got %d", len(alerts))
	}
}

func TestTraceArchiveDiff(t *testing.T) {
	beforeRing := trace.NewRing(100)
	for range 10 {
		beforeRing.Append(trace.TraceEvent{
			Server:  "scribe",
			Tool:    "artifact.list",
			Latency: 50 * time.Millisecond,
		})
	}
	before := trace.Export(beforeRing, "")

	afterRing := trace.NewRing(100)
	for range 10 {
		afterRing.Append(trace.TraceEvent{
			Server:  "scribe",
			Tool:    "artifact.list",
			Latency: 20 * time.Millisecond,
		})
	}
	after := trace.Export(afterRing, "")

	diff := trace.Diff(before, after)
	if diff.EventCountBefore != 10 || diff.EventCountAfter != 10 {
		t.Errorf("counts = %d/%d, want 10/10", diff.EventCountBefore, diff.EventCountAfter)
	}
	if len(diff.LatencyDeltas) == 0 {
		t.Error("should have latency deltas for scribe")
	}
}

// ---------- Harness tests (worktree-based) ----------

// initTestRepo creates a temporary git repo with an initial commit and returns
// the repo root path and a cleanup function.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup %v: %s: %v", args, out, err)
		}
	}

	// Write a minimal Go module so go build/test succeed in worktrees.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Initial commit so worktree creation has a HEAD.
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git commit %v: %s: %v", args, out, err)
		}
	}

	return dir
}

func TestHarnessRunFix_Success(t *testing.T) {
	repoDir := initTestRepo(t)
	wm := vcs.NewWorktreeManager(repoDir, nil)
	h := NewHarness(wm)

	ctx := context.Background()
	result, err := h.RunFix(ctx, "fix-success", []string{"echo hello"})
	if err != nil {
		t.Fatalf("RunFix error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false; error=%s, output=%s", result.Error, result.Output)
	}
	if result.Duration <= 0 {
		t.Error("expected Duration > 0")
	}
	if result.Output == "" {
		t.Error("expected non-empty Output")
	}
	if result.FixID != "fix-success" {
		t.Errorf("FixID = %q, want %q", result.FixID, "fix-success")
	}

	// Cleanup.
	_ = h.Cleanup("fix-success")
}

func TestHarnessRunFix_InstructionFails(t *testing.T) {
	repoDir := initTestRepo(t)
	wm := vcs.NewWorktreeManager(repoDir, nil)
	h := NewHarness(wm)

	ctx := context.Background()
	result, err := h.RunFix(ctx, "fix-fail-instr", []string{"false"})
	if err != nil {
		t.Fatalf("RunFix error: %v", err)
	}

	if result.Success {
		t.Error("expected Success=false for failing instruction")
	}
	if result.Error == "" {
		t.Error("expected non-empty Error when instruction fails")
	}

	// Verify the worktree was cleaned up (Remove was called internally).
	wtPath := wm.Path("fix-fail-instr")
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree dir %s should have been removed, stat err=%v", wtPath, err)
	}
}

func TestHarnessRunFix_CreateError(t *testing.T) {
	// Use a non-existent repo root so Create fails.
	wm := vcs.NewWorktreeManager("/nonexistent/repo/path", nil)
	h := NewHarness(wm)

	ctx := context.Background()
	_, err := h.RunFix(ctx, "fix-create-err", []string{"echo hello"})
	if err == nil {
		t.Fatal("expected error from RunFix when Create fails")
	}
}

func TestHarnessCleanup(t *testing.T) {
	repoDir := initTestRepo(t)
	wm := vcs.NewWorktreeManager(repoDir, nil)
	h := NewHarness(wm)

	// First create a worktree so there's something to clean up.
	ctx := context.Background()
	_, err := h.RunFix(ctx, "fix-cleanup", []string{"echo hello"})
	if err != nil {
		t.Fatalf("RunFix error: %v", err)
	}

	// Now cleanup.
	if err := h.Cleanup("fix-cleanup"); err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}

	// Verify it's gone.
	wtPath := wm.Path("fix-cleanup")
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree dir %s should have been removed after Cleanup", wtPath)
	}
}

func TestHarnessCleanup_Error(t *testing.T) {
	// Create a temp dir that looks like a git repo root but has no actual git
	// metadata, and pre-create the worktree directory so os.Stat succeeds
	// (which causes Remove to return the git error instead of silently pruning).
	fakeRepo := t.TempDir()
	wm := vcs.NewWorktreeManager(fakeRepo, nil)

	// Create the directory that Remove will stat-check.
	wtPath := wm.Path("bad-fix")
	if err := os.MkdirAll(wtPath, 0o755); err != nil {
		t.Fatal(err)
	}

	h := NewHarness(wm)
	err := h.Cleanup("bad-fix")
	if err == nil {
		t.Error("expected error from Cleanup when Remove fails on non-git dir")
	}
}

// ---------- Gate edge case tests ----------

func TestGateValidate_ErrorRateIncreased(t *testing.T) {
	// Before: 1 error out of 10 events.
	beforeRing := trace.NewRing(100)
	for i := range 10 {
		beforeRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Error:     i < 1,
			Latency:   10 * time.Millisecond,
		})
	}
	beforeArchive := trace.Export(beforeRing, "")

	// After: 5 errors out of 10 events, same server+tool so no "new errors".
	afterRing := trace.NewRing(100)
	for i := range 10 {
		afterRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Error:     i < 5,
			Latency:   10 * time.Millisecond,
		})
	}

	result := Validate(beforeArchive, afterRing)
	if result.Verdict != GateFail {
		t.Errorf("verdict = %s, want fail (error rate increased)", result.Verdict)
	}
}

func TestGateValidate_ZeroErrorsBoth(t *testing.T) {
	// Before: no errors.
	beforeRing := trace.NewRing(100)
	for range 10 {
		beforeRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Latency:   10 * time.Millisecond,
		})
	}
	beforeArchive := trace.Export(beforeRing, "")

	// After: no errors.
	afterRing := trace.NewRing(100)
	for range 10 {
		afterRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Latency:   10 * time.Millisecond,
		})
	}

	result := Validate(beforeArchive, afterRing)
	if result.Verdict != GatePass {
		t.Errorf("verdict = %s, want pass (zero errors both before and after)", result.Verdict)
	}
}

func TestGateValidate_ExactBoundary(t *testing.T) {
	// Before: empty.
	beforeRing := trace.NewRing(100)
	beforeArchive := trace.Export(beforeRing, "")

	// After: exactly 5 events (the minimum threshold).
	afterRing := trace.NewRing(100)
	for range 5 {
		afterRing.Append(trace.TraceEvent{
			Component: trace.ComponentMCP,
			Action:    "call" + trace.ActionDoneSuffix,
			Server:    "scribe",
			Tool:      "artifact.list",
			Latency:   10 * time.Millisecond,
		})
	}

	result := Validate(beforeArchive, afterRing)
	// With exactly 5 events (meets threshold), 0 errors after, 0 errors before
	// → error rate unchanged (0 == 0), no new errors → GatePass "no regression detected".
	if result.Verdict != GatePass {
		t.Errorf("verdict = %s, want pass (exactly 5 events, no errors)", result.Verdict)
	}
}

// ---------- Orchestrator.Attempt tests ----------

func TestOrchestratorAttempt_CircuitOpen(t *testing.T) {
	ring := trace.NewRing(100)
	repoDir := initTestRepo(t)
	wm := vcs.NewWorktreeManager(repoDir, nil)
	h := NewHarness(wm)
	orch := NewOrchestrator(ring, h)

	// Fill up circuit breaker with MaxAttemptsPerHour recent attempts.
	orch.mu.Lock()
	for range MaxAttemptsPerHour {
		orch.attempts = append(orch.attempts, time.Now())
	}
	orch.mu.Unlock()

	ctx := context.Background()
	_, err := orch.Attempt(ctx, "test-trigger", []string{"echo fix"})
	if !errors.Is(err, ErrCircuitBreaker) {
		t.Errorf("expected ErrCircuitBreaker, got %v", err)
	}
}

func TestOrchestratorAttempt_BuildFails(t *testing.T) {
	ring := trace.NewRing(100)
	repoDir := initTestRepo(t)

	// Overwrite the Go files with invalid Go code so build fails in the worktree.
	// We'll use instructions that corrupt the Go code.
	wm := vcs.NewWorktreeManager(repoDir, nil)
	h := NewHarness(wm)
	orch := NewOrchestrator(ring, h)

	ctx := context.Background()
	// The instruction writes invalid Go, causing go build to fail.
	record, err := orch.Attempt(ctx, "build-fail-trigger", []string{
		"echo 'invalid go code !!!' > main.go",
	})
	if err != nil {
		t.Fatalf("Attempt returned error: %v", err)
	}

	if record.BuildResult == nil {
		t.Fatal("expected non-nil BuildResult")
	}
	if record.BuildResult.Success {
		t.Error("expected BuildResult.Success=false when go build fails")
	}
	if record.GateResult != nil {
		t.Error("expected GateResult to be nil when build fails")
	}
	if record.Trigger != "build-fail-trigger" {
		t.Errorf("Trigger = %q, want %q", record.Trigger, "build-fail-trigger")
	}
}

func TestOrchestratorHistory(t *testing.T) {
	ring := trace.NewRing(100)
	repoDir := initTestRepo(t)
	wm := vcs.NewWorktreeManager(repoDir, nil)
	h := NewHarness(wm)
	orch := NewOrchestrator(ring, h)

	ctx := context.Background()

	// Run 2 attempts. Both will succeed (instructions are benign echo commands).
	for i := range 2 {
		fixInstr := []string{"echo attempt-" + itoa(i)}
		_, err := orch.Attempt(ctx, "trigger-"+itoa(i), fixInstr)
		if err != nil {
			t.Fatalf("Attempt %d error: %v", i, err)
		}
	}

	history := orch.History()
	if len(history) != 2 {
		t.Fatalf("History() returned %d records, want 2", len(history))
	}

	// Verify returned slice is a copy — mutating it doesn't affect internal state.
	history[0].Trigger = "MUTATED"
	fresh := orch.History()
	if fresh[0].Trigger == "MUTATED" {
		t.Error("History() returned internal slice, not a copy")
	}
}

// itoa is a simple int-to-string helper for test code.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
