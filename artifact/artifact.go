// artifact.go — Universal work-unit primitive with template-enforced sections (GOL-59).
//
// Every work unit in Djinn is an Artifact: plans, tasks, delegations, bugs.
// The Kind field selects a Template from the TemplateRegistry, which validates
// required sections and status transitions. Zero external dependencies (domain layer).
package artifact

import (
	"time"

	"github.com/dpopsuev/parchment"
)

// Status is the lifecycle state of an artifact.
type Status string

// Status constants for all artifact kinds.
const (
	StatusDraft       Status = "draft"
	StatusReady       Status = "ready"
	StatusClaimed     Status = "claimed"
	StatusInProgress  Status = "in_progress"
	StatusComplete    Status = "complete"
	StatusInvalidated Status = "invalidated"
	StatusBlocked     Status = "blocked"
	StatusPending     Status = "pending"
	StatusActive      Status = "active"
	StatusDone        Status = "done"
)

// Artifact is the universal work-unit primitive.
type Artifact struct {
	// Identity
	ID   string `json:"id"`
	Kind string `json:"kind"` // template selector: "plan-segment", "task", etc.

	// Core
	Title   string `json:"title"`
	Status  Status `json:"status"`
	Owner   string `json:"owner,omitempty"`
	Content string `json:"content,omitempty"` // convenience — maps to Sections["content"]

	// Template-enforced sections
	Sections map[string]string `json:"sections,omitempty"`

	// Spatial anchoring (optional per kind)
	Components ComponentMap `json:"components"`

	// Graph
	DependsOn []string `json:"depends_on,omitempty"`
	Children  []string `json:"children,omitempty"`
	Parent    string   `json:"parent,omitempty"`

	// Metadata
	Labels      []string     `json:"labels,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
	Version     int          `json:"version"`
	Priority    int          `json:"priority,omitempty"`

	// Timestamps
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// ComponentMap describes what code an artifact will create or modify.
// Aliased from parchment — single source of truth.
type ComponentMap = parchment.ComponentMap

// Annotation is operator feedback on an artifact.
// Aliased from parchment — single source of truth.
type Annotation = parchment.Annotation
