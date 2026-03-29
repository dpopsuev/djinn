// tokens.go — Mutable token state + ApplyTokens().
//
// Token types and pure conversion live in tui/design/.
// This file holds the mutable global state and the single-writer ApplyTokens()
// function that rebuilds all lipgloss.Style variables across the TUI package.
package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/dpopsuev/djinn/tui/design"
)

// TokenSet is the semantic token type.
type TokenSet = design.TokenSet

// DefaultTokens returns tokens derived from DefaultTheme.
func DefaultTokens() TokenSet { return design.DefaultTokens() }

// TokensFromTheme maps a Theme to a full TokenSet.
func TokensFromTheme(t Theme) TokenSet { return design.TokensFromTheme(t) } //nolint:gocritic // pass-through to design

// ActiveTokens is the live token set. Rebuilt by ApplyTokens().
var ActiveTokens TokenSet

func init() {
	ApplyTokens(DefaultTokens())
}

// ApplyTokens rebuilds all global style variables from the given token set.
// This is the SINGLE WRITER of style vars — no other code should assign them.
func ApplyTokens(ts TokenSet) { //nolint:gocritic // TokenSet is stored as global, copy is intentional
	ActiveTokens = ts

	// Core styles (styles.go vars)
	RedHatRed = ts.AccentFg
	UserStyle = lipgloss.NewStyle().Foreground(ts.UserFg).Bold(true)
	AssistStyle = lipgloss.NewStyle().Foreground(ts.AssistantFg).Bold(true)
	ToolNameStyle = lipgloss.NewStyle().Foreground(ts.ToolNameFg)
	ToolArgStyle = lipgloss.NewStyle().Foreground(ts.ToolArgFg)
	ToolSuccessStyle = lipgloss.NewStyle().Foreground(ts.SuccessFg)
	ErrorStyle = lipgloss.NewStyle().Foreground(ts.ErrorFg)
	LogoStyle = lipgloss.NewStyle().Foreground(ts.AccentFg).Bold(true)

	// Diff styles (diff.go vars)
	diffAddStyle = lipgloss.NewStyle().Foreground(ts.DiffAddFg)
	diffDelStyle = lipgloss.NewStyle().Foreground(ts.DiffDelFg)
	diffHeaderStyle = lipgloss.NewStyle().Foreground(ts.DiffHeaderFg)

	// Health styles (statusline.go vars)
	healthGreen = lipgloss.NewStyle().Foreground(ts.HealthGreenFg)
	healthYellow = lipgloss.NewStyle().Foreground(ts.HealthYellowFg)
	healthRed = lipgloss.NewStyle().Foreground(ts.HealthRedFg)

	// Budget styles (budget.go vars)
	budgetOKStyle = lipgloss.NewStyle().Foreground(ts.HealthGreenFg)
	budgetWarnStyle = lipgloss.NewStyle().Foreground(ts.HealthYellowFg)
	budgetOverStyle = lipgloss.NewStyle().Foreground(ts.HealthRedFg)

	// Coherence zone styles (coherence.go vars)
	zoneColdStyle = lipgloss.NewStyle().Foreground(ts.ZoneColdFg)
	zoneWarmStyle = lipgloss.NewStyle().Foreground(ts.SuccessFg)
	zoneFocusedStyle = lipgloss.NewStyle().Foreground(ts.ZoneFocusedFg)
	zoneHotStyle = lipgloss.NewStyle().Foreground(ts.HealthYellowFg)
	zoneRedlineStyle = lipgloss.NewStyle().Foreground(ts.HealthRedFg)

	// Drift styles (drift.go vars)
	driftGoodStyle = lipgloss.NewStyle().Foreground(ts.SuccessFg)
	driftMidStyle = lipgloss.NewStyle().Foreground(ts.WarningFg)
	driftBadStyle = lipgloss.NewStyle().Foreground(ts.ErrorFg)

	// Dashboard mode indicators (dashboard.go vars)
	modeInsertStyle = lipgloss.NewStyle().Bold(true).Foreground(ts.UserFg)
	modeStreamStyle = lipgloss.NewStyle().Bold(true).Foreground(ts.AssistantFg)
	modeApprovalStyle = lipgloss.NewStyle().Bold(true).Foreground(ts.WarningFg)

	// Focus border (focus.go vars)
	focusBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ts.AccentFg)
	unfocusedBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ts.FocusDimFg)

	// Turn envelope border (turn_envelope.go vars)
	turnBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ts.FocusDimFg)

	// Separator focus (separator.go vars)
	sepFocusStyle = lipgloss.NewStyle().Foreground(ts.AssistantFg)
}
