package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTaskStore_Create(t *testing.T) {
	store := NewTaskStore("")
	task := store.Create("implement feature X")

	if task.ID != "T-001" {
		t.Fatalf("ID = %q, want T-001", task.ID)
	}
	if task.Title != "implement feature X" {
		t.Fatalf("Title = %q, want 'implement feature X'", task.Title)
	}
	if task.Status != StatusPending {
		t.Fatalf("Status = %q, want pending", task.Status)
	}
	if task.Created.IsZero() {
		t.Fatal("Created should be set")
	}
}

func TestTaskStore_CreateIncrements(t *testing.T) {
	store := NewTaskStore("")
	t1 := store.Create("first")
	t2 := store.Create("second")
	t3 := store.Create("third")

	if t1.ID != "T-001" || t2.ID != "T-002" || t3.ID != "T-003" {
		t.Fatalf("IDs = %s, %s, %s — want T-001, T-002, T-003", t1.ID, t2.ID, t3.ID)
	}
}

func TestTaskStore_GetAndUpdate(t *testing.T) {
	store := NewTaskStore("")
	task := store.Create("task A")

	got, ok := store.Get(task.ID)
	if !ok || got.Title != "task A" {
		t.Fatal("Get returned wrong task or not found")
	}

	if err := store.Update(task.ID, StatusActive); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = store.Get(task.ID)
	if got.Status != StatusActive {
		t.Fatalf("Status = %q, want active", got.Status)
	}
}

func TestTaskStore_UpdateNotFound(t *testing.T) {
	store := NewTaskStore("")
	if err := store.Update("T-999", StatusDone); err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestTaskStore_UpdateInvalidStatus(t *testing.T) {
	store := NewTaskStore("")
	task := store.Create("task")
	if err := store.Update(task.ID, "invalid"); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestTaskStore_List(t *testing.T) {
	store := NewTaskStore("")
	store.Create("alpha")
	store.Create("beta")
	store.Create("gamma")

	list := store.List()
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}
	// Should be sorted by ID.
	if list[0].ID != "T-001" || list[1].ID != "T-002" || list[2].ID != "T-003" {
		t.Fatalf("unexpected order: %s, %s, %s", list[0].ID, list[1].ID, list[2].ID)
	}
}

func TestTaskStore_TopoSort_NoDeps(t *testing.T) {
	store := NewTaskStore("")
	store.Create("a")
	store.Create("b")
	store.Create("c")

	sorted := store.TopoSort()
	if len(sorted) != 3 {
		t.Fatalf("len = %d, want 3", len(sorted))
	}
}

func TestTaskStore_TopoSort_WithDeps(t *testing.T) {
	store := NewTaskStore("")
	t1 := store.Create("write spec")
	t2 := store.Create("implement")
	t3 := store.Create("test")

	// implement depends on write spec; test depends on implement
	t2.DependsOn = []string{t1.ID}
	t3.DependsOn = []string{t2.ID}

	sorted := store.TopoSort()
	if len(sorted) != 3 {
		t.Fatalf("len = %d, want 3", len(sorted))
	}

	// Build index.
	idx := make(map[string]int, 3)
	for i, task := range sorted {
		idx[task.ID] = i
	}

	if idx[t1.ID] >= idx[t2.ID] {
		t.Fatalf("spec (%d) should come before implement (%d)", idx[t1.ID], idx[t2.ID])
	}
	if idx[t2.ID] >= idx[t3.ID] {
		t.Fatalf("implement (%d) should come before test (%d)", idx[t2.ID], idx[t3.ID])
	}
}

func TestTaskStore_TopoSort_CycleDoesNotPanic(t *testing.T) {
	store := NewTaskStore("")
	t1 := store.Create("a")
	t2 := store.Create("b")

	// Circular dependency.
	t1.DependsOn = []string{t2.ID}
	t2.DependsOn = []string{t1.ID}

	sorted := store.TopoSort()
	if len(sorted) != 2 {
		t.Fatalf("len = %d, want 2 (cycle should still return all tasks)", len(sorted))
	}
}

func TestTaskStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	// Save.
	store := NewTaskStore(path)
	store.Create("first task")
	store.Create("second task")
	if err := store.Update("T-001", StatusDone); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}

	// Load into fresh store.
	store2 := NewTaskStore(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	list := store2.List()
	if len(list) != 2 {
		t.Fatalf("loaded %d tasks, want 2", len(list))
	}
	if list[0].Status != StatusDone {
		t.Fatalf("T-001 status = %q, want done", list[0].Status)
	}
	if list[1].Status != StatusPending {
		t.Fatalf("T-002 status = %q, want pending", list[1].Status)
	}

	// NextID preserved — next create should be T-003.
	t3 := store2.Create("third task")
	if t3.ID != "T-003" {
		t.Fatalf("ID = %q, want T-003 (nextID should be preserved)", t3.ID)
	}
}

func TestTaskStore_LoadMissing(t *testing.T) {
	store := NewTaskStore("/nonexistent/path/tasks.json")
	if err := store.Load(); err == nil {
		t.Fatal("expected error loading nonexistent file")
	}
}
