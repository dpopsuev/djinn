// envelope_adapters.go — Wrap existing middleware as ToolGate/Enricher/Recorder.
//
// Each adapter is a thin wrapper that bridges an existing type to the
// Envelope's typed interfaces. No new logic — just interface compliance.
package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dpopsuev/djinn/policy"
)

// --- Gates ---

// PolicyGate wraps PolicyEnforcer as a SecurityGate.
// Security by construction: this gate MUST be present for Build() to succeed.
type PolicyGate struct {
	enforcer policy.ToolPolicyEnforcer
	token    policy.CapabilityToken
}

// NewPolicyGate creates a SecurityGate from a PolicyEnforcer + token.
func NewPolicyGate(enforcer policy.ToolPolicyEnforcer, token policy.CapabilityToken) *PolicyGate {
	return &PolicyGate{enforcer: enforcer, token: token}
}

// Check delegates to PolicyEnforcer.Check.
func (g *PolicyGate) Check(ctx context.Context, tool string, input json.RawMessage) (ToolGateResult, error) {
	err := g.enforcer.Check(ctx, g.token, tool, input)
	if err != nil {
		return ToolGateResult{Allowed: false, Reason: err.Error()}, nil
	}
	return ToolGateResult{Allowed: true}, nil
}

// isSecurityGate marks this as a SecurityGate — required by EnvelopeBuilder.
func (g *PolicyGate) isSecurityGate() {}

// Ensure interface compliance.
var _ SecurityGate = (*PolicyGate)(nil)

// --- Enrichers ---

// SymbolEnricher wraps SymbolGraphPopulator. Fires on Edit tool only.
type SymbolEnricher struct {
	populator *SymbolGraphPopulator
}

// NewSymbolEnricher creates an enricher from a SymbolGraphPopulator.
func NewSymbolEnricher(populator *SymbolGraphPopulator) *SymbolEnricher {
	return &SymbolEnricher{populator: populator}
}

// Enrich populates the symbol graph for Edit calls and returns the callers table.
func (e *SymbolEnricher) Enrich(ctx context.Context, tool string, input json.RawMessage) (string, error) {
	if tool != "Edit" {
		return "", nil // only enrich Edit calls
	}

	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &params); err != nil || params.Path == "" {
		return "", nil
	}

	sg, err := e.populator.Populate(ctx, params.Path)
	if err != nil {
		return "", nil // enricher failure is not fatal
	}

	return sg.FormatContext(), nil
}

// Ensure interface compliance.
var _ Enricher = (*SymbolEnricher)(nil)

// --- Recorders ---

// WasteRecorder wraps WasteClassifier as a Recorder.
type WasteRecorder struct {
	classifier *WasteClassifier
}

// NewWasteRecorder creates a recorder from a WasteClassifier.
func NewWasteRecorder(classifier *WasteClassifier) *WasteRecorder {
	return &WasteRecorder{classifier: classifier}
}

// Record classifies the tool call for waste detection.
func (r *WasteRecorder) Record(_ context.Context, tool string, input json.RawMessage, output string, err error, elapsed time.Duration) {
	isError := err != nil
	r.classifier.ClassifyCall(tool, string(input), output, isError, elapsed)
}

// Ensure interface compliance.
var _ Recorder = (*WasteRecorder)(nil)
