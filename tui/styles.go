// Package tui is the Djinn design system — composable TUI components on Bubbletea.
// Zero Djinn domain imports. Only Bubbletea, Bubbles, Lipgloss, Glamour.
package tui

import "github.com/charmbracelet/lipgloss"

// Red Hat Red — primary brand color.
var RedHatRed = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#EE0000"}

// All styles use foreground colors only — no backgrounds.
// This preserves terminal transparency/opacity.
var (
	UserStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"}).
			Bold(true)

	AssistStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"}).
			Bold(true)

	ToolNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#eab308", Dark: "#facc15"})

	ToolArgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#a855f7", Dark: "#c084fc"})

	ToolSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"})

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"})

	DimStyle = lipgloss.NewStyle().Faint(true)

	StatusStyle = lipgloss.NewStyle().Faint(true)

	LogoStyle = lipgloss.NewStyle().
			Foreground(RedHatRed).
			Bold(true)
)

// Djinn logo — Red Hat fedora rendered from rh_logo.svg in block characters.
// The negative space on lines 5-6 is the ribbon (hat band).
const DjinnLogo = `         ████  █████
        ███████████████
       █████████████████
 █████    ███████████████
█████████    ███████████ ████
  ██████████            ███████
      ████████████████████████
            ████████████████`

// FlameFrames — multi-line teardrop flame with hollow center.
// Only the tip dances; the base stays stable. Block chars match the fedora logo.
var FlameFrames = []string{
	"  ▖\n ▐ ▌\n  █",
	"  ▗\n ▐ ▌\n  █",
	" ▗▖\n ▐ ▌\n  █",
	"  ▘\n ▐ ▌\n  █",
	"  ▝\n ▐ ▌\n  █",
	" ▝▘\n ▐ ▌\n  █",
}

// Label constants.
const (
	LabelUser   = "❯ "
	LabelAssist = "djinn"
)
