// context.go — ReviewContext for view awareness (TSK-467).
//
// Shared state injected into the agent's prompt during review mode.
// The agent sees the same context the operator sees.
package review

import (
	"fmt"
	"strings"
)

// Annotation records operator feedback on a specific file.
type Annotation struct {
	File    string `json:"file"`
	Kind    string `json:"kind"` // "+", "-", "~"
	Comment string `json:"comment"`
}

// ReviewContext captures the shared review state for agent prompt injection.
type ReviewContext struct {
	ScopeAnchor   string       `json:"scope_anchor"`
	DriftVerdict  DriftVerdict `json:"drift_verdict"`
	TriggerReason string       `json:"trigger_reason"`
	ChangedFiles  []string     `json:"changed_files"`
	FocusedFile   string       `json:"focused_file"`
	Annotations   []Annotation `json:"annotations,omitempty"`
}

// FormatPrompt renders the review context as a structured prompt block.
func (rc *ReviewContext) FormatPrompt() string {
	var b strings.Builder
	b.WriteString("[REVIEW MODE]\n")
	fmt.Fprintf(&b, "Scope: %s\n", rc.ScopeAnchor)
	fmt.Fprintf(&b, "Drift: %s\n", rc.DriftVerdict)
	if rc.TriggerReason != "" {
		fmt.Fprintf(&b, "Trigger: %s\n", rc.TriggerReason)
	}
	fmt.Fprintf(&b, "Files: %d changed\n", len(rc.ChangedFiles))
	if rc.FocusedFile != "" {
		fmt.Fprintf(&b, "Focus: %s\n", rc.FocusedFile)
	}
	if len(rc.Annotations) > 0 {
		b.WriteString("Annotations:\n")
		for i := range rc.Annotations {
			fmt.Fprintf(&b, "  %s %s: %s\n", rc.Annotations[i].Kind, rc.Annotations[i].File, rc.Annotations[i].Comment)
		}
	}
	return b.String()
}
