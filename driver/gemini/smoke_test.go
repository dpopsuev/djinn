//go:build e2e

// Spec: DJN-SPC-43 — Provider E2E Smoke Tests
package gemini

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func requireGemini(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gemini"); err != nil {
		t.Skip("gemini CLI not found in PATH")
	}
}

func TestSmoke_Gemini_RoundTrip(t *testing.T) {
	requireGemini(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gemini", "-p", "Reply with exactly: PONG")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gemini -p: %v\noutput: %s", err, string(out))
	}

	text := string(out)
	if !strings.Contains(text, "PONG") {
		t.Fatalf("response missing PONG:\n%s", text)
	}
	t.Logf("response: %q", text)
}

func TestSmoke_Gemini_EmptyPrompt(t *testing.T) {
	requireGemini(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gemini", "-p", "")
	out, err := cmd.CombinedOutput()
	// Empty prompt may error or return empty — either is acceptable.
	t.Logf("empty prompt: err=%v output=%q", err, string(out))
}
