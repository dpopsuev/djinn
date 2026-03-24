package staff

import "testing"

func TestMapCommunicationStyle_Defaults(t *testing.T) {
	s := NewMapCommunicationStyle()
	if s.Get("density") != "sparse" {
		t.Fatalf("density = %q", s.Get("density"))
	}
	if s.Get("tone") != "casual" {
		t.Fatalf("tone = %q", s.Get("tone"))
	}
}

func TestMapCommunicationStyle_SetAndGet(t *testing.T) {
	s := NewMapCommunicationStyle()
	s.Set("density", "dense")
	if s.Get("density") != "dense" {
		t.Fatalf("after set: %q", s.Get("density"))
	}
}

func TestMapCommunicationStyle_All(t *testing.T) {
	s := NewMapCommunicationStyle()
	all := s.All()
	if len(all) != 8 {
		t.Fatalf("expected 8 dimensions, got %d", len(all))
	}
	// Modifying the returned map doesn't affect the original.
	all["density"] = "modified"
	if s.Get("density") == "modified" {
		t.Fatal("All() should return a copy")
	}
}

func TestMapCommunicationStyle_UnknownKey(t *testing.T) {
	s := NewMapCommunicationStyle()
	if s.Get("nonexistent") != "" {
		t.Fatal("unknown key should return empty string")
	}
}
