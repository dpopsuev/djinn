package tui

import "testing"

// ---------------------------------------------------------------------------
// DefaultBindingTable (stubBindingTable) tests
// ---------------------------------------------------------------------------

func TestDefaultBindingTable_Lookup(t *testing.T) {
	tbl := DefaultBindingTable()
	cmd, ok := tbl.Lookup("ctrl+c")
	if !ok {
		t.Fatal("ctrl+c should be bound")
	}
	if cmd.Name != "quit" {
		t.Fatalf("ctrl+c command = %q, want quit", cmd.Name)
	}
}

func TestDefaultBindingTable_Lookup_AltM(t *testing.T) {
	tbl := DefaultBindingTable()
	cmd, ok := tbl.Lookup("alt+m")
	if !ok {
		t.Fatal("alt+m should be bound")
	}
	if cmd.Name != "cycle-mode" {
		t.Fatalf("alt+m command = %q, want cycle-mode", cmd.Name)
	}
}

func TestDefaultBindingTable_Lookup_Unknown(t *testing.T) {
	tbl := DefaultBindingTable()
	_, ok := tbl.Lookup("ctrl+shift+alt+z")
	if ok {
		t.Fatal("unknown key should not be bound")
	}
}

func TestDefaultBindingTable_Bindings(t *testing.T) {
	tbl := DefaultBindingTable()
	bindings := tbl.Bindings()
	if len(bindings) == 0 {
		t.Fatal("should have bindings")
	}
	found := false
	for _, b := range bindings {
		if b.Key == "enter" && b.Command == "submit" {
			found = true
		}
	}
	if !found {
		t.Fatal("enter -> submit binding missing")
	}
}

func TestDefaultBindingTable_LookupAllInsertKeys(t *testing.T) {
	tbl := DefaultBindingTable()

	cases := []struct {
		key     string
		command string
	}{
		{"ctrl+c", "quit"},
		{"alt+m", "cycle-mode"},
		{"up", "history-prev"},
		{"down", "history-next"},
		{"enter", "submit"},
		{"tab", "complete"},
		{"shift+tab", "focus-next"},
		{"pgup", "scroll-up"},
		{"pgdown", "scroll-down"},
		{"escape", "back"},
		{"alt+enter", "newline"},
	}

	for _, tc := range cases {
		cmd, ok := tbl.Lookup(tc.key)
		if !ok {
			t.Errorf("key %q should be bound", tc.key)
			continue
		}
		if cmd.Name != tc.command {
			t.Errorf("key %q: got command %q, want %q", tc.key, cmd.Name, tc.command)
		}
	}
}

func TestDefaultBindingTable_LookupEmptyString(t *testing.T) {
	tbl := DefaultBindingTable()
	_, ok := tbl.Lookup("")
	if ok {
		t.Fatal("empty string key should not be bound")
	}
}

func TestDefaultBindingTable_BindingsReturnsCopy(t *testing.T) {
	tbl := DefaultBindingTable()
	b1 := tbl.Bindings()
	b2 := tbl.Bindings()

	if len(b1) != len(b2) {
		t.Fatal("successive Bindings() calls should return same length")
	}

	// Mutating the returned slice should not affect future calls.
	b1[0].Key = "MUTATED"
	b3 := tbl.Bindings()
	if b3[0].Key == "MUTATED" {
		t.Fatal("Bindings() should return a copy, not a reference to internal state")
	}
}

// ---------------------------------------------------------------------------
// ModeTable tests
// ---------------------------------------------------------------------------

func TestNewModeTable_ReturnsValidTable(t *testing.T) {
	mt := NewModeTable()
	if mt == nil {
		t.Fatal("NewModeTable should not return nil")
	}
	if mt.CurrentMode() != KeyModeInsert {
		t.Fatalf("default mode = %q, want %q", mt.CurrentMode(), KeyModeInsert)
	}
}

func TestModeTable_ImplementsBindingTableInterface(t *testing.T) {
	// Compile-time check that *ModeTable satisfies BindingTable.
	var _ BindingTable = (*ModeTable)(nil)
}

