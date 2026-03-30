package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/artifact"
)

func TestGraph_CreateTask(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	id, err := g.Add(artifact.Artifact{
		Kind:   artifact.KindTask,
		Title:  "implement feature X",
		Status: artifact.StatusPending,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	task, err := g.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if task.ID != "T-001" {
		t.Fatalf("ID = %q, want T-001", task.ID)
	}
	if task.Title != "implement feature X" {
		t.Fatalf("Title = %q, want 'implement feature X'", task.Title)
	}
	if task.Status != artifact.StatusPending {
		t.Fatalf("Status = %q, want pending", task.Status)
	}
	if task.Created.IsZero() {
		t.Fatal("Created should be set")
	}
}

func TestGraph_CreateTaskIncrements(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	id1, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "first", Status: artifact.StatusPending})
	id2, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "second", Status: artifact.StatusPending})
	id3, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "third", Status: artifact.StatusPending})

	if id1 != "T-001" || id2 != "T-002" || id3 != "T-003" {
		t.Fatalf("IDs = %s, %s, %s — want T-001, T-002, T-003", id1, id2, id3)
	}
}

func TestGraph_GetAndUpdateStatus(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	id, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "task A", Status: artifact.StatusPending})

	got, err := g.Get(id)
	if err != nil || got.Title != "task A" {
		t.Fatal("Get returned wrong task or not found")
	}

	if err := g.UpdateStatus(id, artifact.StatusActive); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, _ = g.Get(id)
	if got.Status != artifact.StatusActive {
		t.Fatalf("Status = %q, want active", got.Status)
	}
}

func TestGraph_UpdateStatusNotFound(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	if err := g.UpdateStatus("T-999", artifact.StatusDone); err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestGraph_UpdateStatusInvalid(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	id, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "task", Status: artifact.StatusPending})
	if err := g.UpdateStatus(id, "invalid"); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestGraph_ListSorted(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "alpha", Status: artifact.StatusPending})
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "beta", Status: artifact.StatusPending})
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "gamma", Status: artifact.StatusPending})

	list := g.ListSorted()
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}
	// Should be sorted by ID.
	if list[0].ID != "T-001" || list[1].ID != "T-002" || list[2].ID != "T-003" {
		t.Fatalf("unexpected order: %s, %s, %s", list[0].ID, list[1].ID, list[2].ID)
	}
}

func TestGraph_TopoSort_NoDeps(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "a", Status: artifact.StatusPending})
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "b", Status: artifact.StatusPending})
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "c", Status: artifact.StatusPending})

	sorted := g.TopoSort()
	if len(sorted) != 3 {
		t.Fatalf("len = %d, want 3", len(sorted))
	}
}

func TestGraph_TopoSort_WithDeps(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	id1, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "write spec", Status: artifact.StatusPending})
	id2, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "implement", Status: artifact.StatusPending, DependsOn: []string{id1}})
	id3, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "test", Status: artifact.StatusPending, DependsOn: []string{id2}})

	sorted := g.TopoSort()
	if len(sorted) != 3 {
		t.Fatalf("len = %d, want 3", len(sorted))
	}

	// Build index.
	idx := make(map[string]int, 3)
	for i, task := range sorted {
		idx[task.ID] = i
	}

	if idx[id1] >= idx[id2] {
		t.Fatalf("spec (%d) should come before implement (%d)", idx[id1], idx[id2])
	}
	if idx[id2] >= idx[id3] {
		t.Fatalf("implement (%d) should come before test (%d)", idx[id2], idx[id3])
	}
}

func TestGraph_TopoSort_CycleDoesNotPanic(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	id1, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "a", Status: artifact.StatusPending, DependsOn: []string{"T-002"}})
	id2, _ := g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "b", Status: artifact.StatusPending, DependsOn: []string{id1}})
	_ = id2

	sorted := g.TopoSort()
	if len(sorted) != 2 {
		t.Fatalf("len = %d, want 2 (cycle should still return all tasks)", len(sorted))
	}
}

func TestGraph_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	// Save.
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "first task", Status: artifact.StatusPending})
	g.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "second task", Status: artifact.StatusPending})
	if err := g.UpdateStatus("T-001", artifact.StatusDone); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if err := g.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}

	// Load into fresh graph.
	g2 := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	if err := g2.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	list := g2.ListSorted()
	if len(list) != 2 {
		t.Fatalf("loaded %d tasks, want 2", len(list))
	}
	if list[0].Status != artifact.StatusDone {
		t.Fatalf("T-001 status = %q, want done", list[0].Status)
	}
	if list[1].Status != artifact.StatusPending {
		t.Fatalf("T-002 status = %q, want pending", list[1].Status)
	}

	// Counter preserved — next create should be T-003.
	id3, _ := g2.Add(artifact.Artifact{Kind: artifact.KindTask, Title: "third task", Status: artifact.StatusPending})
	if id3 != "T-003" {
		t.Fatalf("ID = %q, want T-003 (counter should be preserved)", id3)
	}
}

func TestGraph_LoadMissing(t *testing.T) {
	g := artifact.NewGraph("tasks", artifact.DefaultRegistry())
	if err := g.Load("/nonexistent/path/tasks.json"); err == nil {
		t.Fatal("expected error loading nonexistent file")
	}
}
