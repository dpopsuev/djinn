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

	mu    sync.Mutex
	execs map[string]context.CancelFunc
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
	}
}

// Execute runs a work plan's stages sequentially, emitting events on a channel.
func (o *SimpleOrchestrator) Execute(ctx context.Context, plan WorkPlan) (<-chan Event, error) {
	ch := make(chan Event, len(plan.Stages)*3+1)
	execCtx, cancel := context.WithCancel(ctx)

	o.mu.Lock()
	o.execs[plan.ID] = cancel
	o.mu.Unlock()

	go func() {
		defer close(ch)
		defer func() {
			o.mu.Lock()
			delete(o.execs, plan.ID)
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
	ch <- Event{ExecID: execID, Kind: StageStarted, Stage: stage.Name}

	o.signalEmit(signal.Signal{
		Workstream: execID,
		Level:      signal.Green,
		Message:    "stage " + stage.Name + " started",
	})

	// Create sandbox
	sandboxID, err := o.createSandbox(ctx, stage.Scope)
	if err != nil {
		return fmt.Errorf("create sandbox: %w", err)
	}
	defer o.destroySandbox(ctx, sandboxID)

	// Start driver
	d := o.driverFactory(stage.Driver)
	if err := d.Start(ctx, sandboxID); err != nil {
		return fmt.Errorf("start driver: %w", err)
	}
	defer d.Stop(ctx)

	// Send prompt and drain responses
	if err := d.Send(ctx, driver.Message{Role: "user", Content: stage.Prompt}); err != nil {
		return fmt.Errorf("send prompt: %w", err)
	}

	recvCh := d.Recv(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-recvCh:
			if !ok {
				goto gateCheck
			}
		}
	}

gateCheck:
	// Run gate
	g := o.gateFactory(stage.Gate)
	if err := g.Validate(ctx, sandboxID); err != nil {
		ch <- Event{ExecID: execID, Kind: GateFailed, Stage: stage.Name, Message: err.Error()}
		return fmt.Errorf("gate %s: %w", stage.Gate.Name, err)
	}
	ch <- Event{ExecID: execID, Kind: GatePassed, Stage: stage.Name}
	ch <- Event{ExecID: execID, Kind: StageCompleted, Stage: stage.Name}

	o.signalEmit(signal.Signal{
		Workstream: execID,
		Level:      signal.Green,
		Message:    "stage " + stage.Name + " completed",
	})

	return nil
}

// Submit is a no-op for the simple orchestrator.
func (o *SimpleOrchestrator) Submit(ctx context.Context, execID string, input ExternalInput) error {
	return nil
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
