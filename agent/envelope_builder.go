// envelope_builder.go — Builder for ToolEnvelope with security enforcement.
//
// Aliased from battery/middleware.Builder — single source of truth.
// Bundles provide djinn-specific pre-configured layer sets.
package agent

import (
	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/djinn/agent/symbol"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/uniform/quality"
)

// EnvelopeBuilder constructs a ToolEnvelope with typed layers.
type EnvelopeBuilder = middleware.Builder

// NewEnvelopeBuilder starts building an envelope around the given executor.
func NewEnvelopeBuilder(executor builtin.ToolExecutor) *EnvelopeBuilder {
	return middleware.NewBuilder(executor)
}

// --- Bundles: pre-configured layer sets ---

// SecurityBundle returns gates for policy enforcement. ALWAYS required.
// Wraps PolicyGate with AsSecurityGate so EnvelopeBuilder recognizes it.
func SecurityBundle(enforcer policy.ToolPolicyEnforcer, token policy.CapabilityToken) []ToolGate {
	return []ToolGate{
		middleware.AsSecurityGate(&PolicyGate{enforcer: enforcer, token: token}),
	}
}

// EnrichmentBundle returns enrichers for pre-edit context.
func EnrichmentBundle(symbols *symbol.SymbolGraphPopulator) []Enricher {
	return []Enricher{
		&SymbolEnricher{populator: symbols},
	}
}

// ObservabilityBundle returns recorders for waste classification.
func ObservabilityBundle(waste *quality.WasteClassifier) []Recorder {
	return []Recorder{
		&WasteRecorder{classifier: waste},
	}
}
