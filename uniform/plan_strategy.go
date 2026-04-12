// plan_strategy.go — Pluggable plan compilation strategies (TSK-427).
//
// PlanStrategy controls HOW a plan is refined: single-agent (Direct)
// or multi-agent debate (Dialectic via Bugle CollectiveStrategy).
// Selected by gear: E0-E1 = Direct, E2+ = Dialectic.
package uniform

import (
	"context"

	"github.com/dpopsuev/djinn/uniform/execution"
)

// Asker is the interface for querying any agent persona.
// Consumer-defined (ISP) — each package defines its own.
// Not role-specific — any persona can satisfy this.
type Asker interface {
	Ask(ctx context.Context, content string) (string, error)
}

// PlanStrategy refines a plan draft. Different strategies produce
// different levels of scrutiny.
type PlanStrategy interface {
	Name() string
	Refine(ctx context.Context, plan string) (string, error)
}

// DirectPlanStrategy is single-agent planning — GenSec produces the plan alone.
// This is the current default behavior.
type DirectPlanStrategy struct{}

func (s *DirectPlanStrategy) Name() string { return "direct" }

func (s *DirectPlanStrategy) Refine(_ context.Context, plan string) (string, error) {
	return plan, nil // pass-through — GenSec already produced the plan
}

// DialecticPlanStrategy uses two agents debating via Bugle's Dialectic
// collective to refine the plan. Thesis proposes, antithesis challenges,
// convergence produces a stronger plan.
type DialecticPlanStrategy struct {
	asker Asker // GenSec for thesis/antithesis prompts
}

// NewDialecticPlanStrategy creates a dialectic planning strategy.
func NewDialecticPlanStrategy(asker Asker) *DialecticPlanStrategy {
	return &DialecticPlanStrategy{asker: asker}
}

func (s *DialecticPlanStrategy) Name() string { return "dialectic" }

func (s *DialecticPlanStrategy) Refine(ctx context.Context, plan string) (string, error) {
	// Thesis: the original plan.
	// Antithesis: ask GenSec to challenge the plan.
	challenge, err := s.asker.Ask(ctx,
		"PLAN REVIEW — Challenge this plan. Find gaps, risks, missing dependencies, "+
			"scope creep, architectural violations. Be adversarial.\n\nPLAN:\n"+plan)
	if err != nil {
		return plan, nil // fallback to original if challenge fails
	}

	// Synthesis: ask GenSec to reconcile thesis + antithesis.
	synthesis, err := s.asker.Ask(ctx,
		"PLAN SYNTHESIS — Reconcile the original plan with the challenges. "+
			"Keep what survives scrutiny, fix what doesn't. Output the refined plan.\n\n"+
			"ORIGINAL PLAN:\n"+plan+"\n\nCHALLENGES:\n"+challenge)
	if err != nil {
		return plan, nil // fallback to original if synthesis fails
	}

	return synthesis, nil
}

// PlanStrategyForGear returns the appropriate strategy based on gear level.
// E0-E1: Direct (single-agent). E2+: Dialectic (multi-agent scrutiny).
func PlanStrategyForGear(gear execution.Gear, asker Asker) PlanStrategy {
	switch gear {
	case execution.GearE2, execution.GearE3, execution.GearAuto:
		if asker != nil {
			return NewDialecticPlanStrategy(asker)
		}
		return &DirectPlanStrategy{}
	default:
		return &DirectPlanStrategy{}
	}
}
