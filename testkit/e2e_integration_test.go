package testkit

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/broker"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	msbsandbox "github.com/dpopsuev/djinn/sandbox/misbah"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/testkit/builders"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tier"
)

const (
	defaultMisbahSocket = "/run/misbah/permission.sock"
	integrationTimeout  = 30 * time.Second
)

// TestE2E_Integration_MisbahSandbox runs the full flow with a real Misbah daemon.
// Skipped if the daemon is not available.
func TestE2E_Integration_MisbahSandbox(t *testing.T) {
	socketPath := os.Getenv("MISBAH_DAEMON_SOCKET")
	if socketPath == "" {
		socketPath = defaultMisbahSocket
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Skipf("Misbah daemon not available at %s: %v", socketPath, err)
	}
	conn.Close()

	workspace := t.TempDir()

	// Real sandbox, stub driver + gate
	sandbox := msbsandbox.New(socketPath, workspace)
	defer sandbox.Close()

	bus := signal.NewSignalBus()
	cordons := broker.NewCordonRegistry()

	orch := orchestrator.NewSimpleOrchestrator(
		sandbox.Create,
		sandbox.Destroy,
		func(cfg driver.DriverConfig) driver.Driver {
			return stubs.NewStubDriver(driver.Message{
				Role:    driver.RoleAssistant,
				Content: "completed",
			})
		},
		func(cfg gate.GateConfig) gate.Gate {
			return stubs.AlwaysPassGate()
		},
		func(s signal.Signal) { bus.Emit(s) },
	)

	op := stubs.NewStubOperatorPort()
	b := broker.NewBroker(broker.BrokerConfig{
		Orchestrator: orch,
		Bus:          bus,
		Cordons:      cordons,
		Operator:     op,
		Sandbox:      sandbox,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return builders.NewWorkPlan(intent.ID).
				AddStage("code", tier.Scope{Level: tier.Mod, Name: "test-module"}, "implement changes").
				Build()
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	b.Start(ctx)

	// Send intent via operator
	op.SendIntent(ari.Intent{ID: "integration-1", Action: "fix"})

	// Wait for result
	deadline := time.Now().Add(integrationTimeout)
	for time.Now().Before(deadline) {
		if len(op.Results()) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	results := op.Results()
	if len(results) == 0 {
		t.Fatal("no results received from integration test")
	}

	// Skip on infrastructure-level failures (network isolation, missing privileges)
	// These are environment issues, not code bugs.
	if !results[0].Success {
		summary := results[0].Summary
		if containsAny(summary, "operation not supported", "permission denied", "network isolation") {
			t.Skipf("skipping: host environment insufficient: %s", summary)
		}
		t.Fatalf("integration result: FAIL: %s", summary)
	}

	t.Logf("integration test passed: %s", results[0].Summary)

	// Verify workstream was tracked
	ws, ok := b.Workstreams().Get("integration-1")
	if !ok {
		t.Fatal("workstream not registered")
	}
	if ws.Status != broker.WorkstreamCompleted {
		t.Fatalf("workstream status = %q, want %q", ws.Status, broker.WorkstreamCompleted)
	}

	// Verify andon is green
	board := b.Andon()
	if board.Level != signal.Green {
		t.Fatalf("andon = %v, want Green", board.Level)
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
