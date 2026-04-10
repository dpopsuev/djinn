package undo_test

import (
	"testing"

	"github.com/dpopsuev/djinn/undo"
	"github.com/dpopsuev/troupe/signal"
	"github.com/dpopsuev/troupe/testkit"
)

// TestSkeleton_CheckpointRollback_AllStubs proves the interfaces compose
// end-to-end using only stubs. Zero real I/O — Forge rule:
// "E2E skeleton before features."
func TestSkeleton_CheckpointRollback_AllStubs(t *testing.T) {
	// 1. Create event log (stub)
	log := testkit.NewStubEventLog()

	// 2. Create undo manager (stub, backed by event log)
	mgr := undo.NewStubManager(log)

	// 3. Simulate agent work: turn + tool call
	log.Emit(signal.Event{Source: "agent", Kind: "turn.start"})
	log.Emit(signal.Event{Source: "tool", Kind: "tool.call", Data: "Write(a.go)"})

	// 4. Checkpoint
	cpIdx := mgr.Checkpoint("after-a.go")
	if cpIdx != 2 {
		t.Fatalf("checkpoint index = %d, want 2", cpIdx)
	}

	// 5. Simulate more agent work (to be undone)
	log.Emit(signal.Event{Source: "tool", Kind: "tool.call", Data: "Write(b.go)"})
	log.Emit(signal.Event{Source: "agent", Kind: "turn.done"})

	// 6. Rollback
	undone, err := mgr.Rollback(cpIdx)
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	// 7. Assert: 2 events undone (Write(b.go) + turn.done)
	if len(undone) != 2 {
		t.Fatalf("undone = %d, want 2", len(undone))
	}
	if undone[0].Data != "Write(b.go)" {
		t.Fatalf("undone[0].Data = %v, want Write(b.go)", undone[0].Data)
	}

	// 8. Assert: manager state updated
	if mgr.Current() != cpIdx {
		t.Fatalf("Current = %d, want %d", mgr.Current(), cpIdx)
	}

	// 9. Assert: call log recorded
	if len(mgr.CheckpointCalls) != 1 || mgr.CheckpointCalls[0] != "after-a.go" {
		t.Fatalf("CheckpointCalls = %v", mgr.CheckpointCalls)
	}
	if len(mgr.RollbackCalls) != 1 || mgr.RollbackCalls[0] != cpIdx {
		t.Fatalf("RollbackCalls = %v", mgr.RollbackCalls)
	}
}

// TestSkeleton_MultipleCheckpoints_RollbackToEarlier proves rollback
// can target any checkpoint, not just the most recent.
func TestSkeleton_MultipleCheckpoints_RollbackToEarlier(t *testing.T) {
	log := testkit.NewStubEventLog()
	mgr := undo.NewStubManager(log)

	log.Emit(signal.Event{Kind: "a"})
	cp1 := mgr.Checkpoint("cp1")

	log.Emit(signal.Event{Kind: "b"})
	log.Emit(signal.Event{Kind: "c"})
	mgr.Checkpoint("cp2")

	log.Emit(signal.Event{Kind: "d"})

	// Rollback to cp1 — should undo b, c, d
	undone, err := mgr.Rollback(cp1)
	if err != nil {
		t.Fatal(err)
	}
	if len(undone) != 3 {
		t.Fatalf("undone = %d, want 3 (b, c, d)", len(undone))
	}
}
