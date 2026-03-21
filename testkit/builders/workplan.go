package builders

import (
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	"github.com/dpopsuev/djinn/tier"
)

// WorkPlanBuilder provides a fluent API for constructing WorkPlans.
type WorkPlanBuilder struct {
	plan orchestrator.WorkPlan
}

// NewWorkPlan starts building a work plan with the given ID.
func NewWorkPlan(id string) *WorkPlanBuilder {
	return &WorkPlanBuilder{
		plan: orchestrator.WorkPlan{ID: id},
	}
}

// AddStage appends a stage to the work plan.
func (b *WorkPlanBuilder) AddStage(name string, scope tier.Scope, prompt string) *WorkPlanBuilder {
	b.plan.Stages = append(b.plan.Stages, orchestrator.Stage{
		Name:   name,
		Scope:  scope,
		Driver: driver.DriverConfig{Model: "stub"},
		Gate:   gate.GateConfig{Name: name + "-gate", Severity: gate.SeverityBlocking},
		Prompt: prompt,
	})
	return b
}

// Build returns the constructed work plan.
func (b *WorkPlanBuilder) Build() orchestrator.WorkPlan {
	return b.plan
}

// StandardFourTierPlan creates a standard 4-stage plan (analyze, code, test, review).
func StandardFourTierPlan(id string) orchestrator.WorkPlan {
	return NewWorkPlan(id).
		AddStage("analyze", tier.Scope{Level: tier.Eco, Name: "root"}, "analyze the codebase").
		AddStage("code", tier.Scope{Level: tier.Com, Name: "impl"}, "implement changes").
		AddStage("test", tier.Scope{Level: tier.Mod, Name: "tests"}, "run tests").
		AddStage("review", tier.Scope{Level: tier.Sys, Name: "review"}, "review changes").
		Build()
}
