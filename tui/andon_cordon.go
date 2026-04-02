// andon_cordon.go — auto-pause logic for critical andon levels.
//
// ShouldCordon bridges the andon visual system to the cordon mechanism.
// When andon is red, executors should be paused. The dashboard emits a
// CordonMsg which the REPL model handles via broker/cordon.go.
package tui

// ShouldCordon returns true if the andon level warrants auto-pausing executors.
// Only red (critical) triggers a cordon — yellow is a warning, not a stop.
func ShouldCordon(level AndonLevel) bool {
	return level == AndonRed
}
