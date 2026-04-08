package arena

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunBuilder_RequiresScenario(t *testing.T) {
	_, err := NewRunBuilder().
		WithReferee(NewStubReferee(CheckResult{Pass: true})).
		WithActor(func(_ context.Context, _ string) (string, error) { return "", nil }).
		Build()
	if !errors.Is(err, ErrMissingScenario) {
		t.Fatalf("err = %v, want ErrMissingScenario", err)
	}
}

func TestRunBuilder_RequiresReferee(t *testing.T) {
	_, err := NewRunBuilder().
		WithScenario(HTTPServiceScenario()).
		WithActor(func(_ context.Context, _ string) (string, error) { return "", nil }).
		Build()
	if !errors.Is(err, ErrMissingReferee) {
		t.Fatalf("err = %v, want ErrMissingReferee", err)
	}
}

func TestRunBuilder_RequiresActor(t *testing.T) {
	_, err := NewRunBuilder().
		WithScenario(HTTPServiceScenario()).
		WithReferee(NewStubReferee(CheckResult{Pass: true})).
		Build()
	if !errors.Is(err, ErrMissingActor) {
		t.Fatalf("err = %v, want ErrMissingActor", err)
	}
}

func TestRunBuilder_BuildsSuccessfully(t *testing.T) {
	runner, err := NewRunBuilder().
		WithScenario(HTTPServiceScenario()).
		WithReferee(NewStubReferee(CheckResult{Pass: true, Score: 0.9})).
		WithActor(func(_ context.Context, _ string) (string, error) { return "done", nil }).
		WithToolLevel(L1AgentSpace).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if runner == nil {
		t.Fatal("runner should not be nil")
	}
}

func TestRunner_Execute_WithStubs(t *testing.T) {
	runner, _ := NewRunBuilder().
		WithScenario(HTTPServiceScenario()).
		WithReferee(NewStubReferee(CheckResult{Pass: true, Score: 0.85})).
		WithActor(func(_ context.Context, _ string) (string, error) { return "built it", nil }).
		WithToolLevel(L0Mothballed).
		Build()

	metrics, err := runner.Execute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !metrics.Pass {
		t.Fatal("expected pass")
	}
	if metrics.Score != 0.85 {
		t.Fatalf("score = %f, want 0.85", metrics.Score)
	}
	if metrics.ScenarioID != "http-service" {
		t.Fatalf("scenario = %q", metrics.ScenarioID)
	}
	if metrics.ToolLevel != L0Mothballed {
		t.Fatalf("tool level = %q", metrics.ToolLevel)
	}
	if metrics.TimeElapsed <= 0 {
		t.Fatal("time should be > 0")
	}
}

// TestHTTPReferee_PassesFixture verifies the referee against a known-good
// HTTP server from testdata/. No LLM — the fixture is a pre-written Go project.
func TestHTTPReferee_PassesFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Copy fixture to temp dir (referee may write build artifacts)
	dir := t.TempDir()
	copyFixture(t, "testdata/http_fixture", dir)

	referee := NewHTTPReferee()
	result, err := referee.Check(context.Background(), "http-service", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pass {
		t.Fatalf("expected pass, errors: %v", result.Errors)
	}
	if result.Score != 1.0 {
		t.Fatalf("score = %f, want 1.0", result.Score)
	}
}

// TestHTTPReferee_FailsBadCode verifies the referee catches build failures.
func TestHTTPReferee_FailsBadCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := t.TempDir()
	// Write invalid Go code
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("not go code"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	referee := NewHTTPReferee()
	result, err := referee.Check(context.Background(), "http-service", dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Pass {
		t.Fatal("expected fail for bad code")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected errors")
	}
}

// TestRunner_E2E_WithHTTPReferee runs the full pipeline:
// stub actor writes fixture → real HTTPReferee compiles+curls → pass.
func TestRunner_E2E_WithHTTPReferee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Stub actor that copies the known-good fixture to workspace
	fixtureActor := func(_ context.Context, input string) (string, error) {
		// Extract workspace from the input (Runner prepends "Workspace: <path>")
		var workspace string
		for _, line := range splitLines(input) {
			if len(line) > 11 && line[:11] == "Workspace: " {
				workspace = line[11:]
				break
			}
		}
		if workspace == "" {
			return "", errors.New("no workspace in input")
		}

		// Copy fixture files
		for _, name := range []string{"main.go", "go.mod"} {
			data, err := os.ReadFile(filepath.Join("testdata", "http_fixture", name))
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(workspace, name), data, 0o644); err != nil {
				return "", err
			}
		}
		return "wrote main.go + go.mod", nil
	}

	runner, err := NewRunBuilder().
		WithScenario(HTTPServiceScenario()).
		WithReferee(NewHTTPReferee()).
		WithActor(fixtureActor).
		WithOperator(NewMockOperator("build the server")).
		WithToolLevel(L0Mothballed).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	metrics, err := runner.Execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !metrics.Pass {
		t.Fatal("expected pass — fixture should compile and serve")
	}
	if metrics.Score != 1.0 {
		t.Fatalf("score = %f, want 1.0", metrics.Score)
	}
	t.Logf("E2E passed: scenario=%s, score=%.1f, time=%v",
		metrics.ScenarioID, metrics.Score, metrics.TimeElapsed)
}

func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read fixture dir %s: %v", src, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", e.Name(), err)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
