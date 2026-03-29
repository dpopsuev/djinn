// narration.go — Agent narration mode (TSK-469).
//
// When enabled, GenSec automatically explains each stop as the operator
// navigates. Generates structured prompts for one-line explanations.
package review

import (
	"fmt"
	"strings"
)

// FormatNarrationPrompt creates a prompt asking GenSec to explain a stop.
func FormatNarrationPrompt(ctx ReviewContext, stopName string) string {
	var b strings.Builder
	b.WriteString("NARRATION REQUEST\n\n")
	fmt.Fprintf(&b, "The operator is reviewing changes and is now looking at: %s\n", stopName)
	fmt.Fprintf(&b, "Scope: %s\n", ctx.ScopeAnchor)
	fmt.Fprintf(&b, "Drift: %s\n", ctx.DriftVerdict)
	b.WriteString("\nProvide a ONE LINE explanation of why this stop was changed ")
	b.WriteString("and how it relates to the original request. Be concise.")
	return b.String()
}
