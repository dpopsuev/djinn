package broker

import (
	"github.com/dpopsuev/djinn/artifact"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/workspace"
)

// ToWorkPlan converts a concrete formation into a WorkPlan.
// Each executor unit becomes a Stage. Reviewers and observers are not
// stages in the sequential MVP — they'll become parallel nodes when
// OrigamiOrchestrator is available.
func ToWorkPlan(f Formation, id string) WorkPlan {
	plan := WorkPlan{ID: id}

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

		plan.Stages = append(plan.Stages, Stage{
			Name:        u.Role + "-" + scopeName,
			Scope:       workspace.TierScope{Level: tierForRole(u.Role), Name: scopeName},
			Driver:      driver.DriverConfig{},
			Gate:        artifact.ContractGateConfig{Name: u.Role + "-gate", Severity: artifact.SeverityBlocking},
			Prompt:      u.TerminatesWhen.Target,
			TimeBudget:  u.Budget.WallClock,
			TokenBudget: u.Budget.Tokens,
		})
	}

	return plan
}

func tierForRole(role string) workspace.TierLevel {
	switch role {
	case RoleLead:
		return workspace.Sys
	case RoleReviewer:
		return workspace.Com
	case RoleExecutor:
		return workspace.Mod
	default:
		return workspace.Mod
	}
}
