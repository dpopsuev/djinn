// triage.go — GenSec queue triage engine (GOL-24, TSK-231-234).
//
// TriagePrompt analyzes an incoming prompt, classifies its complexity,
// checks budget constraints, and decides which gear to recommend.
// DotForState maps triage state to a colored dot indicator for the TUI.
package staff

// TriageState represents the current phase of prompt triage.
type TriageState string

const (
	TriageIdle      TriageState = "idle"      // no prompt queued
	TriageAnalyzing TriageState = "analyzing" // GenSec reading prompt
	TriageRouting   TriageState = "routing"   // deciding which gear/role
	TriageReady     TriageState = "ready"     // decision made, executing
)

// TriageDecision captures what gear GenSec recommends and why.
type TriageDecision struct {
	State         TriageState
	SuggestedGear Gear   // what gear GenSec recommends
	Reason        string // e.g. "complex refactor -> E2"
}

// TriagePrompt analyzes a prompt and returns a triage decision.
//
// Process:
//  1. ClassifyPromptComplexity determines the ideal gear.
//  2. Budget constraints may force a downshift (e.g. E3 -> E1).
//  3. The final decision includes the reason.
func TriagePrompt(prompt string, currentGear Gear, budget float64) *TriageDecision {
	suggested := ClassifyPromptComplexity(prompt)

	reason := gearReason(suggested)

	// Budget constraint: if budget is exhausted or too low, downshift.
	if budget <= 0 {
		capped := constrainForBudget(suggested, GearRead)
		if capped != suggested {
			suggested = capped
			reason += " [budget exhausted, downshift to " + string(suggested) + "]"
		}
	} else if budget < 0.50 {
		capped := constrainForBudget(suggested, GearE0)
		if capped != suggested {
			suggested = capped
			reason += " [low budget, capped at " + string(suggested) + "]"
		}
	}

	return &TriageDecision{
		State:         TriageReady,
		SuggestedGear: suggested,
		Reason:        reason,
	}
}

// constrainForBudget ensures the suggested gear doesn't exceed the ceiling.
func constrainForBudget(suggested, ceiling Gear) Gear {
	if gearRank(suggested) > gearRank(ceiling) {
		return ceiling
	}
	return suggested
}

// gearRank returns an ordinal for comparison: lower = cheaper.
func gearRank(g Gear) int {
	switch g {
	case GearNone:
		return 0
	case GearRead:
		return 1
	case GearPlan:
		return 2
	case GearE0:
		return 3
	case GearE1:
		return 4
	case GearE2:
		return 5
	case GearE3:
		return 6
	case GearAuto:
		return 3 // auto defaults to E0-level cost
	default:
		return 0
	}
}

// gearReason returns a human-readable reason for the gear choice.
func gearReason(g Gear) string {
	switch g {
	case GearRead:
		return "short question -> R"
	case GearPlan:
		return "design/plan prompt -> P"
	case GearE0:
		return "small change -> E0"
	case GearE1:
		return "implementation task -> E1"
	case GearE2:
		return "complex refactor -> E2"
	case GearE3:
		return "major overhaul -> E3"
	default:
		return "default -> " + string(g)
	}
}

// QueueDotIndicator describes the visual dot for triage state.
type QueueDotIndicator struct {
	State TriageState
	Color string // CSS-style color name for TUI rendering
}

// DotForState maps a triage state to its queue indicator dot.
func DotForState(state TriageState) QueueDotIndicator {
	switch state {
	case TriageIdle:
		return QueueDotIndicator{State: state, Color: "green"}
	case TriageAnalyzing:
		return QueueDotIndicator{State: state, Color: "yellow"}
	case TriageRouting:
		return QueueDotIndicator{State: state, Color: "blue"}
	case TriageReady:
		return QueueDotIndicator{State: state, Color: "white"}
	default:
		return QueueDotIndicator{State: state, Color: "green"}
	}
}
