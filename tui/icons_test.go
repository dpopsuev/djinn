package tui

import "testing"

// ═══════════════════════════════════════════════════════════════════════
// RED: Fallback behavior
// ═══════════════════════════════════════════════════════════════════════

func TestIcon_ASCIIFallback(t *testing.T) {
	old := NerdFontsAvailable
	NerdFontsAvailable = false
	defer func() { NerdFontsAvailable = old }()

	if IconCheck.String() != "✓" {
		t.Fatalf("IconCheck ASCII = %q, want ✓", IconCheck.String())
	}
	if IconFile.String() != "F" {
		t.Fatalf("IconFile ASCII = %q, want F", IconFile.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GREEN: Nerd Font mode
// ═══════════════════════════════════════════════════════════════════════

func TestIcon_NerdFontEnabled(t *testing.T) {
	old := NerdFontsAvailable
	NerdFontsAvailable = true
	defer func() { NerdFontsAvailable = old }()

	// Should return Nerd glyph, not ASCII.
	if IconCheck.String() == "✓" {
		t.Fatal("should return Nerd glyph when enabled")
	}
	if IconCheck.String() == "" {
		t.Fatal("Nerd glyph should not be empty")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BLUE: All icons have values
// ═══════════════════════════════════════════════════════════════════════

func TestIcon_AllIconsHaveValues(t *testing.T) {
	icons := []Icon{
		IconFile, IconFolder, IconGit, IconBranch, IconTag,
		IconCheck, IconCross, IconWarning, IconInfo, IconError,
		IconSpinner, IconAgent, IconTool, IconClock, IconBudget,
	}
	for _, icon := range icons {
		if icon.Nerd == "" {
			t.Fatalf("icon has empty Nerd glyph: %+v", icon)
		}
		if icon.ASCII == "" {
			t.Fatalf("icon has empty ASCII fallback: %+v", icon)
		}
	}
}

func TestIcon_StringRoutes(t *testing.T) {
	old := NerdFontsAvailable
	defer func() { NerdFontsAvailable = old }()

	NerdFontsAvailable = false
	ascii := IconGit.String()

	NerdFontsAvailable = true
	nerd := IconGit.String()

	if ascii == nerd {
		t.Fatal("ASCII and Nerd should be different strings")
	}
}
