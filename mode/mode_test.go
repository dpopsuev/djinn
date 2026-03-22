package mode

import "testing"

func TestDefaultRegistry_Get(t *testing.T) {
	r := DefaultRegistry()
	for _, name := range []string{"ask", "plan", "agent", "auto"} {
		d, ok := r.Get(name)
		if !ok {
			t.Fatalf("Get(%q) not found", name)
		}
		if d.Name != name {
			t.Fatalf("Get(%q).Name = %q", name, d.Name)
		}
	}
}

func TestDefaultRegistry_Get_Unknown(t *testing.T) {
	r := DefaultRegistry()
	_, ok := r.Get("yolo")
	if ok {
		t.Fatal("unknown mode should not be found")
	}
}

func TestDefaultRegistry_List(t *testing.T) {
	r := DefaultRegistry()
	list := r.List()
	if len(list) != 4 {
		t.Fatalf("list = %d, want 4", len(list))
	}
}

func TestDefinition_ToolsEnabled(t *testing.T) {
	r := DefaultRegistry()
	ask, _ := r.Get("ask")
	if ask.ToolsEnabled {
		t.Fatal("ask should not have tools")
	}
	auto, _ := r.Get("auto")
	if !auto.ToolsEnabled {
		t.Fatal("auto should have tools")
	}
}

func TestToAgentMode(t *testing.T) {
	m, err := ToAgentMode("auto")
	if err != nil {
		t.Fatal(err)
	}
	if m.String() != "auto" {
		t.Fatalf("mode = %s", m)
	}
}

func TestToAgentMode_Invalid(t *testing.T) {
	_, err := ToAgentMode("yolo")
	if err == nil {
		t.Fatal("expected error")
	}
}
