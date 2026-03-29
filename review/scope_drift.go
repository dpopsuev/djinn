// scope_drift.go — Scope drift GenSec prompt template (TSK-475).
//
// When the scope anchor detects drift, this formats a structured prompt
// for GenSec to interpret whether the drift is acceptable or scope creep.
package review

import (
	"fmt"
	"strings"
)

// FormatScopeDriftPrompt creates a prompt for GenSec to interpret scope drift.
func FormatScopeDriftPrompt(anchor *ScopeAnchor, driftedFiles []string) string {
	var b strings.Builder
	b.WriteString("SCOPE DRIFT INTERPRETATION REQUEST\n\n")
	fmt.Fprintf(&b, "Original request: %s\n", anchor.OriginalRequest)
	fmt.Fprintf(&b, "Expected packages: %s\n", strings.Join(anchor.ExpectedPackages, ", "))
	fmt.Fprintf(&b, "Files outside scope: %s\n\n", strings.Join(driftedFiles, ", "))
	b.WriteString("Is this drift acceptable (the change requires touching related packages) ")
	b.WriteString("or is it scope creep (the agent wandered into unrelated code)?\n\n")
	b.WriteString("Respond with JSON: {\"action\": \"continue|re_scope|cordon\", \"reason\": \"...\", \"confidence\": 0.N}")
	return b.String()
}
