package gemini

import (
	"context"
	"os/exec"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestNew_Options(t *testing.T) {
	d := New(driver.DriverConfig{Model: "gemini-2"}, WithSystemPrompt("test"))
	if d.model != "gemini-2" {
		t.Fatalf("model = %q", d.model)
	}
	if d.systemPrompt != "test" {
		t.Fatalf("prompt = %q", d.systemPrompt)
	}
}

func TestSend_AccumulatesMessages(t *testing.T) {
	d := New(driver.DriverConfig{})
	d.Send(context.Background(), driver.Message{Role: "user", Content: "hello"})
	if len(d.messages) != 1 {
		t.Fatalf("messages = %d", len(d.messages))
	}
}

func TestChat_NoMessages_Error(t *testing.T) {
	d := New(driver.DriverConfig{})
	_, err := d.Chat(context.Background())
	if err == nil {
		t.Fatal("expected error with no messages")
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

func TestChat_Roundtrip(t *testing.T) {
	d := New(driver.DriverConfig{}, WithCommandFactory(
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "printf", "line one\nline two\n")
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
		t.Fatal("no text received")
	}
}

func TestChat_AppendsHistory(t *testing.T) {
	d := New(driver.DriverConfig{}, WithCommandFactory(
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", "response")
		},
	))
	d.Send(context.Background(), driver.Message{Role: "user", Content: "test"})
	ch, _ := d.Chat(context.Background())
	for range ch {
	}
	if len(d.messages) != 2 || d.messages[1].Role != driver.RoleAssistant {
		t.Fatalf("messages = %+v", d.messages)
	}
}

var _ driver.ChatDriver = (*CLIDriver)(nil)
