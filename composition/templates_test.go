package composition

import (
	"errors"
	"testing"
)

func TestTemplateSolo(t *testing.T) {
	f := TemplateSolo()
	if f.Name != "solo" {
		t.Fatalf("Name = %q, want %q", f.Name, "solo")
	}
	if len(f.Units) != 1 {
		t.Fatalf("Units = %d, want 1", len(f.Units))
	}
	if f.Units[0].Role != RoleExecutor {
		t.Fatalf("Role = %q, want %q", f.Units[0].Role, RoleExecutor)
	}
}

func TestTemplateDuo(t *testing.T) {
	f := TemplateDuo()
	if f.Name != "duo" {
		t.Fatalf("Name = %q, want %q", f.Name, "duo")
	}
	if len(f.Units) != 2 {
		t.Fatalf("Units = %d, want 2", len(f.Units))
	}
	if f.Units[0].Role != RoleReviewer {
		t.Fatalf("Unit[0].Role = %q, want %q", f.Units[0].Role, RoleReviewer)
	}
	if f.Units[1].Role != RoleExecutor {
		t.Fatalf("Unit[1].Role = %q, want %q", f.Units[1].Role, RoleExecutor)
	}
	if len(f.Edges) != 1 {
		t.Fatalf("Edges = %d, want 1", len(f.Edges))
	}
	if f.Edges[0].Channel != ChannelPromotionGate {
		t.Fatalf("Edge.Channel = %q, want %q", f.Edges[0].Channel, ChannelPromotionGate)
	}
}

func TestTemplateSquad(t *testing.T) {
	f := TemplateSquad()
	if f.Name != "squad" {
		t.Fatalf("Name = %q, want %q", f.Name, "squad")
	}
	if len(f.Units) != 4 {
		t.Fatalf("Units = %d, want 4", len(f.Units))
	}
	if len(f.Edges) != 3 {
		t.Fatalf("Edges = %d, want 3", len(f.Edges))
	}
}

func TestTemplateByName(t *testing.T) {
	for _, name := range []string{"solo", "duo", "squad"} {
		f, err := TemplateByName(name)
		if err != nil {
			t.Fatalf("TemplateByName(%q): %v", name, err)
		}
		if f.Name != name {
			t.Fatalf("Name = %q, want %q", f.Name, name)
		}
	}
}

func TestTemplateByName_NotFound(t *testing.T) {
	_, err := TemplateByName("platoon")
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound, got %v", err)
	}
}
