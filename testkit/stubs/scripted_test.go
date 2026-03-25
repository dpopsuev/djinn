package stubs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestScriptedDriver_BasicTextStreaming(t *testing.T) {
	drv := NewScriptedDriver(ScriptedStep{
		TextDeltas: []string{"Hello", " ", "world"},
		Usage:      &driver.Usage{InputTokens: 10, OutputTokens: 3},
	})
	drv.Start(context.Background(), "")

	ch, err := drv.Chat(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var texts []string
	var gotDone bool
	var usage *driver.Usage
	for ev := range ch {
		switch ev.Type {
		case driver.EventText:
			texts = append(texts, ev.Text)
		case driver.EventDone:
			gotDone = true
			usage = ev.Usage
		}
	}

	if len(texts) != 3 {
		t.Fatalf("texts = %d, want 3", len(texts))
	}
	if !gotDone {
		t.Fatal("should receive EventDone")
	}
	if usage == nil || usage.OutputTokens != 3 {
		t.Fatal("usage should have OutputTokens=3")
	}
}

func TestScriptedDriver_ThinkingBlock(t *testing.T) {
	drv := NewScriptedDriver(ScriptedStep{
		Thinking:   "Let me think about this...",
		TextDeltas: []string{"The answer is 42"},
	})

	ch, _ := drv.Chat(context.Background())
	var gotThinking bool
	for ev := range ch {
		if ev.Type == driver.EventThinking {
			gotThinking = true
			if ev.Thinking != "Let me think about this..." {
				t.Fatalf("thinking = %q", ev.Thinking)
			}
		}
	}
	if !gotThinking {
		t.Fatal("should receive thinking event")
	}
}

func TestScriptedDriver_ToolCall(t *testing.T) {
	drv := NewScriptedDriver(ScriptedStep{
		ToolCall: &driver.ToolCall{
			ID:    "c1",
			Name:  "Read",
			Input: json.RawMessage(`{"file_path":"main.go"}`),
		},
	})

	ch, _ := drv.Chat(context.Background())
	var gotTool bool
	for ev := range ch {
		if ev.Type == driver.EventToolUse {
			gotTool = true
			if ev.ToolCall.Name != "Read" {
				t.Fatalf("tool = %q", ev.ToolCall.Name)
			}
		}
	}
	if !gotTool {
		t.Fatal("should receive tool call event")
	}
}

func TestScriptedDriver_MultiTurn(t *testing.T) {
	drv := NewScriptedDriver(
		ScriptedStep{TextDeltas: []string{"First response"}},
		ScriptedStep{TextDeltas: []string{"Second response"}},
		ScriptedStep{TextDeltas: []string{"Third response"}},
	)

	for i := range 3 {
		ch, _ := drv.Chat(context.Background())
		var text string
		for ev := range ch {
			if ev.Type == driver.EventText {
				text += ev.Text
			}
		}
		want := []string{"First response", "Second response", "Third response"}
		if text != want[i] {
			t.Fatalf("turn %d: text = %q, want %q", i, text, want[i])
		}
	}

	if drv.StepIndex() != 3 {
		t.Fatalf("stepIdx = %d, want 3", drv.StepIndex())
	}
}

func TestScriptedDriver_ExhaustedSteps(t *testing.T) {
	drv := NewScriptedDriver(ScriptedStep{TextDeltas: []string{"only one"}})

	// First call succeeds.
	ch, _ := drv.Chat(context.Background())
	for range ch {
	}

	// Second call returns done immediately (no more steps).
	ch2, _ := drv.Chat(context.Background())
	for ev := range ch2 {
		if ev.Type != driver.EventDone {
			t.Fatalf("exhausted driver should only send Done, got %q", ev.Type)
		}
	}
}

func TestScriptedDriver_SendRecordsMessages(t *testing.T) {
	drv := NewScriptedDriver()
	drv.Send(context.Background(), driver.Message{Role: "user", Content: "hello"})
	drv.Send(context.Background(), driver.Message{Role: "user", Content: "world"})

	if len(drv.SendLog) != 2 {
		t.Fatalf("SendLog = %d, want 2", len(drv.SendLog))
	}
}

func TestScriptedDriver_ContextWindow(t *testing.T) {
	drv := NewScriptedDriver().WithContextWindow(1000)
	if drv.ContextWindow() != 1000 {
		t.Fatalf("window = %d, want 1000", drv.ContextWindow())
	}
}

func TestScriptedDriver_StartStop(t *testing.T) {
	drv := NewScriptedDriver()
	if drv.Started() {
		t.Fatal("should not be started")
	}
	drv.Start(context.Background(), "")
	if !drv.Started() {
		t.Fatal("should be started")
	}
	drv.Stop(context.Background())
	if !drv.Stopped() {
		t.Fatal("should be stopped")
	}
}
