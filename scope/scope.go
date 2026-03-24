// Package scope implements the three-level scope tree for Djinn.
//
// Three levels:
//   - General (root): sees all ecosystems. Default on startup.
//   - Ecosystem: collection of repos (e.g. aeon = 21 repos).
//   - System: single repo (e.g. aeon/djinn).
//
// The scope tree is navigated via /scope command or -e CLI flag.
// Scope is the TERRITORY. Staff is the ARMY deployed there.
// The General Secretary mailbox follows across all scope levels.
package scope

import (
	"fmt"
	"strings"
)

// Level represents a position in the scope hierarchy.
type Level int

const (
	General   Level = iota // root — sees all ecosystems
	Ecosystem              // collection of repos
	System                 // single repo
)

func (l Level) String() string {
	switch l {
	case General:
		return "general"
	case Ecosystem:
		return "ecosystem"
	case System:
		return "system"
	default:
		return "unknown"
	}
}

// Scope represents a node in the scope tree.
type Scope struct {
	Level    Level
	Name     string   // "general", "aeon", "djinn"
	Repos    []string // repo paths available at this scope
	Children []*Scope
	Parent   *Scope
}

// Path returns the full scope path: "general/aeon/djinn".
func (s *Scope) Path() string {
	if s.Parent == nil {
		return s.Name
	}
	return s.Parent.Path() + "/" + s.Name
}

// Navigator moves through the scope tree.
type Navigator struct {
	root    *Scope
	current *Scope
}

// NewNavigator creates a navigator starting at the general scope.
func NewNavigator() *Navigator {
	root := &Scope{
		Level: General,
		Name:  "general",
	}
	return &Navigator{root: root, current: root}
}

// AddEcosystem registers an ecosystem under the general scope.
func (n *Navigator) AddEcosystem(name string, repos []string) *Scope {
	eco := &Scope{
		Level:  Ecosystem,
		Name:   name,
		Repos:  repos,
		Parent: n.root,
	}
	// Add systems (one per repo).
	for _, repo := range repos {
		parts := strings.Split(repo, "/")
		sysName := parts[len(parts)-1]
		sys := &Scope{
			Level:  System,
			Name:   sysName,
			Repos:  []string{repo},
			Parent: eco,
		}
		eco.Children = append(eco.Children, sys)
	}
	n.root.Children = append(n.root.Children, eco)
	return eco
}

// Current returns the active scope.
func (n *Navigator) Current() *Scope {
	return n.current
}

// Path returns the current scope path.
func (n *Navigator) Path() string {
	return n.current.Path()
}

// Dive moves down one level by name.
// From general: dive into an ecosystem.
// From ecosystem: dive into a system.
func (n *Navigator) Dive(name string) error {
	// Support direct path: "aeon/djinn"
	parts := strings.SplitN(name, "/", 2)
	target := parts[0]

	for _, child := range n.current.Children {
		if child.Name == target {
			n.current = child
			// If there's a second part, dive again.
			if len(parts) > 1 {
				return n.Dive(parts[1])
			}
			return nil
		}
	}
	return fmt.Errorf("scope %q not found under %s", name, n.current.Path())
}

// Climb moves up one level. At general, stays.
func (n *Navigator) Climb() {
	if n.current.Parent != nil {
		n.current = n.current.Parent
	}
}

// Root jumps to the general scope.
func (n *Navigator) Root() {
	n.current = n.root
}

// ChildNames returns the names of children at the current scope.
func (n *Navigator) ChildNames() []string {
	names := make([]string, len(n.current.Children))
	for i, c := range n.current.Children {
		names[i] = c.Name
	}
	return names
}
