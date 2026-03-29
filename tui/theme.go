// theme.go — Re-exports theme types and registry from tui/design.
// Keeps ApplyTheme() here since it mutates tui-level state.
package tui

import "github.com/dpopsuev/djinn/tui/design"

// Theme is the semantic color palette type.
type Theme = design.Theme

// Built-in theme presets.
var (
	DefaultTheme = design.DefaultTheme
	ClaudeTheme  = design.ClaudeTheme
	GeminiTheme  = design.GeminiTheme
	CodexTheme   = design.CodexTheme
)

// RegisterTheme adds or replaces a named theme.
func RegisterTheme(name string, t Theme) { design.RegisterTheme(name, t) } //nolint:gocritic // pass-through to design

// ThemeByName returns a theme by name.
func ThemeByName(name string) Theme { return design.ThemeByName(name) }

// ThemeNames returns all registered theme names.
func ThemeNames() []string { return design.ThemeNames() }

// ActiveTheme is the currently active theme.
var ActiveTheme = DefaultTheme

// ApplyTheme sets the active theme and rebuilds all styles via ApplyTokens.
func ApplyTheme(t Theme) { //nolint:gocritic // Theme is stored as global
	ActiveTheme = t
	ApplyTokens(design.TokensFromTheme(t))
}
