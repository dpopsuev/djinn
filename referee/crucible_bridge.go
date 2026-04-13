// crucible_bridge.go — bridges crucible.CheckResult into Referee scorecard.
//
// WorkspaceCheckRule converts a crucible workspace check into a scored event.
// The old Referee.Check() runs post-run, emits events, new Referee scores them.
//
// GOL-164, TSK-1091
package referee

import "github.com/dpopsuev/troupe/signal"

// EmitWorkspaceResult emits events for a workspace check result.
// Call after crucible.Referee.Check() completes. The Referee will
// score these events like any other.
func EmitWorkspaceResult(log signal.EventLog, pass bool, score float64, errors []string) {
	kind := "workspace.check.pass"
	if !pass {
		kind = "workspace.check.fail"
	}
	log.Emit(signal.Event{
		Source: "referee",
		Kind:   kind,
		Data: map[string]any{
			"pass":   pass,
			"score":  score,
			"errors": errors,
		},
	})
}
