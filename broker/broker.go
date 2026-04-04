package broker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/djinnlog"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/workspace"
)

const (
	alertWorkstreamPrefix = "alert-"
	autoFixIntentPrefix   = "auto-fix-"
)

// Sentinel errors for Broker operations.
var ErrWorkstreamNotFound = errors.New("workstream not found or not running")

// Broker is the central hub that composes the orchestrator, signal bus, cordons,
// and ports. It receives intents, drives orchestration, and reports health.
type Broker struct {
	log *slog.Logger

	orch        Orchestrator
	bus         *signal.SignalBus
	cordons     *CordonRegistry
	workstreams *WorkstreamRegistry
	operator    OperatorPort
	alerts      EventIngressPort
	sandbox     SandboxPort
	metrics     MetricsPort

	planFactory func(ari.Intent) WorkPlan

	mu      sync.Mutex
	running map[string]context.CancelFunc
}

// BrokerConfig holds the dependencies for creating a Broker.
type BrokerConfig struct {
	Logger        *slog.Logger // nil = djinnlog.Nop()
	Orchestrator  Orchestrator
	Bus           *signal.SignalBus
	Cordons       *CordonRegistry
	Operator      OperatorPort
	Alerts        EventIngressPort
	Sandbox       SandboxPort
	Metrics       MetricsPort
	PlanFactory   func(ari.Intent) WorkPlan
	MaxConcurrent int // 0 = unlimited
}

func withMaxConcurrentFromConfig(n int) []RegistryOption {
	if n > 0 {
		return []RegistryOption{WithMaxConcurrent(n)}
	}
	return nil
}

// NewBroker creates a new broker from its dependencies.
func NewBroker(cfg *BrokerConfig) *Broker {
	log := cfg.Logger
	if log == nil {
		log = djinnlog.Nop()
	}
	return &Broker{
		log:         log,
		orch:        cfg.Orchestrator,
		bus:         cfg.Bus,
		cordons:     cfg.Cordons,
		workstreams: NewWorkstreamRegistry(withMaxConcurrentFromConfig(cfg.MaxConcurrent)...),
		operator:    cfg.Operator,
		alerts:      cfg.Alerts,
		sandbox:     cfg.Sandbox,
		metrics:     cfg.Metrics,
		planFactory: cfg.PlanFactory,
		running:     make(map[string]context.CancelFunc),
	}
}

// Start wires up intent listeners, alert listeners, and the black-signal auto-cordon.
func (b *Broker) Start(ctx context.Context) {
	b.operator.OnIntent(func(intent ari.Intent) {
		go b.HandleIntent(ctx, intent)
	})

	// Auto-cordon on Black signals
	b.bus.OnSignal(func(s signal.Signal) {
		if s.Level == signal.Black && len(s.Scope) > 0 {
			b.cordons.Set(s.Scope, s.Message, s.Source)
			// Orange: cordon triggered
			b.log.WarnContext(ctx, "cordon triggered",
				slog.String(djinnlog.KeyAction, "cordon"),
				slog.String(djinnlog.KeyReason, s.Message),
				slog.String(djinnlog.KeySource, s.Source),
			)
		}
	})

	if b.alerts != nil {
		go b.listenAlerts(ctx)
	}

	go b.forwardPermissions(ctx)
}

