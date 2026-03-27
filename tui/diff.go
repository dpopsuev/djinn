// diff.go — colored git diff rendering for terminal output.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Diff styles — set by ApplyTokens(), never hardcode hex here.
var (
	diffAddStyle    lipgloss.Style
	diffDelStyle    lipgloss.Style
	diffHeaderStyle lipgloss.Style
	diffMetaStyle   = lipgloss.NewStyle().Faint(true)
)

// RenderWordDiff highlights individual changed words between two lines.
// Green for additions, red for deletions.
func RenderWordDiff(oldLine, newLine string) string {
	oldWords := strings.Fields(oldLine)
	newWords := strings.Fields(newLine)

	var sb strings.Builder
	maxLen := len(oldWords)
	if len(newWords) > maxLen {
		maxLen = len(newWords)
	}

	for i := range maxLen {
		if i > 0 {
			sb.WriteByte(' ')
		}
		oldW, newW := "", ""
		if i < len(oldWords) {
			oldW = oldWords[i]
		}
		if i < len(newWords) {
			newW = newWords[i]
		}

		switch {
		case oldW == newW:
			sb.WriteString(newW)
		case oldW == "":
			sb.WriteString(diffAddStyle.Render(newW))
		case newW == "":
			sb.WriteString(diffDelStyle.Render(oldW))
		default:
			sb.WriteString(diffDelStyle.Render(oldW))
			sb.WriteByte(' ')
			sb.WriteString(diffAddStyle.Render(newW))
		}
	}
	return sb.String()
}

// RenderDiff applies colors to git diff output.
func RenderDiff(diff string) string {
	var sb strings.Builder
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			sb.WriteString(diffMetaStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			sb.WriteString(diffHeaderStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			sb.WriteString(diffAddStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			sb.WriteString(diffDelStyle.Render(line))
		case strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "):
			sb.WriteString(diffMetaStyle.Render(line))
		default:
			sb.WriteString(line)
		}
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}
