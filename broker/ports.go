package broker

import (
	"context"

	"github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/orchestrator"
	"github.com/dpopsuev/djinn/signal"
	"github.com/dpopsuev/djinn/tier"
)

// OperatorPort is the driving port for operator interactions (intent in, events out).
type OperatorPort interface {
	OnIntent(handler func(ari.Intent))
	EmitProgress(event orchestrator.Event)
	EmitPermission(payload ari.PermissionPayload)
	EmitAndon(board AndonBoard)
	EmitResult(result ari.Result)
	PermissionResponses() <-chan ari.PermissionResponse
}

// EventIngressPort is the driving port for external monitoring alerts.
type EventIngressPort interface {
	Alerts() <-chan ari.Alert
}

// SandboxPort is the driven port for sandbox lifecycle management.
type SandboxPort interface {
	Create(ctx context.Context, scope tier.Scope) (string, error)
	Destroy(ctx context.Context, sandboxID string) error
}

// MetricsPort is the driven port for querying external metrics.
type MetricsPort interface {
	Query(metric string) float64
}

// SignalSinkPort is the driven port for emitting signals to the bus.
type SignalSinkPort interface {
	Emit(s signal.Signal)
}
