package broker

import (
	"strings"
	"time"

	"github.com/dpopsuev/djinn/signal"
)

// SearchResultKind identifies the type of search result.
const (
	ResultKindSignal    = "signal"
	ResultKindWorkstream = "workstream"
	ResultKindCordon    = "cordon"
)

// SearchResult is a single item from a federated search.
type SearchResult struct {
	Kind      string    `json:"kind"`
	ID        string    `json:"id"`
	Summary   string    `json:"summary"`
	Timestamp time.Time `json:"timestamp"`
}

// Search queries all Broker subsystems (signals, workstreams, cordons)
// for entries matching the query string. Returns results sorted by
// recency (newest first).
func (b *Broker) Search(query string) []SearchResult {
	q := strings.ToLower(query)
	var results []SearchResult

	results = append(results, b.searchSignals(q)...)
	results = append(results, b.searchWorkstreams(q)...)
	results = append(results, b.searchCordons(q)...)

	sortByTimestamp(results)
	return results
}

func (b *Broker) searchSignals(q string) []SearchResult {
	var results []SearchResult
	for _, s := range b.bus.Signals() {
		if matchesQuery(q, s.Workstream, s.Message, s.Source, s.Category) {
			results = append(results, SearchResult{
				Kind:      ResultKindSignal,
				ID:        s.Workstream,
				Summary:   formatSignalSummary(s),
				Timestamp: s.Timestamp,
			})
		}
	}
	return results
}

func (b *Broker) searchWorkstreams(q string) []SearchResult {
	var results []SearchResult
	for _, ws := range b.workstreams.All() {
		if matchesQuery(q, ws.ID, ws.IntentID, ws.Action, string(ws.Status)) {
			results = append(results, SearchResult{
				Kind:      ResultKindWorkstream,
				ID:        ws.ID,
				Summary:   ws.Action + " [" + string(ws.Status) + "]",
				Timestamp: ws.StartedAt,
			})
		}
	}
	return results
}

func (b *Broker) searchCordons(q string) []SearchResult {
	var results []SearchResult
	for _, c := range b.cordons.Active() {
		scopes := strings.Join(c.Scope, ", ")
		if matchesQuery(q, scopes, c.Reason, c.Source) {
			results = append(results, SearchResult{
				Kind:      ResultKindCordon,
				ID:        scopes,
				Summary:   c.Reason + " (source: " + c.Source + ")",
				Timestamp: c.Timestamp,
			})
		}
	}
	return results
}

func matchesQuery(q string, fields ...string) bool {
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

func formatSignalSummary(s signal.Signal) string {
	return "[" + s.Level.String() + "] " + s.Message
}

func sortByTimestamp(results []SearchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Timestamp.After(results[j-1].Timestamp); j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}
