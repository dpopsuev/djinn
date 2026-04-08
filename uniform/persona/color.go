// color.go — assigns Bugle ColorIdentity to staff roles for TUI rendering.
package persona

import jIdentity "github.com/dpopsuev/troupe/identity"

// AssignColor creates a ColorIdentity for a role within a scope.
func AssignColor(registry *jIdentity.Registry, role, scope string) (jIdentity.Color, error) {
	return registry.Assign(role, scope)
}
