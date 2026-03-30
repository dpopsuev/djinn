// graph.go — Backward-compatibility shim over artifact.Graph (GOL-59).
//
// All types are aliases to artifact/. PlanGraph wraps artifact.Graph with
// the original API (AddSegment returns string, Annotate takes 3 args, etc.).
// All 13 plan_test.go tests pass unchanged.
package plan

import (
	"github.com/dpopsuev/djinn/artifact"
)

// Type aliases for backward compatibility.
type (
	SegmentStatus = artifact.Status
	Segment       = artifact.Artifact
	ComponentMap  = artifact.ComponentMap
	Annotation    = artifact.Annotation
)

// Status constants re-exported.
const (
	StatusDraft       = artifact.StatusDraft
	StatusReady       = artifact.StatusReady
	StatusClaimed     = artifact.StatusClaimed
	StatusInProgress  = artifact.StatusInProgress
	StatusComplete    = artifact.StatusComplete
	StatusInvalidated = artifact.StatusInvalidated
)

// Sentinel errors re-exported.
var (
	ErrSegmentNotFound   = artifact.ErrNotFound
	ErrAlreadyClaimed    = artifact.ErrAlreadyClaimed
	ErrNotClaimed        = artifact.ErrNotClaimed
	ErrInvalidTransition = artifact.ErrInvalidTransition
)

// PlanGraph wraps artifact.Graph with plan-segment defaults.
type PlanGraph struct {
	G     *artifact.Graph // exported for hub migration
	Title string
}

// NewPlanGraph creates a plan graph backed by artifact.Graph.
func NewPlanGraph(title string) *PlanGraph {
	return &PlanGraph{
		G:     artifact.NewGraph(title, artifact.DefaultRegistry()),
		Title: title,
	}
}

// AddSegment adds a plan segment and returns its ID.
// Sets Kind=plan-segment automatically. Never returns an error (original API).
func (pg *PlanGraph) AddSegment(s Segment) string {
	s.Kind = artifact.KindPlanSegment
	id, _ := pg.G.Add(s) //nolint:errcheck // original API never returned error
	return id
}

// Get returns a segment by ID.
func (pg *PlanGraph) Get(id string) (*Segment, error) {
	return pg.G.Get(id)
}

// Claim gives exclusive ownership to an agent.
func (pg *PlanGraph) Claim(id, owner string) error {
	return pg.G.Claim(id, owner)
}

// Start transitions a claimed segment to in_progress.
func (pg *PlanGraph) Start(id string) error {
	return pg.G.Start(id)
}

// Complete marks a segment as done.
func (pg *PlanGraph) Complete(id string) error {
	return pg.G.Complete(id)
}

// Ready returns segments whose dependencies are all complete.
func (pg *PlanGraph) Ready() []Segment {
	return pg.G.Ready()
}

// DraftGaps returns segments in draft status.
func (pg *PlanGraph) DraftGaps() []Segment {
	return pg.G.DraftGaps()
}

// All returns all segments.
func (pg *PlanGraph) All() []Segment {
	return pg.G.All()
}

// TopoSort returns segments in dependency order.
func (pg *PlanGraph) TopoSort() []Segment {
	return pg.G.TopoSort()
}

// Cascade propagates invalidation via deps + spatial overlap.
func (pg *PlanGraph) Cascade(changedID string) []string {
	return pg.G.Cascade(changedID)
}

// Overlaps returns files shared between two segments' ComponentMaps.
func (pg *PlanGraph) Overlaps(idA, idB string) []string {
	return pg.G.Overlaps(idA, idB)
}

// FillDraft transitions a draft segment to ready with content.
func (pg *PlanGraph) FillDraft(id, content string) error {
	return pg.G.FillDraft(id, content)
}

// Annotate adds operator feedback. Bridges the 3-arg API to artifact.Annotation.
func (pg *PlanGraph) Annotate(id, kind, comment string) {
	pg.G.Annotate(id, artifact.Annotation{Kind: kind, Comment: comment}) //nolint:errcheck // original API was void
}

// Inject adds a new segment linked to an existing one.
func (pg *PlanGraph) Inject(parentID string, s Segment) (string, error) { //nolint:gocritic // matches original API signature
	s.Kind = artifact.KindPlanSegment
	return pg.G.Inject(parentID, s)
}

// Reorder changes a segment's dependencies.
func (pg *PlanGraph) Reorder(id string, newDeps []string) error {
	return pg.G.Reorder(id, newDeps)
}
