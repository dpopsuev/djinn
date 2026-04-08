// color.go — assigns Bugle ColorIdentity to staff roles for TUI rendering.
package persona

import "github.com/dpopsuev/djinn/jerichoport"

// AssignColor creates a ColorIdentity for a role within a scope.
func AssignColor(registry *jerichoport.Registry, role, scope string) (jerichoport.Color, error) {
	return registry.Assign(role, scope)
}
