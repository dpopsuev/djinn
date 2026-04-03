package agent

import (
	"testing"
)

func TestWorkstationManager_CreateAndGet(t *testing.T) {
	mgr := NewWorkstationManager(nil)

	ws := mgr.Create("T-001")
	if ws == nil {
		t.Fatal("Create returned nil")
	}
	if ws.TaskID != "T-001" {
		t.Errorf("TaskID = %q, want T-001", ws.TaskID)
	}
	if ws.ID == "" {
		t.Error("ID should be auto-generated")
	}

	// Get.
	got, ok := mgr.Get(ws.ID)
	if !ok {
		t.Fatal("Get should find the workstation")
	}
	if got != ws {
		t.Error("Get should return the same workstation")
	}

	// Get non-existent.
	_, ok = mgr.Get("WS-999")
	if ok {
		t.Error("Get should return false for non-existent workstation")
	}
}

func TestWorkstationManager_Release(t *testing.T) {
	mgr := NewWorkstationManager(nil)
	ws := mgr.Create("T-001")

	mgr.Release(ws.ID)

	_, ok := mgr.Get(ws.ID)
	if ok {
		t.Error("Get should return false after Release")
	}

	// Release non-existent should not panic.
	mgr.Release("WS-999")
}

func TestWorkstationManager_Assign(t *testing.T) {
	mgr := NewWorkstationManager(nil)
	ws := mgr.Create("T-001")

	if err := mgr.Assign(ws.ID, "agent-1"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if ws.Agent != "agent-1" {
		t.Errorf("Agent = %q, want agent-1", ws.Agent)
	}

	// Assign to occupied workstation should fail.
	if err := mgr.Assign(ws.ID, "agent-2"); err == nil {
		t.Error("Assign to occupied workstation should fail")
	}

	// Assign to non-existent workstation should fail.
	if err := mgr.Assign("WS-999", "agent-3"); err == nil {
		t.Error("Assign to non-existent workstation should fail")
	}
}

func TestWorkstationManager_List(t *testing.T) {
	mgr := NewWorkstationManager(nil)

	if got := mgr.List(); len(got) != 0 {
		t.Errorf("List should be empty, got %d", len(got))
	}

	mgr.Create("T-001")
	mgr.Create("T-002")

	if got := mgr.List(); len(got) != 2 { //nolint:mnd // expected 2
		t.Errorf("List count = %d, want 2", len(got))
	}
}

func TestWorkstationManager_AutoIncrementIDs(t *testing.T) {
	mgr := NewWorkstationManager(nil)

	ws1 := mgr.Create("T-001")
	ws2 := mgr.Create("T-002")

	if ws1.ID == ws2.ID {
		t.Error("workstations should have different IDs")
	}
}
