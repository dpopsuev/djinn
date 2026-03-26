package tui

import (
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// Glyph — RED
// ═══════════════════════════════════════════════════════════════════════

func TestGlyph_UnknownState(t *testing.T) {
	result := Glyph("nonexistent")
	if !strings.Contains(result, "○") {
		t.Fatalf("unknown state should fallback to ○, got %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Glyph — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestGlyph_Done(t *testing.T) {
	if !strings.Contains(Glyph("done"), "⬢") {
		t.Fatal("done should be ⬢")
	}
}

func TestGlyph_Active(t *testing.T) {
	if !strings.Contains(Glyph("active"), "⬡") {
		t.Fatal("active should be ⬡")
	}
}

func TestGlyph_Error(t *testing.T) {
	if !strings.Contains(Glyph("error"), "●") {
		t.Fatal("error should be ●")
	}
}

func TestGlyph_Pending(t *testing.T) {
	if !strings.Contains(Glyph("pending"), "○") {
		t.Fatal("pending should be ○")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// CompactNumber — RED
// ═══════════════════════════════════════════════════════════════════════

func TestCompactNumber_Zero(t *testing.T) {
	if CompactNumber(0) != "0" {
		t.Fatalf("0 → %q", CompactNumber(0))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// CompactNumber — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestCompactNumber_Small(t *testing.T) {
	if CompactNumber(42) != "42" {
		t.Fatalf("42 → %q", CompactNumber(42))
	}
}

func TestCompactNumber_Thousands(t *testing.T) {
	result := CompactNumber(1200)
	if result != "1.2k" {
		t.Fatalf("1200 → %q, want 1.2k", result)
	}
}

func TestCompactNumber_Millions(t *testing.T) {
	result := CompactNumber(3400000)
	if result != "3.4M" {
		t.Fatalf("3400000 → %q, want 3.4M", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// CompactNumber — BLUE
// ═══════════════════════════════════════════════════════════════════════

func TestCompactNumber_ExactThousand(t *testing.T) {
	result := CompactNumber(1000)
	if result != "1.0k" {
		t.Fatalf("1000 → %q, want 1.0k", result)
	}
}

func TestCompactNumber_Negative(t *testing.T) {
	result := CompactNumber(-5000)
	if result != "-5.0k" {
		t.Fatalf("-5000 → %q, want -5.0k", result)
	}
}

func TestCompactNumber_999(t *testing.T) {
	if CompactNumber(999) != "999" {
		t.Fatalf("999 → %q", CompactNumber(999))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Badge — RED
// ═══════════════════════════════════════════════════════════════════════

func TestBadge_ZeroValue(t *testing.T) {
	result := Badge("tokens", 0)
	if !strings.Contains(result, "0") || !strings.Contains(result, "tokens") {
		t.Fatalf("Badge = %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Badge — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestBadge_TokenCount(t *testing.T) {
	result := Badge("tokens", 8200)
	if !strings.Contains(result, "8.2k") || !strings.Contains(result, "tokens") {
		t.Fatalf("Badge = %q, want '8.2k tokens'", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Badge — BLUE
// ═══════════════════════════════════════════════════════════════════════

func TestBadge_LargeNumber(t *testing.T) {
	result := Badge("tokens", 1200000)
	if !strings.Contains(result, "1.2M") {
		t.Fatalf("Badge = %q, want 1.2M", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Hint — RED
// ═══════════════════════════════════════════════════════════════════════

func TestHint_NoBindings(t *testing.T) {
	if Hint() != "" {
		t.Fatalf("empty Hint = %q", Hint())
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Hint — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestHint_ThreeBindings(t *testing.T) {
	result := Hint("enter send", "↑ edit", "esc cancel")
	if !strings.Contains(result, "·") {
		t.Fatalf("Hint = %q, want dots separator", result)
	}
	if !strings.Contains(result, "enter send") || !strings.Contains(result, "esc cancel") {
		t.Fatalf("Hint = %q", result)
	}
}

func TestHint_SingleBinding(t *testing.T) {
	result := Hint("enter submit")
	if !strings.Contains(result, "enter submit") {
		t.Fatalf("Hint = %q", result)
	}
	if strings.Contains(result, "·") {
		t.Fatal("single binding should have no separator")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// HorizontalRule — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestHorizontalRule_Width(t *testing.T) {
	result := HorizontalRule(10)
	if !strings.Contains(result, "──────────") {
		t.Fatalf("HorizontalRule = %q", result)
	}
}

func TestHorizontalRule_Zero(t *testing.T) {
	if HorizontalRule(0) != "" {
		t.Fatal("zero width should be empty")
	}
}
