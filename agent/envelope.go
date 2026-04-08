// envelope.go — Tool Operation Envelope (SPC-118).
//
// Wraps ToolExecutor with three transparent layers:
//   - ToolGate: should the call happen? (can reject)
//   - Enricher: add context before execution (cannot reject)
//   - Recorder: observe after execution (cannot affect result)
//
// ToolEnvelope implements ToolExecutor — drop-in replacement.
// Constructed via EnvelopeBuilder which refuses to build without a SecurityGate.
// Security by construction: no consumer can get an unprotected executor.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// ErrToolDenied is returned when a ToolGate rejects a tool call.
var ErrToolDenied = errors.New("tool denied")

// ToolGate decides if a tool call should proceed. Can reject.
// Aliased from battery/middleware.Gate — single source of truth.
type ToolGate = middleware.Gate

// ToolGateResult is the outcome of a gate check.
// Aliased from battery/middleware.Verdict — single source of truth.
type ToolGateResult = middleware.Verdict

// SecurityGate is a marker interface for gates that enforce security policy.
// Aliased from battery/middleware.SecurityGate — single source of truth.
type SecurityGate = middleware.SecurityGate

// Enricher adds context before execution. Cannot reject the call.
// Aliased from battery/middleware.Enricher — single source of truth.
type Enricher = middleware.Enricher

// Recorder observes after execution. Cannot affect the result.
// Aliased from battery/middleware.Recorder — single source of truth.
type Recorder = middleware.Recorder

// ToolEnvelope wraps a ToolExecutor with Gate/Enrich/Execute/Record pipeline.
// Implements builtin.ToolExecutor — transparent to the agent loop.
type ToolEnvelope struct {
	gates     []ToolGate
	enrichers []Enricher
	recorders []Recorder
	executor  builtin.ToolExecutor
}

// Execute runs the full Gate → Enrich → Execute → Record pipeline.
func (e *ToolEnvelope) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	// 1. Gates: first deny stops the pipeline.
	for _, g := range e.gates {
		result, err := g.Check(ctx, name, input)
		if err != nil {
			return "", fmt.Errorf("gate error: %w", err)
		}
		if !result.Allowed {
			return "", fmt.Errorf("%w: %s", ErrToolDenied, result.Reason)
		}
	}

	// 2. Enrichers: add context. Errors warn but never block.
	var enrichments []string
	for _, en := range e.enrichers {
		enrichment, err := en.Enrich(ctx, name, input)
		if err != nil {
			continue // enricher failure is not fatal
		}
		if enrichment != "" {
			enrichments = append(enrichments, enrichment)
		}
	}

	// 3. Execute the actual tool call.
	start := time.Now()
	output, err := e.executor.Execute(ctx, name, input)
	elapsed := time.Since(start)

	// 4. Recorders: observe. Errors are swallowed (never affect result).
	for _, r := range e.recorders {
		r.Record(ctx, name, input, output, err, elapsed)
	}

	// 5. Append enrichments to output (transparent context injection).
	if len(enrichments) > 0 && err == nil {
		output = output + "\n\n" + strings.Join(enrichments, "\n\n")
	}

	return output, err
}

// All delegates to the wrapped executor.
func (e *ToolEnvelope) All() []builtin.Tool { return e.executor.All() }

// Names delegates to the wrapped executor.
func (e *ToolEnvelope) Names() []string { return e.executor.Names() }

// Ensure interface compliance.
var _ builtin.ToolExecutor = (*ToolEnvelope)(nil)
