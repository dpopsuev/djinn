//go:build e2e

// tier1_conversation_test.go — Tier 1: Conversation
//
// Single agent, text in → text out. No tools. No lifecycle.
// Each story scored by Referee.
//
//	Run: DJINN_PROVIDER=anthropic-api DJINN_MODEL=claude-sonnet-4-6 \
//	     go test -tags e2e -run TestTier1 -v -timeout 120s ./test/e2e/
//
// GOL-166, CMP-24
package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/referee"
	"github.com/dpopsuev/djinn/substrate"
	"github.com/dpopsuev/troupe/execution"
	"github.com/dpopsuev/troupe/signal"

	troupedriver "github.com/dpopsuev/djinn/driver/troupe"
)

// tier1Helper runs a single-agent conversation and scores it.
func tier1Helper(t *testing.T, sc referee.Scorecard, prompt string) referee.Result {
	t.Helper()

	if os.Getenv("DJINN_PROVIDER") == "" {
		t.Skip("DJINN_PROVIDER not set")
	}

	provider, err := execution.NewProviderFromEnv("DJINN_PROVIDER")
	if err != nil {
		t.Fatalf("provider: %v", err)
	}

	model := os.Getenv("DJINN_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	eventLog := signal.NewMemLog()
	ref := referee.New(sc)
	ref.Subscribe(eventLog)

	drv := troupedriver.New(provider, model,
		troupedriver.WithSystemPrompt("You are a helpful assistant. Be concise."),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := drv.Start(ctx, ""); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer drv.Stop(ctx) //nolint:errcheck // test cleanup

	sess := cortex.New("tier1", model, t.TempDir())

	dir := substrate.NewUnifiedDirector(drv, nil,
		substrate.WithSession(sess),
		substrate.WithSystemPrompt("You are a helpful assistant. Be concise. Keep responses under 3 sentences."),
		substrate.WithMaxTurns(3),
		substrate.WithEventLogForDirector(eventLog),
	)

	// Emit operator prompt as event — visible in scorecard chrono dump.
	eventLog.Emit(signal.Event{Kind: "operator.prompt", Source: "operator", Data: prompt})

	handler := &tier1EventHandler{eventLog: eventLog}
	output, err := dir.Run(ctx, prompt, agent.ModeAsk, nil, handler, "gensec")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	t.Logf("Agent response: %s", output)
	return ref.Result()
}

// tier1EventHandler emits cognitive events to EventLog for Referee scoring.
type tier1EventHandler struct {
	eventLog signal.EventLog
}

func (h *tier1EventHandler) OnText(text string) {
	h.eventLog.Emit(signal.Event{Kind: "text", Source: "agent", Data: text})
}
func (h *tier1EventHandler) OnThinking(text string) {
	h.eventLog.Emit(signal.Event{Kind: signal.KindThink, Source: "agent", Data: text})
}
func (h *tier1EventHandler) OnToolCall(call driver.ToolCall) {
	h.eventLog.Emit(signal.Event{Kind: "tool_call", Source: "agent", Data: call.Name})
}
func (h *tier1EventHandler) OnToolResult(_, _, _ string, _ bool) {}
func (h *tier1EventHandler) OnDone(usage *driver.Usage) {
	h.eventLog.Emit(signal.Event{Kind: "done", Source: "agent", Data: usage})
}
func (h *tier1EventHandler) OnError(err error) {
	h.eventLog.Emit(signal.Event{Kind: "error", Source: "agent", Data: err.Error()})
}

// --- Stories ---

// TestTier1_Story01_Hello — agent responds with short text, no tools.
func TestTier1_Story01_Hello(t *testing.T) {
	sc := referee.Scorecard{
		Name:               "tier1_hello",
		Threshold:          15,
		UnknownEventWeight: 0,
		Rules: []referee.ScorecardRule{
			{On: "text", Weight: 10},
			{On: "tool_call", Weight: -10},
			{On: "done", Weight: 10},
			{On: "error", Weight: -20},
		},
	}

	result := tier1Helper(t, sc, "Hello")

	if !result.Pass {
		t.Fatalf("FAIL: %s score=%d threshold=%d events=%d",
			result.Name, result.Score, result.Threshold, len(result.Events))
	}
	t.Logf("PASS: score=%d", result.Score)
}

// TestTier1_Story03_ExplainCode — agent reads/explains without writing.
func TestTier1_Story03_ExplainCode(t *testing.T) {
	sc := referee.Scorecard{
		Name:               "tier1_explain",
		Threshold:          15,
		UnknownEventWeight: 0,
		Rules: []referee.ScorecardRule{
			{On: "text", Weight: 10},
			{On: "tool_call", Weight: -15},
			{On: "done", Weight: 10},
			{On: "error", Weight: -20},
		},
	}

	result := tier1Helper(t, sc, "Explain what a goroutine is in Go, in one sentence.")

	if !result.Pass {
		t.Fatalf("FAIL: %s score=%d threshold=%d",
			result.Name, result.Score, result.Threshold)
	}
	t.Logf("PASS: score=%d", result.Score)
}

// TestTier1_Story07_Brainstorm — agent asks back instead of deciding unilaterally.
func TestTier1_Story07_Brainstorm(t *testing.T) {
	sc := referee.Scorecard{
		Name:               "tier1_brainstorm",
		Threshold:          10,
		UnknownEventWeight: 0,
		Rules: []referee.ScorecardRule{
			{On: "text", Weight: 10},
			{On: "done", Weight: 5},
			{On: "tool_call", Weight: -10},
			{On: "error", Weight: -20},
		},
	}

	result := tier1Helper(t, sc, "What are three good names for a symbol search tool?")

	if !result.Pass {
		t.Fatalf("FAIL: %s score=%d threshold=%d",
			result.Name, result.Score, result.Threshold)
	}
	t.Logf("PASS: score=%d", result.Score)
}
