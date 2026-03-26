// components.go — Layer 2: composite visual molecules.
//
// Compose elements (Layer 1) into reusable widgets. Minimal state.
// Panels (Layer 3) compose these. Never reference panels.
// Dependency: Component → Element → lipgloss. No upward references.
package tui

import (
	"fmt"
	"strings"
)

// SectionHeader renders a labeled section border: ┌─ title ─────────┐
func SectionHeader(title string, width int) string {
	if width < 6 {
		width = 6
	}
	label := " " + title + " "
	remaining := width - len(label) - 2 // ┌ and ┐
	if remaining < 2 {
		remaining = 2
	}
	left := 1
	right := remaining - left
	if right < 0 {
		right = 0
	}
	return DimStyle.Render("┌" + strings.Repeat("─", left) + label + strings.Repeat("─", right) + "┐")
}

// StatusLine renders model info: "Sonnet 4 · 43.2% · 3 files edited"
func StatusLine(model string, contextPct float64, filesEdited int) string {
	parts := []string{model}
	if contextPct > 0 {
		parts = append(parts, fmt.Sprintf("%.1f%%", contextPct))
	}
	if filesEdited > 0 {
		noun := "file"
		if filesEdited != 1 {
			noun = "files"
		}
		parts = append(parts, fmt.Sprintf("%d %s edited", filesEdited, noun))
	}
	return DimStyle.Render(strings.Join(parts, " · "))
}

// ToolStatus renders a tool call status line: ⬢ read main.go (1.2k tokens)
func ToolStatus(name, state string, tokens int) string {
	glyph := Glyph(state)
	label := glyph + " " + name
	if tokens > 0 {
		label += " (" + Badge("tokens", tokens) + ")"
	}
	return label
}

// LabeledBorder wraps content in a bordered box with a title.
//
//	╭─ Title ──────────╮
//	│ content line 1    │
//	│ content line 2    │
//	╰──────────────────╯
func LabeledBorder(title, content string, width int) string {
	if width < 10 {
		width = 10
	}
	inner := width - 4 // │ + space + space + │

	// Title line.
	label := " " + title + " "
	topLen := inner - len(label)
	if topLen < 2 {
		topLen = 2
	}
	leftPad := topLen / 2
	rightPad := topLen - leftPad

	var sb strings.Builder
	sb.WriteString(DimStyle.Render("╭" + strings.Repeat("─", leftPad) + label + strings.Repeat("─", rightPad) + "╮"))
	sb.WriteByte('\n')

	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			continue
		}
		sb.WriteString(DimStyle.Render("│ "))
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	sb.WriteString(DimStyle.Render("╰" + strings.Repeat("─", inner) + "╯"))
	return sb.String()
}
