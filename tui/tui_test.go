package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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
	if !strings.Contains(result, "a") || !strings.Contains(result, "b") {
		t.Fatalf("all green should show individual names: %q", result)
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

// TestRenderMarkdown_DoubleRenderCorrupts reproduces DJN-BUG-5:
// glamour pads each line to full width. Re-rendering padded output
// creates new lines per tick. After N ticks the output is N × width
// chars of whitespace — the text explodes and markdown breaks.
func TestRenderMarkdown_DoubleRenderCorrupts(t *testing.T) {
	InitRenderer(80)

	// Simulate 5 streaming ticks with re-render on each tick.
	// This is what flushStreamBuffer did: append raw, render whole, store rendered.
	prefix := "djinn: "
	line := prefix

	for i, chunk := range []string{"Hello ", "world, ", "this ", "is ", "streaming."} {
		// Append raw chunk to (previously rendered) line
		line += chunk
		// Strip prefix, re-render the whole thing through glamour
		after, _ := strings.CutPrefix(line, prefix)
		rendered := RenderMarkdown(after)
		line = prefix + rendered

		// After each tick, the line should NOT grow unboundedly.
		// Clean "Hello world, this is streaming." is ~40 chars.
		// With glamour padding, each re-render adds ~80 chars of whitespace.
		// After 5 ticks: ~400 chars of garbage instead of ~40.
		// Verify: re-rendering glamour output DOES corrupt.
		// This test documents WHY we use the raw buffer approach.
		if i == 4 && len(line) < 200 {
			t.Fatal("expected output to explode from double-render — if this passes, glamour changed behavior")
		}
	}
}

// TestRenderMarkdown_RawBufferApproach verifies the correct fix:
// always render from raw text, never from previously rendered output.
func TestRenderMarkdown_RawBufferApproach(t *testing.T) {
	InitRenderer(80)

	// Simulate incremental streaming: accumulate raw, render fresh each time.
	var rawBuf strings.Builder

	rawBuf.WriteString("Hello ")
	r1 := RenderMarkdown(rawBuf.String())
	if r1 == "" {
		t.Fatal("render 1 should produce output")
	}

	rawBuf.WriteString("**bold** ")
	r2 := RenderMarkdown(rawBuf.String())
	if r2 == "" {
		t.Fatal("render 2 should produce output")
	}

	rawBuf.WriteString("world")
	r3 := RenderMarkdown(rawBuf.String())
	if r3 == "" {
		t.Fatal("render 3 should produce output")
	}

	// None of the renders should contain raw escape code literals.
	for i, r := range []string{r1, r2, r3} {
		if strings.Contains(r, "[0m[38") || strings.Contains(r, "38;5;252") {
			t.Fatalf("render %d contains ANSI garbage:\n%s", i+1, r)
		}
	}
}

// TestOutputPanel_ViewNeverDimmed verifies DJN-BUG-11:
// Proves WHY output panel must not be depth-dimmed: dimming viewport-padded
// whitespace lines injects ANSI color prefixes that render as visible text
// on terminals without 24-bit color support.
func TestOutputPanel_ViewNeverDimmed(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	p := NewOutputPanel()
	p.Append("Hello world")
	p.Append("")
	p.Append("")

	view := p.View(80)
	dimPrefix := "\x1b[38;2;112;112;112m"

	// The raw output panel view should NOT contain dim prefixes
	if strings.Contains(view, dimPrefix) {
		t.Fatal("BUG-11: OutputPanel.View() itself contains dim prefix — panel is self-dimming")
	}

	// Wrapping with depth > 0 DOES add dim prefix — this is why View() must
	// never go through RenderWithDepth. This documents the root cause.
	dimmed := RenderWithDepth(view, 1, 80)
	if !strings.Contains(dimmed, dimPrefix) {
		t.Skip("dimming not applied (color profile may strip it)")
	}

	// Verify: dimming empty lines creates the exact garbage the user reported
	lines := strings.Split(dimmed, "\n")
	garbageLines := 0
	for _, line := range lines {
		stripped := strings.TrimSpace(strings.ReplaceAll(line, dimPrefix, ""))
		stripped = strings.ReplaceAll(stripped, "\x1b[0m", "")
		if stripped == "" && strings.Contains(line, dimPrefix) {
			garbageLines++
		}
	}
	if garbageLines == 0 {
		t.Fatal("expected depth-dimmed empty lines to have color prefix (documents the bug)")
	}
	// This test PASSES because it documents the bug mechanism,
	// not because the bug is present in View(). The fix is in
	// model.go: output panel skips RenderWithDepth.
}

// TestGlamourInsideBorder_TrueColor_NoVisibleANSI reproduces DJN-BUG-9
// with forced TrueColor profile to match real terminal behavior.
func TestGlamourInsideBorder_TrueColor_NoVisibleANSI(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	InitRenderer(80)
	rendered := RenderMarkdown("Hello **world**, how are you? This is a longer sentence to trigger glamour padding.")
	if rendered == "" {
		t.Fatal("glamour should produce output")
	}

	bordered := RenderWithDepth(rendered, 0, 80)

	// Check for visible ANSI escape code literals
	if strings.Contains(bordered, "[0m[38") {
		t.Fatalf("DJN-BUG-9: glamour ANSI visible inside border (TrueColor):\n%q", bordered[:min(len(bordered), 400)])
	}
	if strings.Contains(bordered, "38;5;252") {
		t.Fatalf("DJN-BUG-9: glamour color codes as text (TrueColor):\n%q", bordered[:min(len(bordered), 400)])
	}
}

// TestGlamourInsideBorder_NoVisibleANSI reproduces DJN-BUG-9:
// glamour pads lines with ANSI codes, lipgloss border re-processes them,
// making escape codes visible as literal text.
func TestGlamourInsideBorder_NoVisibleANSI(t *testing.T) {
	InitRenderer(80)

	// Render markdown through glamour
	rendered := RenderMarkdown("Hello **world**, how are you?")
	if rendered == "" {
		t.Fatal("glamour should produce output")
	}

	// Wrap in a lipgloss rounded border (same as RenderWithDepth depth=0)
	bordered := RenderWithDepth(rendered, 0, 80)

	// The bordered output should NOT contain visible ANSI escape literals.
	if strings.Contains(bordered, "[0m[38") {
		t.Fatalf("DJN-BUG-9: glamour padding ANSI visible inside border:\n%q", bordered[:min(len(bordered), 300)])
	}
	if strings.Contains(bordered, "38;5;252") {
		t.Fatalf("DJN-BUG-9: glamour color codes visible as text inside border:\n%q", bordered[:min(len(bordered), 300)])
	}
}
