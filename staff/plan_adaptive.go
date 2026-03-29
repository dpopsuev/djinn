// plan_adaptive.go — AdaptiveStrategy with checkpoint reconciliation (TSK-428).
//
// After each task completes, checks for drift between planned vs actual.
// If drift exceeds threshold, proposes re-planning via the inner strategy.
package staff

import (
	"context"
	"fmt"

	"github.com/dpopsuev/djinn/tools"
)

// AdaptiveStrategy wraps an inner PlanStrategy with checkpoint logic.
// After each task completion, it checks the task store for drift and
// proposes re-planning if the drift exceeds a threshold.
type AdaptiveStrategy struct {
	inner     PlanStrategy
	store     *tools.TaskStore
	asker     GenSecAgent
	threshold float64 // re-plan if completed/total ratio diverges by this much
}

// NewAdaptiveStrategy creates an adaptive planning strategy.
// Threshold is the drift tolerance (0.0-1.0). Default: 0.2 (20% drift triggers re-plan).
func NewAdaptiveStrategy(inner PlanStrategy, store *tools.TaskStore, asker GenSecAgent, threshold float64) *AdaptiveStrategy {
	if threshold <= 0 {
		threshold = 0.2 //nolint:mnd // 20% drift tolerance
	}
	return &AdaptiveStrategy{
		inner:     inner,
		store:     store,
		asker:     asker,
		threshold: threshold,
	}
}

func (a *AdaptiveStrategy) Name() string { return "adaptive(" + a.inner.Name() + ")" }

// Refine delegates to the inner strategy.
func (a *AdaptiveStrategy) Refine(ctx context.Context, plan string) (string, error) {
	return a.inner.Refine(ctx, plan)
}

// Checkpoint checks for drift after a task completes.
// Returns a re-planned version if drift exceeds threshold, or empty string if on track.
func (a *AdaptiveStrategy) Checkpoint(ctx context.Context, currentPlan string) (revised string, drifted bool, err error) {
	if a.store == nil || a.asker == nil {
		return "", false, nil
	}

	tasks := a.store.List()
	if len(tasks) == 0 {
		return "", false, nil
	}

	completed := 0
	blocked := 0
	for _, t := range tasks {
		switch t.Status {
		case "completed", "done":
			completed++
		case "blocked", "failed":
			blocked++
		}
	}

	// Drift = blocked tasks as fraction of total.
	drift := float64(blocked) / float64(len(tasks))
	if drift < a.threshold {
		return "", false, nil // on track
	}

	// Drift exceeded — ask GenSec to propose re-plan.
	prompt := fmt.Sprintf(
		"ADAPTIVE CHECKPOINT — Plan drift detected.\n\n"+
			"Tasks: %d total, %d completed, %d blocked/failed (drift: %.0f%%)\n"+
			"Threshold: %.0f%%\n\n"+
			"Current plan:\n%s\n\n"+
			"Propose a revised plan that addresses the blocked tasks. "+
			"Keep completed work, re-scope or re-order the remaining tasks.",
		len(tasks), completed, blocked, drift*100, a.threshold*100, currentPlan)

	revised, err = a.asker.Ask(ctx, prompt)
	if err != nil {
		return "", false, fmt.Errorf("adaptive checkpoint: %w", err)
	}

	return revised, true, nil
}
