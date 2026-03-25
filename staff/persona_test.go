package staff

import "testing"

func TestResolvePersona_AllRoles(t *testing.T) {
	for role, personaName := range RolePersona {
		p, ok := ResolvePersona(role)
		if !ok {
			t.Fatalf("role %q → persona %q not found", role, personaName)
		}
		if p.Identity.PersonaName != personaName {
			t.Fatalf("role %q → got %q, want %q", role, p.Identity.PersonaName, personaName)
		}
	}
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
	if len(all) != len(RolePersona) {
		t.Fatalf("resolved %d/%d personas", len(all), len(RolePersona))
	}
}
