// Package tui is the Djinn design system — composable TUI components on Bubbletea.
// Zero Djinn domain imports. Only Bubbletea, Bubbles, Lipgloss, Glamour.
package tui

import "github.com/charmbracelet/lipgloss"

// Red Hat Red — primary brand color. Set by ApplyTokens().
var RedHatRed lipgloss.AdaptiveColor

// All styles use foreground colors only — no backgrounds.
// This preserves terminal transparency/opacity.
// Set by ApplyTokens() at init — never assign hardcoded hex here.
var (
	UserStyle        lipgloss.Style
	AssistStyle      lipgloss.Style
	ToolNameStyle    lipgloss.Style
	ToolArgStyle     lipgloss.Style
	ToolSuccessStyle lipgloss.Style
	ErrorStyle       lipgloss.Style
	DimStyle         = lipgloss.NewStyle().Faint(true) // Faint is not color-based
	StatusStyle      = lipgloss.NewStyle().Faint(true)
	LogoStyle        lipgloss.Style
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

// SpinnerFrames — triangle and diamond cycle, filled and hollow.
var SpinnerFrames = []string{
	"◆",
	"◇",
	"▲",
	"△",
	"◆",
	"◇",
	"▼",
	"▽",
}

// Label variables (var not const — configurable via ApplyGlyphs).
var (
	LabelUser   = "> "
	LabelAssist = "djinn"
)

// Glyph variables for tool call indicators.
// Derived from Glyph() in elements.go — single source of truth.
// These vars exist for backward compat with ApplyGlyphs() and direct references.
var (
	GlyphToolCall    = "λ"
	GlyphToolSuccess = "⬢"
	GlyphToolError   = "●"
)
