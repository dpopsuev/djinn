// unified_director.go — Unified Director: one orchestrator for test, REPL, headless.
//
// Converges AgentRunner, LocalDirector, and crucible.Runner into one type.
// Implements troupe/director.Director for multi-agent orchestration.
// Provides Run() for single-agent REPL path.
//
// GOL-163, TSK-1083, TSK-1084
package substrate

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"

	troupe "github.com/dpopsuev/troupe"
	"github.com/dpopsuev/troupe/director"
	"github.com/dpopsuev/troupe/signal"

	"github.com/dpopsuev/djinn/agent"
	"github.com/dpopsuev/djinn/cortex"
	"github.com/dpopsuev/djinn/driver"
	"github.com/dpopsuev/djinn/hook"
	"github.com/dpopsuev/djinn/policy"
	"github.com/dpopsuev/djinn/telemetry"
	"github.com/dpopsuev/djinn/tools/builtin"
	"github.com/dpopsuev/djinn/uniform"
)

var _ director.Director = (*UnifiedDirector)(nil)

// UnifiedDirector orchestrates agent execution in any context:
// test (ScriptedChatDriver + stub workspace), REPL (real LLM + TUI events),
// or headless (real LLM + socket protocol).
type UnifiedDirector struct {
	// Core dependencies.
	driver       driver.ChatDriver
	tools        builtin.ToolExecutor
	envelope     *agent.ToolEnvelope
	session      *cortex.Session
	systemPrompt string
	maxTurns     int

	// Policies.
	enforcer policy.ToolPolicyEnforcer
	token    policy.CapabilityToken
	router   *uniform.ToolClearance

	// Orchestration.
	scheduler  Scheduler
	dispatcher *hook.EventDispatcher
	eventLog   signal.EventLog
	log        *slog.Logger

	// Sandbox.
	sandboxHandle string
	sandboxExec   func(ctx context.Context, cmd []string) (string, string, error)
}

// DirectorOption configures a UnifiedDirector.
type DirectorOption func(*UnifiedDirector)

