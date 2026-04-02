package widgets

import (
	"testing"

	"github.com/dpopsuev/djinn/review"
)

func TestBudgetPanel_CellSight(t *testing.T) {
	p := NewBudgetPanel()
	p.SetSignals([]review.Signal{
		{Metric: "files_touched", Value: 5, Threshold: 10},
		{Metric: "token_spend", Value: 800, Threshold: 1000},
	})

	cs := p.CellSight()
	if cs.PanelID != "budget" {
		t.Fatalf("PanelID = %q, want budget", cs.PanelID)
	}
	if cs.Kind != "cost" {
		t.Fatalf("Kind = %q, want cost", cs.Kind)
	}

	// Should have 1 public field (signals count) + 2 sensitive (per-signal).
	if len(cs.Fields) != 3 {
		t.Fatalf("Fields count = %d, want 3", len(cs.Fields))
	}

	// Verify signals count is public.
	if cs.Fields[0].Key != "signals" || cs.Fields[0].Sensitive {
		t.Errorf("signals field should be public: got %+v", cs.Fields[0])
	}
	if cs.Fields[0].Value != "2" {
		t.Errorf("signals value = %q, want 2", cs.Fields[0].Value)
	}

	// Verify per-signal fields are sensitive.
	for _, f := range cs.Fields[1:] {
		if !f.Sensitive {
			t.Errorf("field %q should be sensitive", f.Key)
		}
	}
}

func TestBudgetPanel_CellSight_Empty(t *testing.T) {
	p := NewBudgetPanel()
	cs := p.CellSight()
	if cs.PanelID != "budget" {
		t.Fatalf("PanelID = %q, want budget", cs.PanelID)
	}
	// Only the signals count field.
	if len(cs.Fields) != 1 {
		t.Fatalf("Fields count = %d, want 1", len(cs.Fields))
	}
	if cs.Fields[0].Value != "0" {
		t.Errorf("signals value = %q, want 0", cs.Fields[0].Value)
	}
}

func TestBudgetPanel_SightGate(t *testing.T) {
	p := NewBudgetPanel()
	if p.SightGate() {
		t.Fatal("SightGate should be false by default — budget hidden from agents")
	}
}
