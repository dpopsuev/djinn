package troupe

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	anyllm "github.com/mozilla-ai/any-llm-go/providers"

	"github.com/dpopsuev/djinn/driver"
)

// --- Contract ---

func TestTroupeDriver_ImplementsChatDriver(t *testing.T) {
	// Compile-time check is at package level (var _ driver.ChatDriver = ...).
	// Behavioral: all methods callable without panic.
	p := NewStubProvider(TextResponse("hello", 10, 5))
	d := New(p, "test-model")

	ctx := context.Background()
	if err := d.Start(ctx, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "hi"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	ch, err := d.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	drain(ch) // consume events

	d.AppendAssistant(driver.RichMessage{Content: "hello"})
	d.SetSystemPrompt("test")
	if cw := d.ContextWindow(); cw == 0 {
		t.Fatal("ContextWindow returned 0")
	}
	if err := d.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// --- Unit: Text Completion ---

func TestTroupeDriver_TextCompletion(t *testing.T) {
	p := NewStubProvider(TextResponse("Hello, world!", 100, 20))
	d := New(p, "claude-sonnet-4-6")

	ctx := context.Background()
	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "say hello"}) //nolint:errcheck // test setup

	ch, err := d.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	events := drain(ch)

	// Expect: text + done = 2 events
	assertEventType(t, events, 0, driver.EventText)
	if events[0].Text != "Hello, world!" {
		t.Fatalf("text = %q, want %q", events[0].Text, "Hello, world!")
	}

	assertEventType(t, events, 1, driver.EventDone)
	if events[1].Usage == nil {
		t.Fatal("EventDone missing Usage")
	}
	if events[1].Usage.InputTokens != 100 {
		t.Fatalf("input_tokens = %d, want 100", events[1].Usage.InputTokens)
	}
	if events[1].Usage.OutputTokens != 20 {
		t.Fatalf("output_tokens = %d, want 20", events[1].Usage.OutputTokens)
	}
}

// --- Unit: Tool Call Completion ---

func TestTroupeDriver_ToolCallCompletion(t *testing.T) {
	p := NewStubProvider(ToolCallResponse(
		"I'll write the file",
		[]anyllm.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: anyllm.FunctionCall{
					Name:      "Write",
					Arguments: `{"path":"main.go","content":"package main"}`,
				},
			},
		},
		50, 30,
	))
	d := New(p, "test-model")

	ctx := context.Background()
	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "write main.go"}) //nolint:errcheck // test setup

	ch, err := d.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	events := drain(ch)

	// Expect: text + tool_use + done = 3 events
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3", len(events))
	}

	assertEventType(t, events, 0, driver.EventText)
	if events[0].Text != "I'll write the file" {
		t.Fatalf("text = %q", events[0].Text)
	}

	assertEventType(t, events, 1, driver.EventToolUse)
	if events[1].ToolCall == nil {
		t.Fatal("EventToolUse missing ToolCall")
	}
	if events[1].ToolCall.Name != "Write" {
		t.Fatalf("tool name = %q, want Write", events[1].ToolCall.Name)
	}
	if events[1].ToolCall.ID != "call_1" {
		t.Fatalf("tool id = %q, want call_1", events[1].ToolCall.ID)
	}

	var input map[string]string
	if err := json.Unmarshal(events[1].ToolCall.Input, &input); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if input["path"] != "main.go" {
		t.Fatalf("input path = %q, want main.go", input["path"])
	}

	assertEventType(t, events, 2, driver.EventDone)
}

// --- Unit: Multi-Turn History ---

func TestTroupeDriver_MultiTurnHistory(t *testing.T) {
	p := NewStubProvider(
		TextResponse("response 1", 10, 5),
		TextResponse("response 2", 20, 10),
	)
	d := New(p, "test-model", WithSystemPrompt("you are helpful"))

	ctx := context.Background()

	// Turn 1
	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "hello"}) //nolint:errcheck // test setup
	ch, _ := d.Chat(ctx)
	drain(ch)
	d.AppendAssistant(driver.RichMessage{Content: "response 1"})

	// Turn 2
	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "follow up"}) //nolint:errcheck // test setup
	ch, _ = d.Chat(ctx)
	drain(ch)

	// Verify second call has full history
	if len(p.CallLog) != 2 {
		t.Fatalf("calls = %d, want 2", len(p.CallLog))
	}

	msgs := p.CallLog[1].Messages
	// Expect 4 messages: system + user("hello") + assistant("response 1") + user("follow up")
	if len(msgs) != 4 {
		t.Fatalf("turn 2 messages = %d, want 4", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Fatalf("msg[0].role = %q, want system", msgs[0].Role)
	}
	if msgs[1].ContentString() != "hello" {
		t.Fatalf("msg[1] = %q, want hello", msgs[1].ContentString())
	}
	if msgs[2].ContentString() != "response 1" {
		t.Fatalf("msg[2] = %q, want response 1", msgs[2].ContentString())
	}
	if msgs[3].ContentString() != "follow up" {
		t.Fatalf("msg[3] = %q, want follow up", msgs[3].ContentString())
	}
}

