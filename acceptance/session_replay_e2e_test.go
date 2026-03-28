package acceptance

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dpopsuev/djinn/app"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/testkit/stubs"
)

// TestSession_SaveLoadReplay_ToolCallRoundTrip is the E2E acceptance test
// for the entire session lifecycle with tool calls:
//
//	save → load → sanitize → replay → verify proper pairing
//
// This test would have caught BUG-12 (nil tool_use.input), BUG-14 (no sanitize),
// BUG-16 (orphaned tool_use), and BUG-17 (replay drops tool_result blocks).
func TestSession_SaveLoadReplay_ToolCallRoundTrip(t *testing.T) {
	store, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// Build a realistic session with multiple tool call cycles
	sess := session.New("e2e-replay", "claude-sonnet-4-6", "/workspace")
	sess.Name = "e2e-replay"
	sess.Driver = "claude"

	// Turn 1: user prompt
	sess.Append(session.Entry{Role: "user", Content: "list files in this directory"})

	// Turn 2: assistant calls a tool
	sess.Append(session.Entry{
		Role: "assistant",
		Blocks: []driver.ContentBlock{
			driver.NewTextBlock("Let me check."),
			driver.NewToolUseBlock("call-1", "Bash", json.RawMessage(`{"command":"ls"}`)),
		},
	})

	// Turn 3: tool result
	sess.Append(session.Entry{
		Role: "user",
		Blocks: []driver.ContentBlock{
			driver.NewToolResultBlock("call-1", "main.go\ngo.mod\nREADME.md", false),
		},
	})

	// Turn 4: assistant responds
	sess.Append(session.Entry{Role: "assistant", Content: "I see three files: main.go, go.mod, and README.md."})

	// Turn 5: user follow-up
	sess.Append(session.Entry{Role: "user", Content: "read main.go"})

	// Turn 6: assistant calls another tool
	sess.Append(session.Entry{
		Role: "assistant",
		Blocks: []driver.ContentBlock{
			driver.NewToolUseBlock("call-2", "Read", json.RawMessage(`{"path":"main.go"}`)),
		},
	})

	// Turn 7: tool result
	sess.Append(session.Entry{
		Role: "user",
		Blocks: []driver.ContentBlock{
			driver.NewToolResultBlock("call-2", "package main\n\nfunc main() {}", false),
		},
	})

	// Turn 8: assistant responds
	sess.Append(session.Entry{Role: "assistant", Content: "Here's main.go — it has a simple main function."})

	// === SAVE ===
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// === LOAD (triggers Sanitize) ===
	loaded, err := store.Load("e2e-replay")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// === REPLAY into stub driver ===
	stub := &stubs.StubChatDriver{}
	stub.Start(context.Background(), "") //nolint:errcheck // error intentionally ignored

	app.ReplayHistory(context.Background(), stub, loaded)

	// === VERIFY: tool_result blocks were sent via SendRich ===
	if len(stub.SendRichLog) != 2 {
		t.Fatalf("expected 2 SendRich calls (2 tool results), got %d", len(stub.SendRichLog))
	}

	// === VERIFY: each tool_use has a matching tool_result ===
	toolUseIDs := map[string]bool{}
	toolResultIDs := map[string]bool{}

	for _, msg := range stub.SendRichLog {
		for _, block := range msg.Blocks {
			if block.Type == driver.BlockToolResult && block.ToolResult != nil {
				toolResultIDs[block.ToolResult.ToolCallID] = true
			}
		}
	}

	// Check assistant history for tool_use IDs
	for _, msg := range stub.HistoryLog() {
		for _, block := range msg.Blocks {
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				toolUseIDs[block.ToolCall.ID] = true
			}
		}
	}

	for id := range toolUseIDs {
		if !toolResultIDs[id] {
			t.Fatalf("orphaned tool_use %q has no matching tool_result", id)
		}
	}

	// === VERIFY: plain user messages went through Send ===
	if len(stub.SendLog) != 2 {
		t.Fatalf("expected 2 Send calls (2 plain user messages), got %d", len(stub.SendLog))
	}
}

// TestSession_SaveLoadReplay_CorruptToolUse verifies the full chain:
// corrupt session → save → load (sanitize) → replay → no orphans.
// This is the exact bug path from dogfooding: interrupted tool call
// leaves orphaned tool_use, sanitize injects synthetic result,
// replay sends it properly.
func TestSession_SaveLoadReplay_CorruptToolUse(t *testing.T) {
	store, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// Build a CORRUPT session: tool_use with nil input, no tool_result
	sess := session.New("corrupt-e2e", "claude-sonnet-4-6", "/workspace")
	sess.Name = "corrupt-e2e"

	sess.Append(session.Entry{Role: "user", Content: "hello"})
	sess.Append(session.Entry{
		Role: "assistant",
		Blocks: []driver.ContentBlock{
			driver.NewTextBlock("Let me check."),
			driver.NewToolUseBlock("orphan-1", "Bash", nil), // nil input!
		},
	})
	// NO tool_result — session was interrupted here
	sess.Append(session.Entry{Role: "user", Content: "what happened?"})

	// === SAVE corrupt session ===
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// === LOAD (triggers Sanitize — should repair nil input + inject tool_result) ===
	loaded, err := store.Load("corrupt-e2e")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// === REPLAY into stub driver ===
	stub := &stubs.StubChatDriver{}
	stub.Start(context.Background(), "") //nolint:errcheck // error intentionally ignored

	app.ReplayHistory(context.Background(), stub, loaded)

	// === VERIFY: synthetic tool_result was injected and replayed ===
	foundResult := false
	for _, msg := range stub.SendRichLog {
		for _, block := range msg.Blocks {
			if block.Type == driver.BlockToolResult && block.ToolResult != nil {
				if block.ToolResult.ToolCallID == "orphan-1" {
					foundResult = true
					if !block.ToolResult.IsError {
						t.Fatal("synthetic tool_result should be marked as error")
					}
				}
			}
		}
	}
	if !foundResult {
		t.Fatal("corrupt session: orphaned tool_use 'orphan-1' has no matching tool_result after sanitize + replay")
	}

	// === VERIFY: tool_use input was repaired from nil to {} ===
	for _, msg := range stub.HistoryLog() {
		for _, block := range msg.Blocks {
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				if block.ToolCall.ID == "orphan-1" {
					if block.ToolCall.Input == nil || string(block.ToolCall.Input) == "null" {
						t.Fatal("tool_use input should be repaired from nil to {}")
					}
				}
			}
		}
	}
}
