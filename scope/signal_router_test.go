package scope

import "testing"

func TestSignalRouter_SameScope(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	router := NewSignalRouter(tree)

	if !router.ShouldPropagate("/aeon/djinn", "/aeon/djinn") {
		t.Fatal("same scope should be visible")
	}
}

func TestSignalRouter_ParentSeesChild(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	router := NewSignalRouter(tree)

	// Parent ecosystem sees child system signal.
	if !router.ShouldPropagate("/aeon/djinn", "/aeon") {
		t.Fatal("parent should see child signal")
	}
}

func TestSignalRouter_RootSeesAll(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	router := NewSignalRouter(tree)

	if !router.ShouldPropagate("/aeon/djinn", "/") {
		t.Fatal("root should see system signal")
	}
	if !router.ShouldPropagate("/aeon", "/") {
		t.Fatal("root should see ecosystem signal")
	}
}

func TestSignalRouter_SiblingDoesNotSee(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	tree.AddSystem("aeon", "bugle", "/workspace/bugle")
	router := NewSignalRouter(tree)

	if router.ShouldPropagate("/aeon/djinn", "/aeon/bugle") {
		t.Fatal("sibling should NOT see signal")
	}
	if router.ShouldPropagate("/aeon/bugle", "/aeon/djinn") {
		t.Fatal("sibling should NOT see signal (reverse)")
	}
}

func TestSignalRouter_DifferentEcosystem(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddEcosystem("personal", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	tree.AddSystem("personal", "blog", "/workspace/blog")
	router := NewSignalRouter(tree)

	if router.ShouldPropagate("/aeon/djinn", "/personal") {
		t.Fatal("different ecosystem should NOT see signal")
	}
	if router.ShouldPropagate("/personal/blog", "/aeon") {
		t.Fatal("different ecosystem should NOT see signal (reverse)")
	}
}

func TestSignalRouter_GeneralSeesEverything(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddEcosystem("personal", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	tree.AddSystem("personal", "blog", "/workspace/blog")
	router := NewSignalRouter(tree)

	scopes := []string{"/aeon", "/aeon/djinn", "/personal", "/personal/blog"}
	for _, s := range scopes {
		if !router.ShouldPropagate(s, "/") {
			t.Fatalf("general should see signal from %s", s)
		}
	}
}

func TestSignalRouter_ChildDoesNotSeeParent(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddSystem("aeon", "djinn", "/workspace/djinn")
	router := NewSignalRouter(tree)

	// Child should NOT see parent's signals.
	if router.ShouldPropagate("/aeon", "/aeon/djinn") {
		t.Fatal("child should NOT see parent signal")
	}
	if router.ShouldPropagate("/", "/aeon") {
		t.Fatal("child should NOT see root signal")
	}
}

func TestSignalRouter_PrefixBoundary(t *testing.T) {
	tree := NewScopeTree()
	tree.AddEcosystem("aeon", nil)
	tree.AddEcosystem("aeon2", nil)
	router := NewSignalRouter(tree)

	// /aeon2 should NOT match /aeon as parent (prefix boundary check).
	if router.ShouldPropagate("/aeon2", "/aeon") {
		t.Fatal("/aeon should NOT see /aeon2 signal (prefix boundary)")
	}
}
