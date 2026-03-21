package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tier"
)

// SimpleOrchestrator executes stages sequentially with factory functions.
// It avoids importing broker by accepting creation functions directly.
type SimpleOrchestrator struct {
	createSandbox  func(ctx context.Context, scope tier.Scope) (string, error)
	destroySandbox func(ctx context.Context, id string) error
	driverFactory  func(driver.DriverConfig) driver.Driver
	gateFactory    func(gate.GateConfig) gate.Gate
	signalEmit     func(signal.Signal)

	mu     sync.Mutex
	execs  map[string]context.CancelFunc
	inputs map[string]chan ExternalInput
}

// NewSimpleOrchestrator creates a simple sequential orchestrator.
func NewSimpleOrchestrator(
	createSandbox func(ctx context.Context, scope tier.Scope) (string, error),
	destroySandbox func(ctx context.Context, id string) error,
	driverFactory func(driver.DriverConfig) driver.Driver,
	gateFactory func(gate.GateConfig) gate.Gate,
	signalEmit func(signal.Signal),
) *SimpleOrchestrator {
	return &SimpleOrchestrator{
		createSandbox:  createSandbox,
		destroySandbox: destroySandbox,
		driverFactory:  driverFactory,
		gateFactory:    gateFactory,
		signalEmit:     signalEmit,
		execs:          make(map[string]context.CancelFunc),
		inputs:         make(map[string]chan ExternalInput),
	}
}

// Execute runs a work plan's stages sequentially, emitting events on a channel.
func (o *SimpleOrchestrator) Execute(ctx context.Context, plan WorkPlan) (<-chan Event, error) {
	ch := make(chan Event, len(plan.Stages)*3+1)
	execCtx, cancel := context.WithCancel(ctx)

	inputCh := make(chan ExternalInput, 1)
	o.mu.Lock()
	o.execs[plan.ID] = cancel
	o.inputs[plan.ID] = inputCh
	o.mu.Unlock()

	go func() {
		defer close(ch)
		defer func() {
			o.mu.Lock()
			delete(o.execs, plan.ID)
			delete(o.inputs, plan.ID)
			o.mu.Unlock()
		}()

		for _, stage := range plan.Stages {
			if err := o.executeStage(execCtx, plan.ID, stage, ch); err != nil {
				ch <- Event{ExecID: plan.ID, Kind: StageFailed, Stage: stage.Name, Message: err.Error()}
				ch <- Event{ExecID: plan.ID, Kind: ExecutionDone, Message: "failed: " + err.Error()}
				return
			}
		}
		ch <- Event{ExecID: plan.ID, Kind: ExecutionDone, Message: "success"}
	}()

	return ch, nil
}

func (o *SimpleOrchestrator) executeStage(ctx context.Context, execID string, stage Stage, ch chan<- Event) error {
	// Apply time budget if set
	stageCtx := ctx
	if stage.TimeBudget > 0 {
		var cancel context.CancelFunc
		stageCtx, cancel = context.WithTimeout(ctx, stage.TimeBudget)
		defer cancel()
	}

	ch <- Event{ExecID: execID, Kind: StageStarted, Stage: stage.Name}

	o.signalEmit(signal.Signal{
		Workstream: execID,
		Level:      signal.Green,
		Source:     "orchestrator",
		Category:   "lifecycle",
		Message:    "stage " + stage.Name + " started",
	})

	// Create sandbox
	sandboxID, err := o.createSandbox(stageCtx, stage.Scope)
	if err != nil {
		return fmt.Errorf("create sandbox: %w", err)
	}
	defer o.destroySandbox(ctx, sandboxID) // use parent ctx for cleanup

	// Start driver
	d := o.driverFactory(stage.Driver)
	if err := d.Start(stageCtx, sandboxID); err != nil {
		return fmt.Errorf("start driver: %w", err)
	}
	defer d.Stop(ctx) // use parent ctx for cleanup

	// Send prompt and drain responses
	if err := d.Send(stageCtx, driver.Message{Role: "user", Content: stage.Prompt}); err != nil {
		return fmt.Errorf("send prompt: %w", err)
	}

	msgCount := 0
	recvCh := d.Recv(stageCtx)
	for {
		select {
		case <-stageCtx.Done():
			if stageCtx.Err() == context.DeadlineExceeded {
				o.signalEmit(signal.Signal{
					Workstream: execID,
					Level:      signal.Yellow,
					Source:     "orchestrator",
					Category:   "budget",
					Message:    fmt.Sprintf("stage %s exceeded time budget %v", stage.Name, stage.TimeBudget),
				})
			}
			return stageCtx.Err()
		case _, ok := <-recvCh:
			if !ok {
				goto gateCheck
			}
			msgCount++
			if stage.TokenBudget > 0 && msgCount >= stage.TokenBudget {
				o.signalEmit(signal.Signal{
					Workstream: execID,
					Level:      signal.Yellow,
					Source:     "orchestrator",
					Category:   "budget",
					Message:    fmt.Sprintf("stage %s exceeded token budget %d", stage.Name, stage.TokenBudget),
				})
				goto gateCheck
			}
		}
	}

gateCheck:
	// Run gate
	g := o.gateFactory(stage.Gate)
	if err := g.Validate(stageCtx, sandboxID); err != nil {
		ch <- Event{ExecID: execID, Kind: GateFailed, Stage: stage.Name, Message: err.Error()}
		return fmt.Errorf("gate %s: %w", stage.Gate.Name, err)
	}
	ch <- Event{ExecID: execID, Kind: GatePassed, Stage: stage.Name}
	ch <- Event{ExecID: execID, Kind: StageCompleted, Stage: stage.Name}

	o.signalEmit(signal.Signal{
		Workstream: execID,
		Level:      signal.Green,
		Source:     "orchestrator",
		Category:   "lifecycle",
		Message:    "stage " + stage.Name + " completed",
	})

	return nil
}

// Submit sends external input to a running execution.
func (o *SimpleOrchestrator) Submit(ctx context.Context, execID string, input ExternalInput) error {
	o.mu.Lock()
	ch, ok := o.inputs[execID]
	o.mu.Unlock()
	if !ok {
		return fmt.Errorf("execution %q not found", execID)
	}
	select {
	case ch <- input:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Cancel cancels a running execution.
func (o *SimpleOrchestrator) Cancel(ctx context.Context, execID string) error {
	o.mu.Lock()
	cancel, ok := o.execs[execID]
	o.mu.Unlock()
	if !ok {
		return fmt.Errorf("execution %q not found", execID)
	}
	cancel()
	return nil
}
