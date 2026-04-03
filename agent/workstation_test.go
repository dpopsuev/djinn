package agent

import (
	"testing"
)

func TestWorkstation_NewAndVacant(t *testing.T) {
	ws := NewWorkstation("WS-001", "T-001")

	if ws.ID != "WS-001" {
		t.Errorf("ID = %q, want WS-001", ws.ID)
	}
	if ws.TaskID != "T-001" {
		t.Errorf("TaskID = %q, want T-001", ws.TaskID)
	}
	if !ws.IsVacant() {
		t.Error("new workstation should be vacant")
	}
	if ws.Created.IsZero() {
		t.Error("Created should be set")
	}
}

func TestWorkstation_AttachDetach(t *testing.T) {
	ws := NewWorkstation("WS-001", "T-001")

	// Attach.
	if err := ws.Attach("agent-1"); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	if ws.IsVacant() {
		t.Error("workstation should not be vacant after attach")
	}
	if ws.Agent != "agent-1" {
		t.Errorf("Agent = %q, want agent-1", ws.Agent)
	}

	// Detach.
	former := ws.Detach()
	if former != "agent-1" {
		t.Errorf("Detach returned %q, want agent-1", former)
	}
	if !ws.IsVacant() {
		t.Error("workstation should be vacant after detach")
	}
}

func TestWorkstation_AttachWhenOccupied_Fails(t *testing.T) {
	ws := NewWorkstation("WS-001", "T-001")

	if err := ws.Attach("agent-1"); err != nil {
		t.Fatalf("first Attach: %v", err)
	}

	err := ws.Attach("agent-2")
	if err == nil {
		t.Fatal("Attach should fail when occupied")
	}

	if ws.Agent != "agent-1" {
		t.Errorf("Agent should still be agent-1, got %q", ws.Agent)
	}
}

func TestWorkstation_DetachWhenVacant(t *testing.T) {
	ws := NewWorkstation("WS-001", "T-001")

	former := ws.Detach()
	if former != "" {
		t.Errorf("Detach on vacant should return empty, got %q", former)
	}
}
