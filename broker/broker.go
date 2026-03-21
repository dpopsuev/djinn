package broker

import (
	"context"
	"fmt"
	"sync"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/orchestrator"
	"github.com/dpopsuev/djinn/signal"
)

// Broker is the central hub that composes the orchestrator, signal bus, cordons,
// and ports. It receives intents, drives orchestration, and reports health.
type Broker struct {
	orch     orchestrator.Orchestrator
	bus      *signal.SignalBus
	cordons  *CordonRegistry
	operator OperatorPort
	alerts   EventIngressPort
	sandbox  SandboxPort
	metrics  MetricsPort

	planFactory func(ari.Intent) orchestrator.WorkPlan

	mu      sync.Mutex
	running map[string]context.CancelFunc
}

// BrokerConfig holds the dependencies for creating a Broker.
type BrokerConfig struct {
	Orchestrator orchestrator.Orchestrator
	Bus          *signal.SignalBus
	Cordons      *CordonRegistry
	Operator     OperatorPort
	Alerts       EventIngressPort
	Sandbox      SandboxPort
	Metrics      MetricsPort
	PlanFactory  func(ari.Intent) orchestrator.WorkPlan
}

// NewBroker creates a new broker from its dependencies.
func NewBroker(cfg BrokerConfig) *Broker {
	return &Broker{
		orch:        cfg.Orchestrator,
		bus:         cfg.Bus,
		cordons:     cfg.Cordons,
		operator:    cfg.Operator,
		alerts:      cfg.Alerts,
		sandbox:     cfg.Sandbox,
		metrics:     cfg.Metrics,
		planFactory: cfg.PlanFactory,
		running:     make(map[string]context.CancelFunc),
	}
}

// Start wires up intent and alert listeners. Call in a goroutine.
func (b *Broker) Start(ctx context.Context) {
	b.operator.OnIntent(func(intent ari.Intent) {
		go b.HandleIntent(ctx, intent)
	})

	if b.alerts != nil {
		go b.listenAlerts(ctx)
	}
}

// HandleIntent processes a single intent: build plan, execute, relay events.
func (b *Broker) HandleIntent(ctx context.Context, intent ari.Intent) {
	plan := b.planFactory(intent)

	execCtx, cancel := context.WithCancel(ctx)
	b.mu.Lock()
	b.running[plan.ID] = cancel
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.running, plan.ID)
		b.mu.Unlock()
	}()

	ch, err := b.orch.Execute(execCtx, plan)
	if err != nil {
		b.operator.EmitResult(ari.Result{
			IntentID: intent.ID,
			Success:  false,
			Summary:  fmt.Sprintf("execute failed: %v", err),
		})
		return
	}

	var lastEvent orchestrator.Event
	for event := range ch {
		b.operator.EmitProgress(event)
		lastEvent = event
	}

	// Emit final andon
	health := signal.ComputeHealth(b.bus.Signals())
	board := ComputeAndon(health, b.cordons.Active())
	b.operator.EmitAndon(board)

	success := lastEvent.Kind == orchestrator.ExecutionDone && lastEvent.Message == "success"
	b.operator.EmitResult(ari.Result{
		IntentID: intent.ID,
		Success:  success,
		Summary:  lastEvent.Message,
	})
}

// Andon computes the current Andon board.
func (b *Broker) Andon() AndonBoard {
	health := signal.ComputeHealth(b.bus.Signals())
	return ComputeAndon(health, b.cordons.Active())
}

func (b *Broker) listenAlerts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert, ok := <-b.alerts.Alerts():
			if !ok {
				return
			}
			b.handleAlert(ctx, alert)
		}
	}
}

func (b *Broker) handleAlert(ctx context.Context, alert ari.Alert) {
	b.bus.Emit(signal.Signal{
		Workstream: "alert-" + alert.Metric,
		Level:      signal.Red,
		Source:     alert.Source,
		Category:   "performance",
		Message:    fmt.Sprintf("alert: %s = %.2f", alert.Metric, alert.Value),
	})

	// Create autonomous fix intent
	intent := ari.Intent{
		ID:     "auto-fix-" + alert.Metric,
		Action: "fix",
		Payload: map[string]string{
			"metric": alert.Metric,
			"value":  fmt.Sprintf("%.2f", alert.Value),
		},
	}

	go b.HandleIntent(ctx, intent)
}
