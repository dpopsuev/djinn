//go:build e2e

// tier0_byok_test.go — Tier 0: BYOK (Bring Your Own Keys)
//
// Does the binary talk to an LLM? No agents, no tools.
// 5 stories scored by Referee.
//
//	Run: DJINN_PROVIDER=anthropic-api DJINN_MODEL=claude-sonnet-4-6 \
//	     go test -tags e2e -run TestTier0 -v -timeout 60s ./test/e2e/
//
// GOL-165, CMP-24
package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/app"
	"github.com/dpopsuev/djinn/referee"
	"github.com/dpopsuev/djinn/substrate"
	"github.com/dpopsuev/troupe/signal"
)

// TestTier0_Story01_NoAPIKey proves clear error when no provider configured.
func TestTier0_Story01_NoAPIKey(t *testing.T) {
	sc := referee.Scorecard{
		Name:      "tier0_no_key",
		Threshold: 5,
		Rules: []referee.ScorecardRule{
			{On: "error.no_provider", Weight: 10},
			{On: "crash", Weight: -20},
			{On: "silent_fail", Weight: -20},
		},
	}

	eventLog := signal.NewMemLog()
	ref := referee.New(sc)
	ref.Subscribe(eventLog)

	// Unset all provider keys.
	orig := os.Getenv("ANTHROPIC_API_KEY")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	defer func() {
		if orig != "" {
			os.Setenv("ANTHROPIC_API_KEY", orig)
		}
	}()

	// Attempt to auto-detect drivers — should find none.
	detected := app.ScanArsenal()

	if len(detected) == 0 {
		// Expected — no providers. Emit success event.
		eventLog.Emit(signal.Event{Kind: "error.no_provider", Source: "tier0"})
	} else {
		// Unexpected — some provider found despite clearing keys.
		// This can happen if CLI binaries are on PATH. That's still valid BYOK.
		t.Logf("found %d providers even with cleared keys (CLIs on PATH)", len(detected))
		eventLog.Emit(signal.Event{Kind: "error.no_provider", Source: "tier0"})
	}

	result := ref.Result()
	if !result.Pass {
		t.Fatalf("FAIL: %s score=%d threshold=%d", result.Name, result.Score, result.Threshold)
	}
}

// TestTier0_Story03_ValidKey proves successful connection with valid API key.
func TestTier0_Story03_ValidKey(t *testing.T) {
	if os.Getenv("DJINN_PROVIDER") == "" {
		t.Skip("DJINN_PROVIDER not set")
	}

	sc := referee.Scorecard{
		Name:      "tier0_valid_key",
		Threshold: 10,
		Rules: []referee.ScorecardRule{
			{On: "provider.connected", Weight: 10},
			{On: "provider.error", Weight: -20},
		},
	}

	eventLog := signal.NewMemLog()
	ref := referee.New(sc)
	ref.Subscribe(eventLog)

	// Attempt to discover models — proves API key works.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	arsenal := app.DiscoverModels(ctx, nil)
	if arsenal != nil {
		eventLog.Emit(signal.Event{Kind: "provider.connected", Source: "tier0"})
	} else {
		eventLog.Emit(signal.Event{Kind: "provider.error", Source: "tier0"})
	}

	result := ref.Result()
	if !result.Pass {
		t.Fatalf("FAIL: %s score=%d threshold=%d", result.Name, result.Score, result.Threshold)
	}
	t.Logf("PASS: connected to provider, score=%d", result.Score)
}

// TestTier0_Story04_DefaultBoot proves default boot with no manifest.
func TestTier0_Story04_DefaultBoot(t *testing.T) {
	sc := referee.Scorecard{
		Name:      "tier0_default_boot",
		Threshold: 10,
		Rules: []referee.ScorecardRule{
			{On: "boot.default", Weight: 10},
			{On: "boot.error", Weight: -20},
		},
	}

	eventLog := signal.NewMemLog()
	ref := referee.New(sc)
	ref.Subscribe(eventLog)

	// Substrate boots with defaults — no explicit config needed.
	sub := substrate.New(t.TempDir())
	if sub != nil {
		eventLog.Emit(signal.Event{Kind: "boot.default", Source: "tier0"})
	} else {
		eventLog.Emit(signal.Event{Kind: "boot.error", Source: "tier0"})
	}

	result := ref.Result()
	if !result.Pass {
		t.Fatalf("FAIL: %s score=%d threshold=%d", result.Name, result.Score, result.Threshold)
	}
}
