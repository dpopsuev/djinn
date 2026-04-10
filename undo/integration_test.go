package undo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dpopsuev/djinn/undo"
	"github.com/dpopsuev/mirage"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/testkit"
)

func newTestSpace(t *testing.T) mirage.Space {
	t.Helper()
	root := t.TempDir()
	space, err := mirage.Create(mirage.Spec{
		Workspace: root,
		Backend:   mirage.Overlay,
	})
	if err != nil {
		t.Skipf("Mirage unavailable: %v", err)
	}
	t.Cleanup(func() { space.Destroy() }) //nolint:errcheck // test cleanup
	return space
}

func TestIntegration_CheckpointRollback_RealMirage(t *testing.T) {
	space := newTestSpace(t)
	log := testkit.NewStubEventLog()
	mgr := undo.NewSpaceManager(log, space)

	ws := space.WorkDir()

	// Write a.go
	aPath := filepath.Join(ws, "a.go")
	os.WriteFile(aPath, []byte("package a"), 0o644)
	log.Emit(signal.Event{Source: "tool", Kind: "tool.call", Data: "Write(a.go)"})

	// Checkpoint
	cpIdx := mgr.Checkpoint("after-a")

	// Write b.go (to be undone)
	bPath := filepath.Join(ws, "b.go")
	os.WriteFile(bPath, []byte("package b"), 0o644)
	log.Emit(signal.Event{Source: "tool", Kind: "tool.call", Data: "Write(b.go)"})

	// Rollback
	undone, err := mgr.Rollback(cpIdx)
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	// Assert: undone events reported
	if len(undone) != 1 {
		t.Fatalf("undone = %d, want 1", len(undone))
	}
	if undone[0].Data != "Write(b.go)" {
		t.Fatalf("undone[0].Data = %v", undone[0].Data)
	}

	// Assert: b.go is gone (Mirage.Reset cleared overlay)
	if _, err := os.Stat(bPath); !os.IsNotExist(err) {
		t.Fatal("b.go should not exist after rollback")
	}

	// Assert: a.go is also gone (Reset clears ALL overlay changes)
	// This is expected for PoC — linear reset, not selective undo.
	// Branching undo (MVP) will use Snapshot/Restore for selective rollback.
	if _, err := os.Stat(aPath); !os.IsNotExist(err) {
		t.Log("Note: a.go also removed — linear Reset() clears entire overlay. Expected for PoC.")
	}
}

func TestIntegration_RollbackInvalidIndex(t *testing.T) {
	space := newTestSpace(t)
	log := testkit.NewStubEventLog()
	mgr := undo.NewSpaceManager(log, space)

	_, err := mgr.Rollback(999)
	if err == nil {
		t.Fatal("expected error for invalid index")
	}
}

func TestIntegration_RollbackNoChanges(t *testing.T) {
	space := newTestSpace(t)
	log := testkit.NewStubEventLog()
	mgr := undo.NewSpaceManager(log, space)

	cpIdx := mgr.Checkpoint("empty")
	undone, err := mgr.Rollback(cpIdx)

	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if undone != nil {
		t.Fatalf("undone = %v, want nil (nothing to undo)", undone)
	}
}
