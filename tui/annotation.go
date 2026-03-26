// annotation.go — Human-in-the-loop turn annotations for Djinn TUI.
// The operator can mark individual turns as GOOD, BAD, WASTE, BRILLIANT,
// or DANGEROUS. Annotations are stored in memory and surfaced in the UI.
package tui

import (
	"sync"
	"time"
)

// AnnotationKind classifies a human judgement on an agent turn.
type AnnotationKind string

const (
	AnnotationGood      AnnotationKind = "GOOD"
	AnnotationBad       AnnotationKind = "BAD"
	AnnotationWaste     AnnotationKind = "WASTE"
	AnnotationBrilliant AnnotationKind = "BRILLIANT"
	AnnotationDangerous AnnotationKind = "DANGEROUS"
)

// TurnAnnotation records a single human annotation on a turn.
type TurnAnnotation struct {
	TurnIdx int
	Kind    AnnotationKind
	Time    time.Time
}

// AnnotationStore accumulates turn annotations. Thread-safe.
type AnnotationStore struct {
	mu          sync.Mutex
	annotations []TurnAnnotation
}

// NewAnnotationStore creates an empty annotation store.
func NewAnnotationStore() *AnnotationStore {
	return &AnnotationStore{}
}

// Annotate records (or replaces) an annotation for the given turn.
// If the turn already has an annotation, it is overwritten.
func (s *AnnotationStore) Annotate(turnIdx int, kind AnnotationKind) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Replace existing annotation for this turn.
	for i, a := range s.annotations {
		if a.TurnIdx == turnIdx {
			s.annotations[i] = TurnAnnotation{
				TurnIdx: turnIdx,
				Kind:    kind,
				Time:    now,
			}
			return
		}
	}

	s.annotations = append(s.annotations, TurnAnnotation{
		TurnIdx: turnIdx,
		Kind:    kind,
		Time:    now,
	})
}

// GetAnnotation returns the annotation kind for a turn, if any.
func (s *AnnotationStore) GetAnnotation(turnIdx int) (AnnotationKind, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range s.annotations {
		if a.TurnIdx == turnIdx {
			return a.Kind, true
		}
	}
	return "", false
}

// Stats returns a count of annotations per kind.
func (s *AnnotationStore) Stats() map[AnnotationKind]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	counts := make(map[AnnotationKind]int)
	for _, a := range s.annotations {
		counts[a.Kind]++
	}
	return counts
}

// All returns a copy of all annotations, ordered by insertion time.
func (s *AnnotationStore) All() []TurnAnnotation {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TurnAnnotation, len(s.annotations))
	copy(out, s.annotations)
	return out
}
