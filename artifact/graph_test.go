package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func testGraph() *Graph {
	return NewGraph("test", DefaultRegistry())
}

// --- CRUD ---

func TestAdd_PlanSegment_GeneratesID(t *testing.T) {
	g := testGraph()
	id, err := g.Add(Artifact{Kind: KindPlanSegment, Title: "auth handler", Content: "build it"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "seg-1" {
		t.Errorf("id = %q, want seg-1", id)
	}
}

func TestAdd_Task_GeneratesID(t *testing.T) {
	g := testGraph()
	id, err := g.Add(Artifact{Kind: KindTask, Title: "do something"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "T-001" {
		t.Errorf("id = %q, want T-001", id)
	}
}

func TestAdd_IDIncrements(t *testing.T) {
	g := testGraph()
	id1, _ := g.Add(Artifact{Kind: KindTask, Title: "first"})
	id2, _ := g.Add(Artifact{Kind: KindTask, Title: "second"})
	if id1 != "T-001" || id2 != "T-002" {
		t.Errorf("ids = %q, %q; want T-001, T-002", id1, id2)
	}
}

func TestAdd_ValidatesTemplate(t *testing.T) {
	g := testGraph()
	// Plan-segment without content should fail.
	_, err := g.Add(Artifact{Kind: KindPlanSegment, Title: "no content"})
	if !errors.Is(err, ErrMissingSection) {
		t.Errorf("err = %v, want ErrMissingSection", err)
	}
}

func TestAdd_RejectsUnknownKind(t *testing.T) {
	g := testGraph()
	_, err := g.Add(Artifact{Kind: "alien", Title: "unknown"})
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("err = %v, want ErrTemplateNotFound", err)
	}
}

func TestAdd_NormalizesContent(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "test", Content: "via content field"})
	a, _ := g.Get(id)
	if a.Sections["content"] != "via content field" {
		t.Errorf("Sections[content] = %q, want 'via content field'", a.Sections["content"])
	}
}

func TestAdd_DefaultStatus(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindTask, Title: "test"})
	a, _ := g.Get(id)
	if a.Status != StatusDraft {
		t.Errorf("status = %q, want draft", a.Status)
	}
}

func TestGet_NotFound(t *testing.T) {
	g := testGraph()
	_, err := g.Get("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestAll(t *testing.T) {
	g := testGraph()
	g.Add(Artifact{Kind: KindTask, Title: "a"})
	g.Add(Artifact{Kind: KindTask, Title: "b"})
	all := g.All()
	if len(all) != 2 { //nolint:mnd // expected 2
		t.Errorf("All() = %d, want 2", len(all))
	}
}

// --- Status transitions ---

func TestClaim_HappyPath(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "test", Content: "c", Status: StatusReady})
	if err := g.Claim(id, "executor-1"); err != nil {
		t.Fatal(err)
	}
	a, _ := g.Get(id)
	if a.Status != StatusClaimed || a.Owner != "executor-1" {
		t.Errorf("status=%q owner=%q, want claimed/executor-1", a.Status, a.Owner)
	}
}

func TestClaim_WrongStatus(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "test", Content: "c"}) // draft
	err := g.Claim(id, "executor-1")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

func TestClaim_AlreadyClaimed(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "test", Content: "c", Status: StatusReady})
	g.Claim(id, "executor-1")

	// Try to claim again with different owner — should get InvalidTransition
	// because status is now "claimed", not "ready".
	err := g.Claim(id, "executor-2")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

func TestStart_HappyPath(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "test", Content: "c", Status: StatusReady})
	g.Claim(id, "exec")
	if err := g.Start(id); err != nil {
		t.Fatal(err)
	}
	a, _ := g.Get(id)
	if a.Status != StatusInProgress {
		t.Errorf("status = %q, want in_progress", a.Status)
	}
}

func TestComplete_HappyPath(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "test", Content: "c", Status: StatusReady})
	g.Claim(id, "exec")
	g.Start(id)
	if err := g.Complete(id); err != nil {
		t.Fatal(err)
	}
	a, _ := g.Get(id)
	if a.Status != StatusComplete {
		t.Errorf("status = %q, want complete", a.Status)
	}
}

func TestUpdateStatus_Valid(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindTask, Title: "test"})
	if err := g.UpdateStatus(id, StatusActive); err != nil {
		t.Fatal(err)
	}
	a, _ := g.Get(id)
	if a.Status != StatusActive {
		t.Errorf("status = %q, want active", a.Status)
	}
}

