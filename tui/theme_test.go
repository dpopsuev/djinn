package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultTheme_AllFieldsSet(t *testing.T) {
	th := DefaultTheme
	if th.User.Light == "" || th.User.Dark == "" {
		t.Fatal("User color missing")
	}
	if th.Assistant.Light == "" || th.Assistant.Dark == "" {
		t.Fatal("Assistant color missing")
	}
	if th.ToolName.Light == "" || th.ToolName.Dark == "" {
		t.Fatal("ToolName color missing")
	}
	if th.Error.Light == "" || th.Error.Dark == "" {
		t.Fatal("Error color missing")
	}
	if th.Accent.Light == "" || th.Accent.Dark == "" {
		t.Fatal("Accent color missing")
	}
}

func TestApplyTheme_UpdatesStyles(t *testing.T) {
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
	ApplyTheme(custom)
	defer ApplyTheme(DefaultTheme) // restore

	if ActiveTheme.User.Light != "#ff0000" {
		t.Fatal("theme not applied")
	}
	// ApplyTheme must flow through ApplyTokens
	if ActiveTokens.UserFg != custom.User {
		t.Fatal("ApplyTheme did not update ActiveTokens")
	}
	if RedHatRed != custom.Accent {
		t.Fatal("ApplyTheme did not rebuild RedHatRed via tokens")
	}
}

func TestRegisterTheme_RoundTrip(t *testing.T) {
	custom := Theme{
		User:      lipgloss.AdaptiveColor{Light: "#abcdef", Dark: "#abcdef"},
		Assistant: lipgloss.AdaptiveColor{Light: "#abcdef", Dark: "#abcdef"},
		ToolName:  lipgloss.AdaptiveColor{Light: "#abcdef", Dark: "#abcdef"},
		ToolArg:   lipgloss.AdaptiveColor{Light: "#abcdef", Dark: "#abcdef"},
		Success:   lipgloss.AdaptiveColor{Light: "#abcdef", Dark: "#abcdef"},
		Error:     lipgloss.AdaptiveColor{Light: "#abcdef", Dark: "#abcdef"},
		Accent:    lipgloss.AdaptiveColor{Light: "#abcdef", Dark: "#abcdef"},
		FocusDim:  lipgloss.AdaptiveColor{Light: "#abcdef", Dark: "#abcdef"},
	}
	RegisterTheme("neon", custom)
	defer delete(themeRegistry, "neon") // cleanup

	got := ThemeByName("neon")
	if got.User != custom.User {
		t.Fatal("registered theme not returned")
	}
}

func TestThemeByName_UnknownReturnsDefault(t *testing.T) {
	got := ThemeByName("nonexistent")
	if got.User != DefaultTheme.User {
		t.Fatal("unknown name should return DefaultTheme")
	}
}

func TestThemeNames_ContainsBuiltins(t *testing.T) {
	names := ThemeNames()
	want := map[string]bool{"djinn": false, "claude": false, "gemini": false, "codex": false}
	for _, n := range names {
		want[n] = true
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing built-in theme: %s", name)
		}
	}
}
