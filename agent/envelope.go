// envelope.go — Tool Operation Envelope (SPC-118).
//
// All types aliased from battery/middleware — single source of truth.
// Djinn re-exports for backward compatibility during Strangler Fig migration.
package agent

import (
	"github.com/dpopsuev/battery/middleware"
)

// ErrToolDenied is returned when a ToolGate rejects a tool call.
// Re-exported from battery/middleware for backward compatibility.
var ErrToolDenied = middleware.ErrToolDenied

// ToolGate decides if a tool call should proceed. Can reject.
type ToolGate = middleware.Gate

// ToolGateResult is the outcome of a gate check.
type ToolGateResult = middleware.Verdict

// SecurityGate is a marker interface for gates that enforce security policy.
type SecurityGate = middleware.SecurityGate

// Enricher adds context before execution. Cannot reject the call.
type Enricher = middleware.Enricher

// Recorder observes after execution. Cannot affect the result.
type Recorder = middleware.Recorder

// ToolEnvelope wraps a ToolExecutor with Gate/Enrich/Execute/Record pipeline.
type ToolEnvelope = middleware.Envelope

// ErrNoSecurityGate is returned when Build() is called without a SecurityGate.
// Re-exported from battery/middleware for backward compatibility.
var ErrNoSecurityGate = middleware.ErrNoSecurityGate
