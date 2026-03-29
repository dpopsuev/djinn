// hitl.go — Operator plan editing operations (GOL-47).
//
// GenSec calls DraftGaps() to find segments needing operator input.
// Operator fills gaps, annotates, injects new segments, reorders.
package plan

import "fmt"

// FillDraft fills a draft segment with content and marks it ready.
func (pg *PlanGraph) FillDraft(segmentID, content string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	s, ok := pg.segments[segmentID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSegmentNotFound, segmentID)
	}
	if s.Status != StatusDraft {
		return fmt.Errorf("%w: segment is %s, not draft", ErrInvalidTransition, s.Status)
	}

	s.Content = content
	s.Status = StatusReady
	s.Version++
	return nil
}

// Annotate adds operator feedback to a segment.
func (pg *PlanGraph) Annotate(segmentID, kind, comment string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	s, ok := pg.segments[segmentID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSegmentNotFound, segmentID)
	}

	s.Annotations = append(s.Annotations, Annotation{Kind: kind, Comment: comment})
	return nil
}

// Inject adds a new segment to the graph, linked to an existing segment.
func (pg *PlanGraph) Inject(afterID string, s Segment) (string, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if _, ok := pg.segments[afterID]; !ok {
		return "", fmt.Errorf("%w: %s", ErrSegmentNotFound, afterID)
	}

	if s.ID == "" {
		s.ID = fmt.Sprintf("seg-%d", pg.nextID.Add(1))
	}
	if s.Status == "" {
		s.Status = StatusReady
	}
	s.DependsOn = append(s.DependsOn, afterID)
	s.Version = 1
	pg.segments[s.ID] = &s
	return s.ID, nil
}

// Reorder changes the dependencies of a segment.
func (pg *PlanGraph) Reorder(segmentID string, newDeps []string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	s, ok := pg.segments[segmentID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrSegmentNotFound, segmentID)
	}

	// Validate all deps exist.
	for _, dep := range newDeps {
		if _, ok := pg.segments[dep]; !ok {
			return fmt.Errorf("%w: dependency %s", ErrSegmentNotFound, dep)
		}
	}

	s.DependsOn = newDeps
	s.Version++
	return nil
}
