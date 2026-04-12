package substrate

import (
	"testing"

	djinncache "github.com/dpopsuev/djinn/cache"
)

func newTestBoard() *NoteBoard {
	return NewNoteBoard(djinncache.NewMemCache())
}

func TestNoteBoard_LeaveAndRead(t *testing.T) {
	board := newTestBoard()

	board.Leave(Note{From: "operator", To: "coder-1", Key: "heads-up", Title: "Code freeze Thursday", Body: "Don't merge after Thursday."})

	// Pending shows the note.
	pending := board.Pending("coder-1")
	if len(pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(pending))
	}
	if pending[0].Title != "Code freeze Thursday" {
		t.Fatalf("title = %q, want 'Code freeze Thursday'", pending[0].Title)
	}

	// Read expires it.
	note, err := board.Read("coder-1", "heads-up")
	if err != nil {
		t.Fatal(err)
	}
	if note.Body != "Don't merge after Thursday." {
		t.Fatalf("body = %q", note.Body)
	}

	// After read, pending is empty.
	if len(board.Pending("coder-1")) != 0 {
		t.Fatal("note should be expired after read")
	}
}

func TestNoteBoard_Broadcast(t *testing.T) {
	board := newTestBoard()
	board.Leave(Note{From: "operator", To: "*", Key: "announcement", Title: "Using Vertex AI", Body: "Switch to Vertex."})

	// All agents see broadcast.
	if len(board.Pending("coder-1")) != 1 {
		t.Fatal("coder-1 should see broadcast")
	}
	if len(board.Pending("gensec")) != 1 {
		t.Fatal("gensec should see broadcast")
	}

	// One agent reads it — expires for everyone.
	_, err := board.Read("coder-1", "announcement")
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Pending("gensec")) != 0 {
		t.Fatal("broadcast should be expired after any agent reads it")
	}
}

func TestNoteBoard_SaveLatest(t *testing.T) {
	board := newTestBoard()
	board.SaveLatest("coder-1", "Was working on TSK-921", "Step 3/5, found fan-in issue.")

	// New agent spawning in coder-1's role sees the latest.
	pending := board.Pending("coder-1")
	if len(pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(pending))
	}
	if pending[0].Key != "_latest" {
		t.Fatalf("key = %q, want _latest", pending[0].Key)
	}

	// Overwrite latest.
	board.SaveLatest("coder-1", "Updated", "Step 4/5.")
	pending = board.Pending("coder-1")
	if len(pending) != 1 {
		t.Fatal("should have exactly 1 _latest (overwritten)")
	}
	if pending[0].Title != "Updated" {
		t.Fatalf("title = %q, want Updated", pending[0].Title)
	}
}

func TestNoteBoard_ReadNotFound(t *testing.T) {
	board := newTestBoard()
	_, err := board.Read("coder-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing note")
	}
}

func TestNoteBoard_CrossAgent(t *testing.T) {
	board := newTestBoard()
	board.Leave(Note{From: "coder-2", To: "coder-1", Key: "coordinate", Title: "I'm in telemetry/", Body: "Don't touch telemetry/, I'm refactoring it."})

	// coder-1 sees coder-2's note.
	pending := board.Pending("coder-1")
	if len(pending) != 1 {
		t.Fatal("coder-1 should see coder-2's note")
	}

	// coder-2 doesn't see its own note to coder-1.
	if len(board.Pending("coder-2")) != 0 {
		t.Fatal("coder-2 should NOT see note addressed to coder-1")
	}
}

func TestNoteBoard_SharesL2WithCache(t *testing.T) {
	// Notes and file cache share the same L2 — no collision because of "note:" prefix.
	l2 := djinncache.NewMemCache()
	board := NewNoteBoard(l2)

	// File data in L2.
	l2.Put("coder-1", "file:/main.go", []byte("package main"))

	// Note in L2.
	board.Leave(Note{From: "operator", To: "coder-1", Key: "heads-up", Title: "Hi"})

	// Pending only returns notes, not file entries.
	pending := board.Pending("coder-1")
	if len(pending) != 1 {
		t.Fatalf("pending = %d, want 1 (note only, not file)", len(pending))
	}

	// File is still in L2.
	if _, ok := l2.Get("coder-1", "file:/main.go"); !ok {
		t.Fatal("file should still be in L2")
	}
}
