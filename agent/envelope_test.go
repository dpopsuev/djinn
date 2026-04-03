package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/tools/builtin"
)

// --- Test doubles ---

type allowGate struct{}

func (allowGate) Check(_ context.Context, _ string, _ json.RawMessage) (ToolGateResult, error) {
	return ToolGateResult{Allowed: true}, nil
}
func (allowGate) isSecurityGate() {}

var _ SecurityGate = allowGate{}

type denyGate struct{ reason string }

func (g denyGate) Check(_ context.Context, _ string, _ json.RawMessage) (ToolGateResult, error) {
	return ToolGateResult{Allowed: false, Reason: g.reason}, nil
}

type errorGate struct{}

func (errorGate) Check(_ context.Context, _ string, _ json.RawMessage) (ToolGateResult, error) {
	return ToolGateResult{}, errors.New("gate exploded")
}

type appendEnricher struct{ text string }

func (e appendEnricher) Enrich(_ context.Context, _ string, _ json.RawMessage) (string, error) {
	return e.text, nil
}

type failEnricher struct{}

func (failEnricher) Enrich(_ context.Context, _ string, _ json.RawMessage) (string, error) {
	return "", errors.New("enricher failed")
}

type spyRecorder struct {
	calls []string
}

func (r *spyRecorder) Record(_ context.Context, tool string, _ json.RawMessage, _ string, _ error, _ time.Duration) {
	r.calls = append(r.calls, tool)
}

type stubExecutor struct {
	output string
	err    error
}

func (s *stubExecutor) Execute(_ context.Context, _ string, _ json.RawMessage) (string, error) {
	return s.output, s.err
}

func (s *stubExecutor) All() []builtin.Tool { return nil }
func (s *stubExecutor) Names() []string     { return []string{"stub"} }

// --- Tests ---

func TestToolEnvelope_GateAllows(t *testing.T) {
	env := &ToolEnvelope{
		gates:    []ToolGate{allowGate{}},
		executor: &stubExecutor{output: "ok"},
	}
	out, err := env.Execute(context.Background(), "Read", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Fatalf("output = %q, want ok", out)
	}
}

func TestToolEnvelope_GateDenies(t *testing.T) {
	env := &ToolEnvelope{
		gates:    []ToolGate{allowGate{}, denyGate{reason: "protected path"}},
		executor: &stubExecutor{output: "should not reach"},
	}
	_, err := env.Execute(context.Background(), "Write", nil)
	if err == nil {
		t.Fatal("expected denial error")
	}
	if !errors.Is(err, ErrToolDenied) {
		t.Fatalf("expected ErrToolDenied, got %v", err)
	}
}

func TestToolEnvelope_GateError(t *testing.T) {
	env := &ToolEnvelope{
		gates:    []ToolGate{errorGate{}},
		executor: &stubExecutor{output: "should not reach"},
	}
	_, err := env.Execute(context.Background(), "Bash", nil)
	if err == nil {
		t.Fatal("expected gate error")
	}
}

func TestToolEnvelope_EnricherAppendsToOutput(t *testing.T) {
	env := &ToolEnvelope{
		gates:     []ToolGate{allowGate{}},
		enrichers: []Enricher{appendEnricher{text: "<callers>4 callers</callers>"}},
		executor:  &stubExecutor{output: "edit applied"},
	}
	out, err := env.Execute(context.Background(), "Edit", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "edit applied\n\n<callers>4 callers</callers>" {
		t.Fatalf("output = %q, want enrichment appended", out)
	}
}

func TestToolEnvelope_EnricherFailureDoesNotBlock(t *testing.T) {
	env := &ToolEnvelope{
		gates:     []ToolGate{allowGate{}},
		enrichers: []Enricher{failEnricher{}},
		executor:  &stubExecutor{output: "ok"},
	}
	out, err := env.Execute(context.Background(), "Read", nil)
	if err != nil {
		t.Fatalf("enricher failure should not block: %v", err)
	}
	if out != "ok" {
		t.Fatalf("output = %q, want ok (no enrichment on failure)", out)
	}
}

func TestToolEnvelope_RecorderObserves(t *testing.T) {
	spy := &spyRecorder{}
	env := &ToolEnvelope{
		gates:     []ToolGate{allowGate{}},
		recorders: []Recorder{spy},
		executor:  &stubExecutor{output: "ok"},
	}
	_, _ = env.Execute(context.Background(), "Bash", nil)
	if len(spy.calls) != 1 || spy.calls[0] != "Bash" {
		t.Fatalf("recorder calls = %v, want [Bash]", spy.calls)
	}
}

func TestToolEnvelope_GateDenySkipsEnrichersAndRecorders(t *testing.T) {
	spy := &spyRecorder{}
	env := &ToolEnvelope{
		gates:     []ToolGate{denyGate{reason: "no"}},
		enrichers: []Enricher{appendEnricher{text: "should not appear"}},
		recorders: []Recorder{spy},
		executor:  &stubExecutor{output: "should not reach"},
	}
	_, _ = env.Execute(context.Background(), "Write", nil)
	if len(spy.calls) != 0 {
		t.Fatalf("recorder should not fire on deny, got %v", spy.calls)
	}
}

func TestEnvelopeBuilder_Build_RequiresSecurityGate(t *testing.T) {
	_, err := NewEnvelopeBuilder(&stubExecutor{}).Build()
	if !errors.Is(err, ErrNoSecurityGate) {
		t.Fatalf("expected ErrNoSecurityGate, got %v", err)
	}
}

func TestEnvelopeBuilder_Build_WithSecurityGate(t *testing.T) {
	env, err := NewEnvelopeBuilder(&stubExecutor{}).
		WithGate(allowGate{}).
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env == nil {
		t.Fatal("envelope should not be nil")
	}
}

func TestEnvelopeBuilder_WithBundles(t *testing.T) {
	env, err := NewEnvelopeBuilder(&stubExecutor{}).
		WithGates(allowGate{}).
		WithEnrichers(appendEnricher{text: "ctx"}).
		WithRecorders(&spyRecorder{}).
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.gates) != 1 {
		t.Fatalf("gates = %d, want 1", len(env.gates))
	}
	if len(env.enrichers) != 1 {
		t.Fatalf("enrichers = %d, want 1", len(env.enrichers))
	}
	if len(env.recorders) != 1 {
		t.Fatalf("recorders = %d, want 1", len(env.recorders))
	}
}

func TestToolEnvelope_ImplementsToolExecutor(t *testing.T) {
	// Compile-time check is in envelope.go via var _ statement.
	// This test documents the contract.
	env, _ := NewEnvelopeBuilder(&stubExecutor{}).
		WithGate(allowGate{}).
		Build()
	names := env.Names()
	if names == nil {
		t.Fatal("Names() should delegate to executor")
	}
}