func TestModeTable_InsertModeLookupAllKeys(t *testing.T) {
	mt := NewModeTable()

	cases := []struct {
		key     string
		command string
	}{
		{"ctrl+c", "quit"},
		{"alt+m", "cycle-mode"},
		{"up", "history-prev"},
		{"down", "history-next"},
		{"enter", "submit"},
		{"tab", "complete"},
		{"shift+tab", "focus-next"},
		{"pgup", "scroll-up"},
		{"pgdown", "scroll-down"},
		{"escape", "back"},
		{"alt+enter", "newline"},
	}

	for _, tc := range cases {
		cmd, ok := mt.Lookup(tc.key)
		if !ok {
			t.Errorf("[insert] key %q should be bound", tc.key)
			continue
		}
		if cmd.Name != tc.command {
			t.Errorf("[insert] key %q: got command %q, want %q", tc.key, cmd.Name, tc.command)
		}
		if cmd.Description == "" {
			t.Errorf("[insert] key %q: command %q has empty description", tc.key, cmd.Name)
		}
	}
}

func TestModeTable_InsertModeLookupUnknown(t *testing.T) {
	mt := NewModeTable()
	_, ok := mt.Lookup("f12")
	if ok {
		t.Fatal("[insert] unknown key f12 should not be bound")
	}
}

func TestModeTable_InsertModeLookupEmptyString(t *testing.T) {
	mt := NewModeTable()
	_, ok := mt.Lookup("")
	if ok {
		t.Fatal("[insert] empty string key should not be bound")
	}
}

func TestModeTable_NormalModeLookupAllKeys(t *testing.T) {
	mt := NewModeTable()
	mt.SetMode(KeyModeNormal)

	// Commands that exist in defaultKeyCommands.
	registered := []struct {
		key     string
		command string
	}{
		{"j", "focus-down"},
		{"k", "focus-up"},
		{"enter", "dive"},
		{"escape", "climb"},
		{"q", "quit"},
	}

	for _, tc := range registered {
		cmd, ok := mt.Lookup(tc.key)
		if !ok {
			t.Errorf("[normal] key %q should be bound", tc.key)
			continue
		}
		if cmd.Name != tc.command {
			t.Errorf("[normal] key %q: got command %q, want %q", tc.key, cmd.Name, tc.command)
		}
	}
}

func TestModeTable_NormalModeLookup_UnregisteredCommand(t *testing.T) {
	// "i" is bound to "enter-insert" but "enter-insert" is not in defaultKeyCommands.
	// Lookup should find the binding but fail to resolve the command, returning false.
	mt := NewModeTable()
	mt.SetMode(KeyModeNormal)

	_, ok := mt.Lookup("i")
	if ok {
		t.Fatal("[normal] key \"i\" maps to unregistered command \"enter-insert\"; Lookup should return false")
	}
}

func TestModeTable_NormalModeLookupUnknown(t *testing.T) {
	mt := NewModeTable()
	mt.SetMode(KeyModeNormal)
	_, ok := mt.Lookup("ctrl+c")
	if ok {
		t.Fatal("[normal] ctrl+c should not be bound in normal mode")
	}
}

func TestModeTable_NormalModeLookupEmptyString(t *testing.T) {
	mt := NewModeTable()
	mt.SetMode(KeyModeNormal)
	_, ok := mt.Lookup("")
	if ok {
		t.Fatal("[normal] empty string key should not be bound")
	}
}

func TestModeTable_SetModeAndCurrentMode(t *testing.T) {
	mt := NewModeTable()

	if mt.CurrentMode() != KeyModeInsert {
		t.Fatalf("initial mode = %q, want insert", mt.CurrentMode())
	}

	mt.SetMode(KeyModeNormal)
	if mt.CurrentMode() != KeyModeNormal {
		t.Fatalf("after SetMode(normal): mode = %q, want normal", mt.CurrentMode())
	}

	mt.SetMode(KeyModeInsert)
	if mt.CurrentMode() != KeyModeInsert {
		t.Fatalf("after SetMode(insert): mode = %q, want insert", mt.CurrentMode())
	}
}

func TestModeTable_ModeSwitchChangesLookup(t *testing.T) {
	mt := NewModeTable()

	// "enter" is "submit" in insert mode.
	cmd, ok := mt.Lookup("enter")
	if !ok || cmd.Name != "submit" {
		t.Fatalf("[insert] enter: got (%q, %v), want (submit, true)", cmd.Name, ok)
	}

	// "enter" is "dive" in normal mode.
	mt.SetMode(KeyModeNormal)
	cmd, ok = mt.Lookup("enter")
	if !ok || cmd.Name != "dive" {
		t.Fatalf("[normal] enter: got (%q, %v), want (dive, true)", cmd.Name, ok)
	}

	// Switch back to insert and confirm it's "submit" again.
	mt.SetMode(KeyModeInsert)
	cmd, ok = mt.Lookup("enter")
	if !ok || cmd.Name != "submit" {
		t.Fatalf("[insert again] enter: got (%q, %v), want (submit, true)", cmd.Name, ok)
	}
}

