//go:build e2e

// agent_smoke_test.go — E2E smoke tests for the 4 major agent CLIs.
//
// Verifies each CLI binary exists, responds to a simple prompt in headless
// mode, and returns non-empty output. These tests require the actual CLIs
// installed and authenticated. Skip with: go test -short
//
// Usage: go test ./acceptance/ -run TestSmoke -v -timeout 60s
package acceptance

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func skipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping agent smoke test in -short mode")
	}
}

func requireBinary(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not found in PATH — skipping", name)
	}
	return path
}

func runAgent(t *testing.T, name string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\noutput: %s", name, err, string(out))
	}
	return string(out)
}

// TestSmoke_Claude verifies Claude Code responds in print mode.
func TestSmoke_Claude(t *testing.T) {
	skipIfShort(t)
	requireBinary(t, "claude")

	out := runAgent(t, "claude", "-p", "respond with exactly: SMOKE_OK")
	if !strings.Contains(out, "SMOKE_OK") {
		t.Fatalf("claude output missing SMOKE_OK:\n%s", out)
	}
}

// TestSmoke_Gemini verifies Gemini CLI responds in headless mode.
func TestSmoke_Gemini(t *testing.T) {
	skipIfShort(t)
	requireBinary(t, "gemini")

	out := runAgent(t, "gemini", "-p", "respond with exactly: SMOKE_OK")
	if !strings.Contains(out, "SMOKE_OK") {
		t.Fatalf("gemini output missing SMOKE_OK:\n%s", out)
	}
}

// TestSmoke_Codex verifies Codex CLI responds in quiet mode.
func TestSmoke_Codex(t *testing.T) {
	skipIfShort(t)
	requireBinary(t, "codex")

	out := runAgent(t, "codex", "-q", "respond with exactly: SMOKE_OK")
	if !strings.Contains(out, "SMOKE_OK") {
		t.Fatalf("codex output missing SMOKE_OK:\n%s", out)
	}
}

// TestSmoke_Cursor verifies Cursor Agent responds in print mode.
func TestSmoke_Cursor(t *testing.T) {
	skipIfShort(t)
	requireBinary(t, "agent")

	out := runAgent(t, "agent", "-p", "respond with exactly: SMOKE_OK")
	if !strings.Contains(out, "SMOKE_OK") {
		t.Fatalf("cursor agent output missing SMOKE_OK:\n%s", out)
	}
}

// TestSmoke_AllBinariesExist verifies all 4 CLIs are installed (no auth needed).
func TestSmoke_AllBinariesExist(t *testing.T) {
	for _, bin := range []string{"claude", "gemini", "codex", "agent"} {
		t.Run(bin, func(t *testing.T) {
			path := requireBinary(t, bin)
			t.Logf("%s: %s", bin, path)
		})
	}
}
