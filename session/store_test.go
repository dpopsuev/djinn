package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestStore_SaveAndLoad(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	sess := New("sess-1", "claude-sonnet-4-6", "/workspace")
	sess.Name = "myproject"
	sess.Driver = "claude"
	sess.Append(Entry{Role: "user", Content: "hello"})
	sess.Append(Entry{Role: "assistant", Content: "hi there"})

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("myproject")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != "sess-1" {
		t.Fatalf("ID = %q", loaded.ID)
	}
	if loaded.Name != "myproject" {
		t.Fatalf("Name = %q", loaded.Name)
	}
	if loaded.Driver != "claude" {
		t.Fatalf("Driver = %q", loaded.Driver)
	}
	if loaded.Model != "claude-sonnet-4-6" {
		t.Fatalf("Model = %q", loaded.Model)
	}
	if loaded.History.Len() != 2 {
		t.Fatalf("History.Len = %d, want 2", loaded.History.Len())
	}
	if loaded.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt is zero")
	}
}

func TestStore_LoadByID(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	sess := New("sess-no-name", "model", "/work")
	// No Name set — should save/load by ID
	store.Save(sess)

	loaded, err := store.Load("sess-no-name")
	if err != nil {
		t.Fatalf("Load by ID: %v", err)
	}
	if loaded.ID != "sess-no-name" {
		t.Fatalf("ID = %q", loaded.ID)
	}
}

func TestStore_LoadNotFound(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	_, err := store.Load("nonexistent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestStore_List(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	s1 := New("s1", "claude", "/a")
	s1.Name = "alpha"
	s1.Append(Entry{Content: "one"})

	s2 := New("s2", "ollama", "/b")
	s2.Name = "beta"
	s2.Append(Entry{Content: "one"})
	s2.Append(Entry{Content: "two"})

	store.Save(s1)
	store.Save(s2)

	list, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("List = %d, want 2", len(list))
	}

	// Most recently updated first
	if list[0].Name != "beta" {
		t.Fatalf("first = %q, want beta (most recent)", list[0].Name)
	}
	if list[0].Turns != 2 {
		t.Fatalf("beta turns = %d, want 2", list[0].Turns)
	}
}

func TestStore_Delete(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	sess := New("s1", "model", "/work")
	sess.Name = "deleteme"
	store.Save(sess)

	if err := store.Delete("deleteme"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load("deleteme")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatal("should be deleted")
	}
}

func TestStore_DeleteNotFound(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	err := store.Delete("ghost")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestStore_Overwrite(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	sess := New("s1", "model", "/work")
	sess.Name = "overwrite"
	sess.Append(Entry{Content: "v1"})
	store.Save(sess)

	sess.Append(Entry{Content: "v2"})
	store.Save(sess)

	loaded, _ := store.Load("overwrite")
	if loaded.History.Len() != 2 {
		t.Fatalf("after overwrite, turns = %d, want 2", loaded.History.Len())
	}
}

func TestStore_Load_SanitizesNilToolUseInput(t *testing.T) {
	// DJN-BUG-14: sessions with nil tool_use.input should be repaired on load.
	dir := t.TempDir()

	// Write raw JSON with null tool_use input directly to simulate corrupt file.
	rawJSON := `{
		"id": "corrupt",
		"name": "corrupt",
		"model": "test",
		"work_dir": "/workspace",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
		"history": [
			{"role": "user", "content": "hello"},
			{"role": "assistant", "blocks": [
				{"type": "tool_use", "tool_call": {"id": "c1", "name": "Bash", "input": null}}
			]},
			{"role": "user", "blocks": [
				{"type": "tool_result", "tool_result": {"tool_call_id": "c1", "output": "ok"}}
			]}
		]
	}`
	os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte(rawJSON), 0o600) //nolint:errcheck // best-effort write

	store, _ := NewStore(dir)
	loaded, err := store.Load("corrupt")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, entry := range loaded.Entries() {
		for _, block := range entry.Blocks {
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				input := block.ToolCall.Input
				if input == nil || string(input) == "null" {
					t.Fatal("BUG-14: tool_use.input is nil or 'null' after load — should be repaired to {}")
				}
				var parsed any
				if err := json.Unmarshal(input, &parsed); err != nil {
					t.Fatalf("BUG-14: repaired input is not valid JSON: %v", err)
				}
			}
		}
	}
}

func TestStore_Load_SanitizesLargeSession(t *testing.T) {
	// DJN-BUG-14: oversized sessions should be compacted on load.
	dir := t.TempDir()
	store, _ := NewStore(dir)

	sess := New("big", "test", "/workspace")
	sess.Name = "big"
	for i := range 300 {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		sess.Append(Entry{Role: role, Content: "message " + string(rune('0'+i%10))})
	}

	store.Save(sess) //nolint:errcheck // best-effort persist

	loaded, err := store.Load("big")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Session with 300 entries should be compacted to a reasonable size
	if loaded.History.Len() > 250 {
		t.Fatalf("BUG-14: session with 300 entries not compacted on load, got %d", loaded.History.Len())
	}
}

