package builders

import (
	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/workspace"
)

// WorkPlanBuilder provides a fluent API for constructing WorkPlans.
type WorkPlanBuilder struct {
	plan broker.WorkPlan
}

// NewWorkPlan starts building a work plan with the given ID.
func NewWorkPlan(id string) *WorkPlanBuilder {
	return &WorkPlanBuilder{
		plan: broker.WorkPlan{ID: id},
	}
}

// AddStage appends a stage to the work plan.
func (b *WorkPlanBuilder) AddStage(name string, scope workspace.TierScope, prompt string) *WorkPlanBuilder {
	b.plan.Stages = append(b.plan.Stages, broker.Stage{
		Name:   name,
		Scope:  scope,
		Driver: driver.DriverConfig{Model: "stub"},
		Gate:   artifact.ContractGateConfig{Name: name + "-gate", Severity: artifact.SeverityBlocking},
		Prompt: prompt,
	})
	return b
}

// Build returns the constructed work plan.
func (b *WorkPlanBuilder) Build() broker.WorkPlan {
	return b.plan
}

// StandardFourTierPlan creates a standard 4-stage plan (analyze, code, test, review).
func StandardFourTierPlan(id string) broker.WorkPlan {
	return NewWorkPlan(id).
		AddStage("analyze", workspace.TierScope{Level: workspace.Eco, Name: "root"}, "analyze the codebase").
		AddStage("code", workspace.TierScope{Level: workspace.Com, Name: "impl"}, "implement changes").
		AddStage("test", workspace.TierScope{Level: workspace.Mod, Name: "tests"}, "run tests").
		AddStage("review", workspace.TierScope{Level: workspace.Sys, Name: "review"}, "review changes").
		Build()
}
