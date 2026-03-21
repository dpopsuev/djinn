package taskforce

import (
	"errors"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/composition"
	"github.com/dpopsuev/djinn/tier"
	"github.com/dpopsuev/djinn/watchdog"
)

func TestComposer_ClearBand(t *testing.T) {
	c := NewComposer(NewHeuristicClassifier())
	budget := composition.Budget{Tokens: 50000, WallClock: 5 * time.Minute}

	tf, err := c.Compose("tf-1",
		[]tier.Scope{{Level: tier.Mod, Name: "auth"}},
		1, 1, budget,
	)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if tf.Band != Clear {
		t.Fatalf("Band = %v, want Clear", tf.Band)
	}
	if tf.Formation.Name != "solo" {
		t.Fatalf("Formation = %q, want %q", tf.Formation.Name, "solo")
	}
	if len(tf.Formation.Units) != 1 {
		t.Fatalf("Units = %d, want 1", len(tf.Formation.Units))
	}
}

func TestComposer_ComplicatedBand(t *testing.T) {
	c := NewComposer(NewHeuristicClassifier())
	budget := composition.Budget{Tokens: 100000}

	tf, err := c.Compose("tf-2",
		[]tier.Scope{{Level: tier.Com, Name: "auth"}},
		5, 1, budget,
	)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if tf.Band != Complicated {
		t.Fatalf("Band = %v, want Complicated", tf.Band)
	}
	if tf.Formation.Name != "duo" {
		t.Fatalf("Formation = %q, want %q", tf.Formation.Name, "duo")
	}
}

func TestComposer_ComplexBand(t *testing.T) {
	c := NewComposer(NewHeuristicClassifier())
	budget := composition.Budget{Tokens: 200000}

	tf, err := c.Compose("tf-3",
		[]tier.Scope{{Level: tier.Eco, Name: "workspace"}},
		50, 3, budget,
	)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	if tf.Band != Complex {
		t.Fatalf("Band = %v, want Complex", tf.Band)
	}
	if tf.Formation.Name != "squad" {
		t.Fatalf("Formation = %q, want %q", tf.Formation.Name, "squad")
	}
}

func TestComposer_WatchdogAssignment(t *testing.T) {
	c := NewComposer(NewHeuristicClassifier())
	budget := composition.Budget{Tokens: 50000}

	// Clear band: only security + budget
	tf, _ := c.Compose("tf-w", []tier.Scope{{Level: tier.Mod, Name: "fix"}}, 1, 1, budget)
	wda := tf.WatchdogConfig
	if len(wda.Active) != 2 {
		t.Fatalf("Clear band watchdogs = %d, want 2", len(wda.Active))
	}
	hasSecurity := false
	hasBudget := false
	for _, cat := range wda.Active {
		if cat == watchdog.CategorySecurity {
			hasSecurity = true
		}
		if cat == watchdog.CategoryBudget {
			hasBudget = true
		}
	}
	if !hasSecurity || !hasBudget {
		t.Fatalf("Clear band should have security + budget, got %v", wda.Active)
	}
	if wda.Threshold != ThresholdRelaxed {
		t.Fatalf("Clear threshold = %q, want %q", wda.Threshold, ThresholdRelaxed)
	}
}

func TestComposer_NoScopes(t *testing.T) {
	c := NewComposer(NewHeuristicClassifier())
	_, err := c.Compose("tf-err", nil, 1, 1, composition.Budget{Tokens: 50000})
	if !errors.Is(err, ErrNoScopes) {
		t.Fatalf("expected ErrNoScopes, got %v", err)
	}
}

func TestComposer_WithOverride(t *testing.T) {
	c := NewComposer(NewHeuristicClassifier())
	custom := composition.Formation{
		Name: "custom-solo",
		Units: []composition.Unit{
			{Role: composition.RoleExecutor, Scope: composition.UnitScope{RW: []string{"${task.scope}"}}},
		},
	}
	c.WithOverride(Clear, custom)

	tf, err := c.Compose("tf-override",
		[]tier.Scope{{Level: tier.Mod, Name: "fix"}},
		1, 1, composition.Budget{Tokens: 50000},
	)
	if err != nil {
		t.Fatalf("Compose with override: %v", err)
	}
	if tf.Formation.Name != "custom-solo" {
		t.Fatalf("Formation = %q, want %q", tf.Formation.Name, "custom-solo")
	}
}

func TestComposer_Immutability(t *testing.T) {
	c := NewComposer(NewHeuristicClassifier())
	tf, _ := c.Compose("tf-imm", []tier.Scope{{Level: tier.Mod, Name: "auth"}}, 1, 1, composition.Budget{Tokens: 50000})

	if tf.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
	if tf.ID != "tf-imm" {
		t.Fatalf("ID = %q, want %q", tf.ID, "tf-imm")
	}
}
