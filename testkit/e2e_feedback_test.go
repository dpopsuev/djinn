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
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tier"
)

func TestE2E_FeedbackLoop_AlertToFixToRecovery(t *testing.T) {
	// Setup: FeedbackStub with threshold=5.0, initial error_rate=0.3
	feedback := stubs.NewFeedbackStub(5.0, map[string]float64{"error_rate": 0.3})
	sandbox := stubs.NewStubSandbox()
	bus := signal.NewSignalBus()
	cordons := broker.NewCordonRegistry()

	orch := orchestrator.NewSimpleOrchestrator(
		sandbox.Create,
		sandbox.Destroy,
		func(cfg driver.DriverConfig) driver.Driver {
			return stubs.NewStubDriver(driver.Message{Role: "assistant", Content: "fixed"})
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
		Alerts:       feedback, // EventIngressPort (driving)
		Metrics:      feedback, // MetricsPort (driven)
		Sandbox:      sandbox,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return orchestrator.WorkPlan{
				ID: intent.ID,
				Stages: []orchestrator.Stage{
					{Name: "fix", Scope: tier.Scope{Level: tier.Mod, Name: "hotfix"}, Prompt: "fix it"},
				},
			}
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Start(ctx)

	// Step 1: Verify healthy state
	if rate := feedback.Query("error_rate"); rate != 0.3 {
		t.Fatalf("initial error_rate = %f, want 0.3", rate)
	}
	board := b.Andon()
	if board.Level != signal.Green {
		t.Fatalf("initial Andon = %v, want Green", board.Level)
	}

	// Step 2: Breach threshold — alert fires
	feedback.SetMetric("error_rate", 7.2)

	// Wait for the autonomous fix to complete
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(op.Results()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	results := op.Results()
	if len(results) == 0 {
		t.Fatal("no results from autonomous fix")
	}

	// The auto-fix intent should have succeeded (stubs pass everything)
	autoResult := results[0]
	if !autoResult.Success {
		t.Fatalf("auto-fix result.Success = false, Summary = %q", autoResult.Summary)
	}
	if autoResult.IntentID != "auto-fix-error_rate" {
		t.Fatalf("auto-fix IntentID = %q, want %q", autoResult.IntentID, "auto-fix-error_rate")
	}

	// Step 3: Recover
	feedback.RecoverMetric("error_rate", 0.1)
	if rate := feedback.Query("error_rate"); rate != 0.1 {
		t.Fatalf("recovered error_rate = %f, want 0.1", rate)
	}

	// Verify sandbox was created for the fix workstream
	if len(sandbox.Created()) < 1 {
		t.Fatal("expected at least 1 sandbox for auto-fix")
	}
}
