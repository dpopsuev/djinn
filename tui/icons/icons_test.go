package icons

import "testing"

func TestIcon_String_ASCII(t *testing.T) {
	// Default: NerdFontsAvailable = false (env not set in tests).
	icon := File
	if icon.String() != "F" {
		t.Errorf("File.String() = %q, want %q", icon.String(), "F")
	}
}

func TestIcon_AllDefined(t *testing.T) {
	allIcons := []Icon{File, Folder, Git, Branch, Tag, Check, Cross, Warning, Info, Error, Spinner, Agent, Tool, Clock, Budget}

	for i, icon := range allIcons {
		if icon.ASCII == "" {
			t.Errorf("icon %d has empty ASCII fallback", i)
		}
		if icon.Nerd == "" {
			t.Errorf("icon %d has empty Nerd glyph", i)
		}
	}
}
