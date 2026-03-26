// gear.go — Gear system for Djinn staffing model.
// Gears determine the execution mode and how many executors are spawned.
// Auto-shift uses prompt complexity heuristics to pick the right gear.
package staff

import (
	"fmt"
	"strings"
)

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
	return GearNone, fmt.Errorf("unknown gear %q", s)
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

// SupportRoles returns the support roles auto-spawned for this gear.
func (g Gear) SupportRoles() []string {
	switch g {
	case GearE1:
		return []string{"inspector"}
	case GearE2:
		return []string{"scheduler", "inspector"}
	case GearE3:
		return []string{"auditor", "scheduler", "inspector"}
	default:
		return nil
	}
}

// State returns the full GearState for this gear.
func (g Gear) State() GearState {
	return GearState{
		Current:   g,
		Executors: g.Executors(),
		Support:   g.SupportRoles(),
	}
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
