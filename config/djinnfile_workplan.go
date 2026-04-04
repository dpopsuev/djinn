package config

import (
	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/workspace"
)

// ToWorkPlan converts a parsed Djinnfile into an orchestrator WorkPlan.
func (df *Djinnfile) ToWorkPlan(id string) broker.WorkPlan {
	plan := broker.WorkPlan{
		ID:     id,
		Stages: make([]broker.Stage, len(df.Stages)),
	}

	for i := range df.Stages {
		sc := df.Stages[i]
		plan.Stages[i] = broker.Stage{
			Name:  sc.Name,
			Scope: workspace.TierScope{Level: parseTierLevel(sc.Tier), Name: sc.Scope},
			Driver: driver.DriverConfig{
				Model:       df.Driver.Model,
				MaxTokens:   df.Driver.MaxTokens,
				Temperature: df.Driver.Temperature,
			},
			Gate: artifact.ContractGateConfig{
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

func parseTierLevel(s string) workspace.TierLevel {
	switch s {
	case TierEco:
		return workspace.Eco
	case TierSys:
		return workspace.Sys
	case TierCom:
		return workspace.Com
	default:
		return workspace.Mod
	}
}
