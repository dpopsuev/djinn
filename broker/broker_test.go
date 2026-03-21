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
	mu          sync.Mutex
	events      []orchestrator.Event
	results     []ari.Result
	andons      []AndonBoard
	permissions []ari.PermissionPayload
	handler     func(ari.Intent)
	permRespCh  chan ari.PermissionResponse
}

func newTestOperator() *testOperator {
	return &testOperator{
		permRespCh: make(chan ari.PermissionResponse, 10),
	}
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
func (o *testOperator) EmitPermission(payload ari.PermissionPayload) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.permissions = append(o.permissions, payload)
}
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
	return o.permRespCh
}

func (o *testOperator) getResults() []ari.Result {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]ari.Result, len(o.results))
	copy(out, o.results)
	return out
}

func (o *testOperator) getAndons() []AndonBoard {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]AndonBoard, len(o.andons))
	copy(out, o.andons)
	return out
}

func newTestBrokerConfig(op *testOperator, bus *signal.SignalBus) BrokerConfig {
	cordons := NewCordonRegistry()

	orch := orchestrator.NewSimpleOrchestrator(
		func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
		func(ctx context.Context, id string) error { return nil },
		func(cfg driver.DriverConfig) driver.Driver { return newTestDriver() },
		func(cfg gate.GateConfig) gate.Gate { return &testGate{} },
		func(s signal.Signal) { bus.Emit(s) },
	)

	return BrokerConfig{
		Orchestrator: orch,
		Bus:          bus,
		Cordons:      cordons,
		Operator:     op,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return orchestrator.WorkPlan{
				ID: intent.ID,
				Stages: []orchestrator.Stage{
					{Name: "code", Scope: tier.Scope{Level: tier.Mod, Name: "auth"}, Prompt: "code it"},
				},
			}
		},
	}
}

func TestBroker_HandleIntent(t *testing.T) {
	bus := signal.NewSignalBus()
	op := newTestOperator()
	b := NewBroker(newTestBrokerConfig(op, bus))

	b.HandleIntent(context.Background(), ari.Intent{ID: "int-1", Action: "fix"})

	results := op.getResults()
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if !results[0].Success {
		t.Fatalf("Success = false, want true")
	}

	andons := op.getAndons()
	if len(andons) != 1 {
		t.Fatalf("andons = %d, want 1", len(andons))
	}
	if andons[0].Level != signal.Green {
		t.Fatalf("andon level = %v, want Green", andons[0].Level)
	}
}

func TestBroker_WorkstreamTracking(t *testing.T) {
	bus := signal.NewSignalBus()
	op := newTestOperator()
	b := NewBroker(newTestBrokerConfig(op, bus))

	b.HandleIntent(context.Background(), ari.Intent{ID: "ws-track", Action: "implement"})

	ws, ok := b.Workstreams().Get("ws-track")
	if !ok {
		t.Fatal("workstream not registered")
	}
	if ws.Status != WorkstreamCompleted {
		t.Fatalf("Status = %q, want %q", ws.Status, WorkstreamCompleted)
	}
	if ws.IntentID != "ws-track" {
		t.Fatalf("IntentID = %q, want %q", ws.IntentID, "ws-track")
	}
	if ws.Action != "implement" {
		t.Fatalf("Action = %q, want %q", ws.Action, "implement")
	}
	if len(ws.Scopes) != 1 {
		t.Fatalf("Scopes = %d, want 1", len(ws.Scopes))
	}
}

