package artifact

import (
	"testing"
	"time"

	"github.com/dpopsuev/parchment"
)

func TestToParchment_RoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := Artifact{
		ID:      "seg-1",
		Kind:    KindPlanSegment,
		Title:   "implement auth",
		Status:  StatusInProgress,
		Owner:   "executor-1",
		Content: "implement OAuth flow",
		Sections: map[string]string{
			"design":  "use PKCE",
			"content": "implement OAuth flow",
		},
		Components: ComponentMap{
			Files:   []string{"auth.go", "auth_test.go"},
			Symbols: []string{"AuthHandler"},
		},
		DependsOn:   []string{"seg-0"},
		Children:    []string{"T-001"},
		Parent:      "plan-1",
		Labels:      []string{"auth", "security"},
		Annotations: []Annotation{{Kind: "+", Comment: "good approach"}},
		Version:     3,
		Priority:    2,
		Created:     now,
		Updated:     now,
	}

	// Convert to Parchment.
	pa := ToParchment(&original)

	if pa.ID != "seg-1" {
		t.Fatalf("ID = %q, want seg-1", pa.ID)
	}
	if pa.Status != "in_progress" {
		t.Fatalf("Status = %q, want in_progress", pa.Status)
	}
	if pa.Goal != "implement OAuth flow" {
		t.Fatalf("Goal = %q, want content", pa.Goal)
	}
	if len(pa.Sections) != 2 {
		t.Fatalf("Sections = %d, want 2", len(pa.Sections))
	}
	if pa.Priority != "2" {
		t.Fatalf("Priority = %q, want 2", pa.Priority)
	}
	if pa.Extra["owner"] != "executor-1" {
		t.Fatalf("Extra.owner = %v, want executor-1", pa.Extra["owner"])
	}

	// Convert back to djinn.
	roundTripped := FromParchment(pa)

	if roundTripped.ID != original.ID {
		t.Fatalf("ID = %q, want %q", roundTripped.ID, original.ID)
	}
	if roundTripped.Status != original.Status {
		t.Fatalf("Status = %q, want %q", roundTripped.Status, original.Status)
	}
	if roundTripped.Owner != original.Owner {
		t.Fatalf("Owner = %q, want %q", roundTripped.Owner, original.Owner)
	}
	if roundTripped.Content != original.Content {
		t.Fatalf("Content = %q, want %q", roundTripped.Content, original.Content)
	}
	if len(roundTripped.Sections) != len(original.Sections) {
		t.Fatalf("Sections len = %d, want %d", len(roundTripped.Sections), len(original.Sections))
	}
	if roundTripped.Priority != original.Priority {
		t.Fatalf("Priority = %d, want %d", roundTripped.Priority, original.Priority)
	}
	if len(roundTripped.Components.Files) != 2 {
		t.Fatalf("Components.Files = %d, want 2", len(roundTripped.Components.Files))
	}
	if len(roundTripped.Annotations) != 1 {
		t.Fatalf("Annotations = %d, want 1", len(roundTripped.Annotations))
	}
}

func TestFromParchment_Minimal(t *testing.T) {
	pa := &parchment.Artifact{
		ID:    "TSK-1",
		Kind:  "task",
		Title: "minimal",
	}

	a := FromParchment(pa)

	if a.ID != "TSK-1" {
		t.Fatalf("ID = %q", a.ID)
	}
	if a.Created.IsZero() {
		t.Fatal("Created should default to now")
	}
}

func TestToParchment_EmptyExtra(t *testing.T) {
	a := Artifact{
		ID:    "seg-1",
		Kind:  KindPlanSegment,
		Title: "no extras",
	}
	pa := ToParchment(&a)

	if pa.Extra != nil {
		t.Fatalf("Extra should be nil for empty djinn fields, got %v", pa.Extra)
	}
}
