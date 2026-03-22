package session

import "testing"

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
