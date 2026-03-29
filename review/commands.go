// commands.go — Review command constants (TSK-440).
//
// Command names and help text for the review system.
// Actual TUI wiring happens when the review panel is built (Wave 2-3).
package review

// Command names for the review system.
const (
	CmdReview  = "review"  // halt agent, enter review mode
	CmdApprove = "approve" // commit changes, resume agent
	CmdReject  = "reject"  // discard changes, agent retries
	CmdSplit   = "split"   // partial approve/reject
	CmdTour    = "tour"    // launch visual reviewer
)

// CommandHelp returns usage text for review commands.
func CommandHelp(cmd string) string {
	switch cmd {
	case CmdReview:
		return ":review — halt agent and enter review mode"
	case CmdApprove:
		return ":approve [message] — commit reviewed changes and resume"
	case CmdReject:
		return ":reject [reason] — discard changes, agent retries with feedback"
	case CmdSplit:
		return ":split <approve-glob> — approve matching files, reject rest"
	case CmdTour:
		return ":tour [mode] — visual review via data flow circuits (summary|sigs|io|impact|debug|diagram)"
	default:
		return ""
	}
}
