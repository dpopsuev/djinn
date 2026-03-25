package session

import (
	"strings"
	"testing"
)

func TestCompact_KeepsRecent(t *testing.T) {
	sess := New("test", "model", "/work")
	for i := range 10 {
		sess.Append(Entry{Role: "user", Content: "msg " + string(rune('A'+i))})
	}

	before, after := Compact(sess, 4)
	if before != 10 {
		t.Fatalf("before = %d, want 10", before)
	}
	if after > 5 { // 1 summary + 4 recent
		t.Fatalf("after = %d, should be <= 5", after)
	}
	// Last entry should be one of the recent ones
	entries := sess.Entries()
	last := entries[len(entries)-1]
	if last.Content != "msg J" {
		t.Fatalf("last = %q, want msg J", last.Content)
	}
}

func TestCompact_ShortHistoryNoOp(t *testing.T) {
	sess := New("test", "model", "/work")
	sess.Append(Entry{Role: "user", Content: "hello"})

	before, after := Compact(sess, 4)
	if before != after {
		t.Fatalf("short history should be no-op: %d → %d", before, after)
	}
}

func TestCompact_EmptyHistoryNoOp(t *testing.T) {
	sess := New("test", "model", "/work")
	before, after := Compact(sess, 4)
	if before != 0 || after != 0 {
		t.Fatalf("empty: %d → %d", before, after)
	}
}

func TestCompact_SummaryContainsOldContent(t *testing.T) {
	sess := New("test", "model", "/work")
	sess.Append(Entry{Role: "user", Content: "first important message"})
	sess.Append(Entry{Role: "assistant", Content: "first reply"})
	sess.Append(Entry{Role: "user", Content: "second message"})
	sess.Append(Entry{Role: "assistant", Content: "second reply"})
	sess.Append(Entry{Role: "user", Content: "recent 1"})
	sess.Append(Entry{Role: "assistant", Content: "recent 2"})
	sess.Append(Entry{Role: "user", Content: "recent 3"})
	sess.Append(Entry{Role: "assistant", Content: "recent 4"})

	Compact(sess, 4)

	entries := sess.Entries()
	// First entry should be the compacted summary
	if entries[0].Role != "user" {
		t.Fatal("summary should be user role")
	}
	if len(entries[0].Content) == 0 {
		t.Fatal("summary should not be empty")
	}
}

func TestSeedSession_basic(t *testing.T) {
	old := New("old-id", "claude-4", "/workspace")
	old.Driver = "acp"
	old.Mode = "agent"
	old.Workspace = "aeon"
	old.Append(Entry{Role: "user", Content: "msg 1"})
	old.Append(Entry{Role: "assistant", Content: "reply 1"})
	old.Append(Entry{Role: "user", Content: "msg 2"})
	old.Append(Entry{Role: "assistant", Content: "reply 2"})

	seed := SeedSession("new-id", old, "This is a summary of prior work.", 2)

	if seed.ID != "new-id" {
		t.Errorf("ID = %q, want new-id", seed.ID)
	}
	if seed.Model != old.Model {
		t.Errorf("Model = %q, want %q", seed.Model, old.Model)
	}
	if seed.Driver != old.Driver {
		t.Errorf("Driver = %q, want %q", seed.Driver, old.Driver)
	}
	if seed.Workspace != old.Workspace {
		t.Errorf("Workspace = %q, want %q", seed.Workspace, old.Workspace)
	}

	entries := seed.Entries()
	// 1 summary + 2 recent = 3
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}

	// First entry is the summary.
	if !strings.Contains(entries[0].Content, "[Session context]") {
		t.Error("first entry should contain [Session context]")
	}
	if !strings.Contains(entries[0].Content, "summary of prior work") {
		t.Error("first entry should contain summary text")
	}

	// Last two entries are the recent ones from old session.
	if entries[1].Content != "msg 2" {
		t.Errorf("entries[1] = %q, want 'msg 2'", entries[1].Content)
	}
	if entries[2].Content != "reply 2" {
		t.Errorf("entries[2] = %q, want 'reply 2'", entries[2].Content)
	}
}

func TestSeedSession_empty_summary(t *testing.T) {
	old := New("old", "model", "/work")
	old.Append(Entry{Role: "user", Content: "hello"})
	old.Append(Entry{Role: "assistant", Content: "hi"})

	seed := SeedSession("new", old, "", 2)
	entries := seed.Entries()
	// No summary entry, just the 2 recent.
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
}

func TestSeedSession_keepRecent_exceeds_history(t *testing.T) {
	old := New("old", "model", "/work")
	old.Append(Entry{Role: "user", Content: "only"})

	seed := SeedSession("new", old, "summary", 10)
	entries := seed.Entries()
	// 1 summary + 1 entry (all of history).
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
}

func TestExtractSummaryText_basic(t *testing.T) {
	entries := []Entry{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world"},
	}
	text := ExtractSummaryText(entries, 10000)
	if !strings.Contains(text, "user: hello") {
		t.Error("should contain 'user: hello'")
	}
	if !strings.Contains(text, "assistant: world") {
		t.Error("should contain 'assistant: world'")
	}
}

func TestExtractSummaryText_truncates(t *testing.T) {
	entries := []Entry{
		{Role: "user", Content: "short"},
		{Role: "assistant", Content: strings.Repeat("x", 1000)},
	}
	text := ExtractSummaryText(entries, 50)
	// Should only contain the first entry (fits within 50 chars).
	if !strings.Contains(text, "user: short") {
		t.Error("should contain first entry")
	}
	if strings.Contains(text, "assistant:") {
		t.Error("should NOT contain second entry (exceeds limit)")
	}
}
