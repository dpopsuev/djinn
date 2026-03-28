//go:build e2e

package claude

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/driver"
)

// Spec: DJN-SPC-43 — Provider E2E Smoke Tests

func TestSmoke_ClaudeDirect_RoundTrip(t *testing.T) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
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
	t.Logf("response: %q", text)
}

func TestSmoke_ClaudeDirect_AuthFailure(t *testing.T) {
	// Temporarily set invalid key.
	orig := os.Getenv("ANTHROPIC_API_KEY")
	t.Setenv("ANTHROPIC_API_KEY", "sk-invalid-key-for-testing")
	defer func() {
		if orig != "" {
			os.Setenv("ANTHROPIC_API_KEY", orig)
		}
	}()

	d, err := NewAPIDriver(driver.DriverConfig{
		Model:     "claude-sonnet-4-6",
		MaxTokens: 256,
	}, WithLogger(djinnlog.Nop()))
	if err != nil {
		t.Fatalf("NewAPIDriver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := d.Start(ctx, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer d.Stop(ctx) //nolint:errcheck // best-effort shutdown

	if err := d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "hi"}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	_, err = d.Chat(ctx)
	if err == nil {
		t.Fatal("expected auth error")
	}

	var de *driver.DriverError
	if !errors.As(err, &de) {
		t.Fatalf("expected DriverError, got %T: %v", err, err)
	}
	if de.Retryable {
		t.Fatal("auth error should not be retryable")
	}
	t.Logf("auth error: %v", de)
}
