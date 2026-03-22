package session

import (
	"errors"
	"testing"
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
