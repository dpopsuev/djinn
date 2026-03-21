package composition

import (
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	"github.com/dpopsuev/djinn/tier"
)

// ToWorkPlan converts a concrete formation into an orchestrator WorkPlan.
// Each executor unit becomes a Stage. Reviewers and observers are not
// stages in the sequential MVP — they'll become parallel nodes when
// OrigamiOrchestrator is available.
func ToWorkPlan(f Formation, id string) orchestrator.WorkPlan {
	plan := orchestrator.WorkPlan{ID: id}

	for _, u := range f.Units {
		if u.Role == RoleObserver {
			continue
		}

		scopeName := ""
		if len(u.Scope.RW) > 0 {
			scopeName = u.Scope.RW[0]
		} else if len(u.Scope.RO) > 0 {
			scopeName = u.Scope.RO[0]
		}

		plan.Stages = append(plan.Stages, orchestrator.Stage{
			Name:        u.Role + "-" + scopeName,
			Scope:       tier.Scope{Level: tierForRole(u.Role), Name: scopeName},
			Driver:      driver.DriverConfig{},
			Gate:        gate.GateConfig{Name: u.Role + "-gate", Severity: gate.SeverityBlocking},
			Prompt:      u.TerminatesWhen.Target,
			TimeBudget:  u.Budget.WallClock,
			TokenBudget: u.Budget.Tokens,
		})
	}

	return plan
}

func tierForRole(role string) tier.TierLevel {
	switch role {
	case RoleLead:
		return tier.Sys
	case RoleReviewer:
		return tier.Com
	case RoleExecutor:
		return tier.Mod
	default:
		return tier.Mod
	}
}
