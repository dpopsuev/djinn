package scope

import "testing"

func TestScopeTree_NewCreatesRoot(t *testing.T) {
	tree := NewScopeTree()

	if tree.Root == nil {
		t.Fatal("Root is nil")
	}
	if tree.Root.Path != "/" {
		t.Fatalf("Root.Path = %q, want /", tree.Root.Path)
	}
	if tree.Root.Level != LevelGeneral {
		t.Fatalf("Root.Level = %q, want general", tree.Root.Level)
	}
	if tree.Current() != tree.Root {
		t.Fatal("current should be root")
	}
	if tree.Pwd() != "/" {
		t.Fatalf("Pwd = %q, want /", tree.Pwd())
	}
}

func TestScopeTree_AddEcosystemAndSystem(t *testing.T) {
	tree := NewScopeTree()
	eco := tree.AddEcosystem("aeon", []string{"/workspace/djinn", "/workspace/misbah"})

	if eco.Path != "/aeon" {
		t.Fatalf("eco.Path = %q", eco.Path)
	}
	if eco.Level != LevelEcosystem {
		t.Fatalf("eco.Level = %q", eco.Level)
	}
	if len(eco.Repos) != 2 {
		t.Fatalf("eco.Repos = %v", eco.Repos)
	}
	if eco.Parent != tree.Root {
		t.Fatal("eco.Parent should be root")
	}

	sys := tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	if sys == nil {
		t.Fatal("AddSystem returned nil")
	}
	if sys.Path != "/aeon/djinn" {
		t.Fatalf("sys.Path = %q", sys.Path)
	}
	if sys.Level != LevelSystem {
		t.Fatalf("sys.Level = %q", sys.Level)
	}
	if len(sys.Repos) != 1 || sys.Repos[0] != "/workspace/djinn" {
		t.Fatalf("sys.Repos = %v", sys.Repos)
	}
	if sys.Parent != eco {
		t.Fatal("sys.Parent should be eco")
	}
}

func TestScopeTree_AddSystemMissingEcosystem(t *testing.T) {
	tree := NewScopeTree()
	sys := tree.AddSystem("nonexistent", "djinn", "/workspace/djinn")
	if sys != nil {
		t.Fatal("AddSystem should return nil for missing ecosystem")
	}
}

func TestScopeTree_NavigateUp(t *testing.T) {
	tree := NewScopeTree()
	eco := tree.AddEcosystem("aeon", nil)
	tree.SetCurrent(eco)

	node, err := tree.Navigate("..")
	if err != nil {
		t.Fatalf("Navigate ..: %v", err)
	}
	if node != tree.Root {
		t.Fatalf("expected root, got %q", node.Path)
	}

	// ".." at root stays at root.
	tree.SetCurrent(tree.Root)
	node, err = tree.Navigate("..")
	if err != nil {
		t.Fatalf("Navigate .. at root: %v", err)
	}
	if node != tree.Root {
		t.Fatal("should stay at root")
	}
}

func TestScopeTree_NavigateRoot(t *testing.T) {
	tree := NewScopeTree()
	eco := tree.AddEcosystem("aeon", nil)
	tree.SetCurrent(eco)

	node, err := tree.Navigate("/")
	if err != nil {
		t.Fatalf("Navigate /: %v", err)
	}
	if node != tree.Root {
		t.Fatal("should be root")
	}
}

func TestScopeTree_NavigateRelative(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	tree.AddSystem("aeon", "bugle", "/workspace/bugle")

	// From root, navigate relative to aeon/djinn.
	node, err := tree.Navigate("aeon/djinn")
	if err != nil {
		t.Fatalf("Navigate relative: %v", err)
	}
	if node.Path != "/aeon/djinn" {
		t.Fatalf("path = %q", node.Path)
	}
}

func TestScopeTree_NavigateAbsolute(t *testing.T) {
	tree := NewScopeTree()
	eco := tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	tree.SetCurrent(eco)

	// From ecosystem, navigate absolute to /aeon/djinn.
	node, err := tree.Navigate("/aeon/djinn")
	if err != nil {
		t.Fatalf("Navigate absolute: %v", err)
	}
	if node.Path != "/aeon/djinn" {
		t.Fatalf("path = %q", node.Path)
	}
}

func TestScopeTree_NavigateNotFound(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)

	_, err := tree.Navigate("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing scope")
	}

	_, err = tree.Navigate("/aeon/nonexistent")
	if err == nil {
		t.Fatal("expected error for missing system under valid ecosystem")
	}
}

func TestScopeTree_NavigateUpThenDown(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	tree.AddSystem("aeon", "bugle", "/workspace/bugle")

	// Navigate to djinn.
	djinn, _ := tree.Navigate("aeon/djinn")
	tree.SetCurrent(djinn)

	// ".." then "bugle" — navigate to sibling.
	node, err := tree.Navigate("../bugle")
	if err != nil {
		t.Fatalf("Navigate ../bugle: %v", err)
	}
	if node.Path != "/aeon/bugle" {
		t.Fatalf("path = %q, want /aeon/bugle", node.Path)
	}
}

