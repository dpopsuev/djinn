package composition

import "fmt"

const (
	templateScopeVar     = "${task.scope}"
	defaultWorkspacePath = "/workspace"
)

// Sentinel error for template lookup.
var ErrTemplateNotFound = fmt.Errorf("formation template not found")

// TemplateSolo returns a single-executor formation.
func TemplateSolo() Formation {
	return Formation{
		Name: "solo",
		Units: []Unit{
			{
				Role:  RoleExecutor,
				Scope: UnitScope{RO: []string{defaultWorkspacePath}, RW: []string{templateScopeVar}},
				TerminatesWhen: Termination{Type: TermTestsPass, Target: templateScopeVar},
			},
		},
	}
}

// TemplateDuo returns a reviewer + executor formation.
func TemplateDuo() Formation {
	return Formation{
		Name: "duo",
		Units: []Unit{
			{
				Role:  RoleReviewer,
				Scope: UnitScope{RO: []string{templateScopeVar}},
				TerminatesWhen: Termination{Type: TermReviewerApproves},
			},
			{
				Role:  RoleExecutor,
				Scope: UnitScope{RO: []string{defaultWorkspacePath}, RW: []string{templateScopeVar}},
				TerminatesWhen: Termination{Type: TermTestsPass, Target: templateScopeVar},
			},
		},
		Edges: []Edge{
			{From: 1, To: 0, Channel: ChannelPromotionGate},
		},
	}
}

// TemplateSquad returns a lead + 3 executors formation.
func TemplateSquad() Formation {
	return Formation{
		Name: "squad",
		Units: []Unit{
			{
				Role:  RoleLead,
				Scope: UnitScope{RO: []string{templateScopeVar}},
				TerminatesWhen: Termination{Type: TermReviewerApproves},
			},
			{
				Role:  RoleExecutor,
				Scope: UnitScope{RO: []string{defaultWorkspacePath}, RW: []string{templateScopeVar + "/area-1"}},
				TerminatesWhen: Termination{Type: TermTestsPass},
			},
			{
				Role:  RoleExecutor,
				Scope: UnitScope{RO: []string{defaultWorkspacePath}, RW: []string{templateScopeVar + "/area-2"}},
				TerminatesWhen: Termination{Type: TermTestsPass},
			},
			{
				Role:  RoleExecutor,
				Scope: UnitScope{RO: []string{defaultWorkspacePath}, RW: []string{templateScopeVar + "/area-3"}},
				TerminatesWhen: Termination{Type: TermTestsPass},
			},
		},
		Edges: []Edge{
			{From: 1, To: 0, Channel: ChannelPromotionGate},
			{From: 2, To: 0, Channel: ChannelPromotionGate},
			{From: 3, To: 0, Channel: ChannelPromotionGate},
		},
	}
}

// TemplateByName returns a formation template by name.
func TemplateByName(name string) (Formation, error) {
	switch name {
	case "solo":
		return TemplateSolo(), nil
	case "duo":
		return TemplateDuo(), nil
	case "squad":
		return TemplateSquad(), nil
	default:
		return Formation{}, fmt.Errorf("%w: %q", ErrTemplateNotFound, name)
	}
}
