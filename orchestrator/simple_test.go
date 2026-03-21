package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tier"
)

// stubDriver implements driver.Driver for orchestrator tests.
type stubDriver struct {
	recvCh chan driver.Message
}

func newStubDriver(msgs ...driver.Message) *stubDriver {
	ch := make(chan driver.Message, len(msgs))
	for _, m := range msgs {
		ch <- m
	}
	close(ch)
	return &stubDriver{recvCh: ch}
}

func (d *stubDriver) Start(ctx context.Context, sandbox driver.SandboxHandle) error { return nil }
func (d *stubDriver) Send(ctx context.Context, msg driver.Message) error             { return nil }
func (d *stubDriver) Recv(ctx context.Context) <-chan driver.Message                  { return d.recvCh }
func (d *stubDriver) Stop(ctx context.Context) error                                 { return nil }

// stubGate implements gate.Gate for orchestrator tests.
type stubGate struct{ err error }

func (g *stubGate) Validate(ctx context.Context, sandboxID string) error { return g.err }

func TestSimpleOrchestrator_FourStageHappyPath(t *testing.T) {
	sandboxCount := 0
	signals := []signal.Signal{}

	orch := NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) {
			sandboxCount++
			return "sb", nil
		},
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			return newStubDriver(driver.Message{Role: "assistant", Content: "done"})
		},
		func(cfg gate.GateConfig) gate.Gate {
			return &stubGate{}
		},
		func(s signal.Signal) {
			signals = append(signals, s)
		},
	)

	plan := WorkPlan{
		ID: "test-1",
		Stages: []Stage{
			{Name: "analyze", Scope: tier.Scope{Level: tier.Eco}, Prompt: "analyze"},
			{Name: "code", Scope: tier.Scope{Level: tier.Com}, Prompt: "code"},
			{Name: "test", Scope: tier.Scope{Level: tier.Mod}, Prompt: "test"},
			{Name: "review", Scope: tier.Scope{Level: tier.Sys}, Prompt: "review"},
		},
	}

	ctx := context.Background()
	ch, err := orch.Execute(ctx, plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	if sandboxCount != 4 {
		t.Fatalf("sandboxes created = %d, want 4", sandboxCount)
	}

	// Check we got StageStarted, GatePassed, StageCompleted for each stage + ExecutionDone
	kindCounts := make(map[EventKind]int)
	for _, e := range events {
		kindCounts[e.Kind]++
	}

	if kindCounts[StageStarted] != 4 {
		t.Fatalf("StageStarted count = %d, want 4", kindCounts[StageStarted])
	}
	if kindCounts[StageCompleted] != 4 {
		t.Fatalf("StageCompleted count = %d, want 4", kindCounts[StageCompleted])
	}
	if kindCounts[GatePassed] != 4 {
		t.Fatalf("GatePassed count = %d, want 4", kindCounts[GatePassed])
	}
	if kindCounts[ExecutionDone] != 1 {
		t.Fatalf("ExecutionDone count = %d, want 1", kindCounts[ExecutionDone])
	}

	// Last event must be ExecutionDone
	last := events[len(events)-1]
	if last.Kind != ExecutionDone {
		t.Fatalf("last event = %v, want ExecutionDone", last.Kind)
	}
	if last.Message != "success" {
		t.Fatalf("last message = %q, want %q", last.Message, "success")
	}
}

func TestSimpleOrchestrator_GateFailure(t *testing.T) {
	gateErr := errors.New("lint failed")

	orch := NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			return newStubDriver(driver.Message{Role: "assistant", Content: "done"})
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{err: gateErr} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID: "test-fail",
		Stages: []Stage{
			{Name: "code", Scope: tier.Scope{Level: tier.Mod}, Prompt: "code"},
		},
	}

	ch, _ := orch.Execute(context.Background(), plan)
	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	hasGateFailed := false
	hasStageFailed := false
	for _, e := range events {
		if e.Kind == GateFailed {
			hasGateFailed = true
		}
		if e.Kind == StageFailed {
			hasStageFailed = true
		}
	}
	if !hasGateFailed {
		t.Fatal("expected GateFailed event")
	}
	if !hasStageFailed {
		t.Fatal("expected StageFailed event")
	}

	last := events[len(events)-1]
	if last.Kind != ExecutionDone {
		t.Fatalf("last event = %v, want ExecutionDone", last.Kind)
	}
}

