package composition

import (
	"fmt"
	"strings"
)

// Instantiate creates a concrete formation from a template by substituting
// scope variables and dividing the budget.
func Instantiate(tmpl Formation, scope string, totalBudget Budget) (Formation, error) {
	f := Formation{
		Name:      tmpl.Name,
		Units:     make([]Unit, len(tmpl.Units)),
		Edges:     make([]Edge, len(tmpl.Edges)),
		Budget:    totalBudget,
		Observers: make([]Unit, len(tmpl.Observers)),
	}

	for i, u := range tmpl.Units {
		f.Units[i] = substituteUnit(u, scope)
	}
	copy(f.Edges, tmpl.Edges)
	for i, o := range tmpl.Observers {
		f.Observers[i] = substituteUnit(o, scope)
	}

	f.DivideBudget()

	if err := f.Validate(); err != nil {
		return Formation{}, fmt.Errorf("instantiate %q: %w", tmpl.Name, err)
	}

	return f, nil
}

func substituteUnit(u Unit, scope string) Unit {
	out := Unit{
		Role:           u.Role,
		Budget:         u.Budget,
		TerminatesWhen: substituteTermination(u.TerminatesWhen, scope),
	}
	if u.Env != nil {
		out.Env = make(map[string]string, len(u.Env))
		for k, v := range u.Env {
			out.Env[k] = substituteVar(v, scope)
		}
	}
	out.Scope = UnitScope{
		RO: substituteSlice(u.Scope.RO, scope),
		RW: substituteSlice(u.Scope.RW, scope),
	}
	return out
}

func substituteTermination(t Termination, scope string) Termination {
	return Termination{
		Type:   t.Type,
		Target: substituteVar(t.Target, scope),
	}
}

func substituteSlice(paths []string, scope string) []string {
	if paths == nil {
		return nil
	}
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = substituteVar(p, scope)
	}
	return out
}

func substituteVar(s, scope string) string {
	return strings.ReplaceAll(s, templateScopeVar, scope)
}
