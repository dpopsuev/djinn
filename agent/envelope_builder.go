// envelope_builder.go — Builder for ToolEnvelope with security enforcement.
//
// Build() refuses to produce an envelope without a SecurityGate.
// Bundles provide pre-configured layer sets per execution mode.
package agent

import (
	"errors"

	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// ErrNoSecurityGate is returned when Build() is called without a SecurityGate.
var ErrNoSecurityGate = errors.New("envelope: cannot build without a SecurityGate — security by construction")

// EnvelopeBuilder constructs a ToolEnvelope with typed layers.
type EnvelopeBuilder struct {
	gates       []ToolGate
	enrichers   []Enricher
	recorders   []Recorder
	executor    builtin.ToolExecutor
	hasSecurity bool
}

// NewEnvelopeBuilder starts building an envelope around the given executor.
func NewEnvelopeBuilder(executor builtin.ToolExecutor) *EnvelopeBuilder {
	return &EnvelopeBuilder{executor: executor}
}

// WithGate adds a single gate.
func (b *EnvelopeBuilder) WithGate(g ToolGate) *EnvelopeBuilder {
	b.gates = append(b.gates, g)
	if _, ok := g.(SecurityGate); ok {
		b.hasSecurity = true
	}
	return b
}

// WithGates adds multiple gates.
func (b *EnvelopeBuilder) WithGates(gs ...ToolGate) *EnvelopeBuilder {
	for _, g := range gs {
		b.WithGate(g)
	}
	return b
}

// WithEnricher adds a single enricher.
func (b *EnvelopeBuilder) WithEnricher(e Enricher) *EnvelopeBuilder {
	b.enrichers = append(b.enrichers, e)
	return b
}

// WithEnrichers adds multiple enrichers.
func (b *EnvelopeBuilder) WithEnrichers(es ...Enricher) *EnvelopeBuilder {
	b.enrichers = append(b.enrichers, es...)
	return b
}

// WithRecorder adds a single recorder.
func (b *EnvelopeBuilder) WithRecorder(r Recorder) *EnvelopeBuilder {
	b.recorders = append(b.recorders, r)
	return b
}

// WithRecorders adds multiple recorders.
func (b *EnvelopeBuilder) WithRecorders(rs ...Recorder) *EnvelopeBuilder {
	b.recorders = append(b.recorders, rs...)
	return b
}

// Build creates the ToolEnvelope. Returns ErrNoSecurityGate if no SecurityGate
// was registered — security by construction.
func (b *EnvelopeBuilder) Build() (*ToolEnvelope, error) {
	if !b.hasSecurity {
		return nil, ErrNoSecurityGate
	}
	return &ToolEnvelope{
		gates:     b.gates,
		enrichers: b.enrichers,
		recorders: b.recorders,
		executor:  b.executor,
	}, nil
}

// --- Bundles: pre-configured layer sets ---

// SecurityBundle returns gates for policy enforcement. ALWAYS required.
func SecurityBundle(enforcer policy.ToolPolicyEnforcer, token policy.CapabilityToken) []ToolGate {
	return []ToolGate{
		&PolicyGate{enforcer: enforcer, token: token},
	}
}

// EnrichmentBundle returns enrichers for pre-edit context.
func EnrichmentBundle(symbols *SymbolGraphPopulator) []Enricher {
	return []Enricher{
		&SymbolEnricher{populator: symbols},
	}
}

// ObservabilityBundle returns recorders for waste classification.
func ObservabilityBundle(waste *WasteClassifier) []Recorder {
	return []Recorder{
		&WasteRecorder{classifier: waste},
	}
}
