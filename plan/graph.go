// graph.go — PlanGraph: DAG of Segments with cascade and claims (SPC-79, GOL-46).
//
// A Plan is a DAG of Segments. Each Segment has status, owner (claim),
// dependencies, children (nesting), and a ComponentMap describing what
// code it touches. Changes cascade through dependencies and spatial overlaps.
package plan

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// Sentinel errors.
var (
	ErrSegmentNotFound   = errors.New("plan: segment not found")
	ErrAlreadyClaimed    = errors.New("plan: segment already claimed")
	ErrNotClaimed        = errors.New("plan: segment not claimed")
	ErrInvalidTransition = errors.New("plan: invalid status transition")
)

// SegmentStatus represents the lifecycle state of a plan segment.
type SegmentStatus string

const (
	StatusDraft       SegmentStatus = "draft"       // needs operator input
	StatusReady       SegmentStatus = "ready"       // all deps met, can be claimed
	StatusClaimed     SegmentStatus = "claimed"     // agent has exclusive ownership
	StatusInProgress  SegmentStatus = "in_progress" // agent is working
	StatusComplete    SegmentStatus = "complete"    // work done
	StatusInvalidated SegmentStatus = "invalidated" // dependency changed, needs re-plan
)

// Segment is a single unit of work in a PlanGraph.
type Segment struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Status      SegmentStatus `json:"status"`
	Owner       string        `json:"owner,omitempty"` // agent that claimed it
	Content     string        `json:"content"`         // plan text / acceptance criteria
	Components  ComponentMap  `json:"components"`      // what code this segment touches
	DependsOn   []string      `json:"depends_on,omitempty"`
	Children    []string      `json:"children,omitempty"` // nested sub-segments
	Version     int           `json:"version"`
	Annotations []Annotation  `json:"annotations,omitempty"`
}

// ComponentMap describes what code a segment will create or modify.
type ComponentMap struct {
	Directories []string `json:"directories,omitempty"`
	Files       []string `json:"files,omitempty"`
	Symbols     []string `json:"symbols,omitempty"`
}

// Annotation is operator feedback on a segment.
type Annotation struct {
	Kind    string `json:"kind"` // "+", "-", "~"
	Comment string `json:"comment"`
}

// PlanGraph is a DAG of Segments with cascade and claim semantics.
type PlanGraph struct {
	Title    string `json:"title"`
	mu       sync.RWMutex
	segments map[string]*Segment
	nextID   atomic.Int64
}

// NewPlanGraph creates an empty plan graph.
func NewPlanGraph(title string) *PlanGraph {
	return &PlanGraph{
		Title:    title,
		segments: make(map[string]*Segment),
	}
}

// AddSegment adds a segment to the graph.
func (pg *PlanGraph) AddSegment(s Segment) string {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if s.ID == "" {
		s.ID = fmt.Sprintf("seg-%d", pg.nextID.Add(1))
	}
	if s.Status == "" {
		s.Status = StatusDraft
	}
	s.Version = 1
	pg.segments[s.ID] = &s
	return s.ID
}

// Get returns a segment by ID.
func (pg *PlanGraph) Get(id string) (*Segment, error) {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	s, ok := pg.segments[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSegmentNotFound, id)
	}
	return s, nil
}

// Claim gives exclusive ownership of a segment to an agent.
func (pg *PlanGraph) Claim(segmentID, owner string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	s, ok := pg.segments[segmentID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSegmentNotFound, segmentID)
	}
	if s.Status != StatusReady {
		return fmt.Errorf("%w: cannot claim segment in %s state", ErrInvalidTransition, s.Status)
	}
	if s.Owner != "" {
		return fmt.Errorf("%w: owned by %s", ErrAlreadyClaimed, s.Owner)
	}

	s.Owner = owner
	s.Status = StatusClaimed
	return nil
}

// Start transitions a claimed segment to in_progress.
func (pg *PlanGraph) Start(segmentID string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	s, ok := pg.segments[segmentID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSegmentNotFound, segmentID)
	}
	if s.Status != StatusClaimed {
		return fmt.Errorf("%w: cannot start segment in %s state", ErrInvalidTransition, s.Status)
	}
	s.Status = StatusInProgress
	return nil
}

// Complete marks a segment as done.
func (pg *PlanGraph) Complete(segmentID string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	s, ok := pg.segments[segmentID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSegmentNotFound, segmentID)
	}
	if s.Status != StatusInProgress {
		return fmt.Errorf("%w: cannot complete segment in %s state", ErrInvalidTransition, s.Status)
	}
	s.Status = StatusComplete
	s.Version++
	return nil
}

// Ready returns all segments whose dependencies are all complete.
func (pg *PlanGraph) Ready() []Segment {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	var ready []Segment
	for _, s := range pg.segments {
		if s.Status != StatusReady {
			continue
		}
		if pg.depsComplete(s) {
			ready = append(ready, *s)
		}
	}
	return ready
}

// DraftGaps returns segments still in draft status.
func (pg *PlanGraph) DraftGaps() []Segment {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	var gaps []Segment
	for _, s := range pg.segments {
		if s.Status == StatusDraft {
			gaps = append(gaps, *s)
		}
	}
	return gaps
}

// All returns all segments.
func (pg *PlanGraph) All() []Segment {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	result := make([]Segment, 0, len(pg.segments))
	for _, s := range pg.segments {
		result = append(result, *s)
	}
	return result
}

// TopoSort returns segments in dependency order (Kahn's algorithm).
func (pg *PlanGraph) TopoSort() []Segment {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	// Build in-degree map.
	inDegree := make(map[string]int)
	for id := range pg.segments {
		inDegree[id] = 0
	}
	for _, s := range pg.segments {
		for _, dep := range s.DependsOn {
			inDegree[s.ID]++
			_ = dep
		}
	}

	// Queue with zero in-degree.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []Segment
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		s := pg.segments[id]
		sorted = append(sorted, *s)

		// Reduce in-degree of dependents.
		for _, other := range pg.segments {
			for _, dep := range other.DependsOn {
				if dep == id {
					inDegree[other.ID]--
					if inDegree[other.ID] == 0 {
						queue = append(queue, other.ID)
					}
				}
			}
		}
	}

	return sorted
}

func (pg *PlanGraph) depsComplete(s *Segment) bool {
	for _, depID := range s.DependsOn {
		dep, ok := pg.segments[depID]
		if !ok || dep.Status != StatusComplete {
			return false
		}
	}
	return true
}
