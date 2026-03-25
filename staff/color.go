// color.go — assigns Bugle ColorIdentity to staff roles for TUI rendering.
package staff

import "github.com/dpopsuev/djinn/bugleport"

// AssignColor creates a ColorIdentity for a role within a scope.
func AssignColor(registry *bugleport.Registry, role, scope string) (bugleport.ColorIdentity, error) {
	return registry.Assign(role, scope)
}
