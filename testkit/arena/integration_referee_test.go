package arena

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegration_ScriptedAgent_HTTPReferee proves the full pipeline:
// scripted actor writes a known-good HTTP server → referee compiles,
// runs, curls, verifies. No LLM — the actor is a plain function.
func TestIntegration_ScriptedAgent_HTTPReferee(t *testing.T) {
	// Actor that writes the known-good HTTP fixture
	actor := func(ctx context.Context, prompt string) (string, error) {
		// The runner creates workspace in /tmp, passes it to the factory.
		// But since we use WithActor (not WithActorFactory), we need to
		// write to the workspace the runner creates. The runner passes
		// workspace via the Execute flow — but WithActor doesn't get it.
		// We use WithActorFactory instead.
		return "wrote server", nil
	}
	_ = actor // unused — we use factory below

	factory := func(workspace string) (ActorFunc, error) {
		return func(ctx context.Context, prompt string) (string, error) {
			// Write main.go
			mainPath := filepath.Join(workspace, "main.go")
			if err := os.WriteFile(mainPath, []byte(HTTPServiceFixture), 0o644); err != nil {
				return "", err
			}

			// Write go.mod
			modPath := filepath.Join(workspace, "go.mod")
			modContent := "module httpserver\n\ngo 1.22\n"
			if err := os.WriteFile(modPath, []byte(modContent), 0o644); err != nil {
				return "", err
			}

			return "wrote main.go and go.mod", nil
		}, nil
	}

	runner, err := NewRunBuilder().
		WithScenario(HTTPServiceScenario()).
		WithReferee(NewHTTPReferee()).
		WithActorFactory(factory).
		WithToolLevel(L0Mothballed).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metrics, err := runner.Execute(ctx)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !metrics.Pass {
		t.Fatalf("referee FAILED — score=%.1f", metrics.Score)
	}

	t.Logf("PASS — score=%.1f, elapsed=%v", metrics.Score, metrics.TimeElapsed)
}

// TestIntegration_Referee_FailsOnBadCode verifies the referee catches
// compilation failures gracefully.
func TestIntegration_Referee_FailsOnBadCode(t *testing.T) {
	factory := func(workspace string) (ActorFunc, error) {
		return func(ctx context.Context, prompt string) (string, error) {
			// Write invalid Go code
			mainPath := filepath.Join(workspace, "main.go")
			os.WriteFile(mainPath, []byte("package main\n\nfunc main() { BROKEN }\n"), 0o644)

			modPath := filepath.Join(workspace, "go.mod")
			os.WriteFile(modPath, []byte("module bad\n\ngo 1.22\n"), 0o644)

			return "wrote bad code", nil
		}, nil
	}

	runner, err := NewRunBuilder().
		WithScenario(HTTPServiceScenario()).
		WithReferee(NewHTTPReferee()).
		WithActorFactory(factory).
		WithToolLevel(L0Mothballed).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	metrics, err := runner.Execute(ctx)
	if err != nil {
		t.Fatalf("execute should not error: %v", err)
	}

	if metrics.Pass {
		t.Fatal("referee should FAIL on broken code")
	}
	if metrics.Score > 0.1 {
		t.Fatalf("score = %.1f, expected near 0 for build failure", metrics.Score)
	}
	t.Logf("correctly FAILED — score=%.1f", metrics.Score)
}
