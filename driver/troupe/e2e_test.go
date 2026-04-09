//go:build e2e

package troupe

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/troupe/execution"
)

const (
	envDjinnProvider = "DJINN_PROVIDER"
	envDjinnModel    = "DJINN_MODEL"
	defaultModel     = "claude-sonnet-4-6"
)

// TestTroupeDriver_Smoke_RealAPI verifies the adapter works end-to-end
// with a real LLM provider. Provider-agnostic — set DJINN_PROVIDER and
// the corresponding API key. Troupe handles the rest.
//
// Run: DJINN_PROVIDER=claude go test -tags e2e -run TestTroupeDriver_Smoke_RealAPI -v -timeout 60s ./driver/troupe/
func TestTroupeDriver_Smoke_RealAPI(t *testing.T) {
	if os.Getenv(envDjinnProvider) == "" {
		t.Skipf("%s not set — skipping real LLM test", envDjinnProvider)
	}

	provider, err := execution.NewProviderFromEnv(envDjinnProvider)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	model := os.Getenv(envDjinnModel)
	if model == "" {
		model = defaultModel
	}

	d := New(provider, model)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := d.Start(ctx, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer d.Stop(ctx) //nolint:errcheck // test cleanup

	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "Reply with exactly: PONG"}) //nolint:errcheck // test setup

	ch, err := d.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var gotText, gotDone bool
	var text string
	for evt := range ch {
		switch evt.Type {
		case driver.EventText:
			gotText = true
			text += evt.Text
			t.Logf("EventText: %q", evt.Text)
		case driver.EventDone:
			gotDone = true
			if evt.Usage != nil {
				t.Logf("Usage: input=%d output=%d", evt.Usage.InputTokens, evt.Usage.OutputTokens)
			}
		case driver.EventError:
			t.Fatalf("EventError: %s", evt.Error)
		case driver.EventThinking:
			t.Logf("EventThinking: %q", evt.Thinking)
		}
	}

	if !gotText {
		t.Fatal("no EventText received")
	}
	if !gotDone {
		t.Fatal("no EventDone received")
	}
	t.Logf("Full response: %q", text)
}