func TestUpdateStatus_InvalidForKind(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindTask, Title: "test"})
	err := g.UpdateStatus(id, StatusClaimed) // not valid for task
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("err = %v, want ErrInvalidStatus", err)
	}
}

func TestUpdateStatus_NotFound(t *testing.T) {
	g := testGraph()
	err := g.UpdateStatus("nonexistent", StatusActive)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// --- Queries ---

func TestReady_DepsComplete(t *testing.T) {
	g := testGraph()
	idA, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "A", Content: "c", Status: StatusReady})
	g.Add(Artifact{Kind: KindPlanSegment, Title: "B", Content: "c", Status: StatusReady, DependsOn: []string{idA}})

	// B depends on A which is ready (not complete) → B should NOT be in Ready().
	ready := g.Ready()
	if len(ready) != 1 || ready[0].Title != "A" {
		t.Errorf("Ready() = %v, want only A", ready)
	}

	// Complete A → B should now be ready.
	g.Claim(idA, "exec")
	g.Start(idA)
	g.Complete(idA)

	ready = g.Ready()
	if len(ready) != 1 || ready[0].Title != "B" {
		t.Errorf("Ready() after A complete = %v, want only B", ready)
	}
}

func TestDraftGaps(t *testing.T) {
	g := testGraph()
	g.Add(Artifact{Kind: KindPlanSegment, Title: "draft", Content: "c"})                      // draft
	g.Add(Artifact{Kind: KindPlanSegment, Title: "ready", Content: "c", Status: StatusReady}) // ready
	gaps := g.DraftGaps()
	if len(gaps) != 1 || gaps[0].Title != "draft" {
		t.Errorf("DraftGaps() = %v, want only draft", gaps)
	}
}

func TestByKind(t *testing.T) {
	g := testGraph()
	g.Add(Artifact{Kind: KindPlanSegment, Title: "seg", Content: "c"})
	g.Add(Artifact{Kind: KindTask, Title: "task"})

	segs := g.ByKind(KindPlanSegment)
	if len(segs) != 1 || segs[0].Title != "seg" {
		t.Errorf("ByKind(plan-segment) = %v", segs)
	}
}

func TestListSorted(t *testing.T) {
	g := testGraph()
	g.Add(Artifact{Kind: KindTask, Title: "b"})
	g.Add(Artifact{Kind: KindTask, Title: "a"})
	list := g.ListSorted()
	if len(list) != 2 || list[0].ID != "T-001" || list[1].ID != "T-002" { //nolint:mnd // expected 2
		t.Errorf("ListSorted() = %v", list)
	}
}

// --- Cascade ---

func TestCascade_Dependency(t *testing.T) {
	g := testGraph()
	idA, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "A", Content: "c", Status: StatusComplete})
	idB, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "B", Content: "c", Status: StatusReady, DependsOn: []string{idA}})

	affected := g.Cascade(idA)
	if len(affected) != 1 || affected[0] != idB {
		t.Errorf("Cascade = %v, want [%s]", affected, idB)
	}
	b, _ := g.Get(idB)
	if b.Status != StatusInvalidated {
		t.Errorf("B status = %q, want invalidated", b.Status)
	}
}

func TestCascade_SpatialOverlap(t *testing.T) {
	g := testGraph()
	idA, _ := g.Add(Artifact{
		Kind: KindPlanSegment, Title: "A", Content: "c", Status: StatusComplete,
		Components: ComponentMap{Files: []string{"auth/handler.go"}},
	})
	idB, _ := g.Add(Artifact{
		Kind: KindPlanSegment, Title: "B", Content: "c", Status: StatusReady,
		Components: ComponentMap{Files: []string{"auth/handler.go", "auth/middleware.go"}},
	})

	affected := g.Cascade(idA)
	if len(affected) != 1 || affected[0] != idB {
		t.Errorf("Cascade = %v, want [%s]", affected, idB)
	}
}

func TestOverlaps(t *testing.T) {
	g := testGraph()
	idA, _ := g.Add(Artifact{
		Kind: KindPlanSegment, Title: "A", Content: "c",
		Components: ComponentMap{Files: []string{"shared.go", "a_only.go"}},
	})
	idB, _ := g.Add(Artifact{
		Kind: KindPlanSegment, Title: "B", Content: "c",
		Components: ComponentMap{Files: []string{"shared.go", "b_only.go"}},
	})

	shared := g.Overlaps(idA, idB)
	if len(shared) != 1 || shared[0] != "shared.go" {
		t.Errorf("Overlaps = %v, want [shared.go]", shared)
	}
}

// --- HITL ---

