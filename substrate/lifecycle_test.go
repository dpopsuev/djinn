package substrate

import (
	"context"
	"errors"
	"testing"

	"github.com/dpopsuev/djinn/uniform"

	"github.com/dpopsuev/djinn/uniform/persona"
)

func mockExecutor(outputs map[string]string, errs map[string]error) PhaseExecutor {
	return func(ctx context.Context, action PhaseAction) (string, error) {
		return outputs[action.Name], errs[action.Name]
	}
}

func TestParsePhase(t *testing.T) {
	tests := []struct {
		input string
		want  Phase
		ok    bool
	}{
		{"recon", PhaseRecon, true},
		{"execute", PhaseExecute, true},
		{"buffer", PhaseBuffer, true},
		{"verify", PhaseVerify, true},
		{"bogus", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParsePhase(tt.input)
			if ok != tt.ok {
				t.Fatalf("ParsePhase(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("ParsePhase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPhaseIsReadOnly(t *testing.T) {
	tests := []struct {
		phase Phase
		want  bool
	}{
		{PhaseRecon, true},
		{PhaseVerify, true},
		{PhaseExecute, false},
		{PhaseBuffer, false},
	}
	for _, tt := range tests {
		t.Run(tt.phase.String(), func(t *testing.T) {
			if got := tt.phase.IsReadOnly(); got != tt.want {
				t.Fatalf("%s.IsReadOnly() = %v, want %v", tt.phase, got, tt.want)
			}
		})
	}
}

func TestDefaultEnvelopeConfig(t *testing.T) {
	cfg := DefaultEnvelopeConfig()

	if !cfg.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	if cfg.CheckpointEvery != 3 {
		t.Fatalf("CheckpointEvery = %d, want 3", cfg.CheckpointEvery)
	}
	if cfg.DriftThreshold != 0.2 {
		t.Fatalf("DriftThreshold = %f, want 0.2", cfg.DriftThreshold)
	}
	if len(cfg.PreFlightActions) != 3 {
		t.Fatalf("PreFlightActions len = %d, want 3", len(cfg.PreFlightActions))
	}
	if len(cfg.PostFlightActions) != 3 {
		t.Fatalf("PostFlightActions len = %d, want 3", len(cfg.PostFlightActions))
	}
}

func TestDefaultPreFlight(t *testing.T) {
	actions := DefaultPreFlight()
	if len(actions) != 3 {
		t.Fatalf("len = %d, want 3", len(actions))
	}

	wantNames := []string{"architecture scan", "test health", "lint baseline"}
	wantTools := []string{"arch", "test", "arch"}
	for i, a := range actions {
		if a.Name != wantNames[i] {
			t.Errorf("action[%d].Name = %q, want %q", i, a.Name, wantNames[i])
		}
		if a.Tool != wantTools[i] {
			t.Errorf("action[%d].Tool = %q, want %q", i, a.Tool, wantTools[i])
		}
		if a.Required {
			t.Errorf("action[%d].Required = true, want false (pre-flight defaults are optional)", i)
		}
	}
}

func TestDefaultPostFlight(t *testing.T) {
	actions := DefaultPostFlight()
	if len(actions) != 3 {
		t.Fatalf("len = %d, want 3", len(actions))
	}

	wantNames := []string{"architecture audit", "test coverage", "lint hygiene"}
	wantRequired := []bool{true, true, false}
	for i, a := range actions {
		if a.Name != wantNames[i] {
			t.Errorf("action[%d].Name = %q, want %q", i, a.Name, wantNames[i])
		}
		if a.Required != wantRequired[i] {
			t.Errorf("action[%d].Required = %v, want %v", i, a.Required, wantRequired[i])
		}
	}
}

func TestPreFlightAssignment(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig())
	a := env.PreFlightAssignment()

	if a.Role != "gensec" {
		t.Fatalf("Role = %q, want %q", a.Role, "gensec")
	}
	if a.Mode != uniform.ModeAsk {
		t.Fatalf("Mode = %q, want %q", a.Mode, uniform.ModeAsk)
	}
	if len(a.Scope.ReadPaths) != 1 || a.Scope.ReadPaths[0] != "/" {
		t.Fatalf("ReadPaths = %v, want [/]", a.Scope.ReadPaths)
	}
	if a.Scope.WritePaths != nil {
		t.Fatalf("WritePaths = %v, want nil", a.Scope.WritePaths)
	}
	if a.Persona != persona.RolePersona["gensec"] {
		t.Fatalf("Persona = %q, want %q", a.Persona, persona.RolePersona["gensec"])
	}
}

func TestPostFlightAssignment(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig())
	a := env.PostFlightAssignment()

	if a.Role != "inspector" {
		t.Fatalf("Role = %q, want %q", a.Role, "inspector")
	}
	if a.Mode != uniform.ModeAsk {
		t.Fatalf("Mode = %q, want %q", a.Mode, uniform.ModeAsk)
	}
	if len(a.Scope.ReadPaths) != 1 || a.Scope.ReadPaths[0] != "/" {
		t.Fatalf("ReadPaths = %v, want [/]", a.Scope.ReadPaths)
	}
	if a.Scope.WritePaths != nil {
		t.Fatalf("WritePaths = %v, want nil", a.Scope.WritePaths)
	}
	if a.Persona != persona.RolePersona["inspector"] {
		t.Fatalf("Persona = %q, want %q", a.Persona, persona.RolePersona["inspector"])
	}
}

func TestRunPreFlight_Success(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig())
	exec := mockExecutor(
		map[string]string{
			"architecture scan": "ok",
			"test health":       "ok",
			"lint baseline":     "ok",
		},
		nil,
	)

	err := env.RunPreFlight(context.Background(), exec)
	if err != nil {
		t.Fatalf("RunPreFlight() = %v, want nil", err)
	}

	r := env.Result()
	if len(r.PreFlight) != 3 {
		t.Fatalf("PreFlight results = %d, want 3", len(r.PreFlight))
	}
	for i, pr := range r.PreFlight {
		if !pr.Passed() {
			t.Errorf("PreFlight[%d] Passed() = false, want true", i)
		}
	}
}

func TestRunPreFlight_RequiredFailure(t *testing.T) {
	// Make test health required so we can test required failure.
	cfg := DefaultEnvelopeConfig()
	cfg.PreFlightActions[1].Required = true // "test health"

	env := NewEnvelope("goal-1", cfg)
	testErr := errors.New("tests failed")
	exec := mockExecutor(
		map[string]string{
			"architecture scan": "ok",
			"test health":       "",
			"lint baseline":     "ok",
		},
		map[string]error{
			"test health": testErr,
		},
	)

	err := env.RunPreFlight(context.Background(), exec)
	if err == nil {
		t.Fatal("RunPreFlight() = nil, want error")
	}
	if !errors.Is(err, ErrPreFlightFailed) {
		t.Fatalf("error = %v, want ErrPreFlightFailed", err)
	}

	// Should stop at the failure — only 2 results (first succeeded, second failed).
	r := env.Result()
	if len(r.PreFlight) != 2 {
		t.Fatalf("PreFlight results = %d, want 2", len(r.PreFlight))
	}
}

func TestRunPreFlight_OptionalFailure(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig())
	scanErr := errors.New("scan failed")
	exec := mockExecutor(
		map[string]string{
			"architecture scan": "",
			"test health":       "ok",
			"lint baseline":     "ok",
		},
		map[string]error{
			"architecture scan": scanErr,
		},
	)

	err := env.RunPreFlight(context.Background(), exec)
	if err != nil {
		t.Fatalf("RunPreFlight() = %v, want nil (optional failure)", err)
	}

	r := env.Result()
	if len(r.PreFlight) != 3 {
		t.Fatalf("PreFlight results = %d, want 3", len(r.PreFlight))
	}
	if r.PreFlight[0].Passed() {
		t.Error("PreFlight[0] Passed() = true, want false (it errored)")
	}
}

func TestRunPreFlight_Disabled(t *testing.T) {
	cfg := DefaultEnvelopeConfig()
	cfg.Enabled = false

	env := NewEnvelope("goal-1", cfg)
	called := false
	exec := func(ctx context.Context, action PhaseAction) (string, error) {
		called = true
		return "", nil
	}

	err := env.RunPreFlight(context.Background(), exec)
	if err != nil {
		t.Fatalf("RunPreFlight() = %v, want nil", err)
	}
	if called {
		t.Fatal("executor was called, want no calls when disabled")
	}

	r := env.Result()
	if len(r.PreFlight) != 0 {
		t.Fatalf("PreFlight results = %d, want 0", len(r.PreFlight))
	}
}

func TestShouldCheckpoint(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig()) // every 3

	tests := []struct {
		taskIndex int
		want      bool
	}{
		{0, false},
		{1, false},
		{2, true},
		{3, false},
		{4, false},
		{5, true},
	}
	for _, tt := range tests {
		if got := env.ShouldCheckpoint(tt.taskIndex); got != tt.want {
			t.Errorf("ShouldCheckpoint(%d) = %v, want %v", tt.taskIndex, got, tt.want)
		}
	}
}