// --- Unit: Tool Result Round-Trip ---

func TestTroupeDriver_ToolResultRoundTrip(t *testing.T) {
	p := NewStubProvider(
		// Turn 1: assistant requests tool call
		ToolCallResponse("", []anyllm.ToolCall{
			{ID: "call_1", Type: "function", Function: anyllm.FunctionCall{Name: "Read", Arguments: `{"path":"x.go"}`}},
		}, 10, 5),
		// Turn 2: assistant responds after seeing tool result
		TextResponse("file contains package x", 30, 10),
	)
	d := New(p, "test-model")
	ctx := context.Background()

	// Turn 1: user → chat → tool call
	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "read x.go"}) //nolint:errcheck // test setup
	ch, _ := d.Chat(ctx)
	drain(ch)

	// Append assistant's tool call response to history
	d.AppendAssistant(driver.RichMessage{
		Role: driver.RoleAssistant,
		Blocks: []driver.ContentBlock{
			driver.NewToolUseBlock("call_1", "Read", json.RawMessage(`{"path":"x.go"}`)),
		},
	})

	// Send tool result
	d.SendRich(ctx, driver.RichMessage{ //nolint:errcheck // test setup
		Role: driver.RoleUser,
		Blocks: []driver.ContentBlock{
			driver.NewToolResultBlock("call_1", "package x", false),
		},
	})

	// Turn 2: chat → text response
	ch, _ = d.Chat(ctx)
	events := drain(ch)

	assertEventType(t, events, 0, driver.EventText)
	if events[0].Text != "file contains package x" {
		t.Fatalf("text = %q", events[0].Text)
	}

	// Verify tool result was sent correctly
	turn2Msgs := p.CallLog[1].Messages
	// Find the tool result message
	var foundToolResult bool
	for _, m := range turn2Msgs {
		if m.Role == "tool" && m.ToolCallID == "call_1" {
			if m.ContentString() != "package x" {
				t.Fatalf("tool result content = %q, want 'package x'", m.ContentString())
			}
			foundToolResult = true
		}
	}
	if !foundToolResult {
		t.Fatal("tool result message not found in turn 2")
	}
}

// --- Unit: Error Handling ---

func TestTroupeDriver_ErrorHandling(t *testing.T) {
	p := NewStubProvider()
	p.Error = errors.New("rate limit exceeded")
	d := New(p, "test-model")

	ctx := context.Background()
	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "hello"}) //nolint:errcheck // test setup

	ch, err := d.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat should not error, got: %v", err)
	}
	events := drain(ch)

	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	assertEventType(t, events, 0, driver.EventError)
	if events[0].Error != "rate limit exceeded" {
		t.Fatalf("error = %q", events[0].Error)
	}
}

// --- Integration: Tools Passed to Provider ---

func TestTroupeDriver_ToolsPassedToProvider(t *testing.T) {
	p := NewStubProvider(TextResponse("ok", 10, 5))
	tools := []anyllm.Tool{
		{Type: "function", Function: anyllm.Function{Name: "Write", Description: "write a file"}},
		{Type: "function", Function: anyllm.Function{Name: "Read", Description: "read a file"}},
	}
	d := New(p, "test-model", WithTools(tools))

	ctx := context.Background()
	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "test"}) //nolint:errcheck // test setup
	ch, _ := d.Chat(ctx)
	drain(ch)

	if len(p.CallLog) != 1 {
		t.Fatalf("calls = %d, want 1", len(p.CallLog))
	}
	sentTools := p.CallLog[0].Tools
	if len(sentTools) != 2 {
		t.Fatalf("tools = %d, want 2", len(sentTools))
	}
	if sentTools[0].Function.Name != "Write" {
		t.Fatalf("tool[0] = %q, want Write", sentTools[0].Function.Name)
	}
}

// --- Concurrency: Parallel Chat() Calls ---

