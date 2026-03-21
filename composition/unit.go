// Package composition defines composable primitives for agent hierarchy construction.
// Units, formations, edges, budgets, and termination conditions are the building
// blocks from which any formation (solo, duo, squad, or custom) is assembled.
package composition

import (
	"errors"
	"time"
)

// Agent roles within a formation.
const (
	RoleExecutor = "executor"
	RoleReviewer = "reviewer"
	RoleLead     = "lead"
	RoleObserver = "observer"
)

// Termination condition types.
const (
	TermTestsPass        = "tests_pass"
	TermReviewerApproves = "reviewer_approves"
	TermBudgetExhausted  = "budget_exhausted"
	TermTimeout          = "timeout"
	TermManual           = "manual"
)

// Sentinel errors for unit validation.
var (
	ErrEmptyRole  = errors.New("unit role is required")
	ErrNoBudget   = errors.New("unit budget must have tokens or wall clock")
)

// Unit is the primitive building block of a formation.
type Unit struct {
	Role          string
	Scope         UnitScope
	Budget        Budget
	TerminatesWhen Termination
	Env           map[string]string
}

// UnitScope defines filesystem access for a unit.
type UnitScope struct {
	RO []string // read-only paths
	RW []string // read-write paths (must be disjoint across executor units)
}

// Budget defines resource limits for a unit or formation.
type Budget struct {
	Tokens    int
	WallClock time.Duration
}

// IsZero reports whether the budget has no limits set.
func (b Budget) IsZero() bool {
	return b.Tokens == 0 && b.WallClock == 0
}

// Termination defines when a unit's work is considered done.
type Termination struct {
	Type   string // one of Term* constants
	Target string // e.g., test target path
}

// Validate checks that a unit has the required fields.
func (u Unit) Validate() error {
	if u.Role == "" {
		return ErrEmptyRole
	}
	return nil
}

// IsObserver reports whether this unit is a lateral observer.
func (u Unit) IsObserver() bool {
	return u.Role == RoleObserver
}