func TestShouldCheckpoint_Disabled(t *testing.T) {
	cfg := DefaultEnvelopeConfig()
	cfg.Enabled = false
	env := NewEnvelope("goal-1", cfg)

	for i := range 10 {
		if env.ShouldCheckpoint(i) {
			t.Fatalf("ShouldCheckpoint(%d) = true, want false when disabled", i)
		}
	}
}

func TestCheckBuffer_NoDrift(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig()) // threshold 0.2
	cp := env.CheckBuffer(2, 0.1)

	if cp.AfterTask != 2 {
		t.Fatalf("AfterTask = %d, want 2", cp.AfterTask)
	}
	if cp.BudgetUsed != 0.1 {
		t.Fatalf("BudgetUsed = %f, want 0.1", cp.BudgetUsed)
	}
	if cp.Alert {
		t.Fatal("Alert = true, want false (0.1 < 0.2 threshold)")
	}
}

func TestCheckBuffer_HighDrift(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig()) // threshold 0.2
	cp := env.CheckBuffer(5, 0.5)

	if cp.AfterTask != 5 {
		t.Fatalf("AfterTask = %d, want 5", cp.AfterTask)
	}
	if cp.BudgetUsed != 0.5 {
		t.Fatalf("BudgetUsed = %f, want 0.5", cp.BudgetUsed)
	}
	if !cp.Alert {
		t.Fatal("Alert = false, want true (0.5 > 0.2 threshold)")
	}
}

