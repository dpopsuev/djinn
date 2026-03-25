// persona.go — maps Djinn staff roles to Bugle personas.
// Each role gets a stable persona identity that persists across LLM hotswap.
package staff

import "github.com/dpopsuev/djinn/bugleport"

// RolePersona maps a Djinn staff role to a Bugle persona name.
// Persona provides: element, color, position, alignment, step affinity.
var RolePersona = map[string]string{
	"gensec":       "Herald",    // Broker: fast intake, optimistic routing
	"executor":     "Seeker",    // Worker: deep investigator, builds evidence
	"inspector":    "Specter",   // Enforcer: fastest path to contradiction
	"manager":      "Weaver",    // Manager: holistic synthesizer
	"auditor":      "Bulwark",   // Enforcer (pre): precision verifier
	"scheduler":    "Challenger", // Broker (pre): aggressive skeptic
	"externalizer": "Sentinel",  // Worker: steady resolver
	"investigator": "Abyss",    // Enforcer: deep adversary
}

// ResolvePersona returns the Bugle persona for a Djinn role.
// Returns the persona and true if found, zero value and false otherwise.
func ResolvePersona(role string) (bugleport.Persona, bool) {
	personaName, ok := RolePersona[role]
	if !ok {
		return bugleport.Persona{}, false
	}
	return bugleport.PersonaByName(personaName)
}

// AllRolePersonas returns all role→persona mappings that resolve.
func AllRolePersonas() map[string]bugleport.Persona {
	out := make(map[string]bugleport.Persona, len(RolePersona))
	for role, name := range RolePersona {
		if p, ok := bugleport.PersonaByName(name); ok {
			out[role] = p
		}
		_ = name
	}
	return out
}
