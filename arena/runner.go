// runner.go — Arena Runner: compose Scenario + Referee + Operator + Actor into an executable run.
//
// Builder pattern validates composition before execution.
// Runner is generic — knows nothing about HTTP, CLI, or Kernel scenarios.
// The Referee strategy (adapter) brings scenario-specific verification.
package arena

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dpopsuev/djinn/substrate"
)

// Sentinel errors for builder validation.
var (
	ErrMissingScenario = errors.New("arena: scenario is required")
	ErrMissingReferee  = errors.New("arena: referee is required")
	ErrMissingActor    = errors.New("arena: actor is required")
)

// Runner executes a single arena run: operator drives actor, referee verifies.
type Runner struct {
	scenario  Scenario
	referee   Referee
	operator  Operator
	actor     substrate.ActorFunc
	toolLevel ToolLevel
}

// Execute runs the full pipeline: workspace → operator → actor → referee → metrics.
func (r *Runner) Execute(ctx context.Context) (*RunMetrics, error) {
	// 1. Create temp workspace
	workspace, err := os.MkdirTemp("", "arena-"+r.scenario.ID()+"-")
	if err != nil {
		return nil, fmt.Errorf("arena: create workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	// 2. Start timer
	start := time.Now()

	// 3. Feed scenario spec to operator → get prompt
	prompt := r.scenario.Spec()
	if r.operator != nil {
		resp, err := r.operator.Perform(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("arena: operator: %w", err)
		}
		if resp != "" {
			prompt = resp
		}
	}

	// 4. Run actor — agent works in workspace
	// The actor receives the prompt and workspace path as combined input.
	actorInput := fmt.Sprintf("Workspace: %s\n\n%s", workspace, prompt)
	// Actor failure is not fatal — referee still checks what was produced.
	_, _ = r.actor(ctx, actorInput)

	// 5. Referee checks result
	check, checkErr := r.referee.Check(ctx, r.scenario.ID(), workspace)
	if checkErr != nil {
		return nil, fmt.Errorf("arena: referee: %w", checkErr)
	}

	elapsed := time.Since(start)

	// 6. Record metrics
	metrics := &RunMetrics{
		ScenarioID:  r.scenario.ID(),
		ToolLevel:   r.toolLevel,
		TimeElapsed: elapsed,
		Pass:        check.Pass,
		Score:       check.Score,
	}

	return metrics, nil
}

// RunBuilder composes a Runner from parts with validation.
type RunBuilder struct {
	scenario  Scenario
	referee   Referee
	operator  Operator
	actor     substrate.ActorFunc
	toolLevel ToolLevel
}

// NewRunBuilder starts building an arena run.
func NewRunBuilder() *RunBuilder {
	return &RunBuilder{toolLevel: L0Mothballed}
}

func (b *RunBuilder) WithScenario(s Scenario) *RunBuilder {
	b.scenario = s
	return b
}

func (b *RunBuilder) WithReferee(r Referee) *RunBuilder {
	b.referee = r
	return b
}

func (b *RunBuilder) WithOperator(o Operator) *RunBuilder {
	b.operator = o
	return b
}

func (b *RunBuilder) WithActor(a substrate.ActorFunc) *RunBuilder {
	b.actor = a
	return b
}

func (b *RunBuilder) WithToolLevel(l ToolLevel) *RunBuilder {
	b.toolLevel = l
	return b
}

// Build validates and creates the Runner. Fails if required parts are missing.
func (b *RunBuilder) Build() (*Runner, error) {
	if b.scenario == nil {
		return nil, ErrMissingScenario
	}
	if b.referee == nil {
		return nil, ErrMissingReferee
	}
	if b.actor == nil {
		return nil, ErrMissingActor
	}
	return &Runner{
		scenario:  b.scenario,
		referee:   b.referee,
		operator:  b.operator,
		actor:     b.actor,
		toolLevel: b.toolLevel,
	}, nil
}
