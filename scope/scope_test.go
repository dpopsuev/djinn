package scope

import "testing"

func TestNavigator_StartsAtGeneral(t *testing.T) {
	nav := NewNavigator()
	if nav.Current().Level != General {
		t.Fatalf("level = %v", nav.Current().Level)
	}
	if nav.Path() != "general" {
		t.Fatalf("path = %q", nav.Path())
	}
}

func TestNavigator_DiveIntoEcosystem(t *testing.T) {
	nav := NewNavigator()
	nav.AddEcosystem("aeon", []string{"/workspace/djinn", "/workspace/misbah"})

	if err := nav.Dive("aeon"); err != nil {
		t.Fatal(err)
	}
	if nav.Current().Level != Ecosystem {
		t.Fatalf("level = %v", nav.Current().Level)
	}
	if nav.Path() != "general/aeon" {
		t.Fatalf("path = %q", nav.Path())
	}
}

func TestNavigator_DiveIntoSystem(t *testing.T) {
	nav := NewNavigator()
	nav.AddEcosystem("aeon", []string{"/workspace/djinn", "/workspace/misbah"})

	nav.Dive("aeon") //nolint:errcheck
	if err := nav.Dive("djinn"); err != nil {
		t.Fatal(err)
	}
	if nav.Current().Level != System {
		t.Fatalf("level = %v", nav.Current().Level)
	}
	if nav.Path() != "general/aeon/djinn" {
		t.Fatalf("path = %q", nav.Path())
	}
}

func TestNavigator_DirectPath(t *testing.T) {
	nav := NewNavigator()
	nav.AddEcosystem("aeon", []string{"/workspace/djinn"})

	// Direct path from general to system.
	if err := nav.Dive("aeon/djinn"); err != nil {
		t.Fatal(err)
	}
	if nav.Path() != "general/aeon/djinn" {
		t.Fatalf("path = %q", nav.Path())
	}
}

func TestNavigator_Climb(t *testing.T) {
	nav := NewNavigator()
	nav.AddEcosystem("aeon", []string{"/workspace/djinn"})
	nav.Dive("aeon/djinn") //nolint:errcheck

	nav.Climb()
	if nav.Path() != "general/aeon" {
		t.Fatalf("after climb = %q", nav.Path())
	}

	nav.Climb()
	if nav.Path() != "general" {
		t.Fatalf("after second climb = %q", nav.Path())
	}

	// Climbing at root stays at root.
	nav.Climb()
	if nav.Path() != "general" {
		t.Fatal("climb at root should stay")
	}
}

func TestNavigator_Root(t *testing.T) {
	nav := NewNavigator()
	nav.AddEcosystem("aeon", []string{"/workspace/djinn"})
	nav.Dive("aeon/djinn") //nolint:errcheck

	nav.Root()
	if nav.Path() != "general" {
		t.Fatalf("after root = %q", nav.Path())
	}
}

func TestNavigator_DiveNotFound(t *testing.T) {
	nav := NewNavigator()
	if err := nav.Dive("nonexistent"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNavigator_ChildNames(t *testing.T) {
	nav := NewNavigator()
	nav.AddEcosystem("aeon", []string{"/workspace/djinn", "/workspace/misbah"})
	nav.AddEcosystem("hegemony", []string{"/projects/hegemony"})

	names := nav.ChildNames()
	if len(names) != 2 {
		t.Fatalf("children = %v", names)
	}
}

func TestNavigator_SystemHasSingleRepo(t *testing.T) {
	nav := NewNavigator()
	nav.AddEcosystem("aeon", []string{"/workspace/djinn", "/workspace/misbah"})
	nav.Dive("aeon/djinn") //nolint:errcheck

	repos := nav.Current().Repos
	if len(repos) != 1 || repos[0] != "/workspace/djinn" {
		t.Fatalf("system repos = %v", repos)
	}
}
