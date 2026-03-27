// tokens.go — semantic color tokens bridging palette → purpose.
//
// TokenSet maps visual purpose to adaptive colors. All styles read from
// ActiveTokens, never from raw hex. Themes remap tokens via TokensFromTheme().
//
// Layer 0b in the design system: palette (colors.go) → tokens → styles.
package tui

import "github.com/charmbracelet/lipgloss"

// TokenSet maps visual purpose to color. Every styled element reads from here.
type TokenSet struct {
	// Identity — who is speaking
	UserFg      lipgloss.AdaptiveColor
	AssistantFg lipgloss.AdaptiveColor

	// Tool status
	ToolNameFg lipgloss.AdaptiveColor
	ToolArgFg  lipgloss.AdaptiveColor

	// State — what is happening
	SuccessFg lipgloss.AdaptiveColor
	ErrorFg   lipgloss.AdaptiveColor
	WarningFg lipgloss.AdaptiveColor

	// Brand
	AccentFg   lipgloss.AdaptiveColor
	FocusDimFg lipgloss.AdaptiveColor

	// Diff
	DiffAddFg    lipgloss.AdaptiveColor
	DiffDelFg    lipgloss.AdaptiveColor
	DiffHeaderFg lipgloss.AdaptiveColor

	// Health / coherence zones (thermal gradient)
	HealthGreenFg  lipgloss.AdaptiveColor
	HealthYellowFg lipgloss.AdaptiveColor
	HealthRedFg    lipgloss.AdaptiveColor

	// Coherence — extended zones beyond basic health
	ZoneColdFg    lipgloss.AdaptiveColor // blue — fresh context
	ZoneFocusedFg lipgloss.AdaptiveColor // dark green — deep focus
}

// ActiveTokens is the live token set. Rebuilt by ApplyTokens().
var ActiveTokens TokenSet

func init() {
	ApplyTokens(DefaultTokens())
}

// DefaultTokens returns tokens derived from DefaultTheme.
func DefaultTokens() TokenSet {
	return TokensFromTheme(DefaultTheme)
}

// TokensFromTheme maps a Theme to a full TokenSet.
// Theme covers 8 semantic colors; tokens extend with diff, health, and zone colors.
func TokensFromTheme(t Theme) TokenSet {
	return TokenSet{
		// Direct from Theme
		UserFg:      t.User,
		AssistantFg: t.Assistant,
		ToolNameFg:  t.ToolName,
		ToolArgFg:   t.ToolArg,
		SuccessFg:   t.Success,
		ErrorFg:     t.Error,
		AccentFg:    t.Accent,
		FocusDimFg:  t.FocusDim,

		// Extended — derived from Theme semantics
		WarningFg:    t.ToolName, // yellow/warning shares tool name color
		DiffAddFg:    t.Success,  // green = additions
		DiffDelFg:    t.Error,    // red = deletions
		DiffHeaderFg: lipgloss.AdaptiveColor{Light: Teal40, Dark: Teal20}, // diff headers — teal family

		// Health uses the traffic light triad
		HealthGreenFg:  t.Success,
		HealthYellowFg: t.ToolName,
		HealthRedFg:    t.Error,

		// Coherence thermal gradient
		ZoneColdFg:    t.Assistant,                                                   // blue
		ZoneFocusedFg: lipgloss.AdaptiveColor{Light: Green40, Dark: Green30}, // dark green — deep focus zone
	}
}

// ApplyTokens rebuilds all global style variables from the given token set.
// This is the SINGLE WRITER of style vars — no other code should assign them.
func ApplyTokens(ts TokenSet) {
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
