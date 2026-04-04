package broker

import (
	"errors"
	"fmt"
)

// ErrOverlappingScopes indicates two non-observer units have overlapping RW paths.
var ErrOverlappingScopes = errors.New("overlapping RW scopes between units")

// ValidateScopeDisjointness ensures no two non-observer units share RW paths.
func ValidateScopeDisjointness(units []Unit) error {
	for i := range units {
		if units[i].IsObserver() {
			continue
		}
		for j := i + 1; j < len(units); j++ {
			if units[j].IsObserver() {
				continue
			}
			if pathsOverlap(units[i].Scope.RW, units[j].Scope.RW) {
				return fmt.Errorf("%w: unit[%d] and unit[%d]", ErrOverlappingScopes, i, j)
			}
		}
	}
	return nil
}
