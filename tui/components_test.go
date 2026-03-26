package tui

import (
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// SectionHeader — RED
// ═══════════════════════════════════════════════════════════════════════

func TestSectionHeader_EmptyTitle(t *testing.T) {
	result := SectionHeader("", 20)
	if !strings.Contains(result, "┌") || !strings.Contains(result, "┐") {
		t.Fatalf("empty title should still have borders: %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SectionHeader — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestSectionHeader_Normal(t *testing.T) {
	result := SectionHeader("Tasks", 30)
	if !strings.Contains(result, "Tasks") {
		t.Fatalf("missing title: %q", result)
	}
	if !strings.Contains(result, "┌") || !strings.Contains(result, "┐") {
		t.Fatalf("missing borders: %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SectionHeader — BLUE
// ═══════════════════════════════════════════════════════════════════════

func TestSectionHeader_NarrowWidth(t *testing.T) {
	result := SectionHeader("Very Long Title", 5)
	// Should not panic, should produce something.
	if result == "" {
		t.Fatal("narrow width should still produce output")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// ToolStatus — RED
// ═══════════════════════════════════════════════════════════════════════

func TestToolStatus_EmptyName(t *testing.T) {
	result := ToolStatus("", "done", 0)
	// Should contain the glyph at minimum.
	if !strings.Contains(result, "⬢") {
		t.Fatalf("empty name: %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// ToolStatus — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestToolStatus_Done(t *testing.T) {
	result := ToolStatus("read", "done", 1200)
	if !strings.Contains(result, "⬢") {
		t.Fatalf("done should use ⬢: %q", result)
	}
	if !strings.Contains(result, "read") {
		t.Fatalf("missing tool name: %q", result)
	}
	if !strings.Contains(result, "1.2k") {
		t.Fatalf("missing compact tokens: %q", result)
	}
}

func TestToolStatus_Active(t *testing.T) {
	result := ToolStatus("bash", "active", 500)
	if !strings.Contains(result, "⬡") {
		t.Fatalf("active should use ⬡: %q", result)
	}
}

func TestToolStatus_Error(t *testing.T) {
	result := ToolStatus("write", "error", 0)
	if !strings.Contains(result, "●") {
		t.Fatalf("error should use ●: %q", result)
	}
	// No tokens → no badge.
	if strings.Contains(result, "tokens") {
		t.Fatal("zero tokens should not show badge")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// StatusLine — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestStatusLine_Full(t *testing.T) {
	result := StatusLine("Sonnet 4", 43.2, 3)
	if !strings.Contains(result, "Sonnet 4") {
		t.Fatalf("missing model: %q", result)
	}
	if !strings.Contains(result, "43.2%") {
		t.Fatalf("missing pct: %q", result)
	}
	if !strings.Contains(result, "3 files") {
		t.Fatalf("missing files: %q", result)
	}
	if !strings.Contains(result, "·") {
		t.Fatalf("missing separator: %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// StatusLine — BLUE
// ═══════════════════════════════════════════════════════════════════════

func TestStatusLine_ZeroFiles(t *testing.T) {
	result := StatusLine("Opus", 0, 0)
	if !strings.Contains(result, "Opus") {
		t.Fatalf("missing model: %q", result)
	}
	// No pct, no files → just model name.
	if strings.Contains(result, "·") {
		t.Fatalf("no separators when no extra data: %q", result)
	}
}

func TestStatusLine_SingleFile(t *testing.T) {
	result := StatusLine("Haiku", 10.0, 1)
	if !strings.Contains(result, "1 file edited") {
		t.Fatalf("singular 'file': %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// LabeledBorder — GREEN
// ═══════════════════════════════════════════════════════════════════════

func TestLabeledBorder_Content(t *testing.T) {
	result := LabeledBorder("Sprint", "task 1\ntask 2", 40)
	if !strings.Contains(result, "╭") || !strings.Contains(result, "╰") {
		t.Fatalf("missing borders: %q", result)
	}
	if !strings.Contains(result, "Sprint") {
		t.Fatalf("missing title: %q", result)
	}
	if !strings.Contains(result, "task 1") || !strings.Contains(result, "task 2") {
		t.Fatalf("missing content: %q", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// LabeledBorder — BLUE
// ═══════════════════════════════════════════════════════════════════════

func TestLabeledBorder_NarrowWidth(t *testing.T) {
	result := LabeledBorder("T", "c", 5)
	if result == "" {
		t.Fatal("narrow should still produce output")
	}
}

func TestLabeledBorder_EmptyContent(t *testing.T) {
	result := LabeledBorder("Empty", "", 40)
	if !strings.Contains(result, "Empty") {
		t.Fatalf("missing title: %q", result)
	}
	// Should have top and bottom border, no content lines.
	if !strings.Contains(result, "╭") || !strings.Contains(result, "╰") {
		t.Fatal("missing borders")
	}
}
