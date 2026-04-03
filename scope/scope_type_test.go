package scope

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScopeType_Infer_GitRepo(t *testing.T) {
	dir := t.TempDir()

	// Create .git directory to simulate a repo.
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := InferScopeType(dir)
	if got != ScopeSystem {
		t.Fatalf("InferScopeType(%q) = %q, want %q", dir, got, ScopeSystem)
	}
}

func TestScopeType_Infer_GoWork(t *testing.T) {
	dir := t.TempDir()

	// Create go.work file to simulate a multi-repo workspace.
	if err := os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := InferScopeType(dir)
	if got != ScopeEcosystem {
		t.Fatalf("InferScopeType(%q) = %q, want %q", dir, got, ScopeEcosystem)
	}
}

func TestScopeType_Infer_GoWorkPrecedence(t *testing.T) {
	dir := t.TempDir()

	// Both go.work and .git exist — go.work wins (ecosystem).
	if err := os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := InferScopeType(dir)
	if got != ScopeEcosystem {
		t.Fatalf("InferScopeType with both = %q, want %q (go.work takes precedence)", got, ScopeEcosystem)
	}
}

func TestScopeType_Infer_NeitherGlobal(t *testing.T) {
	dir := t.TempDir()

	got := InferScopeType(dir)
	if got != ScopeGlobal {
		t.Fatalf("InferScopeType(empty) = %q, want %q", got, ScopeGlobal)
	}
}

func TestScopeType_Valid(t *testing.T) {
	valid := []ScopeType{ScopeGlobal, ScopeEcosystem, ScopeSystem, ScopePrototype, ScopeOperations}
	for _, st := range valid {
		if !st.Valid() {
			t.Errorf("%q should be valid", st)
		}
	}

	if ScopeType("bogus").Valid() {
		t.Error("bogus should not be valid")
	}
}

func TestScopeType_String(t *testing.T) {
	if ScopeSystem.String() != "system" {
		t.Fatalf("String() = %q", ScopeSystem.String())
	}
}
