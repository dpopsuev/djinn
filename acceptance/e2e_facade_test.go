package acceptance

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/bugleport"
)

// TestE2E_FacadeLifecycle is the canonical example for Bugle's facade API.
// It exercises Staff + AgentHandle for spawning, messaging, state inspection,
// lifecycle management, and clean shutdown — the happy path that every
// Djinn consumer should follow.
func TestE2E_FacadeLifecycle(t *testing.T) {
	ctx := context.Background()

	// 1. Create staff with mock launcher.
	staff := bugleport.NewStaff(newMockLauncher())

	// 2. Spawn GenSec as root agent.
	gensec, err := staff.Spawn(ctx, "gensec", bugleport.LaunchConfig{
		Role:  "gensec",
		Model: "haiku",
	})
	if err != nil {
		t.Fatalf("Spawn gensec: %v", err)
	}
	if gensec.ID() == 0 {
		t.Fatal("gensec ID should not be 0")
	}
	if gensec.Role() != "gensec" {
		t.Fatalf("gensec role = %q, want gensec", gensec.Role())
	}
	if !gensec.IsAlive() {
		t.Fatal("gensec should be alive after spawn")
	}

	// 3. Spawn Executor under GenSec (parent-child relationship).
	executor, err := gensec.Spawn(ctx, "executor", bugleport.LaunchConfig{
		Role:  "executor",
		Model: "opus",
	})
	if err != nil {
		t.Fatalf("Spawn executor: %v", err)
	}
	if executor.Role() != "executor" {
		t.Fatalf("executor role = %q, want executor", executor.Role())
	}

	// Verify parent-child relationship.
	children := gensec.Children()
	if len(children) != 1 {
		t.Fatalf("gensec children = %d, want 1", len(children))
	}
	if children[0].ID() != executor.ID() {
		t.Fatalf("child ID = %d, want executor %d", children[0].ID(), executor.ID())
	}
	parent := executor.Parent()
	if parent == nil {
		t.Fatal("executor parent should not be nil")
	}
	if parent.ID() != gensec.ID() {
		t.Fatalf("executor parent = %d, want gensec %d", parent.ID(), gensec.ID())
	}

	// 4. Executor listens for messages.
	executor.Listen(func(content string) string {
		return "done: " + content
	})

	// 5. GenSec asks Executor via Ask (request-response).
	response, err := executor.Ask(ctx, "review auth module")
	if err != nil {
		t.Fatalf("Ask executor: %v", err)
	}
	if response != "done: review auth module" {
		t.Fatalf("response = %q, want %q", response, "done: review auth module")
	}

	// 6. Check state.
	if !executor.IsAlive() {
		t.Fatal("executor should still be alive")
	}
	if executor.Role() != "executor" {
		t.Fatalf("executor role = %q, want executor", executor.Role())
	}
	if staff.Count() != 2 {
		t.Fatalf("staff count = %d, want 2", staff.Count())
	}

	// 7. Kill executor and wait for exit.
	if err := executor.Kill(ctx); err != nil {
		t.Fatalf("Kill executor: %v", err)
	}
	status, err := executor.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait executor: %v", err)
	}
	if status.Code != bugleport.ExitSuccess {
		t.Fatalf("executor exit code = %d, want ExitSuccess (%d)", status.Code, bugleport.ExitSuccess)
	}

	// 8. Clean shutdown — KillAll stops remaining agents.
	staff.KillAll(ctx)
	if staff.Count() != 0 {
		t.Fatalf("staff count = %d after KillAll, want 0", staff.Count())
	}
}
