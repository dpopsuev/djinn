// agent_suite.go — Testify Suite with invisible Mirage workspace lifecycle.
// Every test method gets an isolated workspace. Setup and teardown are automatic.
// Test authors write suite methods — they never manage workspace lifecycle.
//
// Usage:
//
//	type MyAgentTests struct {
//	    testkit.AgentSuite
//	}
//
//	func (s *MyAgentTests) TestWritesFile() {
//	    // s.Workspace() is automatically created/destroyed
//	    // s.RunAgent("coder-1", []string{"developer"}, "write hello.go")
//	}
//
//	func TestMyAgent(t *testing.T) {
//	    suite.Run(t, new(MyAgentTests))
//	}
package testkit

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	troupedriver "github.com/dpopsuev/djinn/driver/troupe"
	"github.com/dpopsuev/djinn/observe"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/uniform"
	"github.com/dpopsuev/troupe/execution"
	"github.com/dpopsuev/troupe/signal"
	"github.com/stretchr/testify/suite"
)

// AgentSuite provides invisible workspace isolation for agent tests.
// Embed this in your test suite struct. Every test method gets a fresh
// workspace via SetupTest/TearDownTest.
//
// Observability is always-on: every agent run traces tool calls, LLM
// responses, and errors via EventLog → TraceProjection → Observer.
// On test failure, trace events and workspace contents are dumped
// automatically — no opt-in required.
type AgentSuite struct {
	suite.Suite
	ws       *TestWorkspace
	registry *builtin.Registry
	roleReg  *uniform.RoleRegistry
	toolReqs *uniform.ToolRequirements

	// Observability — invisible, always-on.
	eventLog *signal.MemLog
	ring     *telemetry.TraceProjection
	tracer   *telemetry.Tracer
	observer *observe.EventLogObserver

	// Uniform pipeline — metrics, police, cordon, bottleneck.
	signalBus      *telemetry.SignalBus
	metricsHandler *uniform.MetricsHandler
}

// SetupTest creates an isolated workspace before each test method.
func (s *AgentSuite) SetupTest() {
	s.ws = NewTestWorkspace(s.T())

	// Workspace-rooted tools — agent operates inside workspace namespace.
	s.registry = builtin.NewRegistryWithWorkDir(s.ws.Dir())
	builtin.RegisterBuiltinTools(s.registry, s.ws.Dir(), s.ws.Dir())
	s.roleReg = uniform.NewRoleRegistry(uniform.DefaultRoles())
	s.toolReqs = uniform.DefaultToolRequirements()

	// Observability stack — automatic, invisible to test authors.
	s.eventLog = signal.NewMemLog()
	s.ring = telemetry.NewTraceProjection(500).WithEventLog(s.eventLog)
	s.tracer = telemetry.NewTracer(s.ring, telemetry.ComponentAgent)
	s.observer = observe.NewEventLogObserver(s.eventLog)

	// Uniform pipeline — metrics, police, cordon feed into signal bus.
	s.signalBus = telemetry.NewSignalBus()
	metrics := uniform.NewAgentMetrics("test-agent", "test")
	police := uniform.NewAgentPolice(uniform.DefaultCordonConfig())
	s.metricsHandler = uniform.NewMetricsHandler(metrics, police, s.signalBus, nil, uniform.DefaultCordonConfig())
}

// TearDownTest destroys the workspace after each test method.
// On failure, dumps trace events and workspace contents for post-mortem.
func (s *AgentSuite) TearDownTest() {
	if s.T().Failed() {
		s.dumpTrace()
		s.dumpWorkspace()
	}
	if s.ws != nil {
		s.ws.Destroy() //nolint:errcheck // test cleanup
	}
}

// Workspace returns the current test workspace.
func (s *AgentSuite) Workspace() *TestWorkspace { return s.ws }

// WorkDir returns the workspace directory path.
func (s *AgentSuite) WorkDir() string { return s.ws.Dir() }

// Registry returns the tool registry for this test.
func (s *AgentSuite) Registry() *builtin.Registry { return s.registry }

// MakeUniform creates a Uniform for the given persona + roles.
func (s *AgentSuite) MakeUniform(persona string, roles []string, prompt string) *uniform.Uniform {
	return uniform.NewUniform(
		persona, roles,
		s.roleReg, s.toolReqs,
		s.registry.Names(),
		"agent", "",
		prompt,
	)
}

// RunAgent runs an agent with the given persona, roles, and prompt.
// Uses ScriptedChatDriver if no DJINN_PROVIDER set, real LLM otherwise.
// Returns the agent's final response text.
func (s *AgentSuite) RunAgent(persona string, roles []string, prompt string) string {
	s.T().Helper()

	u := s.MakeUniform(persona, roles, "You are a test agent.")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	drv, err := s.createDriver()
	s.Require().NoError(err, "create driver")

	err = drv.Start(ctx, u.SystemContext())
	s.Require().NoError(err, "start driver")
	defer drv.Stop(ctx) //nolint:errcheck // test cleanup

	sess := cortex.New("test-"+persona, os.Getenv("DJINN_PROVIDER"), s.ws.Dir())

	s.metricsHandler.StartTurn()
	result, err := agent.Run(ctx, agent.Config{
		Driver:       drv,
		Tools:        s.registry,
		Session:      sess,
		SystemPrompt: u.SystemContext(),
		MaxTurns:     10,
		ToolsEnabled: true,
		Approve:      agent.AutoApprove,
		Enforcer:     policy.NopToolPolicyEnforcer{},
		Tracer:       s.tracer,
		Handler: &compositeHandler{handlers: []agent.EventHandler{
			&testEventHandler{t: s.T()},
			s.metricsHandler,
		}},
	}, prompt)
	s.Require().NoError(err, "agent.Run")

	return result
}

