package cursor

import (
	"context"
	"os/exec"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestNew_DefaultModel(t *testing.T) {
	d := New(driver.DriverConfig{})
	if d.model != "sonnet-4" {
		t.Fatalf("default model = %q, want sonnet-4", d.model)
	}
}

func TestNew_CustomModel(t *testing.T) {
	d := New(driver.DriverConfig{Model: "gpt-5"})
	if d.model != "gpt-5" {
		t.Fatalf("model = %q, want gpt-5", d.model)
	}
}

func TestNew_Options(t *testing.T) {
	d := New(driver.DriverConfig{}, WithModel("o3"), WithSystemPrompt("be helpful"))
	if d.model != "o3" {
		t.Fatalf("model = %q", d.model)
	}
	if d.systemPrompt != "be helpful" {
		t.Fatalf("prompt = %q", d.systemPrompt)
	}
}

func TestSend_AccumulatesMessages(t *testing.T) {
	d := New(driver.DriverConfig{})
	d.Send(context.Background(), driver.Message{Role: "user", Content: "hello"})
	d.Send(context.Background(), driver.Message{Role: "user", Content: "world"})
	if len(d.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(d.messages))
	}
}

func TestSendRich_ConvertsToPlain(t *testing.T) {
	d := New(driver.DriverConfig{})
	d.SendRich(context.Background(), driver.RichMessage{
		Role:   "user",
		Blocks: []driver.ContentBlock{driver.NewTextBlock("rich text")},
	})
	if len(d.messages) != 1 || d.messages[0].Content != "rich text" {
		t.Fatalf("message = %+v", d.messages)
	}
}

func TestChat_NoMessages_Error(t *testing.T) {
	d := New(driver.DriverConfig{})
	_, err := d.Chat(context.Background())
	if err == nil {
		t.Fatal("expected error with no messages")
	}
}

func TestAppendAssistant(t *testing.T) {
	d := New(driver.DriverConfig{})
	d.AppendAssistant(driver.RichMessage{Role: "assistant", Content: "hi"})
	if len(d.messages) != 1 || d.messages[0].Role != "assistant" {
		t.Fatalf("messages = %+v", d.messages)
	}
}

func TestSetSystemPrompt(t *testing.T) {
	d := New(driver.DriverConfig{})
	d.SetSystemPrompt("new prompt")
	if d.systemPrompt != "new prompt" {
		t.Fatalf("prompt = %q", d.systemPrompt)
	}
}

func TestChat_StreamJSON(t *testing.T) {
	// Mock: echo outputs stream-json lines to stdout.
	d := New(driver.DriverConfig{Model: "test"}, WithCommandFactory(
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", `{"type":"text_delta","content":"hello"}
{"type":"text_delta","content":" world"}
{"type":"done","input_tokens":10,"output_tokens":5}`)
		},
	))
	d.Send(context.Background(), driver.Message{Role: "user", Content: "test"})

	ch, err := d.Chat(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var texts []string
	var gotDone bool
	for evt := range ch {
		switch evt.Type {
		case driver.EventText:
			texts = append(texts, evt.Text)
		case driver.EventDone:
			gotDone = true
			if evt.Usage != nil && evt.Usage.InputTokens == 10 {
				t.Logf("usage: in=%d out=%d", evt.Usage.InputTokens, evt.Usage.OutputTokens)
			}
		}
	}

	if !gotDone {
		t.Fatal("missing done event")
	}
	if len(texts) == 0 {
		t.Fatal("no text events received")
	}
	joined := ""
	for _, s := range texts {
		joined += s
	}
	if joined != "hello world" {
		t.Fatalf("text = %q, want 'hello world'", joined)
	}
}

func TestChat_PlainTextFallback(t *testing.T) {
	// Mock: command outputs plain text (not JSON).
	d := New(driver.DriverConfig{}, WithCommandFactory(
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", "plain response")
		},
	))
	d.Send(context.Background(), driver.Message{Role: "user", Content: "test"})

	ch, err := d.Chat(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var got string
	for evt := range ch {
		if evt.Type == driver.EventText {
			got += evt.Text
		}
	}
	if got == "" {
		t.Fatal("no text from plain output")
	}
}

func TestChat_AppendsAssistantHistory(t *testing.T) {
	d := New(driver.DriverConfig{}, WithCommandFactory(
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", `{"type":"text_delta","content":"response"}`)
		},
	))
	d.Send(context.Background(), driver.Message{Role: "user", Content: "test"})

	ch, _ := d.Chat(context.Background())
	for range ch {
	} // drain

	if len(d.messages) != 2 {
		t.Fatalf("messages = %d, want 2 (user + assistant)", len(d.messages))
	}
	if d.messages[1].Role != driver.RoleAssistant {
		t.Fatalf("role = %q, want assistant", d.messages[1].Role)
	}
}

func TestStartStop_Noop(t *testing.T) {
	d := New(driver.DriverConfig{})
	if err := d.Start(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if err := d.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// Verify CLIDriver implements ChatDriver at compile time.
var _ driver.ChatDriver = (*CLIDriver)(nil)
