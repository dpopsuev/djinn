package tui

import (
	"testing"

	"github.com/dpopsuev/djinn/tui/icons"
)

// ═══════════════════════════════════════════════════════════════════════
// RED: Fallback behavior
// ═══════════════════════════════════════════════════════════════════════

func TestIcon_ASCIIFallback(t *testing.T) {
	old := icons.NerdFontsAvailable
	icons.NerdFontsAvailable = false
	defer func() { icons.NerdFontsAvailable = old }()

	if icons.Check.String() != "✓" {
		t.Fatalf("icons.Check ASCII = %q, want ✓", icons.Check.String())
	}
	if icons.File.String() != "F" {
		t.Fatalf("icons.File ASCII = %q, want F", icons.File.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GREEN: Nerd Font mode
// ═══════════════════════════════════════════════════════════════════════

func TestIcon_NerdFontEnabled(t *testing.T) {
	old := icons.NerdFontsAvailable
	icons.NerdFontsAvailable = true
	defer func() { icons.NerdFontsAvailable = old }()

	// Should return Nerd glyph, not ASCII.
	if icons.Check.String() == "✓" {
		t.Fatal("should return Nerd glyph when enabled")
	}
	if icons.Check.String() == "" {
		t.Fatal("Nerd glyph should not be empty")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BLUE: All icons have values
// ═══════════════════════════════════════════════════════════════════════

func TestIcon_AllIconsHaveValues(t *testing.T) {
	allIcons := []icons.Icon{
		icons.File, icons.Folder, icons.Git, icons.Branch, icons.Tag,
		icons.Check, icons.Cross, icons.Warning, icons.Info, icons.Error,
		icons.Spinner, icons.Agent, icons.Tool, icons.Clock, icons.Budget,
	}
	for _, icon := range allIcons {
		if icon.Nerd == "" {
			t.Fatalf("icon has empty Nerd glyph: %+v", icon)
		}
		if icon.ASCII == "" {
			t.Fatalf("icon has empty ASCII fallback: %+v", icon)
		}
	}
}

func TestIcon_StringRoutes(t *testing.T) {
	old := icons.NerdFontsAvailable
	defer func() { icons.NerdFontsAvailable = old }()

	icons.NerdFontsAvailable = false
	ascii := icons.Git.String()

	icons.NerdFontsAvailable = true
	nerd := icons.Git.String()

	if ascii == nerd {
		t.Fatal("ASCII and Nerd should be different strings")
	}
}