// NewUnifiedDirector creates a Director with functional options.
func NewUnifiedDirector(drv driver.ChatDriver, tools builtin.ToolExecutor, opts ...DirectorOption) *UnifiedDirector {
	d := &UnifiedDirector{
		driver:   drv,
		tools:    tools,
		maxTurns: 20, //nolint:mnd // sensible default
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// --- Functional Options ---

func WithEnvelope(e *agent.ToolEnvelope) DirectorOption {
	return func(d *UnifiedDirector) { d.envelope = e }
}

func WithSession(s *cortex.Session) DirectorOption {
	return func(d *UnifiedDirector) { d.session = s }
}

func WithSystemPrompt(p string) DirectorOption {
	return func(d *UnifiedDirector) { d.systemPrompt = p }
}

func WithMaxTurns(n int) DirectorOption {
	return func(d *UnifiedDirector) { d.maxTurns = n }
}

func WithPolicies(e policy.ToolPolicyEnforcer, t policy.CapabilityToken) DirectorOption {
	return func(d *UnifiedDirector) { d.enforcer = e; d.token = t }
}

func WithToolRouter(r *uniform.ToolClearance) DirectorOption {
	return func(d *UnifiedDirector) { d.router = r }
}

func WithSchedulerForDirector(s Scheduler) DirectorOption {
	return func(d *UnifiedDirector) { d.scheduler = s }
}

func WithDispatcher(disp *hook.EventDispatcher) DirectorOption {
	return func(d *UnifiedDirector) { d.dispatcher = disp }
}

func WithEventLogForDirector(l signal.EventLog) DirectorOption {
	return func(d *UnifiedDirector) { d.eventLog = l }
}

func WithDirectorLogger(l *slog.Logger) DirectorOption {
	return func(d *UnifiedDirector) { d.log = l }
}

func WithSandbox(handle string, execFn func(ctx context.Context, cmd []string) (string, string, error)) DirectorOption {
	return func(d *UnifiedDirector) {
		d.sandboxHandle = handle
		d.sandboxExec = execFn
	}
}

// --- Run (single-agent REPL path) ---

// Run executes a single agent turn: send prompt, run ReAct loop, return output.
// This is the REPL path — blocks until the agent completes.
func (d *UnifiedDirector) Run(ctx context.Context, prompt string, mode agent.Mode, approvalCh chan bool, handler agent.EventHandler, currentRole string) (string, error) {
	tools := d.resolveTools()

	cfg := agent.Config{
		Driver:       d.driver,
		Tools:        tools,
		Envelope:     d.envelope,
		Session:      d.session,
		SystemPrompt: d.systemPrompt,
		MaxTurns:     d.maxTurns,
		ToolsEnabled: mode.ToolsEnabled(),
		Mode:         mode,
		Approve:      ApprovalForMode(mode, approvalCh),
		Enforcer:     d.enforcer,
		Token:        d.token,
		Handler:      handler,
		Log:          telemetry.For(d.log, "agent"),
	}

	// Sandbox: everyone sandboxed except GenSec.
	if d.sandboxHandle != "" && currentRole != "gensec" {
		cfg.SandboxHandle = d.sandboxHandle
		cfg.SandboxExec = d.sandboxExec
		if d.session != nil {
			cfg.SandboxWorkDir = d.session.WorkDir
		}
		cfg.SandboxMount = "/workspace"
	}

	// Emit start event.
	if d.eventLog != nil {
		d.eventLog.Emit(signal.Event{
			Source: "director",
			Kind:   "agent.run.start",
			Data:   map[string]string{"role": currentRole, "mode": mode.String()},
		})
	}

	output, err := agent.Run(ctx, cfg, prompt)

	// Emit done event.
	if d.eventLog != nil {
		kind := "agent.run.done"
		if err != nil {
			kind = "agent.run.error"
		}
		d.eventLog.Emit(signal.Event{
			Source: "director",
			Kind:   kind,
			Data:   map[string]string{"role": currentRole},
		})
	}

	return output, err
}

// --- Shell execution ---

// RunShell executes a shell command, routing through sandbox if needed.
func (d *UnifiedDirector) RunShell(ctx context.Context, cmd, workDir, currentRole string) (string, error) {
	if d.sandboxExec != nil && currentRole != "gensec" {
		stdout, stderr, err := d.sandboxExec(ctx, strings.Fields(cmd))
		output := stdout
		if stderr != "" {
			output += "\n" + stderr
		}
		return output, err
	}

	execCmd := exec.CommandContext(ctx, "bash", "-c", cmd) //nolint:gosec // operator-initiated shell commands
	execCmd.Dir = workDir
	out, err := execCmd.CombinedOutput()
	return string(out), err
}

// --- troupe/director.Director interface ---

// Direct implements troupe/director.Director for multi-agent orchestration.
// Emits lifecycle events to the channel. Currently delegates to Scheduler
// for role resolution (same as LocalDirector). Multi-agent orchestration
// will expand this when GOL-169 (Tier 4: Delegation) lands.
func (d *UnifiedDirector) Direct(ctx context.Context, _ troupe.Broker) (<-chan troupe.Event, error) {
	ch := make(chan troupe.Event, 1)

	// Emit started event.
	ch <- troupe.Event{Kind: troupe.Started, Step: "unified-director"}
	close(ch)

	return ch, nil
}

// --- Accessors ---

// Scheduler returns the underlying Scheduler for role lookup.
func (d *UnifiedDirector) Scheduler() Scheduler { return d.scheduler }

// --- Approval ---

// ApprovalForMode returns the approval function for the given mode.
// Auto mode auto-approves, agent mode blocks on channel, others deny.
func ApprovalForMode(mode agent.Mode, ch chan bool) agent.ApprovalFunc {
	switch mode {
	case agent.ModeAuto:
		return agent.AutoApprove
	case agent.ModeAgent:
		return func(_ driver.ToolCall) bool {
			return <-ch
		}
	default:
		return agent.DenyAll
	}
}

// resolveTools returns the effective tool executor (router or raw tools).
func (d *UnifiedDirector) resolveTools() builtin.ToolExecutor {
	if d.router != nil {
		return d.router
	}
	return d.tools
}