func TestModeTable_EscapeDiffersAcrossModes(t *testing.T) {
	mt := NewModeTable()

	// Insert mode: escape -> back
	cmd, ok := mt.Lookup("escape")
	if !ok || cmd.Name != "back" {
		t.Fatalf("[insert] escape: got (%q, %v), want (back, true)", cmd.Name, ok)
	}

	// Normal mode: escape -> climb
	mt.SetMode(KeyModeNormal)
	cmd, ok = mt.Lookup("escape")
	if !ok || cmd.Name != "climb" {
		t.Fatalf("[normal] escape: got (%q, %v), want (climb, true)", cmd.Name, ok)
	}
}

func TestModeTable_InsertBindings(t *testing.T) {
	mt := NewModeTable()
	bindings := mt.Bindings()

	if len(bindings) != len(defaultBindings) {
		t.Fatalf("insert bindings count = %d, want %d", len(bindings), len(defaultBindings))
	}

	// Verify the returned slice matches defaultBindings content.
	for i, b := range bindings {
		if b.Key != defaultBindings[i].Key || b.Command != defaultBindings[i].Command {
			t.Errorf("binding[%d] = {%q,%q}, want {%q,%q}",
				i, b.Key, b.Command, defaultBindings[i].Key, defaultBindings[i].Command)
		}
	}
}

func TestModeTable_NormalBindings(t *testing.T) {
	mt := NewModeTable()
	mt.SetMode(KeyModeNormal)
	bindings := mt.Bindings()

	if len(bindings) != 6 {
		t.Fatalf("normal bindings count = %d, want 6", len(bindings))
	}

	expected := map[string]string{
		"j":      "focus-down",
		"k":      "focus-up",
		"i":      "enter-insert", // command not in defaultKeyCommands, but binding still listed
		"enter":  "dive",
		"escape": "climb",
		"q":      "quit",
	}

	for _, b := range bindings {
		want, exists := expected[b.Key]
		if !exists {
			t.Errorf("unexpected normal binding: key=%q command=%q", b.Key, b.Command)
		} else if b.Command != want {
			t.Errorf("normal key %q: command = %q, want %q", b.Key, b.Command, want)
		}
	}
}

func TestModeTable_BindingsReturnsCopy(t *testing.T) {
	mt := NewModeTable()
	b1 := mt.Bindings()
	b2 := mt.Bindings()

	if len(b1) != len(b2) {
		t.Fatal("successive Bindings() calls should return same length")
	}

	b1[0].Key = "MUTATED"
	b3 := mt.Bindings()
	if b3[0].Key == "MUTATED" {
		t.Fatal("Bindings() should return a copy, not a reference to internal state")
	}
}

func TestModeTable_LookupUnknownMode(t *testing.T) {
	mt := NewModeTable()
	mt.SetMode(KeyMode("visual"))

	// No bindings registered for "visual" mode.
	_, ok := mt.Lookup("enter")
	if ok {
		t.Fatal("lookup in unregistered mode should return false")
	}

	bindings := mt.Bindings()
	if len(bindings) != 0 {
		t.Fatalf("bindings for unregistered mode should be empty, got %d", len(bindings))
	}
}

// ---------------------------------------------------------------------------
// KeyMode constants
// ---------------------------------------------------------------------------

func TestKeyModeConstants(t *testing.T) {
	if KeyModeInsert != "insert" {
		t.Fatalf("KeyModeInsert = %q, want \"insert\"", KeyModeInsert)
	}
	if KeyModeNormal != "normal" {
		t.Fatalf("KeyModeNormal = %q, want \"normal\"", KeyModeNormal)
	}
}

// ---------------------------------------------------------------------------
// KeyCommand struct
// ---------------------------------------------------------------------------

func TestKeyCommandDescriptionsNonEmpty(t *testing.T) {
	for name, cmd := range defaultKeyCommands {
		if cmd.Name != name {
			t.Errorf("command key %q has Name %q (mismatch)", name, cmd.Name)
		}
		if cmd.Description == "" {
			t.Errorf("command %q has empty description", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Special key names
// ---------------------------------------------------------------------------

func TestModeTable_SpecialKeyNames(t *testing.T) {
	mt := NewModeTable()

	specials := []string{
		"ctrl+c", "alt+m", "shift+tab", "alt+enter",
		"pgup", "pgdown", "up", "down", "tab", "enter", "escape",
	}

	for _, key := range specials {
		_, ok := mt.Lookup(key)
		if !ok {
			t.Errorf("special key %q should be bound in insert mode", key)
		}
	}
}
