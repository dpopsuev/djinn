package ari

import (
	"context"
	"time"
)

// WorkstreamSnapshot is a summary of a workstream for ARI consumers.
type WorkstreamSnapshot struct {
	ID       string `json:"id"`
	IntentID string `json:"intent_id"`
	Action   string `json:"action"`
	Status   string `json:"status"`
	Health   string `json:"health"`
}

// AndonSnapshot is a summary of the factory Andon board for ARI consumers.
type AndonSnapshot struct {
	Level       string               `json:"level"`
	Workstreams []WorkstreamSnapshot `json:"workstreams"`
	Cordons     int                  `json:"cordons"`
}

// SearchResult is a single item from a federated search.
type SearchResult struct {
	Kind      string    `json:"kind"`
	ID        string    `json:"id"`
	Summary   string    `json:"summary"`
	Timestamp time.Time `json:"timestamp"`
}

// Runtime defines the operations the ARI server needs from the Broker.
// This interface breaks the circular dependency between ari/ and broker/.
type Runtime interface {
	HandleIntent(ctx context.Context, intent Intent)
	CancelWorkstream(id string) error
	ClearCordon(paths []string)
	Andon() AndonSnapshot
	ListWorkstreams() []WorkstreamSnapshot
	Search(query string) []SearchResult
}
