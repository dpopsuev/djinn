package plan

import "testing"

func TestAddSegmentAndGet(t *testing.T) {
	pg := NewPlanGraph("test plan")
	id := pg.AddSegment(Segment{Title: "Build auth"})

	s, err := pg.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "Build auth" {
		t.Errorf("title = %q", s.Title)
	}
	if s.Status != StatusDraft {
		t.Errorf("status = %s, want draft", s.Status)
	}
}

func TestClaimAndStart(t *testing.T) {
	pg := NewPlanGraph("test")
	id := pg.AddSegment(Segment{Title: "Task", Status: StatusReady})

	if err := pg.Claim(id, "executor-1"); err != nil {
		t.Fatal(err)
	}
	s, _ := pg.Get(id)
	if s.Owner != "executor-1" {
		t.Errorf("owner = %q", s.Owner)
	}
	if s.Status != StatusClaimed {
		t.Errorf("status = %s, want claimed", s.Status)
	}

	if err := pg.Start(id); err != nil {
		t.Fatal(err)
	}
	s, _ = pg.Get(id)
	if s.Status != StatusInProgress {
		t.Errorf("status = %s, want in_progress", s.Status)
	}
}

func TestComplete(t *testing.T) {
	pg := NewPlanGraph("test")
	id := pg.AddSegment(Segment{Title: "Task", Status: StatusReady})
	pg.Claim(id, "e1")
	pg.Start(id)

	if err := pg.Complete(id); err != nil {
		t.Fatal(err)
	}
	s, _ := pg.Get(id)
	if s.Status != StatusComplete {
		t.Errorf("status = %s, want complete", s.Status)
	}
}

func TestClaimAlreadyClaimed(t *testing.T) {
	pg := NewPlanGraph("test")
	id := pg.AddSegment(Segment{Title: "Task", Status: StatusReady})
	pg.Claim(id, "e1")

	err := pg.Claim(id, "e2")
	if err == nil {
		t.Error("should fail: already claimed")
	}
}

func TestReady(t *testing.T) {
	pg := NewPlanGraph("test")
	a := pg.AddSegment(Segment{Title: "A", Status: StatusComplete})
	pg.AddSegment(Segment{Title: "B", Status: StatusReady, DependsOn: []string{a}})
	pg.AddSegment(Segment{Title: "C", Status: StatusReady, DependsOn: []string{"nonexistent"}})

	ready := pg.Ready()
	if len(ready) != 1 {
		t.Fatalf("expected 1 ready, got %d", len(ready))
	}
	if ready[0].Title != "B" {
		t.Errorf("ready segment = %q, want B", ready[0].Title)
	}
}

func TestDraftGaps(t *testing.T) {
	pg := NewPlanGraph("test")
	pg.AddSegment(Segment{Title: "Ready", Status: StatusReady})
	pg.AddSegment(Segment{Title: "Draft1", Status: StatusDraft})
	pg.AddSegment(Segment{Title: "Draft2", Status: StatusDraft})

	gaps := pg.DraftGaps()
	if len(gaps) != 2 {
		t.Errorf("expected 2 draft gaps, got %d", len(gaps))
	}
}

func TestTopoSort(t *testing.T) {
	pg := NewPlanGraph("test")
	a := pg.AddSegment(Segment{Title: "A", Status: StatusReady})
	b := pg.AddSegment(Segment{Title: "B", Status: StatusReady, DependsOn: []string{a}})
	pg.AddSegment(Segment{Title: "C", Status: StatusReady, DependsOn: []string{b}})

	sorted := pg.TopoSort()
	if len(sorted) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(sorted))
	}
	if sorted[0].Title != "A" {
		t.Errorf("first = %q, want A", sorted[0].Title)
	}
}

func TestCascadeDependency(t *testing.T) {
	pg := NewPlanGraph("test")
	a := pg.AddSegment(Segment{Title: "A", Status: StatusComplete})
	b := pg.AddSegment(Segment{Title: "B", Status: StatusInProgress, DependsOn: []string{a}})
	_ = b

	affected := pg.Cascade(a)
	if len(affected) != 1 {
		t.Fatalf("expected 1 affected, got %d", len(affected))
	}

	s, _ := pg.Get(b)
	if s.Status != StatusInvalidated {
		t.Errorf("B status = %s, want invalidated", s.Status)
	}
}

func TestCascadeSpatialOverlap(t *testing.T) {
	pg := NewPlanGraph("test")
	a := pg.AddSegment(Segment{
		Title:      "A",
		Status:     StatusComplete,
		Components: ComponentMap{Files: []string{"auth/handler.go"}},
	})
	pg.AddSegment(Segment{
		Title:      "B",
		Status:     StatusInProgress,
		Components: ComponentMap{Files: []string{"auth/handler.go", "auth/store.go"}},
	})

	affected := pg.Cascade(a)
	if len(affected) != 1 {
		t.Fatalf("expected 1 spatially affected, got %d", len(affected))
	}
}

func TestOverlaps(t *testing.T) {
	pg := NewPlanGraph("test")
	a := pg.AddSegment(Segment{
		Title:      "A",
		Components: ComponentMap{Files: []string{"auth/handler.go", "auth/store.go"}},
	})
	b := pg.AddSegment(Segment{
		Title:      "B",
		Components: ComponentMap{Files: []string{"auth/handler.go", "config/app.go"}},
	})

	shared := pg.Overlaps(a, b)
	if len(shared) != 1 || shared[0] != "auth/handler.go" {
		t.Errorf("shared = %v, want [auth/handler.go]", shared)
	}
}

func TestFillDraft(t *testing.T) {
	pg := NewPlanGraph("test")
	id := pg.AddSegment(Segment{Title: "Gap", Status: StatusDraft})

	if err := pg.FillDraft(id, "Now I have content"); err != nil {
		t.Fatal(err)
	}
	s, _ := pg.Get(id)
	if s.Status != StatusReady {
		t.Errorf("status = %s, want ready", s.Status)
	}
	if s.Content != "Now I have content" {
		t.Errorf("content = %q", s.Content)
	}
}

func TestAnnotate(t *testing.T) {
	pg := NewPlanGraph("test")
	id := pg.AddSegment(Segment{Title: "Task"})

	pg.Annotate(id, "+", "looks good")
	pg.Annotate(id, "-", "too broad")

	s, _ := pg.Get(id)
	if len(s.Annotations) != 2 {
		t.Errorf("annotations = %d, want 2", len(s.Annotations))
	}
}

func TestInject(t *testing.T) {
	pg := NewPlanGraph("test")
	a := pg.AddSegment(Segment{Title: "A", Status: StatusReady})

	newID, err := pg.Inject(a, Segment{Title: "Injected"})
	if err != nil {
		t.Fatal(err)
	}

	s, _ := pg.Get(newID)
	if s.Title != "Injected" {
		t.Errorf("title = %q", s.Title)
	}
	if len(s.DependsOn) != 1 || s.DependsOn[0] != a {
		t.Errorf("depends_on = %v, want [%s]", s.DependsOn, a)
	}
}

func TestReorder(t *testing.T) {
	pg := NewPlanGraph("test")
	a := pg.AddSegment(Segment{Title: "A"})
	b := pg.AddSegment(Segment{Title: "B"})
	c := pg.AddSegment(Segment{Title: "C", DependsOn: []string{a}})

	if err := pg.Reorder(c, []string{b}); err != nil {
		t.Fatal(err)
	}
	s, _ := pg.Get(c)
	if len(s.DependsOn) != 1 || s.DependsOn[0] != b {
		t.Errorf("deps = %v, want [%s]", s.DependsOn, b)
	}
}