func TestRunPostFlight(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig())
	exec := mockExecutor(
		map[string]string{
			"architecture audit": "clean",
			"test coverage":      "98%",
			"lint hygiene":       "ok",
		},
		nil,
	)

	err := env.RunPostFlight(context.Background(), exec)
	if err != nil {
		t.Fatalf("RunPostFlight() = %v, want nil", err)
	}

	r := env.Result()
	if len(r.PostFlight) != 3 {
		t.Fatalf("PostFlight results = %d, want 3", len(r.PostFlight))
	}
	for i, pr := range r.PostFlight {
		if !pr.Passed() {
			t.Errorf("PostFlight[%d] Passed() = false, want true", i)
		}
	}
}

func TestEnvelopeResult_Copy(t *testing.T) {
	env := NewEnvelope("goal-1", DefaultEnvelopeConfig())
	exec := mockExecutor(
		map[string]string{
			"architecture scan":  "ok",
			"test health":        "ok",
			"lint baseline":      "ok",
			"architecture audit": "ok",
			"test coverage":      "ok",
			"lint hygiene":       "ok",
		},
		nil,
	)

	_ = env.RunPreFlight(context.Background(), exec)
	env.CheckBuffer(2, 0.15)

	r1 := env.Result()
	r2 := env.Result()

	// Mutate the first copy.
	r1.PreFlight = nil
	r1.Buffers = nil

	// Second copy must be unaffected.
	if len(r2.PreFlight) != 3 {
		t.Fatalf("r2.PreFlight len = %d after r1 mutation, want 3", len(r2.PreFlight))
	}
	if len(r2.Buffers) != 1 {
		t.Fatalf("r2.Buffers len = %d after r1 mutation, want 1", len(r2.Buffers))
	}
}

func TestPhaseResultPassed(t *testing.T) {
	pass := PhaseResult{Error: nil}
	if !pass.Passed() {
		t.Fatal("Passed() = false with nil error, want true")
	}

	fail := PhaseResult{Error: errors.New("boom")}
	if fail.Passed() {
		t.Fatal("Passed() = true with error, want false")
	}
}
