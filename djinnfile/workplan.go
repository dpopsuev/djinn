package djinnfile

import (
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	"github.com/dpopsuev/djinn/tier"
)

// ToWorkPlan converts a parsed Djinnfile into an orchestrator WorkPlan.
func (df *Djinnfile) ToWorkPlan(id string) orchestrator.WorkPlan {
	plan := orchestrator.WorkPlan{
		ID:     id,
		Stages: make([]orchestrator.Stage, len(df.Stages)),
	}

	for i, sc := range df.Stages {
		plan.Stages[i] = orchestrator.Stage{
			Name:  sc.Name,
			Scope: tier.Scope{Level: parseTierLevel(sc.Tier), Name: sc.Scope},
			Driver: driver.DriverConfig{
				Model:       df.Driver.Model,
				MaxTokens:   df.Driver.MaxTokens,
				Temperature: df.Driver.Temperature,
			},
			Gate: gate.GateConfig{
				Name:     sc.Gate.Name,
				Severity: sc.Gate.Severity,
			},
			Prompt:      sc.Prompt,
			TimeBudget:  sc.parsedTimeBudget,
			TokenBudget: sc.TokenBudget,
		}
	}

	return plan
}

func parseTierLevel(s string) tier.TierLevel {
	switch s {
	case "eco":
		return tier.Eco
	case "sys":
		return tier.Sys
	case "com":
		return tier.Com
	default:
		return tier.Mod
	}
}
