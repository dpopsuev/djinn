// signal_router.go — scope-aware signal visibility.
//
// Signals propagate upward: a signal emitted at /aeon/djinn is visible
// to listeners at /aeon/djinn, /aeon, and /, but NOT to /aeon/bugle
// or /personal.
//
// Rule: a listener at scope L sees a signal from scope S if and only if
// L is an ancestor of S (or L == S). Ancestor means L is a prefix of S
// when both are treated as path segments.
package scope

import "strings"

// SignalRouter determines whether signals should propagate between scopes.
type SignalRouter struct {
	tree *ScopeTree
}

// NewSignalRouter creates a router backed by the given scope tree.
func NewSignalRouter(tree *ScopeTree) *SignalRouter {
	return &SignalRouter{tree: tree}
}

// ShouldPropagate returns true if a listener at listenerScope should receive
// a signal emitted from signalScope.
//
// Visibility rules:
//   - Same scope: always visible
//   - Parent (ancestor) sees child signals
//   - Root "/" sees everything
//   - Siblings do NOT see each other's signals
//   - Different subtrees do NOT see each other's signals
func (r *SignalRouter) ShouldPropagate(signalScope, listenerScope string) bool {
	// Normalize: ensure leading "/" and no trailing "/".
	sig := normalizePath(signalScope)
	lis := normalizePath(listenerScope)

	// Exact match.
	if sig == lis {
		return true
	}

	// Listener must be an ancestor (prefix) of the signal scope.
	// "/" is ancestor of everything.
	if lis == "/" {
		return true
	}

	// Check that signalScope starts with listenerScope + "/".
	// This ensures /aeon matches /aeon/djinn but NOT /aeon2/djinn.
	return strings.HasPrefix(sig, lis+"/")
}

// normalizePath ensures a path starts with "/" and has no trailing "/".
func normalizePath(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimRight(p, "/")
}
