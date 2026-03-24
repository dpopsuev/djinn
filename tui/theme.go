// theme.go — semantic color palette for the Djinn TUI.
// All styles reference the active theme instead of hardcoded colors.
// Foundation for style presets (claude/codex/gemini/djinn themes).
package tui

import "github.com/charmbracelet/lipgloss"

// Theme defines the semantic color palette.
type Theme struct {
	User      lipgloss.AdaptiveColor // user input text
	Assistant lipgloss.AdaptiveColor // assistant label
	ToolName  lipgloss.AdaptiveColor // tool call name
	ToolArg   lipgloss.AdaptiveColor // tool call arguments
	Success   lipgloss.AdaptiveColor // success indicators
	Error     lipgloss.AdaptiveColor // error messages
	Accent    lipgloss.AdaptiveColor // brand accent (borders, logo)
	FocusDim  lipgloss.AdaptiveColor // unfocused border color
}

// DefaultTheme is the Djinn default palette — Red Hat Red accent, green/blue/yellow/purple semantic colors.
var DefaultTheme = Theme{
	User:      lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"},
	Assistant: lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"},
	ToolName:  lipgloss.AdaptiveColor{Light: "#eab308", Dark: "#facc15"},
	ToolArg:   lipgloss.AdaptiveColor{Light: "#a855f7", Dark: "#c084fc"},
	Success:   lipgloss.AdaptiveColor{Light: "#22c55e", Dark: "#4ade80"},
	Error:     lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"},
	Accent:    lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#EE0000"},
	FocusDim:  lipgloss.AdaptiveColor{Light: "#808080", Dark: "#505050"},
}

// ActiveTheme is the currently active theme. Change this to swap palettes.
var ActiveTheme = DefaultTheme

// ApplyTheme updates all style variables to use the given theme.
func ApplyTheme(t Theme) {
	ActiveTheme = t
	RedHatRed = t.Accent
	UserStyle = lipgloss.NewStyle().Foreground(t.User).Bold(true)
	AssistStyle = lipgloss.NewStyle().Foreground(t.Assistant).Bold(true)
	ToolNameStyle = lipgloss.NewStyle().Foreground(t.ToolName)
	ToolArgStyle = lipgloss.NewStyle().Foreground(t.ToolArg)
	ToolSuccessStyle = lipgloss.NewStyle().Foreground(t.Success)
	ErrorStyle = lipgloss.NewStyle().Foreground(t.Error)
	LogoStyle = lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
}
