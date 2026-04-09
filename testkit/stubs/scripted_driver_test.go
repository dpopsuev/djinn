package stubs

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestScriptedChatDriver_TextOnly(t *testing.T) {
	d := NewScriptedChatDriver(
		ScriptedTurn{Text: "hello"},
		ScriptedTurn{Text: "world"},
	)

	ctx := context.Background()

	// Turn 1
	ch, err := d.Chat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	events := drain(ch)
	assertEvent(t, events, 0, driver.EventText, "hello")
	assertEvent(t, events, 1, driver.EventDone, "")

	// Turn 2
	ch, _ = d.Chat(ctx)
	events = drain(ch)
	assertEvent(t, events, 0, driver.EventText, "world")

	if d.TurnCount() != 2 {
		t.Fatalf("turns = %d, want 2", d.TurnCount())
	}
}

func TestScriptedChatDriver_EmitsToolUse(t *testing.T) {
	d := NewScriptedChatDriver(
		ScriptedTurn{
			Text: "I'll write the file",
			ToolCalls: []driver.ToolCall{
				{ID: "call_1", Name: "Write", Input: MustJSON(map[string]string{"path": "main.go", "content": "package main"})},
			},
		},
		ScriptedTurn{Text: "done"},
	)

	ctx := context.Background()

	// Turn 1: text + tool call
	ch, _ := d.Chat(ctx)
	events := drain(ch)

	if len(events) != 3 { // text + tool_use + done
		t.Fatalf("events = %d, want 3", len(events))
	}
	assertEvent(t, events, 0, driver.EventText, "I'll write the file")
	if events[1].Type != driver.EventToolUse {
		t.Fatalf("event[1].type = %q, want tool_use", events[1].Type)
	}
	if events[1].ToolCall == nil {
		t.Fatal("event[1].ToolCall should not be nil")
	}
	if events[1].ToolCall.Name != "Write" {
		t.Fatalf("tool name = %q, want Write", events[1].ToolCall.Name)
	}
	if events[1].ToolCall.ID != "call_1" {
		t.Fatalf("tool id = %q, want call_1", events[1].ToolCall.ID)
	}

	// Turn 2: text only
	ch, _ = d.Chat(ctx)
	events = drain(ch)
	assertEvent(t, events, 0, driver.EventText, "done")
}

func TestScriptedChatDriver_MultipleToolCalls(t *testing.T) {
	d := NewScriptedChatDriver(
		ScriptedTurn{
			ToolCalls: []driver.ToolCall{
				{ID: "c1", Name: "Write", Input: MustJSON(map[string]string{"path": "a.go"})},
				{ID: "c2", Name: "Write", Input: MustJSON(map[string]string{"path": "b.go"})},
			},
		},
	)

	ch, _ := d.Chat(context.Background())
	events := drain(ch)

	// No text, 2 tool calls, 1 done = 3 events
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3", len(events))
	}
	if events[0].Type != driver.EventToolUse {
		t.Fatalf("event[0] = %q, want tool_use", events[0].Type)
	}
	if events[1].Type != driver.EventToolUse {
		t.Fatalf("event[1] = %q, want tool_use", events[1].Type)
	}
}

func TestScriptedChatDriver_RecordsSendRich(t *testing.T) {
	d := NewScriptedChatDriver(ScriptedTurn{Text: "hi"})

	ctx := context.Background()
	d.SendRich(ctx, driver.RichMessage{ //nolint:errcheck // test
		Role: driver.RoleUser,
		Blocks: []driver.ContentBlock{
			driver.NewToolResultBlock("call_1", "file written", false),
		},
	})

	results := d.ToolResults()
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].ToolCallID != "call_1" {
		t.Fatalf("tool call id = %q, want call_1", results[0].ToolCallID)
	}
	if results[0].Output != "file written" {
		t.Fatalf("output = %q, want 'file written'", results[0].Output)
	}
}

func TestScriptedChatDriver_Exhausted(t *testing.T) {
	d := NewScriptedChatDriver(ScriptedTurn{Text: "only one"})

	d.Chat(context.Background()) //nolint:errcheck // consume turn 1

	// Turn 2: exhausted
	ch, _ := d.Chat(context.Background())
	events := drain(ch)
	assertEvent(t, events, 0, driver.EventText, "(no more turns)")
}

// --- helpers ---

func drain(ch <-chan driver.StreamEvent) []driver.StreamEvent {
	var events []driver.StreamEvent
	for e := range ch {
		events = append(events, e)
	}
	return events
}

func assertEvent(t *testing.T, events []driver.StreamEvent, idx int, typ, text string) {
	t.Helper()
	if idx >= len(events) {
		t.Fatalf("event[%d] out of range (have %d)", idx, len(events))
	}
	if events[idx].Type != typ {
		t.Fatalf("event[%d].type = %q, want %q", idx, events[idx].Type, typ)
	}
	if text != "" && events[idx].Text != text {
		t.Fatalf("event[%d].text = %q, want %q", idx, events[idx].Text, text)
	}
}