func TestScopeTree_Pwd(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")

	if tree.Pwd() != "/" {
		t.Fatalf("Pwd = %q, want /", tree.Pwd())
	}

	node, _ := tree.Navigate("aeon/djinn")
	tree.SetCurrent(node)
	if tree.Pwd() != "/aeon/djinn" {
		t.Fatalf("Pwd = %q, want /aeon/djinn", tree.Pwd())
	}
}

func TestScopeTree_Ls(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddEcosystem("personal", nil)

	children := tree.Ls()
	if len(children) != 2 {
		t.Fatalf("Ls = %d children, want 2", len(children))
	}

	names := make(map[string]bool)
	for _, c := range children {
		names[c.Name] = true
	}
	if !names["aeon"] || !names["personal"] {
		t.Fatalf("Ls = %v", names)
	}
}

func TestScopeTree_RequiresSandbox(t *testing.T) {
	tree := NewScopeTree()
	eco := tree.AddEcosystem("aeon", []string{"/workspace/djinn"})
	sys := tree.AddSystem("aeon", "djinn", "/workspace/djinn")

	if tree.Root.RequiresSandbox() {
		t.Fatal("general should not require sandbox")
	}
	if eco.RequiresSandbox() {
		t.Fatal("ecosystem should not require sandbox")
	}
	if !sys.RequiresSandbox() {
		t.Fatal("system should require sandbox")
	}
}

func TestScopeTree_HasCode(t *testing.T) {
	tree := NewScopeTree()

	if tree.Root.HasCode() {
		t.Fatal("general should not have code")
	}

	eco := tree.AddEcosystem("aeon", []string{"/workspace/djinn"})
	if !eco.HasCode() {
		t.Fatal("ecosystem with repos should have code")
	}

	ecoEmpty := tree.AddEcosystem("empty", nil)
	if ecoEmpty.HasCode() {
		t.Fatal("ecosystem with no repos should not have code")
	}

	sys := tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	if !sys.HasCode() {
		t.Fatal("system should have code")
	}
}

func TestScopeTree_PreviousForCdMinus(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")

	// Initially no previous.
	if tree.Previous() != nil {
		t.Fatal("previous should be nil initially")
	}

	// Simulate cd from root to /aeon/djinn.
	old := tree.Current()
	node, _ := tree.Navigate("aeon/djinn")
	tree.SetPrevious(old)
	tree.SetCurrent(node)

	if tree.Pwd() != "/aeon/djinn" {
		t.Fatalf("Pwd = %q", tree.Pwd())
	}
	if tree.Previous().Path != "/" {
		t.Fatalf("previous = %q, want /", tree.Previous().Path)
	}

	// Simulate "cd -" — swap current and previous.
	prev := tree.Previous()
	tree.SetPrevious(tree.Current())
	tree.SetCurrent(prev)

	if tree.Pwd() != "/" {
		t.Fatalf("Pwd after cd - = %q", tree.Pwd())
	}
	if tree.Previous().Path != "/aeon/djinn" {
		t.Fatalf("previous after cd - = %q", tree.Previous().Path)
	}
}

func TestScopeTree_ComplexTree(t *testing.T) {
	tree := NewScopeTree()

	// Build aeon ecosystem with multiple systems.
	aeonRepos := []string{
		"/workspace/djinn", "/workspace/misbah", "/workspace/scribe",
		"/workspace/locus", "/workspace/limes", "/workspace/lex",
		"/workspace/emcee", "/workspace/origami", "/workspace/macro",
		"/workspace/bugle", "/workspace/tubus", "/workspace/sophia",
		"/workspace/monad", "/workspace/gundog", "/workspace/asterisk",
		"/workspace/kabuki", "/workspace/kami", "/workspace/cherub",
		"/workspace/achilles", "/workspace/universalis", "/workspace/lexicon",
	}
	tree.AddEcosystem("aeon", aeonRepos)

	systems := []string{"djinn", "misbah", "scribe", "locus", "bugle"}
	for _, name := range systems {
		tree.AddSystem("aeon", name, "/workspace/"+name)
	}

	// Verify tree structure.
	if len(tree.Root.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(tree.Root.Children))
	}

	aeon := tree.Root.Children[0]
	if len(aeon.Children) != len(systems) {
		t.Fatalf("aeon children = %d, want %d", len(aeon.Children), len(systems))
	}

	// Navigate to each system.
	for _, name := range systems {
		node, err := tree.Navigate("/aeon/" + name)
		if err != nil {
			t.Fatalf("Navigate to %s: %v", name, err)
		}
		if node.Name != name {
			t.Fatalf("name = %q, want %q", node.Name, name)
		}
		if !node.RequiresSandbox() {
			t.Fatalf("%s should require sandbox", name)
		}
	}

	// Verify root repos aggregation.
	if len(aeon.Repos) != 21 {
		t.Fatalf("aeon repos = %d, want 21", len(aeon.Repos))
	}
}
