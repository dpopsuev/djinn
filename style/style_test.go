package style

import "testing"

func TestDefaultRegistry_Get(t *testing.T) {
	r := DefaultRegistry()
	p, ok := r.Get("djinn")
	if !ok {
		t.Fatal("djinn preset not found")
	}
	if p.Name != "djinn" {
		t.Fatalf("name = %q", p.Name)
	}
	if p.LogoColor != "#EE0000" {
		t.Fatalf("logo color = %q", p.LogoColor)
	}
}

func TestDefaultRegistry_Get_Minimal(t *testing.T) {
	r := DefaultRegistry()
	_, ok := r.Get("minimal")
	if !ok {
		t.Fatal("minimal preset not found")
	}
}

func TestDefaultRegistry_Get_Unknown(t *testing.T) {
	r := DefaultRegistry()
	_, ok := r.Get("neon")
	if ok {
		t.Fatal("unknown preset should not be found")
	}
}

func TestDefaultRegistry_List(t *testing.T) {
	r := DefaultRegistry()
	list := r.List()
	if len(list) != 2 {
		t.Fatalf("list = %d, want 2", len(list))
	}
}

func TestDefaultRegistry_Default(t *testing.T) {
	r := DefaultRegistry()
	d := r.Default()
	if d.Name != "djinn" {
		t.Fatalf("default = %q, want djinn", d.Name)
	}
}

func TestPreset_Labels(t *testing.T) {
	r := DefaultRegistry()
	p := r.Default()
	if p.UserLabel == "" {
		t.Fatal("user label should not be empty")
	}
	if p.AgentLabel == "" {
		t.Fatal("agent label should not be empty")
	}
}
