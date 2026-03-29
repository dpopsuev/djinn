// signal_prompts.go — Structured prompt templates per pillar (NED-16, TSK-472).
//
// Each pillar (budget, drift, performance, lifecycle) has a prompt template
// that formats signal data into a structured GenSec prompt. GenSec responds
// with JSON: {"action": "...", "reason": "...", "confidence": 0.N}.
package staff

import (
	"fmt"
	"strings"

	"github.com/dpopsuev/djinn/signal"
)

// SignalContext provides additional state for prompt enrichment.
type SignalContext struct {
	TasksTotal     int
	TasksDone      int
	BudgetPct      float64 // 0.0-1.0
	ActiveGear     Gear
	RecentSignals  []signal.Signal
	DriftScore     float64 // 0-100 for structure pillar
	DriftViolation string  // most severe violation
	TraceEvidence  string  // formatted recent trace events (from health analyzer)
}

// FormatSignalPrompt creates a structured prompt for GenSec based on a signal and context.
func FormatSignalPrompt(s signal.Signal, ctx SignalContext) string {
	var b strings.Builder

	b.WriteString("SIGNAL INTERPRETATION REQUEST\n\n")
	fmt.Fprintf(&b, "Category: %s\n", s.Category)
	fmt.Fprintf(&b, "Level: %s\n", s.Level)
	fmt.Fprintf(&b, "Source: %s\n", s.Source)
	fmt.Fprintf(&b, "Message: %s\n\n", s.Message)

	switch s.Category {
	case signal.CategoryBudget:
		formatBudgetPrompt(&b, ctx)
	case signal.CategoryDrift:
		formatDriftPrompt(&b, ctx)
	case signal.CategoryPerformance:
		formatPerformancePrompt(&b, ctx)
	case signal.CategoryLifecycle:
		formatLifecyclePrompt(&b, ctx)
	default:
		formatGenericPrompt(&b, ctx)
	}

	if ctx.TraceEvidence != "" {
		b.WriteString("\nTRACE EVIDENCE:\n")
		b.WriteString(ctx.TraceEvidence)
		b.WriteByte('\n')
	}

	b.WriteString("\nRespond with JSON only: {\"action\": \"<action>\", \"reason\": \"<reason>\", \"confidence\": <0.0-1.0>}")
	return b.String()
}

func formatBudgetPrompt(b *strings.Builder, ctx SignalContext) {
	b.WriteString("CONTEXT:\n")
	fmt.Fprintf(b, "  Budget usage: %.0f%%\n", ctx.BudgetPct*100)
	fmt.Fprintf(b, "  Tasks: %d/%d done\n", ctx.TasksDone, ctx.TasksTotal)
	fmt.Fprintf(b, "  Current gear: %s\n", ctx.ActiveGear)
	b.WriteString("\nOPTIONS:\n")
	b.WriteString("  continue  — keep working, budget is manageable\n")
	b.WriteString("  skip      — skip expensive remaining tasks\n")
	b.WriteString("  throttle  — downshift gear to reduce cost\n")
	b.WriteString("  abort     — stop all work, budget critical\n")
}

func formatDriftPrompt(b *strings.Builder, ctx SignalContext) {
	b.WriteString("CONTEXT:\n")
	fmt.Fprintf(b, "  Structure score: %.0f/100\n", ctx.DriftScore)
	if ctx.DriftViolation != "" {
		fmt.Fprintf(b, "  Worst violation: %s\n", ctx.DriftViolation)
	}
	fmt.Fprintf(b, "  Tasks: %d/%d done\n", ctx.TasksDone, ctx.TasksTotal)
	b.WriteString("\nOPTIONS:\n")
	b.WriteString("  continue  — drift is acceptable for current phase\n")
	b.WriteString("  re_scope  — narrow scope to reduce drift\n")
	b.WriteString("  re_plan   — revise plan to address violations\n")
	b.WriteString("  cordon    — pause and escalate to operator\n")
}

func formatPerformancePrompt(b *strings.Builder, ctx SignalContext) {
	b.WriteString("CONTEXT:\n")
	fmt.Fprintf(b, "  Current gear: %s\n", ctx.ActiveGear)
	fmt.Fprintf(b, "  Tasks: %d/%d done\n", ctx.TasksDone, ctx.TasksTotal)
	b.WriteString("\nOPTIONS:\n")
	b.WriteString("  continue  — bottleneck is transient\n")
	b.WriteString("  throttle  — reduce parallelism or tool usage\n")
	b.WriteString("  relay     — context relay to fresh agent\n")
	b.WriteString("  cordon    — pause and escalate to operator\n")
}

func formatLifecyclePrompt(b *strings.Builder, ctx SignalContext) {
	b.WriteString("CONTEXT:\n")
	fmt.Fprintf(b, "  Current gear: %s\n", ctx.ActiveGear)
	fmt.Fprintf(b, "  Tasks: %d/%d done\n", ctx.TasksDone, ctx.TasksTotal)
	b.WriteString("\nOPTIONS:\n")
	b.WriteString("  continue  — lifecycle event is expected\n")
	b.WriteString("  re_plan   — agent failure requires new approach\n")
	b.WriteString("  abort     — unrecoverable failure\n")
}

func formatGenericPrompt(b *strings.Builder, ctx SignalContext) {
	b.WriteString("CONTEXT:\n")
	fmt.Fprintf(b, "  Current gear: %s\n", ctx.ActiveGear)
	fmt.Fprintf(b, "  Tasks: %d/%d done\n", ctx.TasksDone, ctx.TasksTotal)
	b.WriteString("\nOPTIONS:\n")
	b.WriteString("  continue  — no action needed\n")
	b.WriteString("  cordon    — pause and escalate to operator\n")
}
