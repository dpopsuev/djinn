// diff.go — colored git diff rendering for terminal output.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	diffAddStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"})
	diffDelStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"})
	diffHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#06b6d4", Dark: "#22d3ee"})
	diffMetaStyle   = lipgloss.NewStyle().Faint(true)
)

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
