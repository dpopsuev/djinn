package orchestrator

import (
	"context"
	"time"

	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/tier"
)

// EventKind classifies orchestration events.
type EventKind int

const (
	StageStarted EventKind = iota
	StageCompleted
	StageFailed
	GatePassed
	GateFailed
	ExecutionDone
)

func (k EventKind) String() string {
	switch k {
	case StageStarted:
		return "stage_started"
	case StageCompleted:
		return "stage_completed"
	case StageFailed:
		return "stage_failed"
	case GatePassed:
		return "gate_passed"
	case GateFailed:
		return "gate_failed"
	case ExecutionDone:
		return "execution_done"
	default:
		return "unknown"
	}
}

// Event represents an orchestration lifecycle event.
type Event struct {
	ExecID  string
	Kind    EventKind
	Stage   string
	Message string
}

// Stage represents a single phase of a work plan.
type Stage struct {
	Name        string
	Scope       tier.Scope
	Driver      driver.DriverConfig
	Gate        gate.GateConfig
	Prompt      string
	TimeBudget  time.Duration // 0 = unlimited
	TokenBudget int           // 0 = unlimited
}

// WorkPlan describes the full execution plan for an intent.
type WorkPlan struct {
	ID     string
	Stages []Stage
}

// ExternalInput allows injecting data into a running execution.
type ExternalInput struct {
	ExecID  string
	Stage   string
	Payload map[string]string
}

// Orchestrator drives a work plan through its stages.
type Orchestrator interface {
	Execute(ctx context.Context, plan WorkPlan) (<-chan Event, error)
	Submit(ctx context.Context, execID string, input ExternalInput) error
	Cancel(ctx context.Context, execID string) error
}
