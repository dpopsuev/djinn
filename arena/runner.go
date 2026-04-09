// runner.go — Arena Runner: compose Scenario + Referee + Operator + Actor into an executable run.
//
// Builder pattern validates composition before execution.
// Runner is generic — knows nothing about HTTP, CLI, or Kernel scenarios.
// The Referee strategy (adapter) brings scenario-specific verification.
//
// ActorFactory receives the workspace path so tools and session can be
// configured to write there. The Runner creates the workspace, the factory
// creates the actor pointed at it.
package arena

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dpopsuev/djinn/telemetry"
)

// ActorFunc executes agent work in a workspace. Returns output and error.
type ActorFunc func(ctx context.Context, prompt string) (string, error)

// ActorFactory creates an ActorFunc configured for a specific workspace.
// Called by the Runner after creating the workspace directory.
type ActorFactory func(workspace string) (ActorFunc, error)

// Sentinel errors for builder validation.
var (
	ErrMissingScenario = errors.New("arena: scenario is required")
	ErrMissingReferee  = errors.New("arena: referee is required")
	ErrMissingActor    = errors.New("arena: actor or actor factory is required")
)

// Runner executes a single arena run: operator drives actor, referee verifies.
type Runner struct {
	scenario     Scenario
	referee      Referee
	operator     Operator
	actor        ActorFunc    // pre-built actor (workspace baked in)
	actorFactory ActorFactory // OR: factory that receives workspace
	toolLevel    ToolLevel
	log          *slog.Logger
}

// Execute runs the full pipeline: workspace → actor → referee → metrics.
func (r *Runner) Execute(ctx context.Context) (*RunMetrics, error) {
	log := r.log
	if log == nil {
		log = slog.Default()
	}

	// 1. Create temp workspace
	workspace, err := os.MkdirTemp("", "arena-"+r.scenario.ID()+"-")
	if err != nil {
		return nil, fmt.Errorf("arena: create workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	log.InfoContext(ctx, "arena: run started",
		slog.String(telemetry.KeyComponent, r.scenario.ID()),
		slog.String(telemetry.KeyStatus, string(r.toolLevel)),
		slog.String(telemetry.KeyPath, workspace),
	)

	// 2. Resolve actor — factory gets workspace, pre-built actor used as-is
	actor := r.actor
	if r.actorFactory != nil {
		a, err := r.actorFactory(workspace)
		if err != nil {
			return nil, fmt.Errorf("arena: actor factory: %w", err)
		}
		actor = a
	}

	// 3. Start timer
	start := time.Now()

	// 4. Feed scenario spec to operator → get prompt
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

	// 5. Run actor — agent works in workspace
	log.InfoContext(ctx, "arena: actor starting", slog.String(telemetry.KeyCount, fmt.Sprintf("%d", len(prompt))))
	_, actorErr := actor(ctx, prompt)
	if actorErr != nil {
		log.WarnContext(ctx, "arena: actor error (referee still checks)",
			slog.String(telemetry.KeyError, actorErr.Error()),
		)
	}

	// 6. Referee checks result
	check, checkErr := r.referee.Check(ctx, r.scenario.ID(), workspace)
	if checkErr != nil {
		return nil, fmt.Errorf("arena: referee: %w", checkErr)
	}

	elapsed := time.Since(start)

	// 7. Dump workspace artifacts for post-mortem audit
	artifacts := telemetry.DumpWorkspace(workspace)

	// 8. Record metrics
	metrics := &RunMetrics{
		ScenarioID:  r.scenario.ID(),
		ToolLevel:   r.toolLevel,
		TimeElapsed: elapsed,
		Pass:        check.Pass,
		Score:       check.Score,
		Artifacts:   artifacts,
	}

	// 9. Structured log
	metricsJSON, _ := json.Marshal(metrics)
	log.InfoContext(ctx, "arena: run complete", slog.String(telemetry.KeyPerf, string(metricsJSON)))

	// 10. Log each artifact for test output visibility
	for path, content := range artifacts {
		log.InfoContext(ctx, "arena: artifact",
			slog.String(telemetry.KeyPath, path),
			slog.Int(telemetry.KeyCount, len(content)),
		)
	}

	return metrics, nil
}


// RunBuilder composes a Runner from parts with validation.
type RunBuilder struct {
	scenario     Scenario
	referee      Referee
	operator     Operator
	actor        ActorFunc
	actorFactory ActorFactory
	toolLevel    ToolLevel
	log          *slog.Logger
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

// WithActor sets a pre-built actor (workspace already configured).
// Use for stub tests where workspace doesn't matter.
func (b *RunBuilder) WithActor(a ActorFunc) *RunBuilder {
	b.actor = a
	return b
}

// WithActorFactory sets a factory that creates the actor after the workspace exists.
// Use for real runs where tools need to be pointed at the workspace.
func (b *RunBuilder) WithActorFactory(f ActorFactory) *RunBuilder {
	b.actorFactory = f
	return b
}

func (b *RunBuilder) WithToolLevel(l ToolLevel) *RunBuilder {
	b.toolLevel = l
	return b
}

func (b *RunBuilder) WithLogger(l *slog.Logger) *RunBuilder {
	b.log = l
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
	if b.actor == nil && b.actorFactory == nil {
		return nil, ErrMissingActor
	}
	return &Runner{
		scenario:     b.scenario,
		referee:      b.referee,
		operator:     b.operator,
		actor:        b.actor,
		actorFactory: b.actorFactory,
		toolLevel:    b.toolLevel,
		log:          b.log,
	}, nil
}
