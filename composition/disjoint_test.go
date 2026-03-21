package composition

import (
	"errors"
	"testing"
)

func TestValidateScopeDisjointness_NoOverlap(t *testing.T) {
	units := []Unit{
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth"}}},
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/billing"}}},
	}
	if err := ValidateScopeDisjointness(units); err != nil {
		t.Fatalf("disjoint scopes should pass: %v", err)
	}
}

func TestValidateScopeDisjointness_ExactOverlap(t *testing.T) {
	units := []Unit{
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth"}}},
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth"}}},
	}
	if !errors.Is(ValidateScopeDisjointness(units), ErrOverlappingScopes) {
		t.Fatal("exact overlap should fail")
	}
}

func TestValidateScopeDisjointness_PrefixOverlap(t *testing.T) {
	units := []Unit{
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth"}}},
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth/middleware.go"}}},
	}
	if !errors.Is(ValidateScopeDisjointness(units), ErrOverlappingScopes) {
		t.Fatal("prefix overlap should fail")
	}
}

func TestValidateScopeDisjointness_ObserversExempt(t *testing.T) {
	units := []Unit{
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth"}}},
		{Role: RoleObserver, Scope: UnitScope{RW: []string{"pkg/auth"}}}, // observers can overlap
	}
	if err := ValidateScopeDisjointness(units); err != nil {
		t.Fatalf("observer overlap should be exempt: %v", err)
	}
}

func TestValidateScopeDisjointness_PartialNameNoOverlap(t *testing.T) {
	units := []Unit{
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/au"}}},
		{Role: RoleExecutor, Scope: UnitScope{RW: []string{"pkg/auth"}}},
	}
	if err := ValidateScopeDisjointness(units); err != nil {
		t.Fatalf("partial name 'au' should not overlap 'auth': %v", err)
	}
}
