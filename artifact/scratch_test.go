package artifact

import (
	"testing"
)

func TestScratchPaperTemplate_ValidStatuses(t *testing.T) {
	tmpl := ScratchPaperTemplate()

	if tmpl.Kind != KindScratchPaper {
		t.Errorf("Kind = %q, want %q", tmpl.Kind, KindScratchPaper)
	}

	// Verify expected statuses.
	expected := []Status{StatusDraft, StatusActive, StatusComplete}
	if len(tmpl.ValidStatuses) != len(expected) {
		t.Fatalf("ValidStatuses count = %d, want %d", len(tmpl.ValidStatuses), len(expected))
	}
	for i, s := range expected {
		if tmpl.ValidStatuses[i] != s {
			t.Errorf("ValidStatuses[%d] = %q, want %q", i, tmpl.ValidStatuses[i], s)
		}
	}

	// HasStatus checks.
	for _, s := range expected {
		if !tmpl.HasStatus(s) {
			t.Errorf("HasStatus(%q) = false, want true", s)
		}
	}

	// Invalid statuses should fail.
	if tmpl.HasStatus(StatusReady) {
		t.Error("HasStatus(ready) should be false")
	}
	if tmpl.HasStatus(StatusClaimed) {
		t.Error("HasStatus(claimed) should be false")
	}
}

func TestScratchPaperTemplate_IDFormat(t *testing.T) {
	tmpl := ScratchPaperTemplate()
	if tmpl.IDFormat != "SP-%03d" {
		t.Errorf("IDFormat = %q, want SP-%%03d", tmpl.IDFormat)
	}
}

func TestScratchPaperTemplate_RegisteredInDefault(t *testing.T) {
	reg := DefaultRegistry()
	tmpl, err := reg.Get(KindScratchPaper)
	if err != nil {
		t.Fatalf("DefaultRegistry missing scratch-paper: %v", err)
	}
	if tmpl.Kind != KindScratchPaper {
		t.Errorf("Kind = %q, want %q", tmpl.Kind, KindScratchPaper)
	}
}

func TestScratchPaperTemplate_AddToGraph(t *testing.T) {
	g := NewGraph("test", DefaultRegistry())

	id, err := g.Add(Artifact{
		Kind:  KindScratchPaper,
		Title: "task scratch paper",
	})
	if err != nil {
		t.Fatalf("Add scratch paper: %v", err)
	}
	if id == "" {
		t.Fatal("ID should be generated")
	}

	a, err := g.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if a.Kind != KindScratchPaper {
		t.Errorf("Kind = %q, want %q", a.Kind, KindScratchPaper)
	}
}
