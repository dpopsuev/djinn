// server.go — Debug MCP server exposing TraceRing as queryable tools (TSK-480).
//
// Provides djinn_trace tool with actions: list, get, tree, health, stats.
// Runs as a JSON-RPC stdio server, spawned by djinn --debug or :debug on.
package debug

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/dpopsuev/djinn/trace"
)

// Sentinel errors.
var (
	ErrUnknownAction  = errors.New("unknown action")
	ErrIDRequired     = errors.New("id is required")
	ErrNotFound       = errors.New("event not found")
	ErrParentRequired = errors.New("parent_id or id is required")
)

// Server exposes TraceRing data via tool dispatch.
type Server struct {
	ring *trace.Ring
}

// NewServer creates a debug server backed by the given trace ring.
func NewServer(ring *trace.Ring) *Server {
	return &Server{ring: ring}
}

// ToolName is the MCP tool name for the debug server.
const ToolName = "djinn_trace"

// TraceInput is the input schema for the djinn_trace tool.
type TraceInput struct {
	Action    string `json:"action"`              // list, get, tree, health, stats
	ID        string `json:"id,omitempty"`        // for get
	ParentID  string `json:"parent_id,omitempty"` // for tree
	Component string `json:"component,omitempty"` // filter for list
	Limit     int    `json:"limit,omitempty"`     // for list (default: 50)
}

// Handle dispatches a djinn_trace tool call.
func (s *Server) Handle(input TraceInput) (string, error) {
	switch input.Action {
	case "list":
		return s.handleList(input)
	case "get":
		return s.handleGet(input)
	case "tree":
		return s.handleTree(input)
	case "health":
		return s.handleHealth()
	case "stats":
		return s.handleStats()
	default:
		return "", fmt.Errorf("%w: %s (expected: list, get, tree, health, stats)", ErrUnknownAction, input.Action)
	}
}

func (s *Server) handleList(input TraceInput) (string, error) {
	limit := input.Limit
	if limit == 0 {
		limit = 50 //nolint:mnd // sensible default
	}

	var events []trace.TraceEvent
	if input.Component != "" {
		events = s.ring.ByComponent(trace.Component(input.Component))
	} else {
		events = s.ring.Last(limit)
	}

	// Apply limit.
	if len(events) > limit {
		events = events[len(events)-limit:]
	}

	return marshalJSON(events)
}

func (s *Server) handleGet(input TraceInput) (string, error) {
	if input.ID == "" {
		return "", fmt.Errorf("%w for get action", ErrIDRequired)
	}
	event, ok := s.ring.Get(input.ID)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrNotFound, input.ID)
	}
	return marshalJSON(event)
}

func (s *Server) handleTree(input TraceInput) (string, error) {
	parentID := input.ParentID
	if parentID == "" {
		parentID = input.ID
	}
	if parentID == "" {
		return "", fmt.Errorf("%w for tree action", ErrParentRequired)
	}

	// Get the root event.
	root, rootFound := s.ring.Get(parentID)
	children := s.ring.ByParent(parentID)

	type treeNode struct {
		Root     *trace.TraceEvent  `json:"root,omitempty"`
		Children []trace.TraceEvent `json:"children"`
	}

	node := treeNode{Children: children}
	if rootFound {
		node.Root = &root
	}
	return marshalJSON(node)
}

func (s *Server) handleHealth() (string, error) {
	events := s.ring.ByComponent(trace.ComponentMCP)

	// Per-server latency stats.
	type serverHealth struct {
		Server   string    `json:"server"`
		Calls    int       `json:"calls"`
		Errors   int       `json:"errors"`
		AvgMs    float64   `json:"avg_ms"`
		P95Ms    float64   `json:"p95_ms"`
		LastCall time.Time `json:"last_call"`
	}

	serverMap := make(map[string]*struct {
		calls     int
		errors    int
		latencies []float64
		lastCall  time.Time
	})

	for i := range events {
		e := &events[i]
		if e.Action != "result" {
			continue
		}
		s, ok := serverMap[e.Server]
		if !ok {
			s = &struct {
				calls     int
				errors    int
				latencies []float64
				lastCall  time.Time
			}{}
			serverMap[e.Server] = s
		}
		s.calls++
		if e.Error {
			s.errors++
		}
		s.latencies = append(s.latencies, float64(e.Latency.Milliseconds()))
		if e.Timestamp.After(s.lastCall) {
			s.lastCall = e.Timestamp
		}
	}

	health := make([]serverHealth, 0, len(serverMap))
	for name, s := range serverMap {
		avg := 0.0
		p95 := 0.0
		if len(s.latencies) > 0 {
			avg = mean(s.latencies)
			p95 = percentile(s.latencies, 95) //nolint:mnd // p95
		}
		health = append(health, serverHealth{
			Server:   name,
			Calls:    s.calls,
			Errors:   s.errors,
			AvgMs:    avg,
			P95Ms:    p95,
			LastCall: s.lastCall,
		})
	}

	return marshalJSON(health)
}

func (s *Server) handleStats() (string, error) {
	return marshalJSON(s.ring.Stats())
}

func marshalJSON(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func percentile(vals []float64, pct int) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	idx := (pct * len(sorted)) / 100 //nolint:mnd // percentile calculation
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
