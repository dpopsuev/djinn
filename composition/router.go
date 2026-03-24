// Package composition defines routing strategies for multi-agent work distribution.
//
// When the Manager dispatches a batch of work units to multiple backends,
// the RoutingStrategy determines which backend handles each unit.
// This is the lateral execution model from the staffing architecture
// (DJN-DOC-12: Papercup is structural, Liaison is authority).
//
// All implementations are stubs — real routing requires Bugle identity
// and Origami circuit integration.
package composition

import "fmt"

// WorkUnit represents a single piece of work to be routed to a backend.
type WorkUnit struct {
	ID       string
	TaskID   string // Scribe task reference
	Priority int
}

// Backend represents a target for work execution (a Djinn shell/backend pair).
type Backend struct {
	ID       string
	Role     string // executor, inspector, etc.
	Capacity int    // max concurrent units
	Load     int    // current units in progress
}

// RoutingStrategy determines how work units are distributed to backends.
// Different strategies suit different workloads:
//   - Flat: round-robin for independent units (test execution)
//   - Chain: sequential for dependent units (build pipeline)
//   - Affinity: element-matching for persona-aligned work (Bugle future)
type RoutingStrategy interface {
	Route(unit WorkUnit, backends []Backend) (Backend, error)
	Name() string
}

// FlatStrategy distributes round-robin across available backends.
// Simplest strategy — no dependency awareness, no affinity.
type FlatStrategy struct {
	next int
}

func (s *FlatStrategy) Route(_ WorkUnit, backends []Backend) (Backend, error) {
	if len(backends) == 0 {
		return Backend{}, fmt.Errorf("no backends available")
	}
	b := backends[s.next%len(backends)]
	s.next++
	return b, nil
}

func (s *FlatStrategy) Name() string { return "flat" }

// ChainStrategy executes units sequentially on a single backend.
// Used when units have strict ordering dependencies.
type ChainStrategy struct{}

func (s *ChainStrategy) Route(_ WorkUnit, backends []Backend) (Backend, error) {
	if len(backends) == 0 {
		return Backend{}, fmt.Errorf("no backends available")
	}
	// Always route to the first backend — sequential execution.
	return backends[0], nil
}

func (s *ChainStrategy) Name() string { return "chain" }

// Interface compliance.
var (
	_ RoutingStrategy = (*FlatStrategy)(nil)
	_ RoutingStrategy = (*ChainStrategy)(nil)
)
