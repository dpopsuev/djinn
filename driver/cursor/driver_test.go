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
	d := New(driver.DriverConfig{Model: "test"}, WithCommandFactory(
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			// Use bash -c with echo to get proper newlines.
			script := `echo '{"type":"system","subtype":"init","model":"test"}'
echo '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]}}'
echo '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello world"}]}}'
echo '{"type":"result","subtype":"success","usage":{"inputTokens":10,"outputTokens":5}}'`
			return exec.CommandContext(ctx, "bash", "-c", script)
		},
	))
	d.Send(context.Background(), driver.Message{Role: "user", Content: "test"})

	ch, err := d.Chat(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var texts []string
	var gotDone bool
	var usage *driver.Usage
	for evt := range ch {
		switch evt.Type {
		case driver.EventText:
			texts = append(texts, evt.Text)
		case driver.EventDone:
			gotDone = true
			usage = evt.Usage
		}
	}

	if !gotDone {
		t.Fatal("missing done event")
	}
	if len(texts) == 0 {
		t.Fatal("no text events received")
	}
	// First delta = "hello", second delta = " world" (cumulative diff)
	if texts[0] != "hello" {
		t.Fatalf("first delta = %q, want 'hello'", texts[0])
	}
	if texts[1] != " world" {
		t.Fatalf("second delta = %q, want ' world'", texts[1])
	}
	if usage == nil {
		t.Logf("texts received: %v, gotDone: %v", texts, gotDone)
		t.Fatalf("usage is nil — result event may not have parsed correctly")
	}
	if usage.InputTokens != 10 || usage.OutputTokens != 5 {
		t.Fatalf("usage = %+v, want in=10 out=5", usage)
	}
}

func TestChat_PlainTextFallback(t *testing.T) {
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
			script := `echo '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"response"}]}}'
echo '{"type":"result","subtype":"success"}'`
			return exec.CommandContext(ctx, "bash", "-c", script)
		},
	))
	d.Send(context.Background(), driver.Message{Role: "user", Content: "test"})

	ch, _ := d.Chat(context.Background())
	for range ch {
	}

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
