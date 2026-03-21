package composition

import (
	"errors"
	"testing"
	"time"
)

func TestFormation_Validate_Valid(t *testing.T) {
	f := Formation{
		Name: "test",
		Units: []Unit{
			{Role: RoleReviewer, Scope: UnitScope{RO: []string{"pkg/auth"}}},
			{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth"}}},
		},
		Edges: []Edge{{From: 1, To: 0, Channel: ChannelPromotionGate}},
	}
	if err := f.Validate(); err != nil {
		t.Fatalf("valid formation: %v", err)
	}
}

func TestFormation_Validate_Empty(t *testing.T) {
	f := Formation{Name: "empty"}
	if !errors.Is(f.Validate(), ErrEmptyFormation) {
		t.Fatalf("expected ErrEmptyFormation, got %v", f.Validate())
	}
}

func TestFormation_Validate_NoName(t *testing.T) {
	f := Formation{Units: []Unit{{Role: RoleExecutor}}}
	if !errors.Is(f.Validate(), ErrNoFormationName) {
		t.Fatalf("expected ErrNoFormationName, got %v", f.Validate())
	}
}

func TestFormation_Validate_BadEdgeIndex(t *testing.T) {
	f := Formation{
		Name:  "bad-edge",
		Units: []Unit{{Role: RoleExecutor}},
		Edges: []Edge{{From: 0, To: 5, Channel: ChannelPromotionGate}},
	}
	if !errors.Is(f.Validate(), ErrInvalidEdgeIndex) {
		t.Fatalf("expected ErrInvalidEdgeIndex, got %v", f.Validate())
	}
}

func TestFormation_Validate_BadUnitRole(t *testing.T) {
	f := Formation{
		Name:  "bad-unit",
		Units: []Unit{{Role: ""}},
	}
	if !errors.Is(f.Validate(), ErrEmptyRole) {
		t.Fatalf("expected ErrEmptyRole, got %v", f.Validate())
	}
}

func TestFormation_DivideBudget(t *testing.T) {
	f := Formation{
		Name: "divide",
		Units: []Unit{
			{Role: RoleExecutor},
			{Role: RoleExecutor},
			{Role: RoleObserver}, // observers don't get budget
		},
		Budget: Budget{Tokens: 100000, WallClock: time.Hour},
	}
	f.DivideBudget()

	if f.Units[0].Budget.Tokens != 50000 {
		t.Fatalf("unit[0] tokens = %d, want 50000", f.Units[0].Budget.Tokens)
	}
	if f.Units[1].Budget.Tokens != 50000 {
		t.Fatalf("unit[1] tokens = %d, want 50000", f.Units[1].Budget.Tokens)
	}
	if f.Units[2].Budget.Tokens != 0 {
		t.Fatalf("observer should keep zero budget, got %d", f.Units[2].Budget.Tokens)
	}
}

func TestFormation_DivideBudget_PresetUnitsKept(t *testing.T) {
	f := Formation{
		Name: "preset",
		Units: []Unit{
			{Role: RoleExecutor, Budget: Budget{Tokens: 10000}},
			{Role: RoleExecutor},
		},
		Budget: Budget{Tokens: 80000},
	}
	f.DivideBudget()

	if f.Units[0].Budget.Tokens != 10000 {
		t.Fatalf("preset unit should keep 10000, got %d", f.Units[0].Budget.Tokens)
	}
	if f.Units[1].Budget.Tokens != 80000 {
		t.Fatalf("unset unit should get full share 80000, got %d", f.Units[1].Budget.Tokens)
	}
}

func TestFormation_DivideBudget_ZeroBudget(t *testing.T) {
	f := Formation{
		Name:  "zero",
		Units: []Unit{{Role: RoleExecutor}},
	}
	f.DivideBudget() // should not panic
}
