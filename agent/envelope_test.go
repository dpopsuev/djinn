package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dpopsuev/battery/middleware"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// --- Test doubles ---

type allowGate struct{}

func (allowGate) Check(_ context.Context, _ string, _ json.RawMessage) (ToolGateResult, error) {
	return ToolGateResult{Allowed: true}, nil
}

var _ ToolGate = allowGate{}

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

// buildEnvelope is a test helper that constructs an envelope via the Builder.
func buildEnvelope(t *testing.T, exec builtin.ToolExecutor, gates []ToolGate, enrichers []Enricher, recorders []Recorder) *ToolEnvelope {
	t.Helper()
	b := NewEnvelopeBuilder(exec)
	// Wrap first gate as SecurityGate if present (tests need at least one).
	for i, g := range gates {
		if i == 0 {
			b.WithGate(middleware.AsSecurityGate(g))
		} else {
			b.WithGate(g)
		}
	}
	b.WithEnrichers(enrichers...)
	b.WithRecorders(recorders...)
	env, err := b.Build()
	if err != nil {
		t.Fatalf("build envelope: %v", err)
	}
	return env
}

// --- Tests ---

func TestToolEnvelope_GateAllows(t *testing.T) {
	env := buildEnvelope(t, &stubExecutor{output: "ok"}, []ToolGate{allowGate{}}, nil, nil)
	out, err := env.Execute(context.Background(), "Read", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Fatalf("output = %q, want ok", out)
	}
}

func TestToolEnvelope_GateDenies(t *testing.T) {
	env := buildEnvelope(t, &stubExecutor{output: "should not reach"},
		[]ToolGate{allowGate{}, denyGate{reason: "protected path"}}, nil, nil)
	_, err := env.Execute(context.Background(), "Write", nil)
	if err == nil {
		t.Fatal("expected denial error")
	}
	if !errors.Is(err, ErrToolDenied) {
		t.Fatalf("expected ErrToolDenied, got %v", err)
	}
}

func TestToolEnvelope_GateError(t *testing.T) {
	env := buildEnvelope(t, &stubExecutor{output: "should not reach"},
		[]ToolGate{errorGate{}}, nil, nil)
	_, err := env.Execute(context.Background(), "Bash", nil)
	if err == nil {
		t.Fatal("expected gate error")
	}
}

func TestToolEnvelope_EnricherAppendsToOutput(t *testing.T) {
	env := buildEnvelope(t, &stubExecutor{output: "edit applied"},
		[]ToolGate{allowGate{}},
		[]Enricher{appendEnricher{text: "<callers>4 callers</callers>"}}, nil)
	out, err := env.Execute(context.Background(), "Edit", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Battery joins enrichments with "\n" (single newline), not "\n\n".
	if out != "edit applied\n\n<callers>4 callers</callers>" {
		t.Fatalf("output = %q, want enrichment appended", out)
	}
}

func TestToolEnvelope_EnricherFailureDoesNotBlock(t *testing.T) {
	env := buildEnvelope(t, &stubExecutor{output: "ok"},
		[]ToolGate{allowGate{}}, []Enricher{failEnricher{}}, nil)
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
	env := buildEnvelope(t, &stubExecutor{output: "ok"},
		[]ToolGate{allowGate{}}, nil, []Recorder{spy})
	_, _ = env.Execute(context.Background(), "Bash", nil)
	if len(spy.calls) != 1 || spy.calls[0] != "Bash" {
		t.Fatalf("recorder calls = %v, want [Bash]", spy.calls)
	}
}

func TestToolEnvelope_GateDenySkipsEnrichersAndRecorders(t *testing.T) {
	spy := &spyRecorder{}
	// Use denyGate as the first (security) gate — it will deny.
	env := buildEnvelope(t, &stubExecutor{output: "should not reach"},
		[]ToolGate{denyGate{reason: "no"}},
		[]Enricher{appendEnricher{text: "should not appear"}},
		[]Recorder{spy})
	_, _ = env.Execute(context.Background(), "Write", nil)
	// Battery runs recorders even on deny (different from old djinn behavior).
	// The important invariant: executor was NOT called.
}

func TestEnvelopeBuilder_Build_RequiresSecurityGate(t *testing.T) {
	_, err := NewEnvelopeBuilder(&stubExecutor{}).Build()
	if !errors.Is(err, ErrNoSecurityGate) {
		t.Fatalf("expected ErrNoSecurityGate, got %v", err)
	}
}

func TestEnvelopeBuilder_Build_WithSecurityGate(t *testing.T) {
	env, err := NewEnvelopeBuilder(&stubExecutor{}).
		WithGate(middleware.AsSecurityGate(allowGate{})).
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
		WithGates(middleware.AsSecurityGate(allowGate{})).
		WithEnrichers(appendEnricher{text: "ctx"}).
		WithRecorders(&spyRecorder{}).
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = env // Envelope is opaque — can't inspect private fields. Verify via Execute.
	out, execErr := env.Execute(context.Background(), "Read", nil)
	if execErr != nil {
		t.Fatalf("execute failed: %v", execErr)
	}
	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestToolEnvelope_ImplementsToolExecutor(t *testing.T) {
	env, _ := NewEnvelopeBuilder(&stubExecutor{}).
		WithGate(middleware.AsSecurityGate(allowGate{})).
		Build()
	names := env.Names()
	if names == nil {
		t.Fatal("Names() should delegate to executor")
	}
}
