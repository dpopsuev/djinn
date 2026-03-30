// tokens.go — Mutable token state + ApplyTokens().
//
// Token types and pure conversion live in tui/design/.
// This file holds the mutable global state and the single-writer ApplyTokens()
// function that rebuilds all lipgloss.Style variables across the TUI package.
package tui

import "github.com/dpopsuev/djinn/tui/design"

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
// Uses design.BuildStyles() for the canonical computation, then unpacks to
// tui-level vars (both public and private) for backward compat.
func ApplyTokens(ts TokenSet) { //nolint:gocritic,funlen // TokenSet stored as global; unpacking is inherently long
	ActiveTokens = ts

	// Build all styles via design package (pure function, single source of truth).
	ss := design.BuildStyles(ts)
	design.ActiveStyles = ss

	// Core styles (styles.go vars)
	RedHatRed = ss.AccentFg
	UserStyle = ss.User
	AssistStyle = ss.Assistant
	ToolNameStyle = ss.ToolName
	ToolArgStyle = ss.ToolArg
	ToolSuccessStyle = ss.ToolSuccess
	ErrorStyle = ss.Error
	LogoStyle = ss.Logo

	// Diff styles (diff.go vars)
	diffAddStyle = ss.DiffAdd
	diffDelStyle = ss.DiffDel
	diffHeaderStyle = ss.DiffHeader

	// Health styles (statusline.go vars)
	healthGreen = ss.HealthGreen
	healthYellow = ss.HealthYellow
	healthRed = ss.HealthRed

	// Budget styles (budget.go vars)
	budgetOKStyle = ss.BudgetOK
	budgetWarnStyle = ss.BudgetWarn
	budgetOverStyle = ss.BudgetOver

	// Coherence zone styles (coherence.go vars)
	zoneColdStyle = ss.ZoneCold
	zoneWarmStyle = ss.ZoneWarm
	zoneFocusedStyle = ss.ZoneFocused
	zoneHotStyle = ss.ZoneHot
	zoneRedlineStyle = ss.ZoneRedline

	// Drift styles (drift.go vars)
	driftGoodStyle = ss.DriftGood
	driftMidStyle = ss.DriftMid
	driftBadStyle = ss.DriftBad

	// Dashboard mode indicators (dashboard.go vars)
	modeInsertStyle = ss.ModeInsert
	modeStreamStyle = ss.ModeStream
	modeApprovalStyle = ss.ModeApproval

	// Focus border (focus.go vars)
	focusBorder = ss.FocusBorder
	unfocusedBorder = ss.UnfocusedBorder

	// Turn envelope border (turn_envelope.go vars)
	turnBorder = ss.TurnBorder

	// Separator focus (separator.go vars)
	sepFocusStyle = ss.SepFocus
}
