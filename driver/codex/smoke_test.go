//go:build e2e

// Spec: DJN-SPC-43 — Provider E2E Smoke Tests
package codex

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func requireCodex(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex CLI not found in PATH")
	}
}

func TestSmoke_Codex_RoundTrip(t *testing.T) {
	requireCodex(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "codex", "-q", "Reply with exactly: PONG")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("codex -q: %v\noutput: %s", err, string(out))
	}

	text := string(out)
	if !strings.Contains(text, "PONG") {
		t.Fatalf("response missing PONG:\n%s", text)
	}
	t.Logf("response: %q", text)
}

func TestSmoke_Codex_EmptyPrompt(t *testing.T) {
	requireCodex(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "codex", "-q", "")
	out, err := cmd.CombinedOutput()
	t.Logf("empty prompt: err=%v output=%q", err, string(out))
}
