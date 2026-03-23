package staff

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

// NextRole returns the deterministic next role for a given signal.
// Pure function. Zero LLM tokens. The scheduler is a switch statement.
func NextRole(signal Signal) string {
	switch signal {
	case SignalPromptReceived:
		return "gensec"
	case SignalNeedCaptured:
		return "auditor"
	case SignalSpecStamped:
		return "scheduler"
	case SignalTasksPlanned:
		return "executor"
	case SignalExecutorDone:
		return "" // gate fires mechanically, not a role transition
	case SignalGatePassed:
		return "inspector"
	case SignalGateFailed:
		return "executor" // rework
	case SignalInspectorApproved:
		return "gensec" // report back to human
	case SignalInspectorRejected:
		return "executor" // rework with feedback
	default:
		return "gensec"
	}
}
