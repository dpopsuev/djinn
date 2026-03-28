package staff

// Role name constants — used across the staff package for deterministic scheduling.
const (
	RoleGenSec    = "gensec"
	RoleAuditor   = "auditor"
	RoleScheduler = "scheduler"
	RoleExecutor  = "executor"
	RoleInspector = "inspector"
)

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
		return RoleGenSec
	case SignalNeedCaptured:
		return RoleAuditor
	case SignalSpecStamped:
		return RoleScheduler
	case SignalTasksPlanned:
		return RoleExecutor
	case SignalExecutorDone:
		return "" // gate fires mechanically, not a role transition
	case SignalGatePassed:
		return RoleInspector
	case SignalGateFailed:
		return RoleExecutor // rework
	case SignalInspectorApproved:
		return RoleGenSec // report back to human
	case SignalInspectorRejected:
		return RoleExecutor // rework with feedback
	default:
		return RoleGenSec
	}
}
