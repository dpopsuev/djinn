// convert.go — converters between djinn Artifact and Parchment Artifact.
//
// Enables artifacts to flow between djinn's in-memory Graph (session planning)
// and Parchment's persistent Store (Scribe). Shared types (ComponentMap,
// Annotation) are already aliased — these functions bridge the rest.
package artifact

import (
	"fmt"
	"time"

	"github.com/dpopsuev/parchment"
)

// ToParchment converts a djinn Artifact to a Parchment Artifact.
// Fields without a Parchment equivalent are stored in Extra.
func ToParchment(a *Artifact) *parchment.Artifact {
	pa := &parchment.Artifact{
		ID:          a.ID,
		Kind:        a.Kind,
		Title:       a.Title,
		Status:      string(a.Status),
		Parent:      a.Parent,
		DependsOn:   a.DependsOn,
		Labels:      a.Labels,
		Components:  a.Components,
		Annotations: a.Annotations,
		CreatedAt:   a.Created,
		UpdatedAt:   a.Updated,
	}

	// Sections: map[string]string → []Section
	if len(a.Sections) > 0 {
		pa.Sections = make([]parchment.Section, 0, len(a.Sections))
		for name, text := range a.Sections {
			pa.Sections = append(pa.Sections, parchment.Section{Name: name, Text: text})
		}
	}

	// Content → Goal (closest semantic match).
	if a.Content != "" {
		pa.Goal = a.Content
	}

	// Priority: int → string
	if a.Priority != 0 {
		pa.Priority = fmt.Sprintf("%d", a.Priority)
	}

	// djinn-specific fields → Extra
	extra := make(map[string]any)
	if a.Owner != "" {
		extra["owner"] = a.Owner
	}
	if a.Version != 0 {
		extra["version"] = a.Version
	}
	if len(a.Children) > 0 {
		extra["children"] = a.Children
	}
	if len(extra) > 0 {
		pa.Extra = extra
	}

	return pa
}

// FromParchment converts a Parchment Artifact to a djinn Artifact.
// Fields without a djinn equivalent are silently dropped.
func FromParchment(pa *parchment.Artifact) Artifact {
	a := Artifact{
		ID:          pa.ID,
		Kind:        pa.Kind,
		Title:       pa.Title,
		Status:      Status(pa.Status),
		Parent:      pa.Parent,
		DependsOn:   pa.DependsOn,
		Labels:      pa.Labels,
		Components:  pa.Components,
		Annotations: pa.Annotations,
		Created:     pa.CreatedAt,
		Updated:     pa.UpdatedAt,
	}

	// Sections: []Section → map[string]string
	if len(pa.Sections) > 0 {
		a.Sections = make(map[string]string, len(pa.Sections))
		for _, s := range pa.Sections {
			a.Sections[s.Name] = s.Text
		}
	}

	// Goal → Content
	if pa.Goal != "" {
		a.Content = pa.Goal
	}

	// Priority: string → int (best effort)
	if pa.Priority != "" {
		fmt.Sscanf(pa.Priority, "%d", &a.Priority) //nolint:errcheck // best-effort parse
	}

	// Extra → djinn-specific fields
	extractExtra(&a, pa.Extra)

	// Default timestamps.
	if a.Created.IsZero() {
		a.Created = time.Now()
	}
	if a.Updated.IsZero() {
		a.Updated = a.Created
	}

	return a
}

func extractExtra(a *Artifact, extra map[string]any) {
	if extra == nil {
		return
	}
	if owner, ok := extra["owner"].(string); ok {
		a.Owner = owner
	}
	if version, ok := extra["version"].(float64); ok { // JSON numbers are float64
		a.Version = int(version)
	}
	if children, ok := extra["children"].([]any); ok {
		a.Children = make([]string, 0, len(children))
		for _, c := range children {
			if s, ok := c.(string); ok {
				a.Children = append(a.Children, s)
			}
		}
	}
}
