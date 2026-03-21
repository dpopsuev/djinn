package composition

import (
	"errors"
	"testing"
	"time"
)

func TestUnit_Validate(t *testing.T) {
	u := Unit{Role: RoleExecutor}
	if err := u.Validate(); err != nil {
		t.Fatalf("valid unit: %v", err)
	}

	u = Unit{}
	if !errors.Is(u.Validate(), ErrEmptyRole) {
		t.Fatalf("empty role should fail with ErrEmptyRole, got %v", u.Validate())
	}
}

func TestUnit_IsObserver(t *testing.T) {
	if !(Unit{Role: RoleObserver}).IsObserver() {
		t.Fatal("observer role should return true")
	}
	if (Unit{Role: RoleExecutor}).IsObserver() {
		t.Fatal("executor role should return false")
	}
}

func TestBudget_IsZero(t *testing.T) {
	if !(Budget{}).IsZero() {
		t.Fatal("zero budget should be zero")
	}
	if (Budget{Tokens: 100}).IsZero() {
		t.Fatal("non-zero tokens should not be zero")
	}
	if (Budget{WallClock: time.Minute}).IsZero() {
		t.Fatal("non-zero wall clock should not be zero")
	}
}

func TestRoleConstants(t *testing.T) {
	roles := []string{RoleExecutor, RoleReviewer, RoleLead, RoleObserver}
	seen := make(map[string]bool)
	for _, r := range roles {
		if r == "" {
			t.Fatal("role constant is empty")
		}
		if seen[r] {
			t.Fatalf("duplicate role: %q", r)
		}
		seen[r] = true
	}
}

func TestTerminationConstants(t *testing.T) {
	terms := []string{TermTestsPass, TermReviewerApproves, TermBudgetExhausted, TermTimeout, TermManual}
	seen := make(map[string]bool)
	for _, term := range terms {
		if term == "" {
			t.Fatal("termination constant is empty")
		}
		if seen[term] {
			t.Fatalf("duplicate termination: %q", term)
		}
		seen[term] = true
	}
}
