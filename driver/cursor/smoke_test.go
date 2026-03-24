//go:build e2e

// Spec: DJN-SPC-43 — Provider E2E Smoke Tests
package cursor

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func requireCursor(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("agent"); err != nil {
		t.Skip("cursor agent CLI not found in PATH")
	}
}

func TestSmoke_Cursor_RoundTrip(t *testing.T) {
	requireCursor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "agent", "-p", "Reply with exactly: PONG")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent -p: %v\noutput: %s", err, string(out))
	}

	text := string(out)
	if !strings.Contains(text, "PONG") {
		t.Fatalf("response missing PONG:\n%s", text)
	}
	t.Logf("response: %q", text)
}

func TestSmoke_Cursor_EmptyPrompt(t *testing.T) {
	requireCursor(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "agent", "-p", "")
	out, err := cmd.CombinedOutput()
	t.Logf("empty prompt: err=%v output=%q", err, string(out))
}
