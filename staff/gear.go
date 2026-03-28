// gear.go — Gear system for Djinn staffing model.
// Gears determine the execution mode and how many executors are spawned.
// Auto-shift uses prompt complexity heuristics to pick the right gear.
package staff

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for gear parsing.
var ErrUnknownGear = errors.New("unknown gear")

// Gear represents the current execution gear.
type Gear string

const (
	GearNone Gear = "N"  // engine off
	GearRead Gear = "R"  // read-only
	GearPlan Gear = "P"  // plan-only
	GearE0   Gear = "E0" // GenSec solo execute
	GearE1   Gear = "E1" // 1 executor
	GearE2   Gear = "E2" // 2 executors
	GearE3   Gear = "E3" // 3 executors
	GearAuto Gear = "A"  // auto-shift
)

// allGears lists all valid gear values for parsing.
var allGears = []Gear{GearNone, GearRead, GearPlan, GearE0, GearE1, GearE2, GearE3, GearAuto}

// RoleAssignment is a support role to spawn for a gear level.
type RoleAssignment struct {
	Role string
}

// SupportScheduler decides which support agents to spawn for a given gear.
type SupportScheduler interface {
	Plan(gear Gear) []RoleAssignment
}

// SupportSchedulerFunc adapts a plain function to the SupportScheduler interface.
type SupportSchedulerFunc func(Gear) []RoleAssignment

func (f SupportSchedulerFunc) Plan(g Gear) []RoleAssignment { return f(g) }

// defaultSupportScheduler implements the built-in escalation rules.
type defaultSupportScheduler struct{}

func (defaultSupportScheduler) Plan(g Gear) []RoleAssignment {
	switch g {
	case GearE1:
		return []RoleAssignment{{Role: RoleInspector}}
	case GearE2:
		return []RoleAssignment{{Role: RoleScheduler}, {Role: RoleInspector}}
	case GearE3:
		return []RoleAssignment{{Role: RoleAuditor}, {Role: RoleScheduler}, {Role: RoleInspector}}
	default:
		return nil
	}
}

// DefaultSupportScheduler returns the built-in support scheduling strategy.
func DefaultSupportScheduler() SupportScheduler {
	return defaultSupportScheduler{}
}

// GearState captures the current gear and its implications.
type GearState struct {
	Current   Gear
	Executors int
	Support   []string // auto-spawned support roles
}

// ParseGear converts a string to a Gear value. Case-insensitive.
func ParseGear(s string) (Gear, error) {
	upper := strings.ToUpper(strings.TrimSpace(s))
	for _, g := range allGears {
		if string(g) == upper {
			return g, nil
		}
	}
	return GearNone, fmt.Errorf("%w: %q", ErrUnknownGear, s)
}

// Executors returns the number of executor agents this gear spawns.
func (g Gear) Executors() int {
	switch g {
	case GearE1:
		return 1
	case GearE2:
		return 2
	case GearE3:
		return 3
	default:
		return 0
	}
}

// SupportRoles returns the support roles auto-spawned for this gear
// using the DefaultSupportScheduler.
func (g Gear) SupportRoles() []string {
	return roleNames(DefaultSupportScheduler().Plan(g))
}

// State returns the full GearState using the default scheduler.
func (g Gear) State() GearState {
	return g.StateWith(DefaultSupportScheduler())
}

// StateWith returns the GearState using the provided scheduler.
func (g Gear) StateWith(sched SupportScheduler) GearState {
	return GearState{
		Current:   g,
		Executors: g.Executors(),
		Support:   roleNames(sched.Plan(g)),
	}
}

func roleNames(assignments []RoleAssignment) []string {
	if len(assignments) == 0 {
		return nil
	}
	names := make([]string, len(assignments))
	for i, a := range assignments {
		names[i] = a.Role
	}
	return names
}

// ClassifyPromptComplexity uses simple heuristics to choose a gear
// based on prompt content: word count and keyword matching.
func ClassifyPromptComplexity(prompt string) Gear {
	lower := strings.ToLower(prompt)
	words := strings.Fields(lower)
	wordCount := len(words)

	// E3: overhaul, rewrite, migrate
	e3Keywords := []string{"overhaul", "rewrite", "migrate", "migration"}
	for _, kw := range e3Keywords {
		if strings.Contains(lower, kw) {
			return GearE3
		}
	}

	// E2: refactor, restructure + multi-file hints
	e2Keywords := []string{"refactor", "restructure"}
	multiFileHints := []string{"multiple files", "across", "all files", "entire", "codebase"}
	for _, kw := range e2Keywords {
		if strings.Contains(lower, kw) {
			for _, hint := range multiFileHints {
				if strings.Contains(lower, hint) {
					return GearE2
				}
			}
			// Refactor without multi-file hints is still E1
			return GearE1
		}
	}

	// E1: implement, create, build
	e1Keywords := []string{"implement", "create", "build", "add", "write"}
	for _, kw := range e1Keywords {
		if strings.Contains(lower, kw) {
			return GearE1
		}
	}

	// E0: fix, typo, rename — small surgical changes
	e0Keywords := []string{"fix", "typo", "rename", "update", "change", "tweak"}
	for _, kw := range e0Keywords {
		if strings.Contains(lower, kw) {
			return GearE0
		}
	}

	// P: design, plan, spec
	planKeywords := []string{"design", "plan", "spec", "architecture", "proposal"}
	for _, kw := range planKeywords {
		if strings.Contains(lower, kw) {
			return GearPlan
		}
	}

	// R: short questions (< 10 words, ends with ?)
	if wordCount < 10 && strings.Contains(prompt, "?") {
		return GearRead
	}

	// Default: E0 for anything that doesn't match
	return GearE0
}
