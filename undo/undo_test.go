package undo_test

import (
	"testing"

	"github.com/dpopsuev/djinn/undo"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/testkit"
)

func TestCheckpoint_Create(t *testing.T) {
	log := testkit.NewStubEventLog()
	log.Emit(signal.Event{Kind: "turn.start"})
	log.Emit(signal.Event{Kind: "tool.call"})

	mgr := undo.NewStubManager(log)
	idx := mgr.Checkpoint("before-refactor")

	if idx != 2 {
		t.Fatalf("checkpoint index = %d, want 2", idx)
	}

	cps := mgr.Checkpoints()
	if len(cps) != 1 {
		t.Fatalf("checkpoints = %d, want 1", len(cps))
	}
	if cps[0].Name != "before-refactor" {
		t.Fatalf("name = %q", cps[0].Name)
	}
	if cps[0].Index != 2 {
		t.Fatalf("index = %d", cps[0].Index)
	}
}

func TestRollback_ReturnsUndoneEvents(t *testing.T) {
	log := testkit.NewStubEventLog()
	log.Emit(signal.Event{Kind: "turn.start"})
	log.Emit(signal.Event{Kind: "tool.call", Data: "write main.go"})

	mgr := undo.NewStubManager(log)
	cpIdx := mgr.Checkpoint("safe")

	// Agent does more work after checkpoint
	log.Emit(signal.Event{Kind: "tool.call", Data: "write bad.go"})
	log.Emit(signal.Event{Kind: "tool.call", Data: "write worse.go"})

	undone, err := mgr.Rollback(cpIdx)
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if len(undone) != 2 {
		t.Fatalf("undone = %d, want 2", len(undone))
	}
	if undone[0].Data != "write bad.go" {
		t.Fatalf("undone[0] = %v", undone[0].Data)
	}
}

func TestRollback_UpdatesCurrent(t *testing.T) {
	log := testkit.NewStubEventLog()
	mgr := undo.NewStubManager(log)

	if mgr.Current() != -1 {
		t.Fatalf("Current before checkpoint = %d, want -1", mgr.Current())
	}

	log.Emit(signal.Event{Kind: "a"})
	cpIdx := mgr.Checkpoint("cp1")

	if mgr.Current() != cpIdx {
		t.Fatalf("Current after checkpoint = %d, want %d", mgr.Current(), cpIdx)
	}

	log.Emit(signal.Event{Kind: "b"})
	log.Emit(signal.Event{Kind: "c"})
	mgr.Rollback(cpIdx) //nolint:errcheck // test

	if mgr.Current() != cpIdx {
		t.Fatalf("Current after rollback = %d, want %d", mgr.Current(), cpIdx)
	}
}

func TestCheckpoint_Multiple(t *testing.T) {
	log := testkit.NewStubEventLog()
	mgr := undo.NewStubManager(log)

	log.Emit(signal.Event{Kind: "a"})
	mgr.Checkpoint("cp1")

	log.Emit(signal.Event{Kind: "b"})
	log.Emit(signal.Event{Kind: "c"})
	mgr.Checkpoint("cp2")

	cps := mgr.Checkpoints()
	if len(cps) != 2 {
		t.Fatalf("checkpoints = %d, want 2", len(cps))
	}
	if cps[0].Index != 1 {
		t.Fatalf("cp1.Index = %d, want 1", cps[0].Index)
	}
	if cps[1].Index != 3 {
		t.Fatalf("cp2.Index = %d, want 3", cps[1].Index)
	}
}

func TestStubManager_RecordsCallLog(t *testing.T) {
	log := testkit.NewStubEventLog()
	mgr := undo.NewStubManager(log)

	mgr.Checkpoint("a")
	mgr.Checkpoint("b")
	mgr.Rollback(0) //nolint:errcheck // test

	if len(mgr.CheckpointCalls) != 2 {
		t.Fatalf("CheckpointCalls = %d", len(mgr.CheckpointCalls))
	}
	if len(mgr.RollbackCalls) != 1 {
		t.Fatalf("RollbackCalls = %d", len(mgr.RollbackCalls))
	}
}
