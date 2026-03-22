package keybind

import "testing"

func TestDefaultTable_Lookup(t *testing.T) {
	tbl := DefaultTable()
	cmd, ok := tbl.Lookup("ctrl+c")
	if !ok {
		t.Fatal("ctrl+c should be bound")
	}
	if cmd.Name != "quit" {
		t.Fatalf("ctrl+c command = %q, want quit", cmd.Name)
	}
}

func TestDefaultTable_Lookup_AltM(t *testing.T) {
	tbl := DefaultTable()
	cmd, ok := tbl.Lookup("alt+m")
	if !ok {
		t.Fatal("alt+m should be bound")
	}
	if cmd.Name != "cycle-mode" {
		t.Fatalf("alt+m command = %q, want cycle-mode", cmd.Name)
	}
}

func TestDefaultTable_Lookup_Unknown(t *testing.T) {
	tbl := DefaultTable()
	_, ok := tbl.Lookup("ctrl+shift+alt+z")
	if ok {
		t.Fatal("unknown key should not be bound")
	}
}

func TestDefaultTable_Bindings(t *testing.T) {
	tbl := DefaultTable()
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
		t.Fatal("enter → submit binding missing")
	}
}
