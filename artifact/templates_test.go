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

func TestPlanSegmentTemplate_RequiresContent(t *testing.T) {
	tmpl := PlanSegmentTemplate()

	// Without content — should fail.
	a := &Artifact{Kind: KindPlanSegment}
	if err := tmpl.Check(a); err == nil {
		t.Error("expected error for missing content")
	}

	// With Content field — should pass.
	a.Content = "acceptance criteria"
	if err := tmpl.Check(a); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// With Sections["content"] — should also pass.
	a2 := &Artifact{
		Kind:     KindPlanSegment,
		Sections: map[string]string{"content": "via sections"},
	}
	if err := tmpl.Check(a2); err != nil {
		t.Errorf("unexpected error: %v", err)
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
