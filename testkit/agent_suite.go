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
	troupedriver "github.com/dpopsuev/djinn/driver/troupe"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/uniform"
	"github.com/dpopsuev/troupe/execution"
	"github.com/stretchr/testify/suite"
)

// AgentSuite provides invisible workspace isolation for agent tests.
// Embed this in your test suite struct. Every test method gets a fresh
// workspace via SetupTest/TearDownTest.
type AgentSuite struct {
	suite.Suite
	ws       *TestWorkspace
	registry *builtin.Registry
	roleReg  *uniform.RoleRegistry
	toolReqs *uniform.ToolRequirements
}

// SetupTest creates an isolated workspace before each test method.
func (s *AgentSuite) SetupTest() {
	s.ws = NewTestWorkspace(s.T())
	s.registry = builtin.NewRegistry()
	builtin.RegisterBuiltinTools(s.registry, s.ws.Dir(), s.ws.Dir())
	s.roleReg = uniform.NewRoleRegistry(uniform.DefaultRoles())
	s.toolReqs = uniform.DefaultToolRequirements()
}

// TearDownTest destroys the workspace after each test method.
func (s *AgentSuite) TearDownTest() {
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
		fmt.Sprintf("%s ALL file paths MUST be absolute, rooted at %s.", prompt, s.ws.Dir()),
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

	result, err := agent.Run(ctx, agent.Config{
		Driver:       drv,
		Tools:        s.registry,
		Session:      sess,
		SystemPrompt: u.SystemContext(),
		MaxTurns:     10,
		ToolsEnabled: true,
		Approve:      agent.AutoApprove,
		Enforcer:     policy.NopToolPolicyEnforcer{},
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
