// scheduler.go — Config-driven pipeline transitions (OCP fix, DJN-TSK-1058).
//
// Role transitions are defined as data in StaffConfig, not as a switch statement.
// Adding a new role (e.g., CoGS) requires only a YAML change, no code edit.
// Signal → (FromRole, ToRole) mapping. Zero LLM tokens.
package uniform

// Signal represents a pipeline event that triggers role transitions.
type Signal int

const (
	SignalPromptReceived    Signal = iota // human typed something
	SignalNeedCaptured                    // NED-* created in Scribe
	SignalSpecStamped                     // Auditor approved → SPC-*
	SignalTasksPlanned                    // Scheduler created TSK-* batch
	SignalExecutorDone                    // agent loop completed
	SignalGatePassed                      // mechanical gate OK
	SignalGateFailed                      // mechanical gate FAIL
	SignalInspectorApproved               // quality review OK
	SignalInspectorRejected               // quality review FAIL
)

// Transition defines a single signal→role mapping.
// FromRole is optional — empty means "any current role."
type Transition struct {
	Signal   Signal `yaml:"signal"`
	FromRole string `yaml:"from_role,omitempty"` // empty = any
	ToRole   string `yaml:"to_role"`             // empty = no transition (e.g., gate fires)
}

// defaultFallbackRole is the role returned when no transition matches a signal.
const defaultFallbackRole = "gensec"

// DefaultTransitions returns the standard 5-role pipeline transitions.
// Override or extend via StaffConfig.Transitions in YAML.
func DefaultTransitions() []Transition {
	return []Transition{
		{Signal: SignalPromptReceived, ToRole: defaultFallbackRole},
		{Signal: SignalNeedCaptured, ToRole: "auditor"},
		{Signal: SignalSpecStamped, ToRole: "scheduler"},
		{Signal: SignalTasksPlanned, ToRole: "executor"},
		{Signal: SignalExecutorDone, ToRole: ""},
		{Signal: SignalGatePassed, ToRole: "inspector"},
		{Signal: SignalGateFailed, ToRole: "executor"},
		{Signal: SignalInspectorApproved, ToRole: defaultFallbackRole},
		{Signal: SignalInspectorRejected, ToRole: "executor"},
	}
}

// NextRole returns the target role for a given signal using config-driven transitions.
// Checks FromRole-specific transitions first, then any-role transitions.
// Falls back to "gensec" if no transition matches.
func NextRole(signal Signal, transitions ...[]Transition) string {
	table := DefaultTransitions()
	if len(transitions) > 0 && transitions[0] != nil {
		table = transitions[0]
	}

	for _, t := range table {
		if t.Signal == signal {
			return t.ToRole
		}
	}
	return defaultFallbackRole
}
