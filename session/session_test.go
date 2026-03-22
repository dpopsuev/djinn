package session

import (
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestSession_New(t *testing.T) {
	s := New("sess-1", "claude-sonnet-4-6", "/workspace")
	if s.ID != "sess-1" {
		t.Fatalf("ID = %q", s.ID)
	}
	if s.Model != "claude-sonnet-4-6" {
		t.Fatalf("Model = %q", s.Model)
	}
	if s.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
}

func TestSession_AppendAndEntries(t *testing.T) {
	s := New("sess-1", "claude", "/workspace")
	s.Append(Entry{Role: driver.RoleUser, Content: "hello"})
	s.Append(Entry{Role: driver.RoleAssistant, Content: "hi there"})

	entries := s.Entries()
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Content != "hello" {
		t.Fatalf("first entry = %q", entries[0].Content)
	}
	if entries[1].Timestamp.IsZero() {
		t.Fatal("auto-filled timestamp is zero")
	}
}

func TestHistory_TokenBudget(t *testing.T) {
	h := NewHistory(50) // very tight budget

	h.Append(Entry{Content: "short"})    // ~1 token
	h.Append(Entry{Content: "another"})  // ~1 token

	// Add a long entry that should trigger trimming
	long := ""
	for range 200 {
		long += "word "
	}
	h.Append(Entry{Content: long})

	// Should have trimmed oldest entries but kept at least 1
	if h.Len() < 1 {
		t.Fatal("history should have at least 1 entry after trim")
	}
}

func TestHistory_Unlimited(t *testing.T) {
	h := NewHistory(0) // unlimited

	for range 100 {
		h.Append(Entry{Content: "message"})
	}

	if h.Len() != 100 {
		t.Fatalf("unlimited history = %d, want 100", h.Len())
	}
}

func TestHistory_Clear(t *testing.T) {
	h := NewHistory(0)
	h.Append(Entry{Content: "a"})
	h.Append(Entry{Content: "b"})

	h.Clear()
	if h.Len() != 0 {
		t.Fatalf("after Clear, Len = %d", h.Len())
	}
}

func TestHistory_TotalTokens(t *testing.T) {
	h := NewHistory(0)
	h.Append(Entry{Content: "hello world", TokenCount: 3}) // explicit count
	h.Append(Entry{Content: "foo"})                        // estimated

	if h.TotalTokens() < 3 {
		t.Fatalf("TotalTokens = %d, want >= 3", h.TotalTokens())
	}
}

func TestEstimateTokens(t *testing.T) {
	if estimateTokens("") != 1 {
		t.Fatal("empty string should estimate 1 token")
	}
	// "hello world" = 11 chars / 4 = 2 tokens
	if estimateTokens("hello world") < 1 {
		t.Fatal("short string should estimate >= 1")
	}
}

func TestEntry_TextContent_Plain(t *testing.T) {
	e := Entry{Content: "plain text"}
	if e.TextContent() != "plain text" {
		t.Fatalf("TextContent = %q", e.TextContent())
	}
}

func TestEntry_TextContent_Blocks(t *testing.T) {
	e := Entry{
		Blocks: []driver.ContentBlock{
			driver.NewTextBlock("from blocks"),
		},
	}
	if e.TextContent() != "from blocks" {
		t.Fatalf("TextContent = %q", e.TextContent())
	}
}
