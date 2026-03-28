// tui_layout_test.go — responsive layout tests for 4 terminal size breakpoints.
//
// Validates DJN-BUG-35: borders fill width, no gaps, no overflow,
// content adapts to small/medium/large/massive terminals.
package acceptance

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/cli/repl"
	"github.com/dpopsuev/djinn/session"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/tui"
)

func layoutModel(t *testing.T, width, height int) repl.Model {
	t.Helper()
	sess := session.New("layout-test", "test-model", "/workspace")
	m := repl.NewModel(repl.Config{
		Tools:   builtin.NewRegistry(),
		Session: sess,
		Mode:    "agent",
	})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return *toModelPtr(m2)
}

// maxLineWidth returns the longest visible line width (stripping ANSI).
func maxLineWidth(view string) int {
	widest := 0
	for _, line := range strings.Split(view, "\n") {
		w := visibleWidth(line)
		if w > widest {
			widest = w
		}
	}
	return widest
}

// visibleWidth strips ANSI escape codes and returns visible character count.
func visibleWidth(s string) int {
	inEscape := false
	count := 0
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		count++
	}
	return count
}

func TestTUI_Layout_Small(t *testing.T) {
	m := layoutModel(t, 80, 24)
	view := m.View()

	if view == "" {
		t.Fatal("view should not be empty")
	}

	// Borders must not exceed terminal width.
	w := maxLineWidth(view)
	if w > 80 {
		// Debug: show the widest line.
		for _, line := range strings.Split(view, "\n") {
			vw := visibleWidth(line)
			if vw == w {
				t.Logf("widest line (%d visible runes, %d bytes): %q", vw, len(line), line[:min(len(line), 300)])
				t.Logf("lipgloss.Width = %d", lipgloss.Width(line))
				break
			}
		}
		t.Fatalf("line width %d exceeds terminal width 80", w)
	}

	// Should contain the dashboard mode indicator.
	upper := strings.ToUpper(view)
	if !strings.Contains(upper, "INSERT") && !strings.Contains(upper, "GENSEC") {
		t.Fatal("small layout should show mode indicator")
	}
}

func TestTUI_Layout_Medium(t *testing.T) {
	m := layoutModel(t, 120, 40)
	view := m.View()

	// Borders must not exceed terminal width.
	w := maxLineWidth(view)
	if w > 120 {
		t.Fatalf("line width %d exceeds terminal width 120", w)
	}

	// Should contain the full MOTD with logo.
	if !strings.Contains(view, tui.DjinnLogo[:10]) {
		t.Fatal("medium layout should show the logo")
	}

	// Should contain /help hint.
	if !strings.Contains(view, "/help") {
		t.Fatal("medium layout should show /help hint")
	}
}

func TestTUI_Layout_Large(t *testing.T) {
	m := layoutModel(t, 200, 50)
	view := m.View()

	// Borders must not exceed terminal width.
	w := maxLineWidth(view)
	if w > 200 {
		t.Fatalf("line width %d exceeds terminal width 200", w)
	}

	// Should contain the logo.
	if !strings.Contains(view, tui.DjinnLogo[:10]) {
		t.Fatal("large layout should show the logo")
	}
}

func TestTUI_Layout_Massive(t *testing.T) {
	m := layoutModel(t, 300, 60)
	view := m.View()

	// Content should not stretch infinitely — lines should not reach 300 chars.
	// Allow some tolerance (borders + padding).
	w := maxLineWidth(view)
	if w > 300 {
		t.Fatalf("line width %d exceeds terminal width 300", w)
	}

	// View should not be empty.
	if len(view) < 100 {
		t.Fatal("massive layout should produce substantial output")
	}
}

func TestTUI_Layout_BordersFillWidth(t *testing.T) {
	// Test at each breakpoint: the top border line should be exactly terminal width.
	for _, tc := range []struct {
		name          string
		width, height int
	}{
		{"small", 80, 24},
		{"medium", 120, 40},
		{"large", 200, 50},
		{"massive", 300, 60},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := layoutModel(t, tc.width, tc.height)
			view := m.View()
			lines := strings.Split(view, "\n")

			// Find the first border line (contains ╭).
			found := false
			for _, line := range lines {
				if strings.Contains(line, "╭") {
					w := visibleWidth(line)
					if w != tc.width {
						t.Fatalf("%s: border width = %d, want %d", tc.name, w, tc.width)
					}
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("%s: no border line found in view", tc.name)
			}
		})
	}
}

func TestTUI_Layout_NoBorderGaps(t *testing.T) {
	// At 120 cols, there should be no lines with trailing whitespace
	// between borders (the "empty space" bug).
	m := layoutModel(t, 120, 40)
	view := m.View()
	lines := strings.Split(view, "\n")

	for i, line := range lines {
		if line == "" {
			continue
		}
		// Lines with border chars should not have large trailing gaps.
		if strings.Contains(line, "│") {
			trimmed := strings.TrimRight(line, " ")
			gap := len(line) - len(trimmed)
			if gap > 5 {
				t.Fatalf("line %d has %d trailing spaces (border gap):\n%q", i, gap, line[:min(len(line), 100)])
			}
		}
	}
}
