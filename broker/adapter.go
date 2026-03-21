package broker

import (
	"context"

	"github.com/dpopsuev/djinn/ari"
)

// RuntimeAdapter adapts a Broker to the ari.Runtime interface,
// breaking the circular import between ari/ and broker/.
type RuntimeAdapter struct {
	broker *Broker
}

// NewRuntimeAdapter creates a new adapter for the given broker.
func NewRuntimeAdapter(b *Broker) *RuntimeAdapter {
	return &RuntimeAdapter{broker: b}
}

func (a *RuntimeAdapter) HandleIntent(ctx context.Context, intent ari.Intent) {
	a.broker.HandleIntent(ctx, intent)
}

func (a *RuntimeAdapter) CancelWorkstream(id string) error {
	return a.broker.CancelWorkstream(id)
}

func (a *RuntimeAdapter) ClearCordon(paths []string) {
	a.broker.ClearCordon(paths)
}

func (a *RuntimeAdapter) Andon() ari.AndonSnapshot {
	board := a.broker.Andon()
	ws := make([]ari.WorkstreamSnapshot, 0, len(board.Workstreams))
	for _, h := range board.Workstreams {
		ws = append(ws, ari.WorkstreamSnapshot{
			ID:     h.Workstream,
			Health: h.Level.String(),
		})
	}
	return ari.AndonSnapshot{
		Level:       board.Level.String(),
		Workstreams: ws,
		Cordons:     len(board.Cordons),
	}
}

func (a *RuntimeAdapter) ListWorkstreams() []ari.WorkstreamSnapshot {
	all := a.broker.Workstreams().All()
	out := make([]ari.WorkstreamSnapshot, len(all))
	for i, ws := range all {
		out[i] = ari.WorkstreamSnapshot{
			ID:       ws.ID,
			IntentID: ws.IntentID,
			Action:   ws.Action,
			Status:   string(ws.Status),
			Health:   ws.Health.String(),
		}
	}
	return out
}
