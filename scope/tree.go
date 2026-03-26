// tree.go — ScopeTree provides the hierarchical scope navigation for Djinn.
//
// Three levels: General (root) -> Ecosystem -> System.
// Navigate with cd-like semantics: ".." up, "/" root, relative, absolute paths.
// Previous node tracked for "cd -" support.
package scope

import (
	"fmt"
	"strings"
)

// ScopeLevel is the string representation of a scope's position in the hierarchy.
type ScopeLevel string

const (
	LevelGeneral   ScopeLevel = "general"   // / — no code
	LevelEcosystem ScopeLevel = "ecosystem" // /aeon — many repos
	LevelSystem    ScopeLevel = "system"    // /aeon/djinn — one repo
)

// ScopeNode is a node in the scope tree.
type ScopeNode struct {
	Path     string       `json:"path"`
	Name     string       `json:"name"`
	Level    ScopeLevel   `json:"level"`
	Repos    []string     `json:"repos,omitempty"`
	Children []*ScopeNode `json:"children,omitempty"`
	Parent   *ScopeNode   `json:"-"` // avoid circular JSON
}

// RequiresSandbox returns true for system-level scopes, where code execution
// happens and sandbox isolation is needed.
func (n *ScopeNode) RequiresSandbox() bool {
	return n.Level == LevelSystem
}

// HasCode returns true if the scope has any repos associated with it.
func (n *ScopeNode) HasCode() bool {
	return len(n.Repos) > 0
}

// ScopeTree is the three-level hierarchy: General -> Ecosystem -> System.
type ScopeTree struct {
	Root     *ScopeNode
	current  *ScopeNode
	previous *ScopeNode
}

// NewScopeTree creates a tree with a root general node at "/".
func NewScopeTree() *ScopeTree {
	root := &ScopeNode{
		Path:  "/",
		Name:  "/",
		Level: LevelGeneral,
	}
	return &ScopeTree{
		Root:    root,
		current: root,
	}
}

// AddEcosystem adds an ecosystem node under root with the given name and repos.
func (t *ScopeTree) AddEcosystem(name string, repos []string) *ScopeNode {
	node := &ScopeNode{
		Path:   "/" + name,
		Name:   name,
		Level:  LevelEcosystem,
		Repos:  repos,
		Parent: t.Root,
	}
	t.Root.Children = append(t.Root.Children, node)
	return node
}

// AddSystem adds a system node under an ecosystem. Returns nil if the
// ecosystem is not found.
func (t *ScopeTree) AddSystem(ecosystem, name, repoPath string) *ScopeNode {
	for _, child := range t.Root.Children {
		if child.Name == ecosystem && child.Level == LevelEcosystem {
			node := &ScopeNode{
				Path:   "/" + ecosystem + "/" + name,
				Name:   name,
				Level:  LevelSystem,
				Repos:  []string{repoPath},
				Parent: child,
			}
			child.Children = append(child.Children, node)
			return node
		}
	}
	return nil
}

// Navigate implements cd-like semantics:
//   - ".."           → move up one level
//   - "/"            → move to root
//   - "aeon/djinn"   → relative navigation from current
//   - "/aeon/djinn"  → absolute navigation from root
func (t *ScopeTree) Navigate(path string) (*ScopeNode, error) {
	if path == "/" {
		return t.Root, nil
	}

	if path == ".." {
		if t.current.Parent != nil {
			return t.current.Parent, nil
		}
		return t.current, nil
	}

	// Absolute path: start from root.
	var start *ScopeNode
	if strings.HasPrefix(path, "/") {
		start = t.Root
		path = strings.TrimPrefix(path, "/")
	} else {
		start = t.current
	}

	// Walk each segment.
	parts := strings.Split(path, "/")
	node := start
	for _, part := range parts {
		if part == "" {
			continue
		}
		if part == ".." {
			if node.Parent != nil {
				node = node.Parent
			}
			continue
		}
		found := false
		for _, child := range node.Children {
			if child.Name == part {
				node = child
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("scope %q not found under %s", part, node.Path)
		}
	}
	return node, nil
}

// Current returns the active scope node.
func (t *ScopeTree) Current() *ScopeNode {
	return t.current
}

// SetCurrent sets the active scope node.
func (t *ScopeTree) SetCurrent(node *ScopeNode) {
	t.current = node
}

// Pwd returns the current scope path.
func (t *ScopeTree) Pwd() string {
	return t.current.Path
}

// Ls returns the children of the current scope node.
func (t *ScopeTree) Ls() []*ScopeNode {
	return t.current.Children
}

// Previous returns the previously active scope node (for "cd -" support).
func (t *ScopeTree) Previous() *ScopeNode {
	return t.previous
}

// SetPrevious records the previous scope node.
func (t *ScopeTree) SetPrevious(node *ScopeNode) {
	t.previous = node
}
