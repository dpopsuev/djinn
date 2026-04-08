package broker

import (
	"testing"
	"time"

	"github.com/dpopsuev/djinn/uniform"
)

func TestUnitFromAssignment(t *testing.T) {
	a := uniform.Assignment{
		Role: uniform.RoleExecutor,
		Mode: uniform.ModeAgent,
		Scope: uniform.AssignmentScope{
			ReadPaths:  []string{"/src"},
			WritePaths: []string{"/src/out"},
		},
		Budget: uniform.AssignmentBudget{
			MaxTokens:   50000,
			MaxDuration: 5 * time.Minute,
		},
	}

	u := UnitFromAssignment(a)

	if u.Role != uniform.RoleExecutor {
		t.Fatalf("Unit.Role = %q, want %q", u.Role, uniform.RoleExecutor)
	}
	if len(u.Scope.RO) != 1 || u.Scope.RO[0] != "/src" {
		t.Fatalf("Unit.Scope.RO = %v, want [/src]", u.Scope.RO)
	}
	if len(u.Scope.RW) != 1 || u.Scope.RW[0] != "/src/out" {
		t.Fatalf("Unit.Scope.RW = %v, want [/src/out]", u.Scope.RW)
	}
	if u.Budget.Tokens != 50000 {
		t.Fatalf("Unit.Budget.Tokens = %d, want 50000", u.Budget.Tokens)
	}
	if u.Budget.WallClock != 5*time.Minute {
		t.Fatalf("Unit.Budget.WallClock = %v, want 5m", u.Budget.WallClock)
	}
}