// SkipIfNoProvider skips the test if DJINN_PROVIDER is not set.
func (s *AgentSuite) SkipIfNoProvider() {
	if os.Getenv("DJINN_PROVIDER") == "" {
		s.T().Skip("DJINN_PROVIDER not set — skipping real LLM test")
	}
	if os.Getenv("DJINN_MODEL") == "" {
		s.T().Fatal("DJINN_MODEL not set — required for real LLM tests")
	}
}

func (s *AgentSuite) createDriver() (*troupedriver.ChatDriver, error) {
	provider := os.Getenv("DJINN_PROVIDER")
	if provider == "" {
		// No real LLM — tests using RunAgent without SkipIfNoProvider
		// will fail at driver.Start. This is intentional — call SkipIfNoProvider first.
		return nil, fmt.Errorf("DJINN_PROVIDER not set")
	}

	p, err := execution.NewProviderFromEnv("DJINN_PROVIDER")
	if err != nil {
		return nil, err
	}

	model := os.Getenv("DJINN_MODEL")
	return troupedriver.New(p, model, troupedriver.WithBatteryTools(s.registry.All())), nil
}

// Observer returns the EventLog-backed Observer for querying trace events.
func (s *AgentSuite) Observer() *observe.EventLogObserver { return s.observer }

// EventLog returns the underlying MemLog for direct event inspection.
func (s *AgentSuite) EventLog() signal.EventLog { return s.eventLog }

// dumpTrace logs all trace events on test failure.
func (s *AgentSuite) dumpTrace() {
	lines, err := s.observer.Trace(observe.TraceOpts{Last: 100})
	if err != nil {
		s.T().Logf("trace dump failed: %v", err)
		return
	}
	if len(lines) == 0 {
		s.T().Log("=== TRACE DUMP: no events ===")
		return
	}
	s.T().Log("=== TRACE DUMP (last 100 events) ===")
	for _, l := range lines {
		s.T().Logf("  %s [%s] %s — %s (%dms)",
			l.Timestamp.Format("15:04:05.000"),
			l.Source, l.Kind, l.Summary, l.Duration)
	}
}

// dumpWorkspace logs all workspace file contents on test failure.
func (s *AgentSuite) dumpWorkspace() {
	if s.ws == nil {
		return
	}
	files := telemetry.DumpWorkspace(s.ws.Dir())
	if len(files) == 0 {
		s.T().Log("=== WORKSPACE DUMP: empty ===")
		return
	}
	s.T().Log("=== WORKSPACE DUMP ===")
	for path, content := range files {
		s.T().Logf("  --- %s ---\n%s", path, content)
	}
}

// testEventHandler logs agent events to the test's output in real time.
// On failure, -v flag reveals the full event stream.
type testEventHandler struct {
	t interface {
		Helper()
		Logf(format string, args ...any)
	}
}

var _ agent.EventHandler = (*testEventHandler)(nil)

func (h *testEventHandler) OnText(text string) {
	h.t.Helper()
	h.t.Logf("[text] %s", truncateEvent(text))
}

func (h *testEventHandler) OnThinking(text string) {
	h.t.Helper()
	h.t.Logf("[think] %s", truncateEvent(text))
}

func (h *testEventHandler) OnToolCall(call driver.ToolCall) {
	h.t.Helper()
	h.t.Logf("[tool-call] %s id=%s input=%s", call.Name, call.ID, truncateEvent(string(call.Input)))
}

func (h *testEventHandler) OnToolResult(callID, name, output string, isError bool) {
	h.t.Helper()
	h.t.Logf("[tool-result] %s id=%s error=%v output=%s", name, callID, isError, truncateEvent(output))
}

func (h *testEventHandler) OnDone(usage *driver.Usage) {
	h.t.Helper()
	if usage != nil {
		h.t.Logf("[done] input_tokens=%d output_tokens=%d", usage.InputTokens, usage.OutputTokens)
	}
}

func (h *testEventHandler) OnError(err error) {
	h.t.Helper()
	h.t.Logf("[error] %v", err)
}

const maxEventDisplay = 200

func truncateEvent(s string) string {
	if len(s) <= maxEventDisplay {
		return s
	}
	return s[:maxEventDisplay] + "..."
}

// compositeHandler fans out events to multiple handlers.
type compositeHandler struct {
	handlers []agent.EventHandler
}

var _ agent.EventHandler = (*compositeHandler)(nil)

func (c *compositeHandler) OnText(text string) {
	for _, h := range c.handlers {
		h.OnText(text)
	}
}

func (c *compositeHandler) OnThinking(text string) {
	for _, h := range c.handlers {
		h.OnThinking(text)
	}
}

func (c *compositeHandler) OnToolCall(call driver.ToolCall) {
	for _, h := range c.handlers {
		h.OnToolCall(call)
	}
}

func (c *compositeHandler) OnToolResult(callID, name, output string, isError bool) {
	for _, h := range c.handlers {
		h.OnToolResult(callID, name, output, isError)
	}
}

func (c *compositeHandler) OnDone(usage *driver.Usage) {
	for _, h := range c.handlers {
		h.OnDone(usage)
	}
}

func (c *compositeHandler) OnError(err error) {
	for _, h := range c.handlers {
		h.OnError(err)
	}
}

// SignalBus returns the uniform pipeline's signal bus.
func (s *AgentSuite) SignalBus() *telemetry.SignalBus { return s.signalBus }

// MetricsHandler returns the uniform pipeline's metrics handler.
func (s *AgentSuite) MetricsHandler() *uniform.MetricsHandler { return s.metricsHandler }