func TestBroker_Andon(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	b := NewBroker(BrokerConfig{Bus: bus, Cordons: cordons})

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
	op := newTestOperator()
	b := NewBroker(newTestBrokerConfig(op, bus))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Start(ctx)

	op.mu.Lock()
	h := op.handler
	op.mu.Unlock()
	h(ari.Intent{ID: "int-via-start", Action: "fix"})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(op.getResults()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(op.getResults()) < 1 {
		t.Fatal("expected at least 1 result from Start handler")
	}
}

func TestBroker_BlackSignalAutoCordon(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	op := newTestOperator()
	b := NewBroker(BrokerConfig{Bus: bus, Cordons: cordons, Operator: op})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Start(ctx)

	// Emit a Black signal with scope
	bus.Emit(signal.Signal{
		Workstream: "w1",
		Level:      signal.Black,
		Source:     "agent-1",
		Scope:      []string{"auth/middleware.go"},
		Category:   signal.CategorySecurity,
		Message:    "hardcoded API key",
	})

	// Cordon should be set automatically
	overlaps := cordons.Overlaps([]string{"auth/middleware.go"})
	if len(overlaps) != 1 {
		t.Fatalf("expected 1 cordon, got %d", len(overlaps))
	}
	if overlaps[0].Source != "agent-1" {
		t.Fatalf("cordon Source = %q, want %q", overlaps[0].Source, "agent-1")
	}
}

func TestBroker_BlackSignalNoScopeNoCordon(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	op := newTestOperator()
	b := NewBroker(BrokerConfig{Bus: bus, Cordons: cordons, Operator: op})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Start(ctx)

	// Black signal without scope should NOT create a cordon
	bus.Emit(signal.Signal{
		Workstream: "w1",
		Level:      signal.Black,
		Message:    "general failure",
	})

	if len(cordons.Active()) != 0 {
		t.Fatalf("expected 0 cordons for scopeless Black signal, got %d", len(cordons.Active()))
	}
}

func TestBroker_CancelWorkstream(t *testing.T) {
	bus := signal.NewSignalBus()
	op := newTestOperator()

	// Use a blocking driver so the workstream stays running
	blockCh := make(chan struct{})
	cfg := BrokerConfig{
		Orchestrator: orchestrator.NewSimpleOrchestrator(
			func(ctx context.Context, scope tier.Scope) (string, error) { return "sb", nil },
			func(ctx context.Context, id string) error { return nil },
			func(cfg driver.DriverConfig) driver.Driver {
				ch := make(chan driver.Message)
				go func() { <-blockCh; close(ch) }()
				return &testDriver{recvCh: ch}
			},
			func(cfg gate.GateConfig) gate.Gate { return &testGate{} },
			func(s signal.Signal) { bus.Emit(s) },
		),
		Bus:      bus,
		Cordons:  NewCordonRegistry(),
		Operator: op,
		PlanFactory: func(intent ari.Intent) orchestrator.WorkPlan {
			return orchestrator.WorkPlan{
				ID:     intent.ID,
				Stages: []orchestrator.Stage{{Name: "slow", Scope: tier.Scope{Level: tier.Mod}, Prompt: "wait"}},
			}
		},
	}
	b := NewBroker(cfg)

	go b.HandleIntent(context.Background(), ari.Intent{ID: "cancel-me", Action: "fix"})
	time.Sleep(20 * time.Millisecond)

	if err := b.CancelWorkstream("cancel-me"); err != nil {
		t.Fatalf("CancelWorkstream: %v", err)
	}
	close(blockCh)

	// Wait for completion
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(op.getResults()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	ws, ok := b.Workstreams().Get("cancel-me")
	if !ok {
		t.Fatal("workstream should exist after cancel")
	}
	if ws.Status != WorkstreamCancelled {
		t.Fatalf("Status = %q, want %q", ws.Status, WorkstreamCancelled)
	}
}

func TestBroker_CancelNonexistent(t *testing.T) {
	b := NewBroker(BrokerConfig{
		Bus:     signal.NewSignalBus(),
		Cordons: NewCordonRegistry(),
	})

	err := b.CancelWorkstream("nope")
	if err == nil {
		t.Fatal("expected error for nonexistent workstream")
	}
}

func TestBroker_ClearCordon(t *testing.T) {
	bus := signal.NewSignalBus()
	cordons := NewCordonRegistry()
	b := NewBroker(BrokerConfig{Bus: bus, Cordons: cordons})

	cordons.Set([]string{"auth/middleware.go"}, "security", "agent-1")

	if len(cordons.Active()) != 1 {
		t.Fatal("expected 1 active cordon before clear")
	}

	b.ClearCordon([]string{"auth/middleware.go"})

	if len(cordons.Active()) != 0 {
		t.Fatal("expected 0 active cordons after clear")
	}
}

func TestBroker_PermissionForwarding(t *testing.T) {
	bus := signal.NewSignalBus()
	op := newTestOperator()
	cfg := newTestBrokerConfig(op, bus)
	b := NewBroker(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Start(ctx)

	// Send a permission response — it should be forwarded to orchestrator via Submit
	op.permRespCh <- ari.PermissionResponse{ExecID: "some-exec", Approved: true}

	// Give time for the goroutine to process
	time.Sleep(20 * time.Millisecond)

	// No crash = forwarding works (Submit may return "not found" which is fine)
}