func TestStore_Load_SanitizesOrphanedToolUse(t *testing.T) {
	// DJN-BUG-16: tool_use without matching tool_result in next message.
	// Vertex requires strict pairing. Sanitize should inject synthetic results.
	dir := t.TempDir()

	rawJSON := `{
		"id": "orphan",
		"name": "orphan",
		"model": "test",
		"work_dir": "/workspace",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
		"history": [
			{"role": "user", "content": "hello"},
			{"role": "assistant", "blocks": [
				{"type": "text", "text": "Let me check."},
				{"type": "tool_use", "tool_call": {"id": "orphan-1", "name": "Bash", "input": "{}"}}
			]},
			{"role": "user", "content": "what happened?"}
		]
	}`
	os.WriteFile(filepath.Join(dir, "orphan.json"), []byte(rawJSON), 0o600) //nolint:errcheck // best-effort write

	store, _ := NewStore(dir)
	loaded, err := store.Load("orphan")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// After sanitize, there should be a synthetic tool_result between
	// the assistant message (with tool_use) and the user message.
	entries := loaded.Entries()
	foundResult := false
	for _, entry := range entries {
		for _, block := range entry.Blocks {
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
		t.Fatal("BUG-16: orphaned tool_use 'orphan-1' has no matching tool_result after sanitize")
	}
}

func TestStore_Load_SanitizesNullStringInput(t *testing.T) {
	// DJN-BUG-18: json.RawMessage("null") (4 bytes, literal string "null")
	// is different from nil. Sanitize must catch BOTH.
	// This is what actually happens when JSON has "input": null —
	// json.Unmarshal produces json.RawMessage("null"), not nil.
	dir := t.TempDir()

	rawJSON := `{
		"id": "null-str",
		"name": "null-str",
		"model": "test",
		"work_dir": "/workspace",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
		"history": [
			{"role": "user", "content": "hello"},
			{"role": "assistant", "blocks": [
				{"type": "tool_use", "tool_call": {"id": "c1", "name": "Bash", "input": null}}
			]},
			{"role": "user", "blocks": [
				{"type": "tool_result", "tool_result": {"tool_call_id": "c1", "output": "ok"}}
			]}
		]
	}`
	os.WriteFile(filepath.Join(dir, "null-str.json"), []byte(rawJSON), 0o600) //nolint:errcheck // best-effort write

	store, _ := NewStore(dir)
	loaded, err := store.Load("null-str")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// After Load (which calls Sanitize), the tool_use input must be {}
	for _, entry := range loaded.Entries() {
		for _, block := range entry.Blocks {
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				input := block.ToolCall.Input
				inputStr := string(input)
				if input == nil || inputStr == "null" || inputStr == "" {
					t.Fatalf("BUG-18: after Load+Sanitize, tool_use.input = %q (should be {})", inputStr)
				}
				if inputStr != "{}" {
					t.Fatalf("BUG-18: after Load+Sanitize, tool_use.input = %q (expected {})", inputStr)
				}
			}
		}
	}
}

func TestStore_Archive(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	sess := New("s1", "claude-4", "/workspace")
	sess.Name = "archiveme"
	sess.Append(Entry{Role: "user", Content: "hello"})
	store.Save(sess)

	if err := store.Archive(sess); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	// Should no longer appear in active list.
	list, _ := store.List()
	for _, s := range list {
		if s.Name == "archiveme" {
			t.Fatal("archived session should not appear in List()")
		}
	}

	// Should appear in archived list.
	archived, err := store.ListArchived()
	if err != nil {
		t.Fatalf("ListArchived: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("ListArchived = %d, want 1", len(archived))
	}
	if archived[0].Name != "archiveme" {
		t.Fatalf("archived name = %q", archived[0].Name)
	}
}

func TestStore_LoadArchived(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	sess := New("s1", "model", "/work")
	sess.Name = "loadme"
	sess.Append(Entry{Role: "user", Content: "context"})
	store.Save(sess)
	store.Archive(sess)

	loaded, err := store.LoadArchived("loadme")
	if err != nil {
		t.Fatalf("LoadArchived: %v", err)
	}
	if loaded.ID != "s1" {
		t.Errorf("ID = %q, want s1", loaded.ID)
	}
	if loaded.History.Len() != 1 {
		t.Errorf("History.Len = %d, want 1", loaded.History.Len())
	}
}

func TestStore_ListArchived_empty(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	archived, err := store.ListArchived()
	if err != nil {
		t.Fatalf("ListArchived: %v", err)
	}
	if len(archived) != 0 {
		t.Fatalf("ListArchived = %d, want 0", len(archived))
	}
}

func TestStore_LoadArchived_notFound(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	_, err := store.LoadArchived("ghost")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestImport_NilToolUseInputDefaultsToEmptyObject(t *testing.T) {
	// DJN-BUG-15: session file with null tool_use.input should be
	// repaired through the sanitize-on-load path (defense in depth).
	dir := t.TempDir()

	rawJSON := `{
		"id": "imported-test",
		"name": "imported-test",
		"model": "test",
		"work_dir": "/workspace",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
		"history": [
			{"role": "user", "content": "hello"},
			{"role": "assistant", "blocks": [
				{"type": "tool_use", "tool_call": {"id": "c1", "name": "Read", "input": null}}
			]},
			{"role": "user", "blocks": [
				{"type": "tool_result", "tool_result": {"tool_call_id": "c1", "output": "ok"}}
			]}
		]
	}`
	os.WriteFile(filepath.Join(dir, "imported-test.json"), []byte(rawJSON), 0o600) //nolint:errcheck // best-effort write

	store, _ := NewStore(dir)
	loaded, err := store.Load("imported-test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, entry := range loaded.Entries() {
		for _, block := range entry.Blocks {
			if block.Type == driver.BlockToolUse && block.ToolCall != nil {
				input := block.ToolCall.Input
				if input == nil || string(input) == "null" {
					t.Fatal("BUG-15: null tool_use.input not repaired on load")
				}
			}
		}
	}
}
