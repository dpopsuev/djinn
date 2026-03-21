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

// OrchestratorFactory creates an Orchestrator for contract testing.
// Any Orchestrator implementation must pass these tests.
type OrchestratorFactory func(
	createSandbox func(ctx context.Context, scope tier.Scope) (string, error),
	destroySandbox func(ctx context.Context, id string) error,
	driverFactory func(driver.DriverConfig) driver.Driver,
	gateFactory func(gate.GateConfig) gate.Gate,
	signalEmit func(signal.Signal),
) Orchestrator

func simpleFactory(
	createSandbox func(ctx context.Context, scope tier.Scope) (string, error),
	destroySandbox func(ctx context.Context, id string) error,
	driverFactory func(driver.DriverConfig) driver.Driver,
	gateFactory func(gate.GateConfig) gate.Gate,
	signalEmit func(signal.Signal),
) Orchestrator {
	return NewSimpleOrchestrator(createSandbox, destroySandbox, driverFactory, gateFactory, signalEmit)
}

func runContractTests(t *testing.T, factory OrchestratorFactory) {
	t.Run("HappyPath", func(t *testing.T) {
		contractHappyPath(t, factory)
	})
	t.Run("GateFailureStops", func(t *testing.T) {
		contractGateFailureStops(t, factory)
	})
	t.Run("CancelTerminates", func(t *testing.T) {
		contractCancelTerminates(t, factory)
	})
	t.Run("EventsInOrder", func(t *testing.T) {
		contractEventsInOrder(t, factory)
	})
	t.Run("ChannelClosed", func(t *testing.T) {
		contractChannelClosed(t, factory)
	})
}

func TestSimpleOrchestrator_Contract(t *testing.T) {
	runContractTests(t, simpleFactory)
}

func contractHappyPath(t *testing.T, factory OrchestratorFactory) {
	orch := factory(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			return newStubDriver(driver.Message{Role: "assistant", Content: "done"})
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID: "contract-happy",
		Stages: []Stage{
			{Name: "s1", Scope: tier.Scope{Level: tier.Mod}, Prompt: "p1"},
			{Name: "s2", Scope: tier.Scope{Level: tier.Com}, Prompt: "p2"},
		},
	}

	ch, err := orch.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	last := events[len(events)-1]
	if last.Kind != ExecutionDone || last.Message != "success" {
		t.Fatalf("last event = %v/%q, want ExecutionDone/success", last.Kind, last.Message)
	}
}

func contractGateFailureStops(t *testing.T, factory OrchestratorFactory) {
	stageCount := 0
	orch := factory(
		func(ctx context.Context, scope tier.Scope) (string, error) {
			stageCount++
			return "sb", nil
		},
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			return newStubDriver(driver.Message{Role: "assistant", Content: "done"})
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{err: errors.New("fail")} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID: "contract-gate-fail",
		Stages: []Stage{
			{Name: "s1", Scope: tier.Scope{Level: tier.Mod}, Prompt: "p1"},
			{Name: "s2", Scope: tier.Scope{Level: tier.Com}, Prompt: "p2"},
		},
	}

	ch, _ := orch.Execute(context.Background(), plan)
	for range ch {
	}

	if stageCount != 1 {
		t.Fatalf("stages executed = %d, want 1 (gate should stop at first)", stageCount)
	}
}

func contractCancelTerminates(t *testing.T, factory OrchestratorFactory) {
	blockCh := make(chan struct{})
	orch := factory(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			ch := make(chan driver.Message)
			go func() { <-blockCh; close(ch) }()
			return &stubDriver{recvCh: ch}
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID:     "contract-cancel",
		Stages: []Stage{{Name: "block", Scope: tier.Scope{Level: tier.Mod}, Prompt: "wait"}},
	}

	ch, _ := orch.Execute(context.Background(), plan)
	time.Sleep(10 * time.Millisecond)

	if err := orch.Cancel(context.Background(), "contract-cancel"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	close(blockCh)

	var events []Event
	for e := range ch {
		events = append(events, e)
	}
	if len(events) == 0 {
		t.Fatal("expected events after cancel")
	}
}

func contractEventsInOrder(t *testing.T, factory OrchestratorFactory) {
	orch := factory(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			return newStubDriver(driver.Message{Role: "assistant", Content: "done"})
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID:     "contract-order",
		Stages: []Stage{{Name: "s1", Scope: tier.Scope{Level: tier.Mod}, Prompt: "p1"}},
	}

	ch, _ := orch.Execute(context.Background(), plan)
	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	// Must start with StageStarted and end with ExecutionDone
	if events[0].Kind != StageStarted {
		t.Fatalf("first event = %v, want StageStarted", events[0].Kind)
	}
	if events[len(events)-1].Kind != ExecutionDone {
		t.Fatalf("last event = %v, want ExecutionDone", events[len(events)-1].Kind)
	}
}

func contractChannelClosed(t *testing.T, factory OrchestratorFactory) {
	orch := factory(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver {
			return newStubDriver(driver.Message{Role: "assistant", Content: "done"})
		},
		func(cfg gate.GateConfig) gate.Gate { return &stubGate{} },
		func(s signal.Signal) {},
	)

	plan := WorkPlan{
		ID:     "contract-close",
		Stages: []Stage{{Name: "s1", Scope: tier.Scope{Level: tier.Mod}, Prompt: "p1"}},
	}

	ch, _ := orch.Execute(context.Background(), plan)

	// Drain all events
	for range ch {
	}

	// Channel must be closed — reading again should return zero value immediately
	_, ok := <-ch
	if ok {
		t.Fatal("channel should be closed after execution")
	}
}
