package broker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/orchestrator"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tier"
)

// testDriver implements driver.Driver for broker tests.
type testDriver struct {
	recvCh chan driver.Message
}

func newTestDriver() *testDriver {
	ch := make(chan driver.Message, 1)
	ch <- driver.Message{Role: "assistant", Content: "done"}
	close(ch)
	return &testDriver{recvCh: ch}
}

func (d *testDriver) Start(ctx context.Context, sandbox driver.SandboxHandle) error { return nil }
func (d *testDriver) Send(ctx context.Context, msg driver.Message) error             { return nil }
func (d *testDriver) Recv(ctx context.Context) <-chan driver.Message                  { return d.recvCh }
func (d *testDriver) Stop(ctx context.Context) error                                 { return nil }

// testGate implements gate.Gate for broker tests.
type testGate struct{}

func (g *testGate) Validate(ctx context.Context, sandboxID string) error { return nil }

// testOperator implements OperatorPort for broker tests.
type testOperator struct {
	mu      sync.Mutex
	events  []orchestrator.Event
	results []ari.Result
	andons  []AndonBoard
	handler func(ari.Intent)
}

func (o *testOperator) OnIntent(handler func(ari.Intent)) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.handler = handler
}
func (o *testOperator) EmitProgress(event orchestrator.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}
func (o *testOperator) EmitPermission(payload ari.PermissionPayload) {}
func (o *testOperator) EmitAndon(board AndonBoard) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.andons = append(o.andons, board)
}
func (o *testOperator) EmitResult(result ari.Result) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.results = append(o.results, result)
}
func (o *testOperator) PermissionResponses() <-chan ari.PermissionResponse {
	return make(chan ari.PermissionResponse)
}

func TestBroker_HandleIntent(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	op := &testOperator{}

	orch := orchestrator.NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver { return newTestDriver() },
		func(cfg gate.GateConfig) gate.Gate { return &testGate{} },
		func(s signal.Signal) { bus.Emit(s) },
	)

	b := NewBroker(BrokerConfig{
		Orchestrator: orch,
		Bus:          bus,
		Cordons:      cordons,
		Operator:     op,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return orchestrator.WorkPlan{
				ID: intent.ID,
				Stages: []orchestrator.Stage{
					{Name: "code", Scope: tier.Scope{Level: tier.Mod}, Prompt: "code it"},
				},
			}
		},
	})

	b.HandleIntent(context.Background(), ari.Intent{ID: "int-1", Action: "fix"})

	if len(op.results) != 1 {
		t.Fatalf("results = %d, want 1", len(op.results))
	}
	if !op.results[0].Success {
		t.Fatalf("Success = false, want true")
	}
	if len(op.andons) != 1 {
		t.Fatalf("andons = %d, want 1", len(op.andons))
	}
	if op.andons[0].Level != signal.Green {
		t.Fatalf("andon level = %v, want Green", op.andons[0].Level)
	}
}

func TestBroker_Andon(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()

	b := NewBroker(BrokerConfig{
		Bus:     bus,
		Cordons: cordons,
	})

	board := b.Andon()
	if board.Level != signal.Green {
		t.Fatalf("initial Andon = %v, want Green", board.Level)
	}

	bus.Emit(signal.Signal{Workstream: "w1", Level: signal.Red, Message: "failing"})
	board = b.Andon()
	if board.Level != signal.Red {
		t.Fatalf("after red signal Andon = %v, want Red", board.Level)
	}
}

func TestBroker_Start_IntentHandler(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	op := &testOperator{}

	orch := orchestrator.NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver { return newTestDriver() },
		func(cfg gate.GateConfig) gate.Gate { return &testGate{} },
		func(s signal.Signal) { bus.Emit(s) },
	)

	b := NewBroker(BrokerConfig{
		Orchestrator: orch,
		Bus:          bus,
		Cordons:      cordons,
		Operator:     op,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return orchestrator.WorkPlan{
				ID: intent.ID,
				Stages: []orchestrator.Stage{
					{Name: "code", Scope: tier.Scope{Level: tier.Mod}, Prompt: "code it"},
				},
			}
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Start(ctx)

	// Simulate operator sending intent via the registered handler
	op.mu.Lock()
	h := op.handler
	op.mu.Unlock()
	h(ari.Intent{ID: "int-via-start", Action: "fix"})

	// Wait for the async handler to complete
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		op.mu.Lock()
		n := len(op.results)
		op.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	op.mu.Lock()
	defer op.mu.Unlock()
	if len(op.results) < 1 {
		t.Fatal("expected at least 1 result from Start handler")
	}
}
