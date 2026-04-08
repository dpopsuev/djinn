// assignment_adapter.go — converts uniform.Assignment to broker.Unit.
//
// This adapter inverts the dependency: broker imports uniform (not the reverse).
// Extracted from uniform/assignment.go to break the uniform→broker cycle (ISP).
package broker

import "github.com/dpopsuev/djinn/uniform"

// UnitFromAssignment converts a uniform.Assignment to a broker.Unit
// for formation instantiation.
func UnitFromAssignment(a uniform.Assignment) Unit {
	return Unit{
		Role: a.Role,
		Scope: UnitScope{
			RO: a.Scope.ReadPaths,
			RW: a.Scope.WritePaths,
		},
		Budget: Budget{
			Tokens:    a.Budget.MaxTokens,
			WallClock: a.Budget.MaxDuration,
		},
	}
}
