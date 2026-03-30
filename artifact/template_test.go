package artifact

import (
	"errors"
	"testing"
)

func TestTemplateRegistry_RegisterAndGet(t *testing.T) {
	reg := NewTemplateRegistry()
	reg.Register(Template{Kind: "test", IDFormat: "T-%d"})

	tmpl, err := reg.Get("test")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Kind != "test" {
		t.Errorf("Kind = %q, want %q", tmpl.Kind, "test")
	}
}

func TestTemplateRegistry_GetUnknown(t *testing.T) {
	reg := NewTemplateRegistry()
	_, err := reg.Get("nonexistent")
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("err = %v, want ErrTemplateNotFound", err)
	}
}

func TestTemplate_CheckRequiredSections(t *testing.T) {
	tmpl := Template{
		Kind:             "spec",
		RequiredSections: []string{"problem", "decision"},
	}

	// Missing sections.
	a := &Artifact{Kind: "spec", Sections: map[string]string{"problem": "exists"}}
	err := tmpl.Check(a)
	if !errors.Is(err, ErrMissingSection) {
		t.Errorf("err = %v, want ErrMissingSection", err)
	}

	// All sections present.
	a.Sections["decision"] = "chosen"
	if err := tmpl.Check(a); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTemplate_CheckContentFallback(t *testing.T) {
	tmpl := Template{
		Kind:             "plan-segment",
		RequiredSections: []string{"content"},
	}

	// Content field satisfies "content" section requirement.
	a := &Artifact{Kind: "plan-segment", Content: "acceptance criteria"}
	if err := tmpl.Check(a); err != nil {
		t.Errorf("Content field should satisfy 'content' section: %v", err)
	}
}

func TestTemplate_CheckCustomValidate(t *testing.T) {
	errCustom := errors.New("custom failure")
	tmpl := Template{
		Kind: "strict",
		Validate: func(a *Artifact) error {
			if a.Title == "" {
				return errCustom
			}
			return nil
		},
	}

	a := &Artifact{Kind: "strict", Title: ""}
	if err := tmpl.Check(a); !errors.Is(err, errCustom) {
		t.Errorf("err = %v, want custom failure", err)
	}

	a.Title = "has title"
	if err := tmpl.Check(a); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTemplate_CheckKindMismatch(t *testing.T) {
	tmpl := Template{Kind: "task"}
	a := &Artifact{Kind: "plan-segment"}
	if err := tmpl.Check(a); !errors.Is(err, ErrKindMismatch) {
		t.Errorf("err = %v, want ErrKindMismatch", err)
	}
}

func TestTemplate_HasStatus(t *testing.T) {
	tmpl := Template{
		Kind:          "task",
		ValidStatuses: []Status{StatusPending, StatusActive, StatusDone, StatusBlocked},
	}

	if !tmpl.HasStatus(StatusPending) {
		t.Error("should have StatusPending")
	}
	if tmpl.HasStatus(StatusClaimed) {
		t.Error("should NOT have StatusClaimed")
	}
}

func TestTemplateRegistry_Check(t *testing.T) {
	reg := NewTemplateRegistry()
	reg.Register(Template{
		Kind:             "task",
		RequiredSections: nil,
	})

	a := &Artifact{Kind: "task", Title: "do something"}
	if err := reg.Check(a); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	a.Kind = "unknown"
	if err := reg.Check(a); !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("err = %v, want ErrTemplateNotFound", err)
	}
}
