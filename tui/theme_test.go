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
}
