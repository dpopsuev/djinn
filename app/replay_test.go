package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
)

func TestReplayHistory_ToolResultViaSendRich(t *testing.T) {
	// DJN-BUG-17: user entries with tool_result Blocks must go through
	// SendRich, not Send. Send strips Blocks → orphaned tool_use.
	sess := session.New("replay-test", "test-model", "/workspace")
	sess.Append(session.Entry{Role: "user", Content: "hello"})
	sess.Append(session.Entry{
		Role: "assistant",
		Blocks: []driver.ContentBlock{
			driver.NewTextBlock("Let me check."),
			driver.NewToolUseBlock("call-1", "Bash", json.RawMessage(`{"cmd":"ls"}`)),
		},
	})
	sess.Append(session.Entry{
		Role: "user",
		Blocks: []driver.ContentBlock{
			driver.NewToolResultBlock("call-1", "file1.go\nfile2.go", false),
		},
	})
	sess.Append(session.Entry{Role: "assistant", Content: "I see two files."})

	stub := &stubs.StubChatDriver{}
	stub.Start(context.Background(), "") //nolint:errcheck

	ReplayHistory(context.Background(), stub, sess)

	// The tool_result entry MUST go through SendRich
	if len(stub.SendRichLog) == 0 {
		t.Fatal("BUG-17: tool_result user entry was NOT sent via SendRich — blocks are lost")
	}

	// Verify the tool_result block is present with correct ID
	found := false
	for _, msg := range stub.SendRichLog {
		for _, block := range msg.Blocks {
			if block.Type == driver.BlockToolResult && block.ToolResult != nil {
				if block.ToolResult.ToolCallID == "call-1" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("BUG-17: tool_result block for call-1 not found in SendRich calls")
	}
}

func TestReplayHistory_PlainUserViaSend(t *testing.T) {
	// Plain user text (no Blocks) should go through Send, not SendRich.
	sess := session.New("plain-test", "test-model", "/workspace")
	sess.Append(session.Entry{Role: "user", Content: "hello"})
	sess.Append(session.Entry{Role: "assistant", Content: "hi"})

	stub := &stubs.StubChatDriver{}
	stub.Start(context.Background(), "") //nolint:errcheck

	ReplayHistory(context.Background(), stub, sess)

	if len(stub.SendLog) != 1 {
		t.Fatalf("plain user entry: SendLog = %d, want 1", len(stub.SendLog))
	}
	if len(stub.SendRichLog) != 0 {
		t.Fatalf("plain user entry should NOT use SendRich, got %d calls", len(stub.SendRichLog))
	}
}
