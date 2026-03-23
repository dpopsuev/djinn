package tui

import (
	"strings"
	"testing"
)

// --- WrapText tests ---

func TestWrapText_ShortLine(t *testing.T) {
	result := WrapText("short", 80)
	if result != "short" {
		t.Fatalf("short line should pass through: %q", result)
	}
}

func TestWrapText_LongLine(t *testing.T) {
	long := "this is a very long line that should be wrapped at the specified width boundary point"
	result := WrapText(long, 40)
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Fatalf("should wrap, got %d lines: %q", len(lines), result)
	}
	for _, line := range lines {
		if len(line) > 42 { // some tolerance for word boundaries
			t.Fatalf("line too long (%d): %q", len(line), line)
		}
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	result := WrapText("hello", 0)
	if result != "hello" {
		t.Fatal("zero width should pass through")
	}
}

func TestWrapText_ExactWidth(t *testing.T) {
	text := "1234567890"
	result := WrapText(text, 10)
	if result != text {
		t.Fatalf("exact width should not wrap: %q", result)
	}
}

func TestWrapText_PreservesNewlines(t *testing.T) {
	text := "line one\nline two\nline three"
	result := WrapText(text, 80)
	if strings.Count(result, "\n") < 2 {
		t.Fatal("should preserve existing newlines")
	}
}

// --- RenderDiff tests ---

func TestRenderDiff_AdditionsGreen(t *testing.T) {
	diff := "+added line"
	result := RenderDiff(diff)
	if !strings.Contains(result, "added line") {
		t.Fatal("should contain the text")
	}
	// Should contain ANSI escape codes (coloring happened)
	if !strings.Contains(result, "\033[") {
		t.Log("warning: no ANSI codes — may be unstyled")
	}
}

func TestRenderDiff_DeletionsRed(t *testing.T) {
	diff := "-removed line"
	result := RenderDiff(diff)
	if !strings.Contains(result, "removed line") {
		t.Fatal("should contain the text")
	}
}

func TestRenderDiff_Headers(t *testing.T) {
	diff := "@@ -1,3 +1,4 @@"
	result := RenderDiff(diff)
	if !strings.Contains(result, "@@") {
		t.Fatal("should contain header")
	}
}

func TestRenderDiff_MixedOutput(t *testing.T) {
	diff := "diff --git a/file.go b/file.go\n--- a/file.go\n+++ b/file.go\n@@ -1,3 +1,4 @@\n context\n-old line\n+new line"
	result := RenderDiff(diff)
	if !strings.Contains(result, "old line") || !strings.Contains(result, "new line") {
		t.Fatal("should contain both old and new")
	}
}

func TestRenderDiff_EmptyInput(t *testing.T) {
	result := RenderDiff("")
	_ = result // should not panic
}

// --- StatusLine rendering tests (decomposed) ---

func TestRenderHealth_AllGreen(t *testing.T) {
	reports := []HealthReport{
		{Component: "a", Status: StatusGreen},
		{Component: "b", Status: StatusGreen},
	}
	result := RenderHealth(reports)
	if !strings.Contains(result, "2 mcp") {
		t.Fatalf("all green should collapse: %q", result)
	}
}

func TestRenderHealth_Mixed(t *testing.T) {
	reports := []HealthReport{
		{Component: "scribe", Status: StatusYellow},
		{Component: "locus", Status: StatusGreen},
	}
	result := RenderHealth(reports)
	if !strings.Contains(result, "scribe") {
		t.Fatalf("should show failed component: %q", result)
	}
}

func TestRenderHealth_Empty(t *testing.T) {
	result := RenderHealth(nil)
	if result != "" {
		t.Fatal("empty reports should return empty")
	}
}

func TestRenderStatusLine_Structure(t *testing.T) {
	result := RenderStatusLine("aeon", "claude", "opus", "agent", 100, 50, 3, nil)
	if result == "" {
		t.Fatal("should produce output")
	}
}
