// template.go — Template enforcement for artifact kinds (GOL-59).
//
// Templates are Go code (Validate functions), not data schemas.
// The TemplateRegistry maps Kind strings to Templates. Graph.Add
// validates against the template before storing.
package artifact

import (
	"errors"
	"fmt"
	"slices"
)

// Sentinel errors for template validation.
var (
	ErrTemplateNotFound = errors.New("artifact: template not found")
	ErrMissingSection   = errors.New("artifact: missing required section")
	ErrKindMismatch     = errors.New("artifact: kind does not match template")
)

// Template defines validation rules for an artifact kind.
type Template struct {
	Kind             string                  // must match Artifact.Kind
	RequiredSections []string                // sections that must be non-empty
	ValidStatuses    []Status                // allowed statuses for this kind
	IDFormat         string                  // fmt format string, e.g. "seg-%d" or "T-%03d"
	Validate         func(a *Artifact) error // custom validation (nil = no custom)
}

// Check validates an artifact against this template.
func (t *Template) Check(a *Artifact) error {
	if a.Kind != t.Kind {
		return fmt.Errorf("%w: got %q, want %q", ErrKindMismatch, a.Kind, t.Kind)
	}
	for _, name := range t.RequiredSections {
		if a.Sections == nil || a.Sections[name] == "" {
			// Check Content field as fallback for "content" section.
			if name == "content" && a.Content != "" {
				continue
			}
			return fmt.Errorf("%w: %s", ErrMissingSection, name)
		}
	}
	if t.Validate != nil {
		return t.Validate(a)
	}
	return nil
}

// HasStatus checks whether a status is valid for this template.
func (t *Template) HasStatus(s Status) bool {
	return slices.Contains(t.ValidStatuses, s)
}

// TemplateRegistry maps Kind strings to Templates.
type TemplateRegistry struct {
	templates map[string]*Template
}

// NewTemplateRegistry creates an empty registry.
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{templates: make(map[string]*Template)}
}

// Register adds a template to the registry, keyed by Kind.
func (r *TemplateRegistry) Register(t Template) {
	r.templates[t.Kind] = &t
}

// Get returns a template by kind.
func (r *TemplateRegistry) Get(kind string) (*Template, error) {
	t, ok := r.templates[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrTemplateNotFound, kind)
	}
	return t, nil
}

// Check validates an artifact against its kind's template.
func (r *TemplateRegistry) Check(a *Artifact) error {
	t, err := r.Get(a.Kind)
	if err != nil {
		return err
	}
	return t.Check(a)
}
