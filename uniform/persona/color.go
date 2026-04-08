// color.go — assigns Bugle ColorIdentity to staff roles for TUI rendering.
package persona

import jSymbol "github.com/dpopsuev/jericho/symbol"

// AssignColor creates a ColorIdentity for a role within a scope.
func AssignColor(registry *jSymbol.Registry, role, scope string) (jSymbol.Color, error) {
	return registry.Assign(role, scope)
}
