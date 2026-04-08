package persona

import "testing"

func TestResolvePersona_KnownRoles(t *testing.T) {
	resolved := 0
	for role, personaName := range RolePersona {
		p, ok := ResolvePersona(role)
		if ok {
			resolved++
			if p.Name != personaName {
				t.Errorf("role %q → got %q, want %q", role, p.Name, personaName)
			}
		}
	}
	if resolved == 0 {
		t.Fatal("no personas resolved — Troupe identity package may be misconfigured")
	}
	t.Logf("resolved %d/%d roles", resolved, len(RolePersona))
}

func TestResolvePersona_Unknown(t *testing.T) {
	_, ok := ResolvePersona("nonexistent")
	if ok {
		t.Fatal("unknown role should return false")
	}
}

func TestAllRolePersonas(t *testing.T) {
	all := AllRolePersonas()
	if len(all) == 0 {
		t.Fatal("should resolve at least some personas")
	}
	// Troupe has 5 archetypes vs Jericho's 8 personas.
	// Not all role mappings resolve — this is expected during migration.
	t.Logf("resolved %d/%d personas", len(all), len(RolePersona))
}
