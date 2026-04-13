package referee

import (
	"testing"

	"github.com/dpopsuev/troupe/signal"
)

// TestE2E_HelloWorld_Pass proves the Referee scores a simple text-only agent run.
// Hello World: text=+10, tool_call=-10, short response (done with low tokens)=+5.
// Threshold=15. Agent responds with text only → should PASS.
func TestE2E_HelloWorld_Pass(t *testing.T) {
	sc := Scorecard{
		Name:               "hello_world",
		Threshold:          15,
		UnknownEventWeight: 0,
		Rules: []ScorecardRule{
			{On: "agent.cognitive.think", Weight: 5},
			{On: "agent.cognitive.decide", Weight: -10}, // tool decision = bad for hello
			{On: "tool.executed", Weight: -10},
			{On: "agent.run.start", Weight: 0},
			{On: "agent.run.done", Weight: 10},
		},
	}

	eventLog := signal.NewMemLog()
	ref := New(sc)
	ref.Subscribe(eventLog)

	// Simulate a good hello world run: think → done (no tools).
	eventLog.Emit(signal.Event{Kind: "agent.run.start", Source: "director"})
	eventLog.Emit(signal.Event{Kind: "agent.cognitive.think", Source: "gensec"})
	eventLog.Emit(signal.Event{Kind: "agent.run.done", Source: "director"})

	result := ref.Result()

	if !result.Pass {
		t.Fatalf("expected PASS, got score=%d threshold=%d", result.Score, result.Threshold)
	}
	if result.Score != 15 { // start(0) + think(5) + done(10)
		t.Errorf("score = %d, want 15", result.Score)
	}
	if len(result.Events) != 3 {
		t.Errorf("events = %d, want 3", len(result.Events))
	}
}

// TestE2E_HelloWorld_Fail proves tool calls make hello world FAIL.
func TestE2E_HelloWorld_Fail(t *testing.T) {
	sc := Scorecard{
		Name:               "hello_world",
		Threshold:          15,
		UnknownEventWeight: 0,
		Rules: []ScorecardRule{
			{On: "agent.cognitive.think", Weight: 5},
			{On: "agent.cognitive.decide", Weight: -10},
			{On: "tool.executed", Weight: -10},
			{On: "agent.run.done", Weight: 10},
		},
	}

	eventLog := signal.NewMemLog()
	ref := New(sc)
	ref.Subscribe(eventLog)

	// Bad hello: agent calls tools (decides + executes).
	eventLog.Emit(signal.Event{Kind: "agent.cognitive.think", Source: "gensec"})
	eventLog.Emit(signal.Event{Kind: "agent.cognitive.decide", Source: "gensec"})
	eventLog.Emit(signal.Event{Kind: "tool.executed", Source: "tool"})
	eventLog.Emit(signal.Event{Kind: "agent.run.done", Source: "director"})

	result := ref.Result()

	if result.Pass {
		t.Fatalf("expected FAIL, got score=%d", result.Score)
	}
	// Expected: think 5, decide -10, tool -10, done 10 → total -5.
	if result.Score != -5 {
		t.Errorf("score = %d, want -5", result.Score)
	}
}

// TestE2E_UnknownEvents proves unknown events get penalized.
func TestE2E_UnknownEvents(t *testing.T) {
	sc := Scorecard{
		Name:               "strict",
		Threshold:          0,
		UnknownEventWeight: -5,
		Rules: []ScorecardRule{
			{On: "agent.run.done", Weight: 10},
		},
	}

	eventLog := signal.NewMemLog()
	ref := New(sc)
	ref.Subscribe(eventLog)

	// Unknown events penalized.
	eventLog.Emit(signal.Event{Kind: "something.weird", Source: "x"})
	eventLog.Emit(signal.Event{Kind: "another.unknown", Source: "y"})
	eventLog.Emit(signal.Event{Kind: "agent.run.done", Source: "director"})

	result := ref.Result()
	// Expected: two unknowns at -5 each, one done at 10 → total 0.
	if result.Score != 0 {
		t.Errorf("score = %d, want 0", result.Score)
	}
	if !result.Pass { // 0 >= 0 threshold
		t.Error("expected PASS (score == threshold)")
	}
}

// TestE2E_Buckets proves bucket summary groups by event kind.
func TestE2E_Buckets(t *testing.T) {
	sc := Scorecard{
		Name:      "buckets",
		Threshold: 0,
		Rules: []ScorecardRule{
			{On: "tool.executed", Weight: 5},
		},
	}

	eventLog := signal.NewMemLog()
	ref := New(sc)
	ref.Subscribe(eventLog)

	eventLog.Emit(signal.Event{Kind: "tool.executed", Source: "tool"})
	eventLog.Emit(signal.Event{Kind: "tool.executed", Source: "tool"})
	eventLog.Emit(signal.Event{Kind: "tool.executed", Source: "tool"})

	result := ref.Result()
	bucket := result.Buckets["tool.executed"]
	if bucket.Count != 3 {
		t.Errorf("bucket count = %d, want 3", bucket.Count)
	}
	if bucket.TotalWeight != 15 {
		t.Errorf("bucket weight = %d, want 15", bucket.TotalWeight)
	}
}

// TestE2E_ParseScorecard proves YAML → Scorecard roundtrip.
func TestE2E_ParseScorecard(t *testing.T) {
	yaml := `
name: hello_world
threshold: 20
unknown_event_weight: -2
rules:
  - on: text
    weight: 10
  - on: tool_call
    weight: -10
  - on: done
    weight: 5
    condition: tokens_out < 100
`
	sc, err := ParseScorecard([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseScorecard: %v", err)
	}
	if sc.Name != "hello_world" {
		t.Errorf("name = %q", sc.Name)
	}
	if sc.Threshold != 20 {
		t.Errorf("threshold = %d", sc.Threshold)
	}
	if len(sc.Rules) != 3 {
		t.Fatalf("rules = %d, want 3", len(sc.Rules))
	}
	if sc.UnknownEventWeight != -2 {
		t.Errorf("unknown_event_weight = %d", sc.UnknownEventWeight)
	}
}

// TestE2E_Reset proves Referee can be reused.
func TestE2E_Reset(t *testing.T) {
	sc := Scorecard{Name: "reuse", Threshold: 0, Rules: []ScorecardRule{{On: "x", Weight: 5}}}
	eventLog := signal.NewMemLog()
	ref := New(sc)
	ref.Subscribe(eventLog)

	eventLog.Emit(signal.Event{Kind: "x"})
	if ref.Score() != 5 {
		t.Fatalf("score after first = %d", ref.Score())
	}

	ref.Reset()
	if ref.Score() != 0 {
		t.Fatalf("score after reset = %d", ref.Score())
	}
}
