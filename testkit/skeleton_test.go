package testkit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dpopsuev/djinn/assignment"
	"github.com/dpopsuev/djinn/budget"
	"github.com/dpopsuev/djinn/discourse"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/substrate"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/vessel"
	"github.com/dpopsuev/djinn/vezir"
	troupeTestkit "github.com/dpopsuev/troupe/testkit"
)

// TestSkeleton_AllStubsCompose proves every new interface connects
// end-to-end using only stubs. Zero real I/O. Forge rule:
// "E2E skeleton before features."
func TestSkeleton_AllStubsCompose(t *testing.T) {
	// 1. Event log (stub)
	eventLog := troupeTestkit.NewStubEventLog()

	// 2. Vessel (stub — agent harness)
	tools := substrate.NewStubSubstrate(nil, eventLog).Tools()
	v := vessel.NewStubVessel(tools, eventLog, t.TempDir())

	// 3. Verify Vessel provides what agents need
	if v.Tools() == nil {
		// nil tools is OK for skeleton — stub substrate returns nil
		t.Log("Tools() is nil — expected for stub, real impl provides tools")
	}
	if v.EventLog() == nil {
		t.Fatal("EventLog() should not be nil")
	}
	if v.WorkDir() == "" {
		t.Fatal("WorkDir() should not be empty")
	}

	// 4. ScriptedChatDriver (stub — LLM)
	drv := stubs.NewScriptedChatDriver(
		stubs.ScriptedTurn{Text: "I'll help", ToolCalls: []driver.ToolCall{
			{ID: "c1", Name: "Write", Input: json.RawMessage(`{"path":"test.go","content":"package main"}`)},
		}},
		stubs.ScriptedTurn{Text: "done"},
	)
	if err := drv.Start(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	// 5. Vezir (stub — control plane)
	vz := vezir.NewStubVezir()
	health := vz.Health()
	if !health.Substrate.Running {
		t.Fatal("Vezir stub should report Substrate running")
	}

	// 6. Budget (stub — observer + controller)
	budgetObs := &budget.StubObserver{UsageVal: 0.5}
	budgetCtl := &budget.StubController{ThrottleVal: false}
	if budgetObs.Exceeded() {
		t.Fatal("Budget should not be exceeded at 50%")
	}
	if budgetCtl.ShouldThrottle() {
		t.Fatal("Budget should not throttle")
	}

	// 7. Discourse (stub — planning)
	forum := discourse.NewStubForum()
	threadID, err := forum.Post("refactor", "should we split the driver?")
	if err != nil {
		t.Fatal(err)
	}
	if err := forum.Reply(threadID, "yes, use TroupeChatDriver"); err != nil {
		t.Fatal(err)
	}
	threads := forum.Threads("refactor")
	if len(threads) != 1 || len(threads[0].Messages) != 2 {
		t.Fatalf("Discourse: threads=%d, messages=%d", len(threads), len(threads[0].Messages))
	}

	// 8. Assignment (stub — execution)
	mgr := assignment.NewStubManager()
	assignID, err := mgr.Assign("executor-1", "write main.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Update(assignID, assignment.InProgress, ""); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Update(assignID, assignment.Done, "wrote 42 bytes"); err != nil {
		t.Fatal(err)
	}
	done := mgr.List(assignment.Done)
	if len(done) != 1 {
		t.Fatalf("Assignments done=%d, want 1", len(done))
	}

	// 9. Close vessel
	if err := v.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !v.Closed {
		t.Fatal("Vessel should be closed")
	}

	t.Log("Skeleton PASSES — all 7 interfaces compose with stubs")
}
