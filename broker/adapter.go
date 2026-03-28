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
	for i := range board.Workstreams {
		ws = append(ws, ari.WorkstreamSnapshot{
			ID:     board.Workstreams[i].Workstream,
			Health: board.Workstreams[i].Level.String(),
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
	for i := range all {
		out[i] = ari.WorkstreamSnapshot{
			ID:       all[i].ID,
			IntentID: all[i].IntentID,
			Action:   all[i].Action,
			Status:   string(all[i].Status),
			Health:   all[i].Health.String(),
		}
	}
	return out
}

func (a *RuntimeAdapter) Search(query string) []ari.SearchResult {
	brokerResults := a.broker.Search(query)
	out := make([]ari.SearchResult, len(brokerResults))
	for i, r := range brokerResults {
		out[i] = ari.SearchResult{
			Kind:      r.Kind,
			ID:        r.ID,
			Summary:   r.Summary,
			Timestamp: r.Timestamp,
		}
	}
	return out
}
