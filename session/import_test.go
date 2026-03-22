package session

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestJSONL(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	f, _ := os.Create(path)
	for _, line := range lines {
		f.WriteString(line + "\n")
	}
	f.Close()
	return path
}

func TestImportClaudeSession_BasicMessages(t *testing.T) {
	path := writeTestJSONL(t,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello"}]},"timestamp":"2026-03-22T10:00:00Z"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi there"}]},"timestamp":"2026-03-22T10:00:01Z"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"fix the bug"}]},"timestamp":"2026-03-22T10:00:02Z"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"done"}]},"timestamp":"2026-03-22T10:00:03Z"}`,
	)

	sess, err := ImportClaudeSession(path, 0)
	if err != nil {
		t.Fatalf("ImportClaudeSession: %v", err)
	}

	if sess.History.Len() != 4 {
		t.Fatalf("history = %d, want 4", sess.History.Len())
	}

	entries := sess.Entries()
	if entries[0].Role != "user" || entries[0].Content != "hello" {
		t.Fatalf("entry[0] = %q/%q", entries[0].Role, entries[0].Content)
	}
	if entries[1].Role != "assistant" || entries[1].Content != "hi there" {
		t.Fatalf("entry[1] = %q/%q", entries[1].Role, entries[1].Content)
	}
}

func TestImportClaudeSession_SkipsNonMessages(t *testing.T) {
	path := writeTestJSONL(t,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello"}]},"timestamp":"2026-03-22T10:00:00Z"}`,
		`{"type":"file-history-snapshot","messageId":"abc","snapshot":{}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]},"timestamp":"2026-03-22T10:00:01Z"}`,
	)

	sess, err := ImportClaudeSession(path, 0)
	if err != nil {
		t.Fatalf("ImportClaudeSession: %v", err)
	}

	if sess.History.Len() != 2 {
		t.Fatalf("history = %d, want 2 (snapshots skipped)", sess.History.Len())
	}
}

func TestImportClaudeSession_SkipsToolResults(t *testing.T) {
	path := writeTestJSONL(t,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"file contents"}]},"timestamp":"2026-03-22T10:00:00Z"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"real message"}]},"timestamp":"2026-03-22T10:00:01Z"}`,
	)

	sess, err := ImportClaudeSession(path, 0)
	if err != nil {
		t.Fatalf("ImportClaudeSession: %v", err)
	}

	// Should have 1 entry (tool_result-only messages skipped)
	if sess.History.Len() != 1 {
		t.Fatalf("history = %d, want 1", sess.History.Len())
	}
	if sess.Entries()[0].Content != "real message" {
		t.Fatalf("content = %q", sess.Entries()[0].Content)
	}
}

func TestImportClaudeSession_Compaction(t *testing.T) {
	// Create 100 messages, compact to fit budget
	var lines []string
	for i := range 100 {
		if i%2 == 0 {
			lines = append(lines, `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"message `+string(rune('A'+i%26))+`"}]},"timestamp":"2026-03-22T10:00:00Z"}`)
		} else {
			lines = append(lines, `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"response `+string(rune('A'+i%26))+`"}]},"timestamp":"2026-03-22T10:00:00Z"}`)
		}
	}
	path := writeTestJSONL(t, lines...)

	// Import with small token budget — should compact
	sess, err := ImportClaudeSession(path, 500)
	if err != nil {
		t.Fatalf("ImportClaudeSession: %v", err)
	}

	// Should have fewer entries than 100 due to compaction
	if sess.History.Len() >= 100 {
		t.Fatalf("history = %d, should be compacted below 100", sess.History.Len())
	}
	if sess.History.Len() == 0 {
		t.Fatal("history should not be empty after compaction")
	}
}

func TestImportClaudeSession_FileNotFound(t *testing.T) {
	_, err := ImportClaudeSession("/nonexistent/path.jsonl", 0)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestImportClaudeSession_SetsMetadata(t *testing.T) {
	path := writeTestJSONL(t,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hi"}]},"timestamp":"2026-03-22T10:00:00Z"}`,
	)

	sess, err := ImportClaudeSession(path, 0)
	if err != nil {
		t.Fatalf("ImportClaudeSession: %v", err)
	}

	if sess.ID == "" {
		t.Fatal("session ID should be set")
	}
	if sess.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
}
