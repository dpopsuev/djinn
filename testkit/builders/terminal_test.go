package builders

import "testing"

func TestDjinnBuilder_Defaults(t *testing.T) {
	d := NewDjinnBuilder().Build()
	s := d.Status()

	if s.Operation != "agent" {
		t.Errorf("default Operation = %q, want %q", s.Operation, "agent")
	}
	if s.AgentCap != 1 {
		t.Errorf("default AgentCap = %d, want 1", s.AgentCap)
	}
}

func TestDjinnBuilder_WithOperation(t *testing.T) {
	d := NewDjinnBuilder().WithOperation("ask").Build()
	s := d.Status()

	if s.Operation != "ask" {
		t.Errorf("Operation = %q, want %q", s.Operation, "ask")
	}
}

func TestDjinnBuilder_WithCapacity(t *testing.T) {
	d := NewDjinnBuilder().WithCapacity(5).Build()
	s := d.Status()

	if s.AgentCap != 5 {
		t.Errorf("AgentCap = %d, want 5", s.AgentCap)
	}
}

func TestDjinnBuilder_Chaining(t *testing.T) {
	d := NewDjinnBuilder().
		WithOperation("plan").
		WithCapacity(3).
		Build()

	s := d.Status()
	if s.Operation != "plan" {
		t.Errorf("Operation = %q, want %q", s.Operation, "plan")
	}
	if s.AgentCap != 3 {
		t.Errorf("AgentCap = %d, want 3", s.AgentCap)
	}
}
