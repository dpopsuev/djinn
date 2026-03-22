package repl

import "github.com/charmbracelet/lipgloss"

// All styles use foreground colors only — no backgrounds.
// This preserves terminal transparency/opacity.
// AdaptiveColor picks the right shade for dark/light terminals.

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"}).
			Bold(true)

	assistStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"}).
			Bold(true)

	toolNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#eab308", Dark: "#facc15"})

	toolArgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#a855f7", Dark: "#c084fc"})

	toolSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"})

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"})

	dimStyle = lipgloss.NewStyle().Faint(true)

	statusStyle = lipgloss.NewStyle().Faint(true)
)

// Label constants.
const (
	labelUser    = "> "
	labelAssist  = "djinn"
	labelTool    = "tool"
	labelError   = "error"
)
