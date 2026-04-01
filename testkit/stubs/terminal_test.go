package stubs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/terminal"
)

func TestTestOperator_SubmitAndReceiveEvent(t *testing.T) {
	op := NewTestOperator()

	// Wire a handler that emits an output event on submit.
	op.Djinn().OnSubmit = func(_ context.Context, prompt string) error {
		op.Djinn().Emit(terminal.ViewEvent{
			Kind: terminal.EventOutput,
			Text: "echo: " + prompt,
		})
		return nil
	}

	ctx := context.Background()
	if err := op.Submit(ctx, "hello"); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ev, ok := op.WaitForEvent(terminal.EventOutput, time.Second)
	if !ok {
		t.Fatal("timed out waiting for EventOutput")
	}
	if ev.Text != "echo: hello" {
		t.Errorf("Text = %q, want %q", ev.Text, "echo: hello")
	}
}

func TestTestOperator_NoHandler(t *testing.T) {
	op := NewTestOperator()
	err := op.Submit(context.Background(), "hello")
	if !errors.Is(err, terminal.ErrNoSubmitHandler) {
		t.Errorf("Submit without handler = %v, want ErrNoSubmitHandler", err)
	}
}

func TestTestOperator_Command(t *testing.T) {
	op := NewTestOperator()
	ctx := context.Background()

	out, err := op.Command(ctx, "op", nil)
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	if out != "operation: agent" {
		t.Errorf("op query = %q, want %q", out, "operation: agent")
	}
}

func TestTestOperator_Events(t *testing.T) {
	op := NewTestOperator()
	d := op.Djinn()

	d.Emit(terminal.ViewEvent{Kind: terminal.EventOutput, Text: "one"})
	d.Emit(terminal.ViewEvent{Kind: terminal.EventToolCall, Tool: "Read"})
	d.Emit(terminal.ViewEvent{Kind: terminal.EventDone})

	// Give channel a moment to deliver.
	time.Sleep(10 * time.Millisecond)

	events := op.Events()
	if len(events) != 3 {
		t.Fatalf("Events() = %d, want 3", len(events))
	}
	if events[0].Kind != terminal.EventOutput {
		t.Errorf("events[0].Kind = %s, want output", events[0].Kind)
	}
	if events[1].Kind != terminal.EventToolCall {
		t.Errorf("events[1].Kind = %s, want tool_call", events[1].Kind)
	}
	if events[2].Kind != terminal.EventDone {
		t.Errorf("events[2].Kind = %s, want done", events[2].Kind)
	}
}

func TestTestOperator_DrainEvents(t *testing.T) {
	op := NewTestOperator()
	d := op.Djinn()

	d.Emit(terminal.ViewEvent{Kind: terminal.EventOutput, Text: "a"})
	d.Emit(terminal.ViewEvent{Kind: terminal.EventOutput, Text: "b"})

	time.Sleep(10 * time.Millisecond)

	drained := op.DrainEvents()
	if len(drained) != 2 {
		t.Fatalf("DrainEvents = %d, want 2", len(drained))
	}

	// Second drain should still return all collected events.
	drained2 := op.DrainEvents()
	if len(drained2) != 2 {
		t.Fatalf("second DrainEvents = %d, want 2 (accumulated)", len(drained2))
	}
}

func TestTestOperator_WaitForEvent_Timeout(t *testing.T) {
	op := NewTestOperator()
	_, ok := op.WaitForEvent(terminal.EventDone, 50*time.Millisecond)
	if ok {
		t.Error("WaitForEvent should return false on timeout")
	}
}

func TestTestOperator_WaitForEvent_SkipsOtherKinds(t *testing.T) {
	op := NewTestOperator()
	d := op.Djinn()

	go func() {
		time.Sleep(10 * time.Millisecond)
		d.Emit(terminal.ViewEvent{Kind: terminal.EventOutput, Text: "noise"})
		d.Emit(terminal.ViewEvent{Kind: terminal.EventToolCall, Tool: "Bash"})
		d.Emit(terminal.ViewEvent{Kind: terminal.EventDone})
	}()

	ev, ok := op.WaitForEvent(terminal.EventDone, time.Second)
	if !ok {
		t.Fatal("timed out waiting for EventDone")
	}
	if ev.Kind != terminal.EventDone {
		t.Errorf("Kind = %s, want done", ev.Kind)
	}

	// The skipped events should still be collected.
	events := op.Events()
	if len(events) < 3 {
		t.Errorf("Events() = %d, want >= 3", len(events))
	}
}

func TestTestOperator_Status(t *testing.T) {
	op := NewTestOperator()
	d := op.Djinn()

	d.SetTokens(200, 100)
	d.SetTurns(5)
	d.SetAgentCount(3)

	s := d.Status()
	if s.TokensIn != 200 {
		t.Errorf("TokensIn = %d, want 200", s.TokensIn)
	}
	if s.Turns != 5 {
		t.Errorf("Turns = %d, want 5", s.Turns)
	}
	if s.AgentCount != 3 {
		t.Errorf("AgentCount = %d, want 3", s.AgentCount)
	}
}
