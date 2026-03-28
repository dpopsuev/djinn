package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultTokens_MatchesTheme(t *testing.T) {
	ts := DefaultTokens()

	// Direct Theme fields must round-trip exactly.
	checks := []struct {
		name      string
		got, want lipgloss.AdaptiveColor
	}{
		{"UserFg", ts.UserFg, DefaultTheme.User},
		{"AssistantFg", ts.AssistantFg, DefaultTheme.Assistant},
		{"ToolNameFg", ts.ToolNameFg, DefaultTheme.ToolName},
		{"ToolArgFg", ts.ToolArgFg, DefaultTheme.ToolArg},
		{"SuccessFg", ts.SuccessFg, DefaultTheme.Success},
		{"ErrorFg", ts.ErrorFg, DefaultTheme.Error},
		{"AccentFg", ts.AccentFg, DefaultTheme.Accent},
		{"FocusDimFg", ts.FocusDimFg, DefaultTheme.FocusDim},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestTokensFromTheme_CustomDiffers(t *testing.T) {
	custom := Theme{
		User:      lipgloss.AdaptiveColor{Light: "#ff0000", Dark: "#ff0000"},
		Assistant: lipgloss.AdaptiveColor{Light: "#00ff00", Dark: "#00ff00"},
		ToolName:  lipgloss.AdaptiveColor{Light: "#0000ff", Dark: "#0000ff"},
		ToolArg:   lipgloss.AdaptiveColor{Light: "#ffff00", Dark: "#ffff00"},
		Success:   lipgloss.AdaptiveColor{Light: "#00ffff", Dark: "#00ffff"},
		Error:     lipgloss.AdaptiveColor{Light: "#ff00ff", Dark: "#ff00ff"},
		Accent:    lipgloss.AdaptiveColor{Light: "#111111", Dark: "#222222"},
		FocusDim:  lipgloss.AdaptiveColor{Light: "#333333", Dark: "#444444"},
	}
	ts := TokensFromTheme(custom)
	defaults := DefaultTokens()

	if ts.UserFg == defaults.UserFg {
		t.Error("custom token should differ from default")
	}
	if ts.UserFg != custom.User {
		t.Error("custom token should match custom theme")
	}
}

func TestDefaultTokens_ExtendedFieldsPopulated(t *testing.T) {
	ts := DefaultTokens()

	extended := []struct {
		name string
		c    lipgloss.AdaptiveColor
	}{
		{"DiffAddFg", ts.DiffAddFg},
		{"DiffDelFg", ts.DiffDelFg},
		{"DiffHeaderFg", ts.DiffHeaderFg},
		{"HealthGreenFg", ts.HealthGreenFg},
		{"HealthYellowFg", ts.HealthYellowFg},
		{"HealthRedFg", ts.HealthRedFg},
		{"WarningFg", ts.WarningFg},
		{"ZoneColdFg", ts.ZoneColdFg},
		{"ZoneFocusedFg", ts.ZoneFocusedFg},
	}
	for _, c := range extended {
		if c.c.Light == "" || c.c.Dark == "" {
			t.Errorf("%s: has empty Light or Dark value", c.name)
		}
	}
}

func TestApplyTokens_RebuildsCoreStyles(t *testing.T) {
	custom := Theme{
		User:      lipgloss.AdaptiveColor{Light: "#aaaaaa", Dark: "#bbbbbb"},
		Assistant: lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#dddddd"},
		ToolName:  lipgloss.AdaptiveColor{Light: "#eeeeee", Dark: "#ffffff"},
		ToolArg:   lipgloss.AdaptiveColor{Light: "#111111", Dark: "#222222"},
		Success:   lipgloss.AdaptiveColor{Light: "#333333", Dark: "#444444"},
		Error:     lipgloss.AdaptiveColor{Light: "#555555", Dark: "#666666"},
		Accent:    lipgloss.AdaptiveColor{Light: "#777777", Dark: "#888888"},
		FocusDim:  lipgloss.AdaptiveColor{Light: "#999999", Dark: "#aaaaaa"},
	}
	ts := TokensFromTheme(custom)
	ApplyTokens(ts)
	defer ApplyTokens(DefaultTokens()) // restore

	if ActiveTokens.UserFg != custom.User {
		t.Error("ActiveTokens.UserFg not updated")
	}
	if RedHatRed != custom.Accent {
		t.Error("RedHatRed not updated by ApplyTokens")
	}
}

func TestApplyTokens_RebuildsDiffStyles(t *testing.T) {
	custom := Theme{
		User:      lipgloss.AdaptiveColor{Light: "#aaaaaa", Dark: "#bbbbbb"},
		Assistant: lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#dddddd"},
		ToolName:  lipgloss.AdaptiveColor{Light: "#eeeeee", Dark: "#ffffff"},
		ToolArg:   lipgloss.AdaptiveColor{Light: "#111111", Dark: "#222222"},
		Success:   lipgloss.AdaptiveColor{Light: "#333333", Dark: "#444444"},
		Error:     lipgloss.AdaptiveColor{Light: "#555555", Dark: "#666666"},
		Accent:    lipgloss.AdaptiveColor{Light: "#777777", Dark: "#888888"},
		FocusDim:  lipgloss.AdaptiveColor{Light: "#999999", Dark: "#aaaaaa"},
	}
	ApplyTokens(TokensFromTheme(custom))
	defer ApplyTokens(DefaultTokens())

	// Verify diff styles were rebuilt (they should use Success/Error, not hardcoded)
	if ActiveTokens.DiffAddFg != custom.Success {
		t.Error("DiffAddFg should derive from Success")
	}
	if ActiveTokens.DiffDelFg != custom.Error {
		t.Error("DiffDelFg should derive from Error")
	}
}
