package uniform

import (
	"testing"

	"github.com/dpopsuev/djinn/jerichoport"
)

func TestAssignDisplay(t *testing.T) {
	sw := &StaffWorld{
		Registry: jerichoport.NewRegistry(),
	}

	d, err := sw.AssignDisplay("executor", "refactor")
	if err != nil {
		t.Fatalf("AssignDisplay: %v", err)
	}
	if d.Name == "" {
		t.Error("Display.Name should be a heraldic color name")
	}
	if d.Color == "" {
		t.Error("Display.Color should be a hex string")
	}
	if d.Color[0] != '#' {
		t.Errorf("Display.Color = %q, want hex starting with #", d.Color)
	}
}

func TestAssignDisplay_Unique(t *testing.T) {
	sw := &StaffWorld{
		Registry: jerichoport.NewRegistry(),
	}

	d1, err := sw.AssignDisplay("executor", "auth")
	if err != nil {
		t.Fatalf("first AssignDisplay: %v", err)
	}
	d2, err := sw.AssignDisplay("inspector", "auth")
	if err != nil {
		t.Fatalf("second AssignDisplay: %v", err)
	}

	// Two different assignments should get different colors.
	if d1.Color == d2.Color {
		t.Errorf("two agents got same hex %q — registry should assign unique colors", d1.Color)
	}
	if d1.Name == d2.Name {
		t.Errorf("two agents got same name %q — registry should assign unique names", d1.Name)
	}
}
