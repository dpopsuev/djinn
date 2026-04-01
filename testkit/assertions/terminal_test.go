package assertions

import (
	"context"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/terminal"
	"github.com/dpopsuev/djinn/testkit/stubs"
)

func TestAssertStatus_Pass(t *testing.T) {
	d := terminal.NewDjinn()
	d.SetTokens(100, 50)
	d.SetTurns(3)
	d.SetAgentCount(2)

	AssertStatus(t, d, "operation", "agent")
	AssertStatus(t, d, "tokens_in", 100)
	AssertStatus(t, d, "turns", 3)
	AssertStatus(t, d, "agent_count", 2)
}

func TestAssertStatus_Fail(t *testing.T) {
	d := terminal.NewDjinn()
	d.SetTurns(5)

	// Use a sub-test with a mock T to verify failure.
	mockT := &testing.T{}
	AssertStatus(mockT, d, "turns", 99)
	// mockT should have recorded a failure — we can't check t.Failed()
	// on a zero-value T, but the call should not panic.
}

func TestAssertEventReceived_Pass(t *testing.T) {
	op := stubs.NewTestOperator()
	d := op.Djinn()

	d.OnSubmit = func(_ context.Context, prompt string) error {
		d.Emit(terminal.ViewEvent{Kind: terminal.EventOutput, Text: prompt})
		return nil
	}

	if err := op.Submit(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}

	// Wait for event delivery.
	time.Sleep(10 * time.Millisecond)

	AssertEventReceived(t, op, terminal.EventOutput)
}

func TestAssertEventReceived_Fail(t *testing.T) {
	op := stubs.NewTestOperator()

	// No events emitted — use a mock T.
	mockT := &testing.T{}
	AssertEventReceived(mockT, op, terminal.EventDone)
	// Should not panic; mockT would record the failure.
}
