//go:build e2e

package claude

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
)

// Spec: DJN-SPC-43 — Provider E2E Smoke Tests

func TestSmoke_Vertex_RoundTrip(t *testing.T) {
	project := os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")
	if project == "" {
		t.Skip("ANTHROPIC_VERTEX_PROJECT_ID not set")
	}

	d, err := NewAPIDriver(driver.DriverConfig{
		Model:     "claude-sonnet-4-6",
		MaxTokens: 256,
	}, WithLogger(djinnlog.Nop()))
	if err != nil {
		t.Fatalf("NewAPIDriver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := d.Start(ctx, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer d.Stop(ctx) //nolint:errcheck // best-effort shutdown

	if err := d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "Reply with exactly: PONG"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	events, err := d.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var text string
	var gotDone bool
	for evt := range events {
		switch evt.Type {
		case driver.EventText:
			text += evt.Text
		case driver.EventDone:
			gotDone = true
		case driver.EventError:
			t.Fatalf("stream error: %s", evt.Error)
		}
	}

	if !gotDone {
		t.Fatal("missing done event")
	}
	if text == "" {
		t.Fatal("empty response")
	}
	t.Logf("Vertex response: %q", text)
}

func TestSmoke_Vertex_ExpiredAuth(t *testing.T) {
	project := os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")
	if project == "" {
		t.Skip("ANTHROPIC_VERTEX_PROJECT_ID not set")
	}
	if os.Getenv("DJINN_TEST_EXPIRED_AUTH") == "" {
		t.Skip("set DJINN_TEST_EXPIRED_AUTH=1 to test expired auth")
	}

	d, err := NewAPIDriver(driver.DriverConfig{
		Model:     "claude-sonnet-4-6",
		MaxTokens: 256,
	}, WithLogger(djinnlog.Nop()))
	if err != nil {
		t.Fatalf("NewAPIDriver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = d.Start(ctx, "")
	if err == nil {
		t.Fatal("expected auth error with expired token")
	}
	t.Logf("auth error: %v", err)
}
