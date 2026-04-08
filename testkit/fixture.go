// fixture.go — AgentFixture: pre-configured agent for E2E tests.
//
// Shared factory so E2E tests don't each build agent configs from scratch.
// Default: StubChatDriver + StubSpace + all builtin tools.
// Override: real driver via DJINN_PROVIDER for live E2E.
package testkit

import (
	"context"
	"log/slog"
	"testing"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/contextmgr"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/substrate"
	"github.com/dpopsuev/djinn/testkit/stubs"
	"github.com/dpopsuev/djinn/tools/builtin"
)

// AgentFixture is a pre-configured agent for testing.
type AgentFixture struct {
	t         *testing.T
	workspace *TestWorkspace
	tools     builtin.ToolExecutor
	driver    driver.ChatDriver
	session   *contextmgr.Session
	mode      agent.Mode
	maxTurns  int
}

// AgentOpt configures an AgentFixture.
type AgentOpt func(*AgentFixture)

// WithDriver sets a real ChatDriver (for live E2E).
func WithDriver(d driver.ChatDriver) AgentOpt {
	return func(f *AgentFixture) { f.driver = d }
}

// WithTools overrides the tool registry.
func WithTools(t builtin.ToolExecutor) AgentOpt {
	return func(f *AgentFixture) { f.tools = t }
}

// WithMode sets the agent mode.
func WithMode(m agent.Mode) AgentOpt {
	return func(f *AgentFixture) { f.mode = m }
}

// WithMaxTurns sets the maximum turns.
func WithMaxTurns(n int) AgentOpt {
	return func(f *AgentFixture) { f.maxTurns = n }
}

// NewAgentFixture creates a pre-configured agent for testing.
// Default: StubChatDriver, all builtin tools, agent mode, 10 turns.
func NewAgentFixture(t *testing.T, opts ...AgentOpt) *AgentFixture {
	t.Helper()
	ws := NewTestWorkspace(t)

	f := &AgentFixture{
		t:         t,
		workspace: ws,
		tools:     builtin.NewRegistry(),
		driver:    &stubs.StubChatDriver{},
		session:   contextmgr.New("test", "test-model", ws.Dir()),
		mode:      agent.ModeAgent,
		maxTurns:  10,
	}
	for _, opt := range opts {
		opt(f)
	}

	// Register builtin tools pointed at workspace
	builtin.RegisterBuiltinTools(f.tools.(*builtin.Registry), ws.Dir(), t.TempDir())

	return f
}

// Run executes the agent with the given prompt and returns the result.
func (f *AgentFixture) Run(ctx context.Context, prompt string) (string, error) {
	actorFn := substrate.AgentActorFunc(substrate.AgentActorConfig{
		Driver:   f.driver,
		Tools:    f.tools,
		Session:  f.session,
		MaxTurns: f.maxTurns,
		Mode:     f.mode,
		Approve:  agent.AutoApprove,
		Enforcer: policy.NopToolPolicyEnforcer{},
		Log:      slog.Default(),
	})
	return actorFn(ctx, prompt)
}

// Workspace returns the isolated test workspace.
func (f *AgentFixture) Workspace() *TestWorkspace { return f.workspace }

// Dir returns the workspace directory path.
func (f *AgentFixture) Dir() string { return f.workspace.Dir() }