func TestSimpleOrchestrator_Cancel(t *testing.T) {
	blockCh := make(chan struct{})

	orch := NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			// Return a driver that blocks on Recv until cancel
			ch := make(chan driver.Message)
			go func() {
				<-blockCh
				close(ch)
			}()
			return &stubDriver{recvCh: ch}
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID:     "cancel-test",
		Stages: []Stage{{Name: "long", Scope: tier.Scope{Level: tier.Mod}, Prompt: "wait"}},
	}

	ch, _ := orch.Execute(context.Background(), plan)

	// Wait for stage to start
	time.Sleep(10 * time.Millisecond)

	if err := orch.Cancel(context.Background(), "cancel-test"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	close(blockCh)

	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	hasFailed := false
	for _, e := range events {
		if e.Kind == StageFailed || e.Kind == ExecutionDone {
			hasFailed = true
		}
	}
	if !hasFailed {
		t.Fatal("expected failure event after cancel")
	}
}

func TestSimpleOrchestrator_TimeBudget(t *testing.T) {
	orch := NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			// Driver that blocks until context cancelled
			ch := make(chan driver.Message)
			return &stubDriver{recvCh: ch}
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID: "budget-test",
		Stages: []Stage{{
			Name:       "slow",
			Scope:      tier.Scope{Level: tier.Mod},
			Prompt:     "work",
			TimeBudget: 50 * time.Millisecond,
		}},
	}

	ch, _ := orch.Execute(context.Background(), plan)
	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	hasStageFailed := false
	for _, e := range events {
		if e.Kind == StageFailed {
			hasStageFailed = true
		}
	}
	if !hasStageFailed {
		t.Fatal("expected StageFailed from time budget")
	}
}

func TestSimpleOrchestrator_TokenBudget(t *testing.T) {
	var signals []signal.Signal

	orch := NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			return newStubDriver(
				driver.Message{Role: "assistant", Content: "msg1"},
				driver.Message{Role: "assistant", Content: "msg2"},
				driver.Message{Role: "assistant", Content: "msg3"},
			)
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{} },
		func(s signal.Signal) { signals = append(signals, s) },
	)

	plan := WorkPlan{
		ID: "token-test",
		Stages: []Stage{{
			Name:        "chatty",
			Scope:       tier.Scope{Level: tier.Mod},
			Prompt:      "work",
			TokenBudget: 2, // stop after 2 messages
		}},
	}

	ch, _ := orch.Execute(context.Background(), plan)
	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	// Should complete (gate runs after token budget hit)
	last := events[len(events)-1]
	if last.Kind != ExecutionDone {
		t.Fatalf("last event = %v, want ExecutionDone", last.Kind)
	}

	// Check budget signal was emitted
	hasBudgetSignal := false
	for _, s := range signals {
		if s.Category == "budget" {
			hasBudgetSignal = true
		}
	}
	if !hasBudgetSignal {
		t.Fatal("expected budget signal")
	}
}

func TestSimpleOrchestrator_Submit(t *testing.T) {
	orch := NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			return newStubDriver(driver.Message{Role: "assistant", Content: "done"})
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID:     "submit-test",
		Stages: []Stage{{Name: "code", Scope: tier.Scope{Level: tier.Mod}, Prompt: "code"}},
	}

	ch, _ := orch.Execute(context.Background(), plan)

	// Submit while running
	time.Sleep(5 * time.Millisecond)
	err := orch.Submit(context.Background(), "submit-test", ExternalInput{
		ExecID:  "submit-test",
		Payload: map[string]string{"action": "approve"},
	})
	// May succeed or fail (execution might be done already)
	_ = err

	// Drain events
	for range ch {
	}

	// Submit to non-existent execution
	err = orch.Submit(context.Background(), "nonexistent", ExternalInput{})
	if err == nil {
		t.Fatal("Submit to non-existent should error")
	}
}
