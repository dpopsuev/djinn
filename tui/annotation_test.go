package tui

import "testing"

func TestAnnotationStore_NewIsEmpty(t *testing.T) {
	s := NewAnnotationStore()
	all := s.All()
	if len(all) != 0 {
		t.Fatalf("new store should be empty, got %d", len(all))
	}
	stats := s.Stats()
	if len(stats) != 0 {
		t.Fatalf("new store stats should be empty, got %+v", stats)
	}
}

func TestAnnotationStore_AnnotateAndGet(t *testing.T) {
	s := NewAnnotationStore()
	s.Annotate(1, AnnotationGood)
	s.Annotate(2, AnnotationBad)

	kind, ok := s.GetAnnotation(1)
	if !ok || kind != AnnotationGood {
		t.Fatalf("turn 1: kind = %q, ok = %v", kind, ok)
	}
	kind, ok = s.GetAnnotation(2)
	if !ok || kind != AnnotationBad {
		t.Fatalf("turn 2: kind = %q, ok = %v", kind, ok)
	}
}

func TestAnnotationStore_GetMissing(t *testing.T) {
	s := NewAnnotationStore()
	_, ok := s.GetAnnotation(99)
	if ok {
		t.Fatal("expected not-found for unannotated turn")
	}
}

func TestAnnotationStore_OverwriteAnnotation(t *testing.T) {
	s := NewAnnotationStore()
	s.Annotate(1, AnnotationGood)
	s.Annotate(1, AnnotationDangerous) // overwrite

	kind, ok := s.GetAnnotation(1)
	if !ok || kind != AnnotationDangerous {
		t.Fatalf("overwritten turn 1: kind = %q, want DANGEROUS", kind)
	}
	// Should still be only 1 annotation total.
	all := s.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 annotation after overwrite, got %d", len(all))
	}
}

func TestAnnotationStore_Stats(t *testing.T) {
	s := NewAnnotationStore()
	s.Annotate(1, AnnotationGood)
	s.Annotate(2, AnnotationGood)
	s.Annotate(3, AnnotationBad)
	s.Annotate(4, AnnotationWaste)
	s.Annotate(5, AnnotationBrilliant)

	stats := s.Stats()
	if stats[AnnotationGood] != 2 {
		t.Fatalf("GOOD = %d, want 2", stats[AnnotationGood])
	}
	if stats[AnnotationBad] != 1 {
		t.Fatalf("BAD = %d, want 1", stats[AnnotationBad])
	}
	if stats[AnnotationWaste] != 1 {
		t.Fatalf("WASTE = %d, want 1", stats[AnnotationWaste])
	}
	if stats[AnnotationBrilliant] != 1 {
		t.Fatalf("BRILLIANT = %d, want 1", stats[AnnotationBrilliant])
	}
	if stats[AnnotationDangerous] != 0 {
		t.Fatalf("DANGEROUS = %d, want 0", stats[AnnotationDangerous])
	}
}

func TestAnnotationStore_AllReturnsCopy(t *testing.T) {
	s := NewAnnotationStore()
	s.Annotate(1, AnnotationGood)

	all1 := s.All()
	all1[0].Kind = AnnotationBad // mutate the copy

	// Original should be unchanged.
	kind, _ := s.GetAnnotation(1)
	if kind != AnnotationGood {
		t.Fatal("All() should return a copy, mutation should not affect store")
	}
}

func TestAnnotationStore_MultipleAnnotations(t *testing.T) {
	s := NewAnnotationStore()
	for i := 0; i < 10; i++ {
		s.Annotate(i, AnnotationGood)
	}
	all := s.All()
	if len(all) != 10 {
		t.Fatalf("expected 10 annotations, got %d", len(all))
	}
}

func TestAnnotationKindConstants(t *testing.T) {
	// Verify constants are distinct.
	kinds := []AnnotationKind{
		AnnotationGood, AnnotationBad, AnnotationWaste,
		AnnotationBrilliant, AnnotationDangerous,
	}
	seen := make(map[AnnotationKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Fatalf("duplicate annotation kind: %q", k)
		}
		seen[k] = true
	}
}
