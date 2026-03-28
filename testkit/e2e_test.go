package testkit

import (
	"context"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/testkit/assertions"
	"github.com/dpopsuev/djinn/testkit/builders"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tier"
)

func TestE2E_StandardFlow_AllStubs(t *testing.T) {
	// Wire all stubs
	sandbox := stubs.NewStubSandbox()
	bus := signal.NewSignalBus()
	cordons := broker.NewCordonRegistry()

	stubDriverInstance := stubs.NewStubDriver(
		driver.Message{Role: "assistant", Content: "done"},
	)

	orch := orchestrator.NewSimpleOrchestrator(
		sandbox.Create,
		sandbox.Destroy,
		func(cfg driver.DriverConfig) driver.Driver {
			// Return a fresh stub driver for each stage (closed channel)
			d := stubs.NewStubDriver(driver.Message{Role: "assistant", Content: "done"})
			_ = stubDriverInstance // reference kept for verification
			return d
		},
		func(cfg gate.GateConfig) gate.Gate {
			return stubs.AlwaysPassGate()
		},
		func(s signal.Signal) { bus.Emit(s) },
	)

	op := stubs.NewStubOperatorPort()

	b := broker.NewBroker(&broker.BrokerConfig{
		Orchestrator: orch,
		Bus:          bus,
		Cordons:      cordons,
		Operator:     op,
		Sandbox:      sandbox,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return builders.StandardFourTierPlan(intent.ID)
		},
	})

	ctx := context.Background()
	b.Start(ctx)

	// Send intent
	op.SendIntent(ari.Intent{ID: "e2e-1", Action: "implement"})

	// Wait for result
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(op.Results()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	results := op.Results()
	if len(results) == 0 {
		t.Fatal("no results received")
	}

	if !results[0].Success {
		t.Fatalf("result.Success = false, Summary = %q", results[0].Summary)
	}

	// Assert event order
	events := op.Events()
	assertions.AssertEventOrder(t, events, []orchestrator.EventKind{
		orchestrator.StageStarted,
		orchestrator.StageCompleted,
		orchestrator.StageStarted,
		orchestrator.StageCompleted,
		orchestrator.StageStarted,
		orchestrator.StageCompleted,
		orchestrator.StageStarted,
		orchestrator.StageCompleted,
		orchestrator.ExecutionDone,
	})

	// Assert Andon is green
	andons := op.Andons()
	if len(andons) == 0 {
		t.Fatal("no andon boards emitted")
	}
	if andons[0].Level != signal.Green {
		t.Fatalf("andon level = %v, want Green", andons[0].Level)
	}

	// Assert 4 sandboxes created and destroyed
	created := sandbox.Created()
	destroyed := sandbox.Destroyed()
	if len(created) != 4 {
		t.Fatalf("sandboxes created = %d, want 4", len(created))
	}
	if len(destroyed) != 4 {
		t.Fatalf("sandboxes destroyed = %d, want 4", len(destroyed))
	}

	// Assert no cordons
	assertions.AssertCordonCount(t, cordons, 0)
}

func TestE2E_GateFailure_StopsExecution(t *testing.T) {
	sandbox := stubs.NewStubSandbox()
	bus := signal.NewSignalBus()
	cordons := broker.NewCordonRegistry()

	orch := orchestrator.NewSimpleOrchestrator(
		sandbox.Create,
		sandbox.Destroy,
		func(cfg driver.DriverConfig) driver.Driver {
			return stubs.NewStubDriver(driver.Message{Role: "assistant", Content: "done"})
		},
		func(cfg gate.GateConfig) gate.Gate {
			return stubs.AlwaysFailGate("quality too low")
		},
		func(s signal.Signal) { bus.Emit(s) },
	)

	op := stubs.NewStubOperatorPort()

	b := broker.NewBroker(&broker.BrokerConfig{
		Orchestrator: orch,
		Bus:          bus,
		Cordons:      cordons,
		Operator:     op,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return builders.NewWorkPlan(intent.ID).
				AddStage("code", tier.Scope{Level: tier.Mod, Name: "auth"}, "code it").
				AddStage("test", tier.Scope{Level: tier.Mod, Name: "tests"}, "test it").
				Build()
		},
	})

	ctx := context.Background()
	b.Start(ctx)

	op.SendIntent(ari.Intent{ID: "fail-1", Action: "fix"})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(op.Results()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	results := op.Results()
	if len(results) == 0 {
		t.Fatal("no results received")
	}
	if results[0].Success {
		t.Fatal("expected failure result")
	}

	// Only 1 sandbox should have been created (failed at first gate)
	if len(sandbox.Created()) != 1 {
		t.Fatalf("sandboxes created = %d, want 1 (gate stops at first stage)", len(sandbox.Created()))
	}
}