func TestTroupeDriver_ConcurrentChatSafe(t *testing.T) {
	// Build enough responses for concurrent calls
	var responses []*anyllm.ChatCompletion
	for range 10 {
		responses = append(responses, TextResponse("ok", 10, 5))
	}
	p := NewStubProvider(responses...)
	d := New(p, "test-model")

	ctx := context.Background()
	d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "hello"}) //nolint:errcheck // test setup

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	for range 10 {
		wg.Go(func() {
			ch, err := d.Chat(ctx)
			if err != nil {
				errs <- err
				return
			}
			drain(ch)
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent Chat error: %v", err)
	}
}

// --- Security: Malformed Provider Response ---

func TestTroupeDriver_MalformedProviderResponse(t *testing.T) {
	t.Run("nil_choices", func(t *testing.T) {
		p := NewStubProvider(&anyllm.ChatCompletion{
			Choices: nil,
			Usage:   &anyllm.Usage{},
		})
		d := New(p, "test-model")
		ctx := context.Background()
		d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "test"}) //nolint:errcheck // test setup
		ch, _ := d.Chat(ctx)
		events := drain(ch)

		assertEventType(t, events, 0, driver.EventError)
		if events[0].Error == "" {
			t.Fatal("expected non-empty error for nil choices")
		}
	})

	t.Run("empty_choices", func(t *testing.T) {
		p := NewStubProvider(&anyllm.ChatCompletion{
			Choices: []anyllm.Choice{},
			Usage:   &anyllm.Usage{},
		})
		d := New(p, "test-model")
		ctx := context.Background()
		d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "test"}) //nolint:errcheck // test setup
		ch, _ := d.Chat(ctx)
		events := drain(ch)

		assertEventType(t, events, 0, driver.EventError)
	})

	t.Run("nil_usage", func(t *testing.T) {
		p := NewStubProvider(&anyllm.ChatCompletion{
			Choices: []anyllm.Choice{{
				Message:      anyllm.Message{Role: "assistant", Content: "ok"},
				FinishReason: "stop",
			}},
			Usage: nil, // no usage info
		})
		d := New(p, "test-model")
		ctx := context.Background()
		d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "test"}) //nolint:errcheck // test setup
		ch, _ := d.Chat(ctx)
		events := drain(ch)

		// Should still work — EventDone with nil Usage
		if len(events) < 2 {
			t.Fatalf("events = %d, want >= 2", len(events))
		}
		assertEventType(t, events, 0, driver.EventText)
		assertEventType(t, events, 1, driver.EventDone)
		if events[1].Usage != nil {
			t.Fatal("expected nil Usage when provider returns nil")
		}
	})

	t.Run("empty_tool_call_arguments", func(t *testing.T) {
		p := NewStubProvider(ToolCallResponse("", []anyllm.ToolCall{
			{ID: "c1", Type: "function", Function: anyllm.FunctionCall{Name: "Noop", Arguments: ""}},
		}, 10, 5))
		d := New(p, "test-model")
		ctx := context.Background()
		d.Send(ctx, driver.Message{Role: driver.RoleUser, Content: "test"}) //nolint:errcheck // test setup
		ch, _ := d.Chat(ctx)
		events := drain(ch)

		// Should emit tool call even with empty arguments
		var found bool
		for _, e := range events {
			if e.Type == driver.EventToolUse {
				found = true
				if e.ToolCall.Name != "Noop" {
					t.Fatalf("tool name = %q", e.ToolCall.Name)
				}
			}
		}
		if !found {
			t.Fatal("EventToolUse not emitted for empty-argument tool call")
		}
	})
}

// --- Context Window ---

func TestTroupeDriver_ContextWindow(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"claude-opus-4-6", 1_000_000},
		{"claude-sonnet-4-6", 200_000},
		{"gpt-4o", 200_000},
		{"claude-haiku-4-5", 200_000},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			d := New(NewStubProvider(), tt.model)
			if got := d.ContextWindow(); got != tt.want {
				t.Fatalf("ContextWindow(%q) = %d, want %d", tt.model, got, tt.want)
			}
		})
	}
}

// --- helpers ---

func drain(ch <-chan driver.StreamEvent) []driver.StreamEvent {
	var events []driver.StreamEvent
	for e := range ch {
		events = append(events, e)
	}
	return events
}

func assertEventType(t *testing.T, events []driver.StreamEvent, idx int, typ string) {
	t.Helper()
	if idx >= len(events) {
		t.Fatalf("event[%d] out of range (have %d events)", idx, len(events))
	}
	if events[idx].Type != typ {
		t.Fatalf("event[%d].type = %q, want %q", idx, events[idx].Type, typ)
	}
}
