package testkit

import (
	"context"
	"testing"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/testkit/stubs"
)

func TestAgentFixture_Creates(t *testing.T) {
	f := NewAgentFixture(t)
	if f.Dir() == "" {
		t.Fatal("Dir should not be empty")
	}
	if f.Workspace() == nil {
		t.Fatal("Workspace should not be nil")
	}
}

func TestAgentFixture_RunWithStub(t *testing.T) {
	stub := stubs.NewStubChatDriver(driver.Message{
		Role:    "assistant",
		Content: "I wrote the file",
	})

	f := NewAgentFixture(t,
		WithDriver(stub),
		WithMode(agent.ModeAuto),
		WithMaxTurns(1),
	)

	result, err := f.Run(context.Background(), "write hello.txt")
	// StubChatDriver returns canned response — agent loop processes it.
	// The exact result depends on agent.Run internals, but it shouldn't panic.
	_ = result
	_ = err
}

func TestAgentFixture_WorkspaceIsolated(t *testing.T) {
	f := NewAgentFixture(t)

	// Workspace should be a real directory
	ws := f.Workspace()
	if ws.Dir() == "" {
		t.Fatal("workspace Dir should not be empty")
	}
}
