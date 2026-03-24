package acp

import (
	"context"
	"os/exec"
	"testing"

	"github.com/dpopsuev/djinn/driver"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
}

// mockACPServer is a bash script that simulates an ACP agent.
const mockACPServer = `#!/bin/bash
# Read and respond to JSON-RPC messages from stdin.
while IFS= read -r line; do
  method=$(echo "$line" | grep -o '"method":"[^"]*"' | cut -d'"' -f4)
  id=$(echo "$line" | grep -o '"id":[0-9]*' | cut -d: -f2)

  case "$method" in
    initialize)
      echo '{"jsonrpc":"2.0","id":'$id',"result":{"protocolVersion":1,"agentInfo":{"name":"mock","version":"0.1.0"}}}'
      ;;
    session/new)
      echo '{"jsonrpc":"2.0","id":'$id',"result":{"sessionId":"test-session-1"}}'
      ;;
    session/prompt)
      echo '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"test-session-1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hello "}}}}'
      echo '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"test-session-1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"world"}}}}'
      echo '{"jsonrpc":"2.0","id":'$id',"result":{"stopReason":"end_turn"}}'
      ;;
    session/cancel)
      exit 0
      ;;
  esac
done
`

func TestNew_ValidAgent(t *testing.T) {
	for _, name := range []string{"cursor", "claude", "gemini", "codex"} {
		d, err := New(name)
		if err != nil {
			t.Fatalf("New(%q): %v", name, err)
		}
		if d.agentName != name {
			t.Fatalf("agent = %q, want %q", d.agentName, name)
		}
	}
}

func TestNew_InvalidAgent(t *testing.T) {
	_, err := New("unknown")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestACPDriver_FullLifecycle(t *testing.T) {
	d, err := New("cursor", WithCommandFactory(
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "bash", "-c", mockACPServer)
		},
	))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Start — handshake.
	if err := d.Start(ctx, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if d.sessionID != "test-session-1" {
		t.Fatalf("sessionID = %q", d.sessionID)
	}

	// Send + Chat.
	d.Send(ctx, driver.Message{Role: "user", Content: "hello"})
	ch, err := d.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var texts []string
	var gotDone bool
	for evt := range ch {
		switch evt.Type {
		case driver.EventText:
			texts = append(texts, evt.Text)
		case driver.EventDone:
			gotDone = true
		case driver.EventError:
			t.Fatalf("error: %s", evt.Error)
		}
	}

	if !gotDone {
		t.Fatal("missing done event")
	}
	if len(texts) != 2 || texts[0] != "hello " || texts[1] != "world" {
		t.Fatalf("texts = %v, want [hello , world]", texts)
	}

	// History should have user + assistant.
	if len(d.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(d.messages))
	}
	if d.messages[1].Role != driver.RoleAssistant {
		t.Fatalf("role = %q", d.messages[1].Role)
	}
	if d.messages[1].Content != "hello world" {
		t.Fatalf("content = %q", d.messages[1].Content)
	}

	// Stop.
	d.Stop(ctx) //nolint:errcheck
}

func TestACPDriver_ChatNoMessages(t *testing.T) {
	d, _ := New("cursor")
	_, err := d.Chat(context.Background())
	if err == nil {
		t.Fatal("expected error with no messages")
	}
}

func TestACPDriver_SendRich(t *testing.T) {
	d, _ := New("cursor")
	d.SendRich(context.Background(), driver.RichMessage{
		Role:   "user",
		Blocks: []driver.ContentBlock{driver.NewTextBlock("rich")},
	})
	if len(d.messages) != 1 || d.messages[0].Content != "rich" {
		t.Fatalf("messages = %+v", d.messages)
	}
}

func TestACPDriver_AppendAssistant(t *testing.T) {
	d, _ := New("cursor")
	d.AppendAssistant(driver.RichMessage{Role: "assistant", Content: "hi"})
	if len(d.messages) != 1 {
		t.Fatalf("messages = %d", len(d.messages))
	}
}

// Verify ACPDriver implements ChatDriver at compile time.
var _ driver.ChatDriver = (*ACPDriver)(nil)