func TestFillDraft(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindPlanSegment, Title: "test", Content: "placeholder"})
	if err := g.FillDraft(id, "real content"); err != nil {
		t.Fatal(err)
	}
	a, _ := g.Get(id)
	if a.Status != StatusReady {
		t.Errorf("status = %q, want ready", a.Status)
	}
	if a.Content != "real content" {
		t.Errorf("content = %q, want 'real content'", a.Content)
	}
}

func TestAnnotate(t *testing.T) {
	g := testGraph()
	id, _ := g.Add(Artifact{Kind: KindTask, Title: "test"})
	if err := g.Annotate(id, Annotation{Kind: "+", Comment: "LGTM"}); err != nil {
		t.Fatal(err)
	}
	a, _ := g.Get(id)
	if len(a.Annotations) != 1 || a.Annotations[0].Kind != "+" {
		t.Errorf("annotations = %v", a.Annotations)
	}
}

func TestInject(t *testing.T) {
	g := testGraph()
	parentID, _ := g.Add(Artifact{Kind: KindTask, Title: "parent"})
	childID, err := g.Inject(parentID, Artifact{Kind: KindTask, Title: "child"})
	if err != nil {
		t.Fatal(err)
	}
	child, _ := g.Get(childID)
	if len(child.DependsOn) != 1 || child.DependsOn[0] != parentID {
		t.Errorf("child.DependsOn = %v, want [%s]", child.DependsOn, parentID)
	}
}

func TestReorder(t *testing.T) {
	g := testGraph()
	idA, _ := g.Add(Artifact{Kind: KindTask, Title: "A"})
	idB, _ := g.Add(Artifact{Kind: KindTask, Title: "B"})
	idC, _ := g.Add(Artifact{Kind: KindTask, Title: "C", DependsOn: []string{idA}})

	if err := g.Reorder(idC, []string{idB}); err != nil {
		t.Fatal(err)
	}
	c, _ := g.Get(idC)
	if len(c.DependsOn) != 1 || c.DependsOn[0] != idB {
		t.Errorf("C.DependsOn = %v, want [%s]", c.DependsOn, idB)
	}
}

// --- TopoSort ---

func TestTopoSort_NoDeps(t *testing.T) {
	g := testGraph()
	g.Add(Artifact{Kind: KindTask, Title: "B"})
	g.Add(Artifact{Kind: KindTask, Title: "A"})
	sorted := g.TopoSort()
	if len(sorted) != 2 { //nolint:mnd // expected 2
		t.Fatalf("TopoSort = %d, want 2", len(sorted))
	}
	// Deterministic: sorted by ID.
	if sorted[0].ID != "T-001" {
		t.Errorf("first = %q, want T-001", sorted[0].ID)
	}
}

func TestTopoSort_WithDeps(t *testing.T) {
	g := testGraph()
	idA, _ := g.Add(Artifact{Kind: KindTask, Title: "A"})
	g.Add(Artifact{Kind: KindTask, Title: "B", DependsOn: []string{idA}})

	sorted := g.TopoSort()
	if sorted[0].Title != "A" || sorted[1].Title != "B" {
		t.Errorf("TopoSort = [%s, %s], want [A, B]", sorted[0].Title, sorted[1].Title)
	}
}

func TestTopoSort_CycleDoesNotPanic(t *testing.T) {
	g := testGraph()
	idA, _ := g.Add(Artifact{Kind: KindTask, Title: "A"})
	idB, _ := g.Add(Artifact{Kind: KindTask, Title: "B", DependsOn: []string{idA}})
	// Create cycle: A depends on B.
	a, _ := g.Get(idA)
	a.DependsOn = []string{idB}

	sorted := g.TopoSort()
	if len(sorted) != 2 { //nolint:mnd // expected 2
		t.Fatalf("TopoSort with cycle = %d, want 2", len(sorted))
	}
}

// --- Persistence ---

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")

	g := testGraph()
	g.Add(Artifact{Kind: KindTask, Title: "persisted"})
	if err := g.Save(path); err != nil {
		t.Fatal(err)
	}

	g2 := testGraph()
	if err := g2.Load(path); err != nil {
		t.Fatal(err)
	}
	all := g2.All()
	if len(all) != 1 || all[0].Title != "persisted" {
		t.Errorf("loaded = %v", all)
	}
}

func TestLoad_Missing(t *testing.T) {
	g := testGraph()
	err := g.Load("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error loading missing file")
	}
}

func TestSave_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "graph.json")

	g := testGraph()
	g.Add(Artifact{Kind: KindTask, Title: "test"})
	if err := g.Save(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}
