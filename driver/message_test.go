package driver

import (
	"encoding/json"
	"testing"
)

func TestRichMessage_TextContent_PlainFallback(t *testing.T) {
	m := RichMessage{Role: RoleAssistant, Content: "hello"}
	if m.TextContent() != "hello" {
		t.Fatalf("TextContent = %q, want %q", m.TextContent(), "hello")
	}
}

func TestRichMessage_TextContent_FromBlocks(t *testing.T) {
	m := RichMessage{
		Role: RoleAssistant,
		Blocks: []ContentBlock{
			NewTextBlock("first "),
			NewThinkingBlock("let me think..."),
			NewTextBlock("second"),
		},
	}
	if m.TextContent() != "first second" {
		t.Fatalf("TextContent = %q, want %q", m.TextContent(), "first second")
	}
}

func TestRichMessage_ToolCalls(t *testing.T) {
	input := json.RawMessage(`{"path": "main.go"}`)
	m := RichMessage{
		Role: RoleAssistant,
		Blocks: []ContentBlock{
			NewTextBlock("I'll read the file"),
			NewToolUseBlock("call-1", "Read", input),
			NewToolUseBlock("call-2", "Bash", json.RawMessage(`{"cmd": "go test"}`)),
		},
	}

	if !m.HasToolCalls() {
		t.Fatal("HasToolCalls = false, want true")
	}

	calls := m.ToolCalls()
	if len(calls) != 2 {
		t.Fatalf("ToolCalls = %d, want 2", len(calls))
	}
	if calls[0].Name != "Read" {
		t.Fatalf("call[0].Name = %q, want %q", calls[0].Name, "Read")
	}
	if calls[1].Name != "Bash" {
		t.Fatalf("call[1].Name = %q, want %q", calls[1].Name, "Bash")
	}
}

func TestRichMessage_NoToolCalls(t *testing.T) {
	m := RichMessage{
		Role:   RoleAssistant,
		Blocks: []ContentBlock{NewTextBlock("just text")},
	}
	if m.HasToolCalls() {
		t.Fatal("HasToolCalls = true, want false")
	}
	if len(m.ToolCalls()) != 0 {
		t.Fatal("ToolCalls should be empty")
	}
}

func TestRichMessage_ThinkingContent(t *testing.T) {
	m := RichMessage{
		Role: RoleAssistant,
		Blocks: []ContentBlock{
			NewThinkingBlock("step 1... "),
			NewTextBlock("result"),
			NewThinkingBlock("step 2"),
		},
	}
	if m.ThinkingContent() != "step 1... step 2" {
		t.Fatalf("ThinkingContent = %q", m.ThinkingContent())
	}
}

func TestRichMessage_ToMessage(t *testing.T) {
	m := RichMessage{
		Role:   RoleAssistant,
		Blocks: []ContentBlock{NewTextBlock("converted")},
	}
	plain := m.ToMessage()
	if plain.Role != RoleAssistant {
		t.Fatalf("Role = %q", plain.Role)
	}
	if plain.Content != "converted" {
		t.Fatalf("Content = %q", plain.Content)
	}
}

func TestNewToolResultBlock(t *testing.T) {
	b := NewToolResultBlock("call-1", "file contents here", false)
	if b.Type != BlockToolResult {
		t.Fatalf("Type = %q", b.Type)
	}
	if b.ToolResult.ToolCallID != "call-1" {
		t.Fatalf("ToolCallID = %q", b.ToolResult.ToolCallID)
	}
	if b.ToolResult.IsError {
		t.Fatal("IsError should be false")
	}
}

func TestNewToolResultBlock_Error(t *testing.T) {
	b := NewToolResultBlock("call-2", "permission denied", true)
	if !b.ToolResult.IsError {
		t.Fatal("IsError should be true")
	}
}

func TestStreamEvent_Types(t *testing.T) {
	events := []StreamEvent{
		{Type: EventText, Text: "hello"},
		{Type: EventThinking, Thinking: "reasoning..."},
		{Type: EventToolUse, ToolCall: &ToolCall{ID: "c1", Name: "Read"}},
		{Type: EventDone, Usage: &Usage{InputTokens: 100, OutputTokens: 50}},
		{Type: EventError, Error: "timeout"},
	}

	if events[0].Type != EventText {
		t.Fatal("first event should be text")
	}
	if events[3].Usage.InputTokens != 100 {
		t.Fatal("usage not set")
	}
	if events[4].Error != "timeout" {
		t.Fatal("error not set")
	}
}
