package acceptance

import (
	"testing"

	"github.com/dpopsuev/djinn/session"
)

func TestSession_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store, _ := session.NewStore(dir)

	sess := session.New("lifecycle-test", "claude-opus", "/workspace")
	sess.Name = "lifecycle-test"
	sess.Append(session.Entry{Role: "user", Content: "hello"})
	sess.Append(session.Entry{Role: "assistant", Content: "hi"})

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("lifecycle-test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.History.Len() != 2 {
		t.Fatalf("history = %d, want 2", loaded.History.Len())
	}
}

func TestSession_ResumeRestoresDriverModel(t *testing.T) {
	dir := t.TempDir()
	store, _ := session.NewStore(dir)

	sess := session.New("resume-test", "claude-opus-4-6", "/workspace")
	sess.Name = "resume-test"
	sess.Driver = "claude"
	store.Save(sess)

	loaded, _ := store.Load("resume-test")
	if loaded.Driver != "claude" {
		t.Fatalf("driver = %q, want claude", loaded.Driver)
	}
	if loaded.Model != "claude-opus-4-6" {
		t.Fatalf("model = %q", loaded.Model)
	}
}

func TestSession_ResumeRestoresWorkDirs(t *testing.T) {
	dir := t.TempDir()
	store, _ := session.NewStore(dir)

	sess := session.New("workdir-test", "model", "/workspace")
	sess.Name = "workdir-test"
	sess.WorkDirs = []string{"/home/user/project", "/home/user/lib"}
	store.Save(sess)

	loaded, _ := store.Load("workdir-test")
	if len(loaded.WorkDirs) != 2 {
		t.Fatalf("WorkDirs = %v", loaded.WorkDirs)
	}
}

func TestSession_KillDeletes(t *testing.T) {
	dir := t.TempDir()
	store, _ := session.NewStore(dir)

	sess := session.New("kill-test", "model", "/workspace")
	sess.Name = "kill-test"
	store.Save(sess)

	if err := store.Delete("kill-test"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load("kill-test")
	if err == nil {
		t.Fatal("session should be gone after kill")
	}
}

func TestSession_ListShowsSessions(t *testing.T) {
	dir := t.TempDir()
	store, _ := session.NewStore(dir)

	s1 := session.New("s1", "model", "/work")
	s1.Name = "alpha"
	store.Save(s1)

	s2 := session.New("s2", "model", "/work")
	s2.Name = "beta"
	store.Save(s2)

	list, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %d, want 2", len(list))
	}
}