// HandleIntent processes a single intent: build plan, execute, relay events.
func (b *Broker) HandleIntent(ctx context.Context, intent ari.Intent) {
	plan := b.planFactory(intent)
	startedAt := time.Now()

	scopes := make([]workspace.TierScope, len(plan.Stages))
	for i := range plan.Stages {
		scopes[i] = plan.Stages[i].Scope
	}

	// Yellow: workstream created
	b.log.InfoContext(ctx, "workstream created",
		slog.String(djinnlog.KeyAction, "create"),
		slog.String(djinnlog.KeyIntentID, intent.ID),
		slog.Int(djinnlog.KeyCount, len(plan.Stages)),
	)

	wsBus := signal.NewSignalBus()
	ws := &WorkstreamInfo{
		ID:        plan.ID,
		IntentID:  intent.ID,
		Action:    intent.Action,
		Status:    WorkstreamRunning,
		Scopes:    scopes,
		Health:    signal.Green,
		Bus:       wsBus,
		StartedAt: startedAt,
	}

	registered, queuePos := b.workstreams.TryRegister(ws)
	if !registered {
		// Orange: queue full / rejected
		b.log.WarnContext(ctx, "workstream queued",
			slog.String(djinnlog.KeyIntentID, intent.ID),
			slog.Int(djinnlog.KeyQueuePos, queuePos),
			slog.String(djinnlog.KeyReason, "concurrency limit reached"),
		)
		b.operator.EmitResult(ari.Result{
			IntentID: intent.ID,
			Success:  false,
			Summary:  fmt.Sprintf("queued at position %d", queuePos),
		})
		return
	}

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
		// Orange: execution error
		b.log.WarnContext(ctx, "workstream execution error",
			slog.String(djinnlog.KeyIntentID, intent.ID),
			slog.String(djinnlog.KeyError, err.Error()),
		)
		b.workstreams.Complete(plan.ID, WorkstreamFailed)
		b.operator.EmitResult(ari.Result{
			IntentID: intent.ID,
			Success:  false,
			Summary:  fmt.Sprintf("execute failed: %v", err),
		})
		return
	}

	var lastEvent Event
	for event := range ch {
		b.operator.EmitProgress(event)
		lastEvent = event
	}

	// Emit final andon
	health := signal.ComputeHealth(b.bus.Signals())
	board := ComputeAndon(health, b.cordons.Active())
	b.operator.EmitAndon(board)

	success := lastEvent.Kind == ExecutionDone && lastEvent.Message == ExecutionSuccess
	if success {
		b.workstreams.Complete(plan.ID, WorkstreamCompleted)
		// Yellow: workstream completed
		b.log.InfoContext(ctx, "workstream completed",
			slog.String(djinnlog.KeyIntentID, intent.ID),
			slog.String(djinnlog.KeyStatus, string(WorkstreamCompleted)),
			slog.Duration(djinnlog.KeyDuration, time.Since(startedAt)),
		)
	} else {
		b.workstreams.Complete(plan.ID, WorkstreamFailed)
		// Orange: workstream failed
		b.log.WarnContext(ctx, "workstream failed",
			slog.String(djinnlog.KeyIntentID, intent.ID),
			slog.String(djinnlog.KeyStatus, string(WorkstreamFailed)),
			slog.String(djinnlog.KeyReason, lastEvent.Message),
			slog.Duration(djinnlog.KeyDuration, time.Since(startedAt)),
		)
	}

	b.operator.EmitResult(ari.Result{
		IntentID: intent.ID,
		Success:  success,
		Summary:  lastEvent.Message,
	})

	// Dequeue next pending workstream if any
	if next := b.workstreams.Dequeue(); next != nil {
		go b.HandleIntent(ctx, ari.Intent{
			ID:     next.IntentID,
			Action: next.Action,
		})
	}
}

// CancelWorkstream cancels a running workstream by its plan ID.
func (b *Broker) CancelWorkstream(id string) error {
	b.mu.Lock()
	cancel, ok := b.running[id]
	b.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrWorkstreamNotFound, id)
	}
	cancel()
	b.workstreams.Complete(id, WorkstreamCanceled)

	// Yellow: workstream canceled
	b.log.InfoContext(context.Background(), "workstream canceled",
		slog.String(djinnlog.KeyAction, "cancel"),
		slog.String(djinnlog.KeyWorkstreamID, id),
		slog.String(djinnlog.KeyStatus, string(WorkstreamCanceled)),
	)
	return nil
}

// ClearCordon clears cordons overlapping the given paths.
func (b *Broker) ClearCordon(paths []string) {
	b.cordons.Clear(paths)
}

// Andon computes the current Andon board.
func (b *Broker) Andon() AndonBoard {
	health := signal.ComputeHealth(b.bus.Signals())
	return ComputeAndon(health, b.cordons.Active())
}

// Workstreams returns the workstream registry.
func (b *Broker) Workstreams() *WorkstreamRegistry {
	return b.workstreams
}

func (b *Broker) forwardPermissions(ctx context.Context) {
	ch := b.operator.PermissionResponses()
	for {
		select {
		case <-ctx.Done():
			return
		case resp, ok := <-ch:
			if !ok {
				return
			}
			_ = b.orch.Submit(ctx, resp.ExecID, ExternalInput{
				ExecID:  resp.ExecID,
				Payload: map[string]string{"approved": fmt.Sprintf("%t", resp.Approved)},
			})
		}
	}
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
		Workstream: alertWorkstreamPrefix + alert.Metric,
		Level:      signal.Red,
		Source:     alert.Source,
		Category:   signal.CategoryPerformance,
		Message:    fmt.Sprintf("alert: %s = %.2f", alert.Metric, alert.Value),
	})

	intent := ari.Intent{
		ID:     autoFixIntentPrefix + alert.Metric,
		Action: "fix",
		Payload: map[string]string{
			"metric": alert.Metric,
			"value":  fmt.Sprintf("%.2f", alert.Value),
		},
	}

	go b.HandleIntent(ctx, intent)
}
