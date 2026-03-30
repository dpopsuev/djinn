package artifact

import "testing"

func TestDefaultRegistry_HasBothKinds(t *testing.T) {
	reg := DefaultRegistry()

	if _, err := reg.Get(KindPlanSegment); err != nil {
		t.Errorf("missing plan-segment template: %v", err)
	}
	if _, err := reg.Get(KindTask); err != nil {
		t.Errorf("missing task template: %v", err)
	}
}

func TestPlanSegmentTemplate_NoRequiredSections(t *testing.T) {
	tmpl := PlanSegmentTemplate()

	// Draft segments can be created without content — FillDraft adds it later.
	a := &Artifact{Kind: KindPlanSegment, Title: "draft segment"}
	if err := tmpl.Check(a); err != nil {
		t.Errorf("draft segment should pass validation: %v", err)
	}

	// With Content — also passes.
	a.Content = "acceptance criteria"
	if err := tmpl.Check(a); err != nil {
		t.Errorf("segment with content should pass: %v", err)
	}
}

func TestTaskTemplate_NoRequiredSections(t *testing.T) {
	tmpl := TaskTemplate()
	a := &Artifact{Kind: KindTask, Title: "do something"}
	if err := tmpl.Check(a); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPlanSegmentTemplate_Statuses(t *testing.T) {
	tmpl := PlanSegmentTemplate()

	planStatuses := []Status{StatusDraft, StatusReady, StatusClaimed, StatusInProgress, StatusComplete, StatusInvalidated}
	for _, s := range planStatuses {
		if !tmpl.HasStatus(s) {
			t.Errorf("plan-segment should have status %q", s)
		}
	}
	if tmpl.HasStatus(StatusBlocked) {
		t.Error("plan-segment should NOT have StatusBlocked")
	}
}

func TestTaskTemplate_Statuses(t *testing.T) {
	tmpl := TaskTemplate()

	taskStatuses := []Status{StatusPending, StatusActive, StatusDone, StatusBlocked}
	for _, s := range taskStatuses {
		if !tmpl.HasStatus(s) {
			t.Errorf("task should have status %q", s)
		}
	}
	if tmpl.HasStatus(StatusClaimed) {
		t.Error("task should NOT have StatusClaimed")
	}
}

func TestIDFormats(t *testing.T) {
	if PlanSegmentTemplate().IDFormat != "seg-%d" {
		t.Errorf("plan-segment IDFormat = %q, want seg-%%d", PlanSegmentTemplate().IDFormat)
	}
	if TaskTemplate().IDFormat != "T-%03d" {
		t.Errorf("task IDFormat = %q, want T-%%03d", TaskTemplate().IDFormat)
	}
}
