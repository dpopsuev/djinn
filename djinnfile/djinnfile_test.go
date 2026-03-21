package djinnfile

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dpopsuev/djinn/gate"
	"github.com/dpopsuev/djinn/tier"
)

const validDjinnfile = `{
  "version": "1",
  "name": "auth-refactor",
  "stages": [
    {
      "name": "analyze",
      "tier": "eco",
      "scope": "workspace",
      "prompt": "analyze the auth subsystem",
      "gate": {"name": "analysis-gate", "severity": "warning"}
    },
    {
      "name": "implement",
      "tier": "mod",
      "scope": "auth",
      "prompt": "implement the fix",
      "time_budget": "3m",
      "token_budget": 100,
      "gate": {"name": "unit-gate", "severity": "blocking"}
    }
  ],
  "driver": {
    "model": "claude-opus-4-6",
    "max_tokens": 8192
  }
}`

func TestParse_Valid(t *testing.T) {
	df, err := Parse(strings.NewReader(validDjinnfile))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if df.Name != "auth-refactor" {
		t.Fatalf("Name = %q, want %q", df.Name, "auth-refactor")
	}
	if df.Version != "1" {
		t.Fatalf("Version = %q, want %q", df.Version, "1")
	}
	if len(df.Stages) != 2 {
		t.Fatalf("Stages = %d, want 2", len(df.Stages))
	}
	if df.Stages[0].Name != "analyze" {
		t.Fatalf("Stage[0].Name = %q, want %q", df.Stages[0].Name, "analyze")
	}
	if df.Stages[0].Tier != "eco" {
		t.Fatalf("Stage[0].Tier = %q, want %q", df.Stages[0].Tier, "eco")
	}
	if df.Stages[1].parsedTimeBudget != 3*time.Minute {
		t.Fatalf("Stage[1].parsedTimeBudget = %v, want 3m", df.Stages[1].parsedTimeBudget)
	}
	if df.Stages[1].TokenBudget != 100 {
		t.Fatalf("Stage[1].TokenBudget = %d, want 100", df.Stages[1].TokenBudget)
	}
	if df.Driver.Model != "claude-opus-4-6" {
		t.Fatalf("Driver.Model = %q, want %q", df.Driver.Model, "claude-opus-4-6")
	}
	if df.Driver.MaxTokens != 8192 {
		t.Fatalf("Driver.MaxTokens = %d, want 8192", df.Driver.MaxTokens)
	}
}

func TestParse_Defaults(t *testing.T) {
	minimal := `{"stages": [{"name": "code", "prompt": "do it"}]}`
	df, err := ParseBytes([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if df.Version != "1" {
		t.Fatalf("default Version = %q, want %q", df.Version, "1")
	}
	if df.Driver.Model != DefaultModel {
		t.Fatalf("default Model = %q, want %q", df.Driver.Model, DefaultModel)
	}
	if df.Stages[0].Tier != "mod" {
		t.Fatalf("default Tier = %q, want %q", df.Stages[0].Tier, "mod")
	}
	if df.Stages[0].Gate.Severity != gate.SeverityBlocking {
		t.Fatalf("default Severity = %q, want %q", df.Stages[0].Gate.Severity, gate.SeverityBlocking)
	}
	if df.Stages[0].Gate.Name != "code-gate" {
		t.Fatalf("default Gate.Name = %q, want %q", df.Stages[0].Gate.Name, "code-gate")
	}
	if df.Stages[0].parsedTimeBudget != DefaultTimeBudgetMod {
		t.Fatalf("default TimeBudget = %v, want %v", df.Stages[0].parsedTimeBudget, DefaultTimeBudgetMod)
	}
}

func TestParse_DefaultTimeBudgetPerTier(t *testing.T) {
	tests := []struct {
		tier string
		want time.Duration
	}{
		{"eco", DefaultTimeBudgetEco},
		{"sys", DefaultTimeBudgetSys},
		{"com", DefaultTimeBudgetCom},
		{"mod", DefaultTimeBudgetMod},
	}
	for _, tt := range tests {
		input := `{"stages": [{"name": "s", "tier": "` + tt.tier + `"}]}`
		df, err := ParseBytes([]byte(input))
		if err != nil {
			t.Fatalf("Parse(%s): %v", tt.tier, err)
		}
		if df.Stages[0].parsedTimeBudget != tt.want {
			t.Fatalf("tier %s: budget = %v, want %v", tt.tier, df.Stages[0].parsedTimeBudget, tt.want)
		}
	}
}

func TestParse_NoStages(t *testing.T) {
	_, err := ParseBytes([]byte(`{"stages": []}`))
	if !errors.Is(err, ErrNoStages) {
		t.Fatalf("err = %v, want ErrNoStages", err)
	}
}

func TestParse_NoStageName(t *testing.T) {
	_, err := ParseBytes([]byte(`{"stages": [{"prompt": "do it"}]}`))
	if !errors.Is(err, ErrNoStageName) {
		t.Fatalf("err = %v, want ErrNoStageName", err)
	}
}

func TestParse_InvalidTimeBudget(t *testing.T) {
	input := `{"stages": [{"name": "s", "time_budget": "not-a-duration"}]}`
	_, err := ParseBytes([]byte(input))
	if err == nil {
		t.Fatal("expected error for invalid time_budget")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := Parse(strings.NewReader("{bad json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestToWorkPlan(t *testing.T) {
	df, err := Parse(strings.NewReader(validDjinnfile))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	plan := df.ToWorkPlan("plan-1")

	if plan.ID != "plan-1" {
		t.Fatalf("ID = %q, want %q", plan.ID, "plan-1")
	}
	if len(plan.Stages) != 2 {
		t.Fatalf("Stages = %d, want 2", len(plan.Stages))
	}

	s0 := plan.Stages[0]
	if s0.Name != "analyze" {
		t.Fatalf("Stage[0].Name = %q, want %q", s0.Name, "analyze")
	}
	if s0.Scope.Level != tier.Eco {
		t.Fatalf("Stage[0].Scope.Level = %v, want Eco", s0.Scope.Level)
	}
	if s0.Scope.Name != "workspace" {
		t.Fatalf("Stage[0].Scope.Name = %q, want %q", s0.Scope.Name, "workspace")
	}
	if s0.Prompt != "analyze the auth subsystem" {
		t.Fatalf("Stage[0].Prompt = %q", s0.Prompt)
	}
	if s0.Driver.Model != "claude-opus-4-6" {
		t.Fatalf("Stage[0].Driver.Model = %q, want %q", s0.Driver.Model, "claude-opus-4-6")
	}

	s1 := plan.Stages[1]
	if s1.TimeBudget != 3*time.Minute {
		t.Fatalf("Stage[1].TimeBudget = %v, want 3m", s1.TimeBudget)
	}
	if s1.TokenBudget != 100 {
		t.Fatalf("Stage[1].TokenBudget = %d, want 100", s1.TokenBudget)
	}
	if s1.Gate.Name != "unit-gate" {
		t.Fatalf("Stage[1].Gate.Name = %q, want %q", s1.Gate.Name, "unit-gate")
	}
}
