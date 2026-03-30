package elements

import "testing"

func TestGlyph_AllStates(t *testing.T) {
	for _, state := range []string{StateDone, StateActive, StateError, StatePending, "unknown"} {
		result := Glyph(state)
		if result == "" {
			t.Errorf("Glyph(%q) returned empty string", state)
		}
	}
}

func TestBadge(t *testing.T) {
	result := Badge("tokens", 8150)
	if result != "8.2k tokens" {
		t.Errorf("Badge = %q, want %q", result, "8.2k tokens")
	}
}

func TestCompactNumber(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{42, "42"},
		{1200, "1.2k"},
		{3400000, "3.4M"},
		{-1200, "-1.2k"},
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
	}
	for _, tt := range tests {
		got := CompactNumber(tt.n)
		if got != tt.want {
			t.Errorf("CompactNumber(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestHint_Empty(t *testing.T) {
	if Hint() != "" {
		t.Error("Hint() with no args should return empty")
	}
}

func TestHint_Multiple(t *testing.T) {
	result := Hint("a", "b")
	if result == "" {
		t.Error("Hint with args should return non-empty")
	}
}

func TestHorizontalRule_ZeroWidth(t *testing.T) {
	if HorizontalRule(0) != "" {
		t.Error("HorizontalRule(0) should return empty")
	}
	if HorizontalRule(-1) != "" {
		t.Error("HorizontalRule(-1) should return empty")
	}
}

func TestHorizontalRule_Positive(t *testing.T) {
	result := HorizontalRule(5)
	if result == "" {
		t.Error("HorizontalRule(5) should return non-empty")
	}
}

func TestDim(t *testing.T) {
	result := Dim("faded")
	if result == "" {
		t.Error("Dim should return non-empty")
	}
}
